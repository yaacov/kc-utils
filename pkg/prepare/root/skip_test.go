//go:build linux

package root

import "testing"

func TestSkipFSType(t *testing.T) {
	t.Run("skips crypto and bitlocker", func(t *testing.T) {
		for _, ft := range []string{"crypto_LUKS", "CRYPTO_LUKS", "crypto_luks", "BitLocker", "bitlocker"} {
			if !skipFSType(ft) {
				t.Fatalf("skipFSType(%q) = false, want true", ft)
			}
		}
	})
	t.Run("skips swap variants", func(t *testing.T) {
		for _, ft := range []string{"swap", "linux-swap", "linux-swap(v1)"} {
			if !skipFSType(ft) {
				t.Fatalf("skipFSType(%q) = false, want true", ft)
			}
		}
	})
	t.Run("skips empty and unknown", func(t *testing.T) {
		for _, ft := range []string{"", "unknown"} {
			if !skipFSType(ft) {
				t.Fatalf("skipFSType(%q) = false, want true", ft)
			}
		}
	})
	t.Run("allows real filesystems", func(t *testing.T) {
		for _, ft := range []string{"ext4", "xfs", "ntfs", "btrfs"} {
			if skipFSType(ft) {
				t.Fatalf("skipFSType(%q) = true, want false", ft)
			}
		}
	})
}
