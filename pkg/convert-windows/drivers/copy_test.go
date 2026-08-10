//go:build linux

package drivers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yaacov/kc-utils/pkg/convert-windows/driversource"
)

func TestCopyStagesOnlyListedFiles(t *testing.T) {
	src := t.TempDir()
	guest := t.TempDir()

	mustWrite := func(name, content string) string {
		t.Helper()
		p := filepath.Join(src, name)
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}

	inf := mustWrite("netkvm.inf", "inf")
	sys := mustWrite("netkvm.sys", "sys")
	exe := mustWrite("netkvmp.exe", "exe")
	mustWrite("noise.dll", "noise")

	names, err := Copy(guest, []driversource.DriverFile{{
		Name:    "netkvm",
		SrcPath: src,
		InfPath: inf,
		Files:   []string{inf, sys, exe},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 1 || names[0] != "netkvm" {
		t.Fatalf("names=%v", names)
	}

	virtio := filepath.Join(guest, "Windows", "Drivers", "VirtIO")
	for _, name := range []string{"netkvm.inf", "netkvm.sys", "netkvmp.exe"} {
		if _, err := os.Stat(filepath.Join(virtio, name)); err != nil {
			t.Fatalf("expected %s: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(virtio, "noise.dll")); err == nil {
		t.Fatal("noise.dll should not be staged")
	}
}

func TestCopyMirrorsBootCriticalSys(t *testing.T) {
	src := t.TempDir()
	guest := t.TempDir()

	inf := filepath.Join(src, "viostor.inf")
	sys := filepath.Join(src, "viostor.sys")
	if err := os.WriteFile(inf, []byte("inf"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sys, []byte("sys"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Copy(guest, []driversource.DriverFile{{
		Name:  "viostor",
		Files: []string{inf, sys},
	}}); err != nil {
		t.Fatal(err)
	}

	sysDst := filepath.Join(guest, "Windows", "System32", "drivers", "viostor.sys")
	if _, err := os.Stat(sysDst); err != nil {
		t.Fatalf("expected system32 mirror: %v", err)
	}
}

func TestCopySkipsEmptyFilesList(t *testing.T) {
	guest := t.TempDir()
	names, err := Copy(guest, []driversource.DriverFile{{Name: "broken"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 0 {
		t.Fatalf("names=%v", names)
	}
}

func TestCopyPreservesPackageRelativeSubdir(t *testing.T) {
	src := t.TempDir()
	guest := t.TempDir()

	inf := filepath.Join(src, "netkvm.inf")
	sys := filepath.Join(src, "bin", "netkvm.sys")
	if err := os.MkdirAll(filepath.Dir(sys), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inf, []byte("inf"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sys, []byte("sys"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Copy(guest, []driversource.DriverFile{{
		Name:    "netkvm",
		SrcPath: src,
		InfPath: inf,
		Files:   []string{inf, sys},
	}}); err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(guest, "Windows", "Drivers", "VirtIO", "bin", "netkvm.sys")
	if _, err := os.Stat(dst); err != nil {
		t.Fatalf("expected staged subdir file: %v", err)
	}
}
