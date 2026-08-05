package copy

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
)

// grainDecompressor reuses a single zlib reader across grains to avoid
// per-grain allocations of the internal flate dictionary (~44 KiB each).
type grainDecompressor struct {
	br *bytes.Reader
	zr io.ReadCloser
}

func newGrainDecompressor() *grainDecompressor {
	return &grainDecompressor{br: bytes.NewReader(nil)}
}

func (d *grainDecompressor) decompress(compressed, buf []byte) ([]byte, error) {
	d.br.Reset(compressed)

	if d.zr == nil {
		zr, err := zlib.NewReader(d.br)
		if err != nil {
			return nil, err
		}
		d.zr = zr
	} else {
		if err := d.zr.(zlib.Resetter).Reset(d.br, nil); err != nil {
			return nil, err
		}
	}

	n, err := io.ReadFull(d.zr, buf)
	if err == io.ErrUnexpectedEOF {
		return buf[:n], nil
	}
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

// Stream-optimized VMDK format constants.
const (
	vmdkMagic      = 0x564d444b // "VMDK" (little-endian: 'K','D','M','V')
	sectorSize     = 512
	grainMarkerEOS = 0 // end-of-stream marker type
)

// sparseHeader field offsets within the 512-byte on-disk header.
const (
	hdrOffMagic    = 0
	hdrOffVersion  = 4
	hdrOffCapacity = 12
	hdrOffGrain    = 20
	hdrOffOverHead = 64
)

const grainMarkerSize = 12 // uint64 LBA + uint32 Size

// StreamToRaw reads a stream-optimized VMDK from r and writes the
// decompressed raw disk image to w. The writer must support Seek for
// sparse output (zero regions are skipped). onProgress is called with
// bytes written so far and total capacity; it may be nil.
func StreamToRaw(ctx context.Context, r io.Reader, w io.WriteSeeker, onProgress func(written, total int64)) error {
	var hdrBuf [sectorSize]byte
	if _, err := io.ReadFull(r, hdrBuf[:]); err != nil {
		return fmt.Errorf("vmdk: read header: %w", err)
	}

	magic := binary.LittleEndian.Uint32(hdrBuf[hdrOffMagic:])
	if magic != vmdkMagic {
		return fmt.Errorf("vmdk: bad magic 0x%08x (expected 0x%08x)", magic, vmdkMagic)
	}
	version := binary.LittleEndian.Uint32(hdrBuf[hdrOffVersion:])
	if version < 1 || version > 3 {
		return fmt.Errorf("vmdk: unsupported version %d", version)
	}

	capacity := binary.LittleEndian.Uint64(hdrBuf[hdrOffCapacity:])
	grainSizeSectors := binary.LittleEndian.Uint64(hdrBuf[hdrOffGrain:])
	overHead := binary.LittleEndian.Uint64(hdrBuf[hdrOffOverHead:])

	totalBytes := int64(capacity) * sectorSize
	grainBytes := int64(grainSizeSectors) * sectorSize
	if grainBytes == 0 {
		return fmt.Errorf("vmdk: grain size is zero")
	}

	slog.Debug("vmdk stream parameters",
		"capacity", totalBytes,
		"grainBytes", grainBytes,
		"overhead", overHead,
	)

	// Skip the rest of the overhead area (descriptor, etc.) to reach grain data.
	skipSectors := int64(overHead) - 1
	if skipSectors > 0 {
		if _, err := io.CopyN(io.Discard, r, skipSectors*sectorSize); err != nil {
			return fmt.Errorf("vmdk: skip overhead: %w", err)
		}
	}

	// compBuf starts at grainBytes: compressed grains are almost always
	// smaller than the original. Grows on demand up to compBufLimit.
	compBufLimit := int(grainBytes) * 2
	compBuf := make([]byte, grainBytes)
	grainBuf := make([]byte, grainBytes)
	var markerBuf [grainMarkerSize]byte
	var padBuf [sectorSize]byte
	decomp := newGrainDecompressor()
	var written int64

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		if _, err := io.ReadFull(r, markerBuf[:]); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			return fmt.Errorf("vmdk: read grain marker: %w", err)
		}
		lba := binary.LittleEndian.Uint64(markerBuf[0:8])
		size := binary.LittleEndian.Uint32(markerBuf[8:12])

		if size == 0 {
			// Metadata or EOS marker. Read the type field to decide.
			var typeBuf [4]byte
			if _, err := io.ReadFull(r, typeBuf[:]); err != nil {
				if err == io.EOF || err == io.ErrUnexpectedEOF {
					break
				}
				return fmt.Errorf("vmdk: read marker type: %w", err)
			}
			markerType := binary.LittleEndian.Uint32(typeBuf[:])
			// Type 0 = EOS, 1 = grain table, 2 = grain directory, 3 = footer.
			if markerType == grainMarkerEOS {
				break
			}
			// Metadata markers occupy a full sector. We've read 12 + 4 = 16
			// bytes so far; skip the remaining padding to the sector boundary.
			const metaMarkerHdr = grainMarkerSize + 4 // 16 bytes consumed
			markerPad := sectorSize - metaMarkerHdr
			if _, err := io.CopyN(io.Discard, r, int64(markerPad)); err != nil {
				return fmt.Errorf("vmdk: skip marker padding (type %d): %w", markerType, err)
			}
			// For grain table / grain directory / footer markers, LBA holds
			// the number of sectors of metadata that follow.
			metaBytes := int64(lba) * sectorSize
			slog.Debug("vmdk: skipping metadata marker",
				"type", markerType,
				"sectors", lba,
				"bytes", metaBytes,
			)
			if metaBytes > 0 {
				if _, err := io.CopyN(io.Discard, r, metaBytes); err != nil {
					return fmt.Errorf("vmdk: skip metadata (type %d): %w", markerType, err)
				}
			}
			continue
		}

		// Compressed grain data.
		dataSize := int(size)
		if dataSize > compBufLimit {
			return fmt.Errorf("vmdk: compressed grain at LBA %d claims %d bytes (limit %d)", lba, dataSize, compBufLimit)
		}
		if dataSize > len(compBuf) {
			compBuf = make([]byte, dataSize)
		}
		if _, err := io.ReadFull(r, compBuf[:dataSize]); err != nil {
			return fmt.Errorf("vmdk: read grain data at LBA %d: %w", lba, err)
		}

		// Pad to sector boundary (stream-optimized grains are sector-aligned).
		padBytes := (sectorSize - (grainMarkerSize+dataSize)%sectorSize) % sectorSize
		if padBytes > 0 {
			if _, err := io.ReadFull(r, padBuf[:padBytes]); err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
				return fmt.Errorf("vmdk: read grain padding at LBA %d: %w", lba, err)
			}
		}

		decompressed, err := decomp.decompress(compBuf[:dataSize], grainBuf)
		if err != nil {
			return fmt.Errorf("vmdk: decompress grain at LBA %d: %w", lba, err)
		}

		offset := int64(lba) * sectorSize
		if _, err := w.Seek(offset, io.SeekStart); err != nil {
			return fmt.Errorf("vmdk: seek to offset %d: %w", offset, err)
		}
		if _, err := w.Write(decompressed); err != nil {
			return fmt.Errorf("vmdk: write grain at LBA %d: %w", lba, err)
		}

		written += int64(len(decompressed))
		if onProgress != nil {
			onProgress(written, totalBytes)
		}
	}

	return nil
}
