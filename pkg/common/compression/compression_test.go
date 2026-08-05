package compression

import (
	"bytes"
	"testing"
)

func TestDetectFormatGzip(t *testing.T) {
	data := []byte{0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00}
	if got := DetectFormat(data); got != FormatGzip {
		t.Errorf("got %v, want FormatGzip", got)
	}
}

func TestDetectFormatXZ(t *testing.T) {
	data := []byte{0xfd, '7', 'z', 'X', 'Z', 0x00}
	if got := DetectFormat(data); got != FormatXZ {
		t.Errorf("got %v, want FormatXZ", got)
	}
}

func TestDetectFormatZstd(t *testing.T) {
	data := []byte{0x28, 0xb5, 0x2f, 0xfd, 0x00, 0x00}
	if got := DetectFormat(data); got != FormatZstd {
		t.Errorf("got %v, want FormatZstd", got)
	}
}

func TestDetectFormatLZ4(t *testing.T) {
	data := []byte{0x02, 0x21, 0x00, 0x00, 0x00, 0x00}
	if got := DetectFormat(data); got != FormatLZ4 {
		t.Errorf("got %v, want FormatLZ4", got)
	}
}

func TestDetectFormatUncompressed(t *testing.T) {
	data := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	if got := DetectFormat(data); got != FormatUncompressed {
		t.Errorf("got %v, want FormatUncompressed", got)
	}
}

func TestGzipRoundTrip(t *testing.T) {
	original := []byte("hello world, this is test data for compression")
	compressed, err := Compress(original, FormatGzip)
	if err != nil {
		t.Fatalf("compress: %v", err)
	}
	decompressed, err := Decompress(compressed, FormatGzip)
	if err != nil {
		t.Fatalf("decompress: %v", err)
	}
	if !bytes.Equal(original, decompressed) {
		t.Error("round-trip data mismatch")
	}
}

func TestXZRoundTrip(t *testing.T) {
	original := []byte("xz round-trip test data for compression verification")
	compressed, err := Compress(original, FormatXZ)
	if err != nil {
		t.Fatalf("compress: %v", err)
	}
	decompressed, err := Decompress(compressed, FormatXZ)
	if err != nil {
		t.Fatalf("decompress: %v", err)
	}
	if !bytes.Equal(original, decompressed) {
		t.Error("xz round-trip data mismatch")
	}
}

func TestZstdRoundTrip(t *testing.T) {
	original := []byte("zstd round-trip test data for compression verification")
	compressed, err := Compress(original, FormatZstd)
	if err != nil {
		t.Fatalf("compress: %v", err)
	}
	decompressed, err := Decompress(compressed, FormatZstd)
	if err != nil {
		t.Fatalf("decompress: %v", err)
	}
	if !bytes.Equal(original, decompressed) {
		t.Error("zstd round-trip data mismatch")
	}
}

func TestLZ4RoundTrip(t *testing.T) {
	original := []byte("lz4 round-trip test data for compression verification")
	compressed, err := Compress(original, FormatLZ4)
	if err != nil {
		t.Fatalf("compress: %v", err)
	}
	decompressed, err := Decompress(compressed, FormatLZ4)
	if err != nil {
		t.Fatalf("decompress: %v", err)
	}
	if !bytes.Equal(original, decompressed) {
		t.Error("lz4 round-trip data mismatch")
	}
}

func TestFormatString(t *testing.T) {
	tests := []struct {
		format Format
		want   string
	}{
		{FormatGzip, "gzip"},
		{FormatXZ, "xz"},
		{FormatZstd, "zstd"},
		{FormatLZ4, "lz4"},
		{FormatUncompressed, "uncompressed"},
	}
	for _, tt := range tests {
		if got := tt.format.String(); got != tt.want {
			t.Errorf("Format(%d).String() = %q, want %q", tt.format, got, tt.want)
		}
	}
}

func TestDetectFormatShortData(t *testing.T) {
	if got := DetectFormat([]byte{0x1f}); got != FormatUncompressed {
		t.Errorf("short data: got %v, want FormatUncompressed", got)
	}
}

func TestDetectFormatCPIO(t *testing.T) {
	data := []byte("0707deadbeef")
	if got := DetectFormat(data); got != FormatUncompressed {
		t.Errorf("CPIO magic: got %v, want FormatUncompressed", got)
	}
}

func TestUncompressedRoundTrip(t *testing.T) {
	original := []byte("uncompressed data pass-through")
	out, err := Compress(original, FormatUncompressed)
	if err != nil {
		t.Fatalf("compress: %v", err)
	}
	if !bytes.Equal(original, out) {
		t.Error("uncompressed compress should return copy")
	}
	dec, err := Decompress(out, FormatUncompressed)
	if err != nil {
		t.Fatalf("decompress: %v", err)
	}
	if !bytes.Equal(original, dec) {
		t.Error("uncompressed decompress should return copy")
	}
}

func TestDecompressUnknownFormat(t *testing.T) {
	_, err := Decompress([]byte("data"), Format(99))
	if err == nil {
		t.Error("expected error for unknown format")
	}
}

func TestCompressUnknownFormat(t *testing.T) {
	_, err := Compress([]byte("data"), Format(99))
	if err == nil {
		t.Error("expected error for unknown format")
	}
}

func TestGzipDetectAfterCompress(t *testing.T) {
	compressed, err := Compress([]byte("detect test"), FormatGzip)
	if err != nil {
		t.Fatalf("compress: %v", err)
	}
	if got := DetectFormat(compressed); got != FormatGzip {
		t.Errorf("DetectFormat after gzip compress = %v, want FormatGzip", got)
	}
}

func TestXZDetectAfterCompress(t *testing.T) {
	compressed, err := Compress([]byte("detect test"), FormatXZ)
	if err != nil {
		t.Fatalf("compress: %v", err)
	}
	if got := DetectFormat(compressed); got != FormatXZ {
		t.Errorf("DetectFormat after xz compress = %v, want FormatXZ", got)
	}
}

func TestZstdDetectAfterCompress(t *testing.T) {
	compressed, err := Compress([]byte("detect test"), FormatZstd)
	if err != nil {
		t.Fatalf("compress: %v", err)
	}
	if got := DetectFormat(compressed); got != FormatZstd {
		t.Errorf("DetectFormat after zstd compress = %v, want FormatZstd", got)
	}
}
