package directory

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yaacov/kc-utils/pkg/convert-linux/guestagent"
)

func writePkg(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFindPackagesELExact(t *testing.T) {
	base := t.TempDir()
	writePkg(t, filepath.Join(base, "rpm", "el8", "x86_64"), "qemu-guest-agent-8.0-1.el8.x86_64.rpm")
	writePkg(t, filepath.Join(base, "rpm", "el9", "x86_64"), "qemu-guest-agent-9.1.0-1.el9.x86_64.rpm")
	writePkg(t, filepath.Join(base, "rpm", "el10", "x86_64"), "qemu-guest-agent-10.0-1.el10.x86_64.rpm")

	src := &DirectorySource{BasePath: base}
	pkgs, err := src.FindPackages(guestagent.FindRequest{
		Name: "qemu-guest-agent", Format: "rpm", Arch: "x86_64",
		Distro: "rhel", MajorVersion: 9,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("got %d packages, want 1", len(pkgs))
	}
	if pkgs[0].FileName != "qemu-guest-agent-9.1.0-1.el9.x86_64.rpm" {
		t.Errorf("got %q", pkgs[0].FileName)
	}
	if pkgs[0].ELTag != "el9" {
		t.Errorf("ELTag = %q, want el9", pkgs[0].ELTag)
	}
}

func TestFindPackagesELFallbackLower(t *testing.T) {
	base := t.TempDir()
	writePkg(t, filepath.Join(base, "rpm", "el8", "x86_64"), "qemu-guest-agent-8.0-1.el8.x86_64.rpm")
	// el9 missing — el10 guest with only el8 should fall back to el8
	writePkg(t, filepath.Join(base, "rpm", "el10", "x86_64"), "qemu-guest-agent-10.0-1.el10.x86_64.rpm")

	src := &DirectorySource{BasePath: base}
	pkgs, err := src.FindPackages(guestagent.FindRequest{
		Name: "qemu-guest-agent", Format: "rpm", Arch: "x86_64",
		Distro: "centos", MajorVersion: 9,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("got %d packages, want 1", len(pkgs))
	}
	if pkgs[0].ELTag != "el8" {
		t.Errorf("ELTag = %q, want el8 (nearest lower)", pkgs[0].ELTag)
	}
}

func TestFindPackagesELEmpty(t *testing.T) {
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, "rpm"), 0o755); err != nil {
		t.Fatal(err)
	}
	src := &DirectorySource{BasePath: base}
	pkgs, err := src.FindPackages(guestagent.FindRequest{
		Name: "qemu-guest-agent", Format: "rpm", Arch: "x86_64",
		Distro: "rhel", MajorVersion: 9,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 0 {
		t.Fatalf("got %d packages, want 0", len(pkgs))
	}
}

func TestFindPackagesLegacyFlatRPM(t *testing.T) {
	base := t.TempDir()
	writePkg(t, filepath.Join(base, "rpm", "x86_64"), "qemu-guest-agent-9.1.0-1.el9.x86_64.rpm")
	writePkg(t, filepath.Join(base, "rpm", "x86_64"), "unrelated-1.0.x86_64.rpm")

	src := &DirectorySource{BasePath: base}
	pkgs, err := src.FindPackages(guestagent.FindRequest{
		Name: "qemu-guest-agent", Format: "rpm", Arch: "x86_64",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("got %d packages, want 1", len(pkgs))
	}
	if pkgs[0].FileName != "qemu-guest-agent-9.1.0-1.el9.x86_64.rpm" {
		t.Errorf("got %q", pkgs[0].FileName)
	}
}

func TestFindPackagesDEB(t *testing.T) {
	base := t.TempDir()
	writePkg(t, filepath.Join(base, "deb", "amd64"), "qemu-guest-agent_9.1.0-1_amd64.deb")

	src := &DirectorySource{BasePath: base}
	pkgs, err := src.FindPackages(guestagent.FindRequest{
		Name: "qemu-guest-agent", Format: "deb", Arch: "x86_64",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("got %d packages, want 1 (arch mapping x86_64->amd64)", len(pkgs))
	}
}

func TestFindPackagesNoarch(t *testing.T) {
	base := t.TempDir()
	writePkg(t, filepath.Join(base, "rpm", "noarch"), "cloud-init-23.4-1.el9.noarch.rpm")

	src := &DirectorySource{BasePath: base}
	pkgs, err := src.FindPackages(guestagent.FindRequest{
		Name: "cloud-init", Format: "rpm", Arch: "x86_64",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("got %d packages, want 1 (noarch)", len(pkgs))
	}
}

func TestFindPackagesNotAvailable(t *testing.T) {
	src := &DirectorySource{BasePath: "/nonexistent/path"}
	if src.Available() {
		t.Error("should not be available")
	}
	pkgs, err := src.FindPackages(guestagent.FindRequest{
		Name: "anything", Format: "rpm", Arch: "x86_64",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 0 {
		t.Errorf("got %d packages from nonexistent path", len(pkgs))
	}
}

func TestELMajorCandidates(t *testing.T) {
	supported := []int{10, 9, 8}
	got := elMajorCandidates(9, supported)
	want := []int{9, 8}
	if len(got) != len(want) || got[0] != 9 || got[1] != 8 {
		t.Errorf("elMajorCandidates(9) = %v, want %v", got, want)
	}
	got = elMajorCandidates(10, supported)
	if got[0] != 10 || got[1] != 9 || got[2] != 8 {
		t.Errorf("elMajorCandidates(10) = %v", got)
	}
}

func TestFindPackagesAarch64RPM(t *testing.T) {
	base := t.TempDir()
	writePkg(t, filepath.Join(base, "rpm", "el9", "aarch64"), "qemu-guest-agent-9.1.0-1.el9.aarch64.rpm")

	src := &DirectorySource{BasePath: base}
	pkgs, err := src.FindPackages(guestagent.FindRequest{
		Name: "qemu-guest-agent", Format: "rpm", Arch: "aarch64",
		Distro: "rhel", MajorVersion: 9,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("got %d packages, want 1", len(pkgs))
	}
	if pkgs[0].FileName != "qemu-guest-agent-9.1.0-1.el9.aarch64.rpm" {
		t.Errorf("got %q", pkgs[0].FileName)
	}
}

func TestFindPackagesAarch64DEB(t *testing.T) {
	base := t.TempDir()
	writePkg(t, filepath.Join(base, "deb", "arm64"), "qemu-guest-agent_9.1.0-1_arm64.deb")

	src := &DirectorySource{BasePath: base}
	pkgs, err := src.FindPackages(guestagent.FindRequest{
		Name: "qemu-guest-agent", Format: "deb", Arch: "aarch64",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("got %d packages, want 1 (arch mapping aarch64->arm64)", len(pkgs))
	}
}

func TestAvailableELMajorsScansDirectory(t *testing.T) {
	base := t.TempDir()
	for _, dir := range []string{"el8", "el9", "el10", "el11"} {
		if err := os.MkdirAll(filepath.Join(base, "rpm", dir, "x86_64"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	src := &DirectorySource{BasePath: base}
	got := src.availableELMajors()
	if len(got) != 4 || got[0] != 11 || got[1] != 10 || got[2] != 9 || got[3] != 8 {
		t.Errorf("availableELMajors() = %v, want [11 10 9 8]", got)
	}
}

func TestArchVariants(t *testing.T) {
	if v := archVariants("deb", "x86_64"); len(v) != 1 || v[0] != "amd64" {
		t.Errorf("deb x86_64 -> %v, want [amd64]", v)
	}
	if v := archVariants("deb", "aarch64"); len(v) != 1 || v[0] != "arm64" {
		t.Errorf("deb aarch64 -> %v, want [arm64]", v)
	}
	if v := archVariants("rpm", "x86_64"); len(v) != 1 || v[0] != "x86_64" {
		t.Errorf("rpm x86_64 -> %v, want [x86_64]", v)
	}
}

func TestFindPackagesAmazonLinuxMappedEL9(t *testing.T) {
	base := t.TempDir()
	writePkg(t, filepath.Join(base, "rpm", "el9", "x86_64"), "qemu-guest-agent-9.1.0-1.el9.x86_64.rpm")
	writePkg(t, filepath.Join(base, "rpm", "el10", "x86_64"), "qemu-guest-agent-10.0-1.el10.x86_64.rpm")

	src := &DirectorySource{BasePath: base}
	pkgs, err := src.FindPackages(guestagent.FindRequest{
		Name: "qemu-guest-agent", Format: "rpm", Arch: "x86_64",
		Distro: "amzn", MajorVersion: 9,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("got %d packages, want 1", len(pkgs))
	}
	if pkgs[0].ELTag != "el9" {
		t.Errorf("ELTag = %q, want el9 (not el10 for mapped AL major)", pkgs[0].ELTag)
	}
}
