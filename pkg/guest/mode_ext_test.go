//go:build linux

package guest_test

import (
	"strings"
	"testing"

	"github.com/yaacov/kc-utils/pkg/guest"

	_ "github.com/yaacov/kc-utils/pkg/guest/direct"
	_ "github.com/yaacov/kc-utils/pkg/guest/guestfs"
)

func TestParseMode(t *testing.T) {
	m, err := guest.ParseMode("direct")
	if err != nil || m != guest.ModeDirect {
		t.Fatalf("ParseMode(direct) = %q, %v", m, err)
	}
	m, err = guest.ParseMode("guestfs")
	if err != nil || m != guest.ModeGuestfs {
		t.Fatalf("ParseMode(guestfs) = %q, %v", m, err)
	}
	m, err = guest.ParseMode("")
	if err != nil || m != guest.ModeDirect {
		t.Fatalf("ParseMode(\"\") = %q, %v", m, err)
	}
	_, err = guest.ParseMode("nope")
	if err == nil || !strings.Contains(err.Error(), "available:") {
		t.Fatalf("ParseMode(nope) error = %v", err)
	}

	names := guest.AvailableBackends()
	joined := strings.Join(names, ",")
	if !strings.Contains(joined, "direct") || !strings.Contains(joined, "guestfs") {
		t.Fatalf("missing backends: %v", names)
	}
}
