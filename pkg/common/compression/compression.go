package compression

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"

	"github.com/klauspost/compress/zstd"
	"github.com/pierrec/lz4/v4"
	"github.com/ulikunitz/xz"
)

// Format identifies a compression format.
type Format int

const (
	FormatUncompressed Format = iota
	FormatGzip
	FormatXZ
	FormatZstd
	FormatLZ4
)

func (f Format) String() string {
	switch f {
	case FormatGzip:
		return "gzip"
	case FormatXZ:
		return "xz"
	case FormatZstd:
		return "zstd"
	case FormatLZ4:
		return "lz4"
	default:
		return "uncompressed"
	}
}

// DetectFormat identifies compression format from magic bytes.
func DetectFormat(data []byte) Format {
	if len(data) < 6 {
		return FormatUncompressed
	}
	switch {
	case data[0] == 0x1f && data[1] == 0x8b:
		return FormatGzip
	case data[0] == 0xfd && len(data) >= 6 && string(data[1:6]) == "7zXZ\x00":
		return FormatXZ
	case data[0] == 0x28 && data[1] == 0xb5 && data[2] == 0x2f && data[3] == 0xfd:
		return FormatZstd
	case data[0] == 0x02 && data[1] == 0x21:
		return FormatLZ4
	case len(data) >= 4 && string(data[0:4]) == "0707":
		return FormatUncompressed
	default:
		return FormatUncompressed
	}
}

// Decompress decompresses data in the given format.
// For FormatUncompressed, returns a copy of the input.
func Decompress(data []byte, format Format) ([]byte, error) {
	switch format {
	case FormatUncompressed:
		return append([]byte(nil), data...), nil
	case FormatGzip:
		return decompressGzip(data)
	case FormatXZ:
		return decompressXZ(data)
	case FormatZstd:
		return decompressZstd(data)
	case FormatLZ4:
		return decompressLZ4(data)
	default:
		return nil, fmt.Errorf("unknown compression format: %d", format)
	}
}

// Compress compresses data in the given format.
func Compress(data []byte, format Format) ([]byte, error) {
	switch format {
	case FormatUncompressed:
		return append([]byte(nil), data...), nil
	case FormatGzip:
		return compressGzip(data)
	case FormatXZ:
		return compressXZ(data)
	case FormatZstd:
		return compressZstd(data)
	case FormatLZ4:
		return compressLZ4(data)
	default:
		return nil, fmt.Errorf("unknown compression format: %d", format)
	}
}

func decompressGzip(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("gzip reader: %w", err)
	}
	defer r.Close()
	return io.ReadAll(r)
}

func compressGzip(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(data); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decompressXZ(data []byte) ([]byte, error) {
	r, err := xz.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("xz reader: %w", err)
	}
	return io.ReadAll(r)
}

func compressXZ(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w, err := xz.NewWriter(&buf)
	if err != nil {
		return nil, fmt.Errorf("xz writer: %w", err)
	}
	if _, err := w.Write(data); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decompressZstd(data []byte) ([]byte, error) {
	dec, err := zstd.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("zstd reader: %w", err)
	}
	defer dec.Close()
	return io.ReadAll(dec)
}

func compressZstd(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	enc, err := zstd.NewWriter(&buf)
	if err != nil {
		return nil, fmt.Errorf("zstd writer: %w", err)
	}
	if _, err := enc.Write(data); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decompressLZ4(data []byte) ([]byte, error) {
	r := lz4.NewReader(bytes.NewReader(data))
	return io.ReadAll(r)
}

func compressLZ4(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := lz4.NewWriter(&buf)
	if _, err := w.Write(data); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
