//go:build linux

package ntfsfix

import "testing"

func TestHeadsForSize(t *testing.T) {
	tests := []struct {
		name      string
		sizeBytes int64
		want      uint16
	}{
		{"below 2.0GiB", 2114445311, 0x40},
		{"at 2.0GiB boundary", 2114445312, 0x80},
		{"mid range", 3000000000, 0x80},
		{"at 3.9GiB boundary", 4228374780, 0xFF},
		{"large disk (10 GB)", 10 * 1024 * 1024 * 1024, 0xFF},
		{"zero size", 0, 0x40},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := headsForSize(tt.sizeBytes)
			if got != tt.want {
				t.Errorf("headsForSize(%d) = %d, want %d", tt.sizeBytes, got, tt.want)
			}
		})
	}
}
