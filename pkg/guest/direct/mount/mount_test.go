//go:build linux

package mount

import "testing"

func TestMountOptions(t *testing.T) {
	tests := []struct {
		name     string
		fstype   string
		readOnly bool
		want     string
	}{
		{"rw default", "vfat", false, "nodev,nosuid,noexec"},
		{"ro default", "vfat", true, "nodev,nosuid,noexec,ro"},
		{"rw ext4", "ext4", false, "nodev,nosuid,noexec,norecovery"},
		{"ro xfs", "xfs", true, "nodev,nosuid,noexec,ro,norecovery"},
		{"ro ntfs3", "ntfs3", true, "nodev,nosuid,noexec,ro,force"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mountOptions(tt.fstype, tt.readOnly)
			if got != tt.want {
				t.Fatalf("mountOptions(%q, %v) = %q, want %q", tt.fstype, tt.readOnly, got, tt.want)
			}
		})
	}
}
