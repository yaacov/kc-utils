//go:build darwin

package guest_test

import (
	"strings"
	"testing"

	"github.com/yaacov/kc-utils/pkg/guest"

	_ "github.com/yaacov/kc-utils/pkg/guest/direct"
	_ "github.com/yaacov/kc-utils/pkg/guest/guestfs"
	_ "github.com/yaacov/kc-utils/pkg/guest/qemu"
)

func TestDarwinAvailableBackends(t *testing.T) {
	names := guest.AvailableBackends()
	joined := strings.Join(names, ",")
	if !strings.Contains(joined, "qemu") {
		t.Fatalf("darwin should register qemu, got %v", names)
	}
	if strings.Contains(joined, "direct") || strings.Contains(joined, "guestfs") {
		t.Fatalf("darwin must not register linux-only backends, got %v", names)
	}
	if _, err := guest.ParseMode("direct"); err == nil {
		t.Fatal("ParseMode(direct) should fail on darwin")
	}
	if _, err := guest.ParseMode("qemu"); err != nil {
		t.Fatalf("ParseMode(qemu): %v", err)
	}
}
