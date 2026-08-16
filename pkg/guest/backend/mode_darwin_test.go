//go:build darwin

package backend_test

import (
	"strings"
	"testing"

	"github.com/yaacov/kc-utils/pkg/guest/backend"

	_ "github.com/yaacov/kc-utils/pkg/guest/plugins/direct"
	_ "github.com/yaacov/kc-utils/pkg/guest/plugins/guestfs"
	_ "github.com/yaacov/kc-utils/pkg/guest/plugins/qemu"
)

func TestDarwinAvailableBackends(t *testing.T) {
	names := backend.AvailableBackends()
	joined := strings.Join(names, ",")
	if !strings.Contains(joined, "qemu") {
		t.Fatalf("darwin should register qemu, got %v", names)
	}
	if strings.Contains(joined, "direct") || strings.Contains(joined, "guestfs") {
		t.Fatalf("darwin must not register linux-only backends, got %v", names)
	}
	if _, err := backend.ParseMode("direct"); err == nil {
		t.Fatal("ParseMode(direct) should fail on darwin")
	}
	if _, err := backend.ParseMode("qemu"); err != nil {
		t.Fatalf("ParseMode(qemu): %v", err)
	}
}
