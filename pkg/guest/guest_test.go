//go:build unix

package guest

import (
	"testing"

	"github.com/yaacov/kc-utils/pkg/backend/plugins/direct"
)

func TestNormalizeGuestPath(t *testing.T) {
	if normalizeGuestPath("etc/fstab") != "/etc/fstab" {
		t.Fatalf("got %q", normalizeGuestPath("etc/fstab"))
	}
	if normalizeGuestPath("/etc/../etc/fstab") != "/etc/fstab" {
		t.Fatalf("got %q", normalizeGuestPath("/etc/../etc/fstab"))
	}
}

func TestDirectFSRoundTrip(t *testing.T) {
	dir := t.TempDir()
	b := direct.NewMounted(nil, dir, nil)
	if err := b.WriteFile("/etc/hostname", []byte("guest\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	data, err := b.ReadFile("/etc/hostname")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "guest\n" {
		t.Fatalf("got %q", data)
	}
	if !b.Exists("/etc/hostname") || !b.IsDir("/etc") {
		t.Fatal("exists/isdir")
	}
	entries, err := b.ReadDir("/etc")
	if err != nil || len(entries) != 1 || entries[0].Name != "hostname" {
		t.Fatalf("readdir: %#v err=%v", entries, err)
	}
}

func TestCheckoutDirect(t *testing.T) {
	dir := t.TempDir()
	g := &Guest{rootPath: dir, backendName: BackendDirect, backend: direct.NewMounted(nil, dir, nil)}
	if err := g.WriteFile("/Windows/System32/config/SYSTEM", []byte("hive"), 0o644); err != nil {
		t.Fatal(err)
	}
	host, err := g.Checkout("/Windows/System32/config/SYSTEM")
	if err != nil {
		t.Fatal(err)
	}
	if host != g.HostPath("/Windows/System32/config/SYSTEM") {
		t.Fatalf("direct checkout should be HostPath, got %s", host)
	}
	if err := g.Checkin("/Windows/System32/config/SYSTEM", host); err != nil {
		t.Fatal(err)
	}
}

func TestMkdirCreatesParents(t *testing.T) {
	dir := t.TempDir()
	g := &Guest{rootPath: dir, backendName: BackendDirect, backend: direct.NewMounted(nil, dir, nil)}
	if err := g.Mkdir("a/b/c", 0o755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	if !g.IsDir("/a/b/c") {
		t.Fatal("expected /a/b/c to exist as a directory")
	}
	if !g.IsDir("/a") || !g.IsDir("/a/b") {
		t.Fatal("expected intermediate parents to be created")
	}
}
