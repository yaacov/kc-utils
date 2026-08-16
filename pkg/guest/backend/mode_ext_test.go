//go:build linux

package backend_test

import (
	"strings"
	"testing"

	"github.com/yaacov/kc-utils/pkg/guest/backend"

	_ "github.com/yaacov/kc-utils/pkg/guest/plugins/direct"
	_ "github.com/yaacov/kc-utils/pkg/guest/plugins/guestfs"
	_ "github.com/yaacov/kc-utils/pkg/guest/plugins/qemu"
)

func TestParseMode(t *testing.T) {
	m, err := backend.ParseMode("direct")
	if err != nil || m != backend.ModeDirect {
		t.Fatalf("ParseMode(direct) = %q, %v", m, err)
	}
	m, err = backend.ParseMode("guestfs")
	if err != nil || m != backend.ModeGuestfs {
		t.Fatalf("ParseMode(guestfs) = %q, %v", m, err)
	}
	m, err = backend.ParseMode("qemu")
	if err != nil || m != backend.ModeQemu {
		t.Fatalf("ParseMode(qemu) = %q, %v", m, err)
	}
	m, err = backend.ParseMode("")
	if err == nil {
		t.Fatalf("ParseMode(\"\") = %q, want error", m)
	}
	_, err = backend.ParseMode("nope")
	if err == nil || !strings.Contains(err.Error(), "available:") {
		t.Fatalf("ParseMode(nope) error = %v", err)
	}

	names := backend.AvailableBackends()
	joined := strings.Join(names, ",")
	if !strings.Contains(joined, "direct") || !strings.Contains(joined, "guestfs") || !strings.Contains(joined, "qemu") {
		t.Fatalf("missing backends: %v", names)
	}
}
