//go:build unix

package kernel

import (
	"os"
	"path/filepath"
	"testing"
)

func mkModule(t *testing.T, root, version, subdir, name string) {
	t.Helper()
	dir := filepath.Join(root, "lib", "modules", version, "kernel", "drivers", subdir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), nil, 0o644); err != nil {
		t.Fatal(err)
	}
}

func mkUsrModule(t *testing.T, root, version, subdir, name string) {
	t.Helper()
	dir := filepath.Join(root, "usr", "lib", "modules", version, "kernel", "drivers", subdir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), nil, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestProbeModulesVirtio(t *testing.T) {
	root := t.TempDir()
	ver := "5.14.0-284"
	mkModule(t, root, ver, "virtio", "virtio_pci.ko")

	hasVirtio, isXenPV := ProbeModules(root, ver)
	if !hasVirtio {
		t.Error("expected HasVirtio=true when virtio_pci.ko exists")
	}
	if isXenPV {
		t.Error("expected IsXenPV=false when virtio_pci.ko exists")
	}
}

func TestProbeModulesVirtioCompressed(t *testing.T) {
	root := t.TempDir()
	ver := "6.1.0-18"
	mkModule(t, root, ver, "virtio", "virtio_pci.ko.xz")

	hasVirtio, isXenPV := ProbeModules(root, ver)
	if !hasVirtio {
		t.Error("expected HasVirtio=true for compressed virtio_pci.ko.xz")
	}
	if isXenPV {
		t.Error("expected IsXenPV=false")
	}
}

func TestProbeModulesXenPVOnly(t *testing.T) {
	root := t.TempDir()
	ver := "4.18.0-100"
	mkModule(t, root, ver, "block", "xen-blkfront.ko")

	hasVirtio, isXenPV := ProbeModules(root, ver)
	if hasVirtio {
		t.Error("expected HasVirtio=false when only xen-blkfront exists")
	}
	if !isXenPV {
		t.Error("expected IsXenPV=true when xen-blkfront exists without virtio_pci")
	}
}

func TestProbeModulesBothVirtioAndXen(t *testing.T) {
	root := t.TempDir()
	ver := "5.14.0-284"
	mkModule(t, root, ver, "virtio", "virtio_pci.ko")
	mkModule(t, root, ver, "block", "xen-blkfront.ko")

	hasVirtio, isXenPV := ProbeModules(root, ver)
	if !hasVirtio {
		t.Error("expected HasVirtio=true")
	}
	if isXenPV {
		t.Error("expected IsXenPV=false when both virtio_pci and xen-blkfront exist")
	}
}

func TestProbeModulesNoModules(t *testing.T) {
	root := t.TempDir()
	ver := "5.14.0-284"
	if err := os.MkdirAll(filepath.Join(root, "lib", "modules", ver), 0o755); err != nil {
		t.Fatal(err)
	}

	hasVirtio, isXenPV := ProbeModules(root, ver)
	if hasVirtio {
		t.Error("expected HasVirtio=false when no modules present")
	}
	if isXenPV {
		t.Error("expected IsXenPV=false when no modules present")
	}
}

func TestProbeModulesUsrLibModules(t *testing.T) {
	root := t.TempDir()
	ver := "5.14.0-284"
	mkUsrModule(t, root, ver, "virtio", "virtio_pci.ko")

	hasVirtio, isXenPV := ProbeModules(root, ver)
	if !hasVirtio {
		t.Error("expected HasVirtio=true when modules live under usr/lib/modules")
	}
	if isXenPV {
		t.Error("expected IsXenPV=false")
	}
}

func TestProbeModulesVirtioBlk(t *testing.T) {
	root := t.TempDir()
	ver := "5.14.0-421"
	mkModule(t, root, ver, "block", "virtio_blk.ko")

	hasVirtio, isXenPV := ProbeModules(root, ver)
	if !hasVirtio {
		t.Error("expected HasVirtio=true when virtio_blk.ko exists")
	}
	if isXenPV {
		t.Error("expected IsXenPV=false")
	}
}

func TestProbeModulesVirtioNet(t *testing.T) {
	root := t.TempDir()
	ver := "5.14.0-421"
	mkModule(t, root, ver, "net", "virtio_net.ko")

	hasVirtio, isXenPV := ProbeModules(root, ver)
	if !hasVirtio {
		t.Error("expected HasVirtio=true when virtio_net.ko exists in net/")
	}
	if isXenPV {
		t.Error("expected IsXenPV=false")
	}
}

func TestProbeModulesVirtioNetUnderVirtioDir(t *testing.T) {
	root := t.TempDir()
	ver := "6.1.0-100"
	mkModule(t, root, ver, "virtio", "virtio_net.ko.zst")

	hasVirtio, isXenPV := ProbeModules(root, ver)
	if !hasVirtio {
		t.Error("expected HasVirtio=true when virtio_net lives under virtio/ dir")
	}
	if isXenPV {
		t.Error("expected IsXenPV=false")
	}
}

func TestProbeModulesXenPVWithVirtioBlk(t *testing.T) {
	root := t.TempDir()
	ver := "5.14.0-284"
	mkModule(t, root, ver, "block", "xen-blkfront.ko")
	mkModule(t, root, ver, "block", "virtio_blk.ko")

	hasVirtio, isXenPV := ProbeModules(root, ver)
	if !hasVirtio {
		t.Error("expected HasVirtio=true")
	}
	if isXenPV {
		t.Error("expected IsXenPV=false when xen-blkfront AND virtio_blk both exist")
	}
}

func TestProbeModulesMissingDriversDir(t *testing.T) {
	root := t.TempDir()
	ver := "5.14.0-421"
	// Create only the version directory, no kernel/drivers tree.
	if err := os.MkdirAll(filepath.Join(root, "lib", "modules", ver), 0o755); err != nil {
		t.Fatal(err)
	}

	hasVirtio, isXenPV := ProbeModules(root, ver)
	if hasVirtio {
		t.Error("expected HasVirtio=false when kernel/drivers dir is missing")
	}
	if isXenPV {
		t.Error("expected IsXenPV=false when kernel/drivers dir is missing")
	}
}
