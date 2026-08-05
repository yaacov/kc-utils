package overlay

import (
	"strings"
	"testing"
)

func TestOverlayFilename(t *testing.T) {
	got := overlayFilename("/dev/block0", 0)
	if !strings.HasPrefix(got, "overlay-0-") {
		t.Fatalf("got %q", got)
	}
}

func TestRestorePaths(t *testing.T) {
	disk := &Disk{BackingPath: "/dev/block0", Path: "/var/tmp/v2v/overlay-0-block0.qcow2"}
	restorePaths([]*Overlay{{Disk: disk, BackingPath: "/dev/block0"}})
	if disk.Path != "/dev/block0" {
		t.Fatalf("path = %q, want backing path", disk.Path)
	}
}
