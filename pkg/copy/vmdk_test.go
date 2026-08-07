package copy

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/binary"
	"io"
	"testing"
)

// buildTestVMDK creates a minimal stream-optimized VMDK with one compressed grain.
func buildTestVMDK(t *testing.T, grainSizeSectors uint64, lba uint64, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer

	// Write 512-byte header manually.
	var hdr [sectorSize]byte
	binary.LittleEndian.PutUint32(hdr[hdrOffMagic:], vmdkMagic)
	binary.LittleEndian.PutUint32(hdr[hdrOffVersion:], 1)
	binary.LittleEndian.PutUint64(hdr[hdrOffCapacity:], grainSizeSectors*4)
	binary.LittleEndian.PutUint64(hdr[hdrOffGrain:], grainSizeSectors)
	binary.LittleEndian.PutUint64(hdr[hdrOffOverHead:], 1) // 1 sector overhead (header only)
	buf.Write(hdr[:])

	// Compress the grain data.
	var compressed bytes.Buffer
	zw := zlib.NewWriter(&compressed)
	if _, err := zw.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	// Write grain marker + compressed data.
	var marker [grainMarkerSize]byte
	binary.LittleEndian.PutUint64(marker[0:8], lba)
	binary.LittleEndian.PutUint32(marker[8:12], uint32(compressed.Len()))
	buf.Write(marker[:])
	buf.Write(compressed.Bytes())

	// Pad to sector boundary.
	markerAndData := grainMarkerSize + compressed.Len()
	padBytes := (sectorSize - markerAndData%sectorSize) % sectorSize
	buf.Write(make([]byte, padBytes))

	// Write EOS marker (lba=0, size=0) + type=0.
	var eos [grainMarkerSize + 4]byte // 12 + 4 = 16
	buf.Write(eos[:])

	return buf.Bytes()
}

func TestStreamToRaw(t *testing.T) {
	grainSize := uint64(2) // 2 sectors = 1024 bytes
	grainBytes := int(grainSize) * sectorSize
	payload := make([]byte, grainBytes)
	for i := range payload {
		payload[i] = byte(i % 256)
	}

	vmdk := buildTestVMDK(t, grainSize, 0, payload)

	var out seekBuffer
	if err := StreamToRaw(context.Background(), bytes.NewReader(vmdk), &out, nil); err != nil {
		t.Fatalf("StreamToRaw: %v", err)
	}

	if len(out.buf) < grainBytes {
		t.Fatalf("output too short: %d < %d", len(out.buf), grainBytes)
	}
	for i := 0; i < grainBytes; i++ {
		if out.buf[i] != payload[i] {
			t.Fatalf("mismatch at byte %d: got %d want %d", i, out.buf[i], payload[i])
		}
	}
}

func TestStreamToRawAtOffset(t *testing.T) {
	grainSize := uint64(2)
	grainBytes := int(grainSize) * sectorSize
	payload := bytes.Repeat([]byte{0xAB}, grainBytes)
	lba := uint64(4) // offset = 4 * 512 = 2048

	vmdk := buildTestVMDK(t, grainSize, lba, payload)

	var out seekBuffer
	if err := StreamToRaw(context.Background(), bytes.NewReader(vmdk), &out, nil); err != nil {
		t.Fatalf("StreamToRaw: %v", err)
	}

	expectedOffset := int(lba) * sectorSize
	if len(out.buf) < expectedOffset+grainBytes {
		t.Fatalf("output too short: %d < %d", len(out.buf), expectedOffset+grainBytes)
	}
	for i := 0; i < expectedOffset; i++ {
		if out.buf[i] != 0 {
			t.Fatalf("expected zero at offset %d, got %d", i, out.buf[i])
		}
	}
	for i := 0; i < grainBytes; i++ {
		if out.buf[expectedOffset+i] != 0xAB {
			t.Fatalf("mismatch at data offset %d", i)
		}
	}
}

func TestStreamToRawWithMetadataMarkers(t *testing.T) {
	grainSize := uint64(2) // 2 sectors = 1024 bytes
	grainBytes := int(grainSize) * sectorSize

	payload1 := bytes.Repeat([]byte{0xAA}, grainBytes)
	payload2 := bytes.Repeat([]byte{0xBB}, grainBytes)
	lba1 := uint64(0)
	lba2 := uint64(grainSize) // immediately after first grain

	// Capacity must cover both grains.
	capacity := grainSize * 4

	var buf bytes.Buffer

	// Header (1 sector).
	var hdr [sectorSize]byte
	binary.LittleEndian.PutUint32(hdr[hdrOffMagic:], vmdkMagic)
	binary.LittleEndian.PutUint32(hdr[hdrOffVersion:], 1)
	binary.LittleEndian.PutUint64(hdr[hdrOffCapacity:], capacity)
	binary.LittleEndian.PutUint64(hdr[hdrOffGrain:], grainSize)
	binary.LittleEndian.PutUint64(hdr[hdrOffOverHead:], 1)
	buf.Write(hdr[:])

	// Grain 1: marker + compressed data + padding.
	writeGrain := func(lba uint64, data []byte) {
		var compressed bytes.Buffer
		zw := zlib.NewWriter(&compressed)
		if _, err := zw.Write(data); err != nil {
			t.Fatal(err)
		}
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}
		var marker [grainMarkerSize]byte
		binary.LittleEndian.PutUint64(marker[0:8], lba)
		binary.LittleEndian.PutUint32(marker[8:12], uint32(compressed.Len()))
		buf.Write(marker[:])
		buf.Write(compressed.Bytes())
		total := grainMarkerSize + compressed.Len()
		pad := (sectorSize - total%sectorSize) % sectorSize
		buf.Write(make([]byte, pad))
	}

	writeGrain(lba1, payload1)

	// Grain table metadata marker (type 1): occupies a full sector header
	// + 2 sectors of fake metadata payload.
	const gtMetaSectors = 2
	var metaMarker [sectorSize]byte // full sector, zero-padded
	binary.LittleEndian.PutUint64(metaMarker[0:8], gtMetaSectors)
	binary.LittleEndian.PutUint32(metaMarker[8:12], 0)  // size=0 → metadata
	binary.LittleEndian.PutUint32(metaMarker[12:16], 1) // type=1 (grain table)

	buf.Write(metaMarker[:])
	buf.Write(make([]byte, gtMetaSectors*sectorSize)) // dummy GT payload

	writeGrain(lba2, payload2)

	// EOS marker.
	var eos [grainMarkerSize + 4]byte
	buf.Write(eos[:])

	// Parse and verify both grains survive the metadata marker.
	var out seekBuffer
	if err := StreamToRaw(context.Background(), bytes.NewReader(buf.Bytes()), &out, nil); err != nil {
		t.Fatalf("StreamToRaw: %v", err)
	}

	off1 := int(lba1) * sectorSize
	off2 := int(lba2) * sectorSize
	if len(out.buf) < off2+grainBytes {
		t.Fatalf("output too short: %d < %d", len(out.buf), off2+grainBytes)
	}
	for i := 0; i < grainBytes; i++ {
		if out.buf[off1+i] != 0xAA {
			t.Fatalf("grain1 mismatch at offset %d: got 0x%02x want 0xAA", i, out.buf[off1+i])
		}
	}
	for i := 0; i < grainBytes; i++ {
		if out.buf[off2+i] != 0xBB {
			t.Fatalf("grain2 mismatch at offset %d: got 0x%02x want 0xBB", i, out.buf[off2+i])
		}
	}
}

func TestStreamToRawBadMagic(t *testing.T) {
	vmdk := make([]byte, sectorSize)
	binary.LittleEndian.PutUint32(vmdk[0:4], 0xDEADBEEF)
	err := StreamToRaw(context.Background(), bytes.NewReader(vmdk), &seekBuffer{}, nil)
	if err == nil {
		t.Fatal("expected error for bad magic")
	}
}

func TestStreamToRawProgress(t *testing.T) {
	grainSize := uint64(2)
	grainBytes := int(grainSize) * sectorSize
	payload := make([]byte, grainBytes)
	vmdk := buildTestVMDK(t, grainSize, 0, payload)

	var progressCalls int
	var lastWritten, lastTotal int64
	err := StreamToRaw(context.Background(), bytes.NewReader(vmdk), &seekBuffer{}, func(written, total int64) {
		progressCalls++
		lastWritten = written
		lastTotal = total
	})
	if err != nil {
		t.Fatal(err)
	}
	if progressCalls == 0 {
		t.Fatal("expected at least one progress call")
	}
	if lastWritten != int64(grainBytes) {
		t.Fatalf("last written=%d want %d", lastWritten, grainBytes)
	}
	if lastTotal != int64(grainSize*4)*sectorSize {
		t.Fatalf("total=%d want %d", lastTotal, int64(grainSize*4)*sectorSize)
	}
}

func TestStreamToRawRejectsHugeGrainSize(t *testing.T) {
	var hdr [sectorSize]byte
	binary.LittleEndian.PutUint32(hdr[hdrOffMagic:], vmdkMagic)
	binary.LittleEndian.PutUint32(hdr[hdrOffVersion:], 1)
	binary.LittleEndian.PutUint64(hdr[hdrOffCapacity:], 8)
	// Larger than maxGrainBytes / sectorSize — must fail before buffer alloc.
	binary.LittleEndian.PutUint64(hdr[hdrOffGrain:], uint64(maxGrainBytes/sectorSize)+1)
	binary.LittleEndian.PutUint64(hdr[hdrOffOverHead:], 1)

	err := StreamToRaw(context.Background(), bytes.NewReader(hdr[:]), &seekBuffer{}, nil)
	if err == nil {
		t.Fatal("expected error for oversized grain")
	}
}

func TestStreamToRawRejectsGrainBeyondCapacity(t *testing.T) {
	grainSize := uint64(2)
	grainBytes := int(grainSize) * sectorSize
	payload := bytes.Repeat([]byte{0xCD}, grainBytes)
	// Capacity is grainSize*4 sectors; LBA past that must be rejected.
	lba := grainSize * 4
	vmdk := buildTestVMDK(t, grainSize, lba, payload)

	err := StreamToRaw(context.Background(), bytes.NewReader(vmdk), &seekBuffer{}, nil)
	if err == nil {
		t.Fatal("expected error for grain beyond capacity")
	}
}

// seekBuffer is an in-memory io.WriteSeeker for testing.
type seekBuffer struct {
	buf []byte
	pos int64
}

func (s *seekBuffer) Write(p []byte) (int, error) {
	end := int(s.pos) + len(p)
	if end > len(s.buf) {
		s.buf = append(s.buf, make([]byte, end-len(s.buf))...)
	}
	copy(s.buf[s.pos:], p)
	s.pos += int64(len(p))
	return len(p), nil
}

func (s *seekBuffer) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		s.pos = offset
	case io.SeekCurrent:
		s.pos += offset
	case io.SeekEnd:
		s.pos = int64(len(s.buf)) + offset
	}
	if s.pos < 0 {
		s.pos = 0
	}
	if int(s.pos) > len(s.buf) {
		s.buf = append(s.buf, make([]byte, int(s.pos)-len(s.buf))...)
	}
	return s.pos, nil
}
