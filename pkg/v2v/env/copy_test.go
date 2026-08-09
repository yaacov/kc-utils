package env

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	kccopy "github.com/yaacov/kc-utils/pkg/copy"
)

func TestResolveCopySourcesFromEnv(t *testing.T) {
	cfg := &Config{
		DiskPath: "[ds] vm/a.vmdk,[ds] vm/b.vmdk",
	}
	disks, err := ResolveCopySources(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(disks) != 2 || disks[0] != "[ds] vm/a.vmdk" {
		t.Fatalf("got %v", disks)
	}
}

func TestResolveCopySourcesMissing(t *testing.T) {
	cfg := &Config{}
	if _, err := ResolveCopySources(cfg); err == nil {
		t.Fatal("expected error when no disk paths configured")
	}
}

func TestNeedsCopyFlagOnly(t *testing.T) {
	if !NeedsCopy(&Config{IsInPlace: false}) {
		t.Fatal("expected NeedsCopy true when IsInPlace=false (default copy)")
	}
	if NeedsCopy(&Config{IsInPlace: true}) {
		t.Fatal("expected NeedsCopy false when IsInPlace=true")
	}
	if !NeedsCopy(&Config{IsInPlace: false, Source: "ec2"}) {
		t.Fatal("expected NeedsCopy true for IsInPlace=false regardless of source")
	}
}

func setupCopyTestTargets(t *testing.T, empty bool) (dir string, restore func()) {
	t.Helper()
	dir = t.TempDir()
	mountDir := filepath.Join(dir, "disk0")
	if err := os.Mkdir(mountDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if !empty {
		img := filepath.Join(mountDir, "disk.img")
		data := make([]byte, copyEmptyThreshold()+1)
		data[0] = 1
		if err := os.WriteFile(img, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	restore = kccopy.SetTargetGlobs(filepath.Join(dir, "block*"), filepath.Join(dir, "disk*"))
	return dir, restore
}

// copyEmptyThreshold mirrors pkg/copy emptyThreshold (1 MiB) for test fixtures.
func copyEmptyThreshold() int { return 1 << 20 }

func TestValidateCopyModeCopyOK(t *testing.T) {
	_, restore := setupCopyTestTargets(t, true)
	defer restore()

	cfg := &Config{
		IsInPlace:  false,
		Source:     "vSphere",
		LibvirtURL: "vpx://user@vcenter/dc/host/esxi",
		VmName:     "my-vm",
	}
	if err := ValidateCopyMode(cfg); err != nil {
		t.Fatalf("expected OK: %v", err)
	}
}

func TestValidateCopyModeCopyPopulatedFails(t *testing.T) {
	_, restore := setupCopyTestTargets(t, false)
	defer restore()

	cfg := &Config{
		IsInPlace:  false,
		Source:     "vSphere",
		LibvirtURL: "vpx://user@vcenter/dc/host/esxi",
		VmName:     "my-vm",
	}
	err := ValidateCopyMode(cfg)
	if err == nil || !strings.Contains(err.Error(), "already populated") {
		t.Fatalf("expected populated mismatch error, got %v", err)
	}
}

func TestValidateCopyModeInPlaceOK(t *testing.T) {
	_, restore := setupCopyTestTargets(t, false)
	defer restore()

	cfg := &Config{IsInPlace: true}
	if err := ValidateCopyMode(cfg); err != nil {
		t.Fatalf("expected OK: %v", err)
	}
}

func TestValidateCopyModeInPlaceEmptyFails(t *testing.T) {
	_, restore := setupCopyTestTargets(t, true)
	defer restore()

	cfg := &Config{IsInPlace: true}
	err := ValidateCopyMode(cfg)
	if err == nil || !strings.Contains(err.Error(), "PVC targets are empty") {
		t.Fatalf("expected empty mismatch error, got %v", err)
	}
}

func TestValidateCopyModeCopyWrongSource(t *testing.T) {
	_, restore := setupCopyTestTargets(t, true)
	defer restore()

	cfg := &Config{
		IsInPlace:  false,
		Source:     "ec2",
		LibvirtURL: "vpx://user@vcenter/dc/host/esxi",
		VmName:     "my-vm",
	}
	err := ValidateCopyMode(cfg)
	if err == nil || !strings.Contains(err.Error(), "vSphere") {
		t.Fatalf("expected vSphere source error, got %v", err)
	}
}

func TestValidateCopyModeNoTargets(t *testing.T) {
	dir := t.TempDir()
	restore := kccopy.SetTargetGlobs(filepath.Join(dir, "block*"), filepath.Join(dir, "disk*"))
	defer restore()

	err := ValidateCopyMode(&Config{IsInPlace: true})
	if err == nil || !strings.Contains(err.Error(), "no PVC targets") {
		t.Fatalf("expected no targets error, got %v", err)
	}
}

func TestValidateCopySourceCountOK(t *testing.T) {
	_, restore := setupCopyTestTargets(t, true)
	defer restore()

	if err := ValidateCopySourceCount([]string{"[ds] vm/a.vmdk"}); err != nil {
		t.Fatalf("expected OK: %v", err)
	}
}

func TestValidateCopySourceCountMismatch(t *testing.T) {
	_, restore := setupCopyTestTargets(t, true)
	defer restore()

	err := ValidateCopySourceCount([]string{"[ds] vm/a.vmdk", "[ds] vm/b.vmdk"})
	if err == nil || !strings.Contains(err.Error(), "disk count mismatch") {
		t.Fatalf("expected count mismatch error, got %v", err)
	}
}

func TestBuildCopyInputKeepsSourceDisks(t *testing.T) {
	cfg := &Config{
		LibvirtURL:      "vpx://user@vcenter/dc/host/esxi",
		VmName:          "my-vm",
		Fingerprint:     "aa:bb",
		Workdir:         "/tmp/work",
		CopyConcurrency: 2,
		Source:          "ec2", // skip inventory lookup
	}
	sources := []string{"[ds] vm/a.vmdk"}
	in := BuildCopyInput(cfg, sources)
	if len(in.SourceDisks) != 1 || in.SourceDisks[0] != sources[0] {
		t.Fatalf("SourceDisks = %v, want debug metadata retained", in.SourceDisks)
	}
	if in.Host != "vcenter" || in.VMName != cfg.VmName {
		t.Fatalf("unexpected input: %+v", in)
	}
	if in.Datacenter != "dc" {
		t.Fatalf("Datacenter = %q, want %q", in.Datacenter, "dc")
	}
	if in.Insecure {
		t.Fatal("Insecure should be false for URL without no_verify")
	}
	if in.CaBundle != DefaultCaBundle {
		t.Fatalf("CaBundle = %q, want %q", in.CaBundle, DefaultCaBundle)
	}
}

func TestBuildCopyInputJSONIncludesTLSFields(t *testing.T) {
	cfg := &Config{
		LibvirtURL:  "vpx://user@vcenter/dc/host/esxi?no_verify=1",
		VmName:      "my-vm",
		Fingerprint: "aa:bb",
		CaBundle:    DefaultCaBundle,
	}
	in := BuildCopyInput(cfg, []string{"[ds] vm/a.vmdk"})
	data, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if raw["insecure"] != true {
		t.Fatalf("insecure = %v, want true in JSON", raw["insecure"])
	}
	if raw["ca_bundle"] != DefaultCaBundle {
		t.Fatalf("ca_bundle = %v, want %q in JSON", raw["ca_bundle"], DefaultCaBundle)
	}
}

func TestBuildCopyInputCaBundleFromURL(t *testing.T) {
	cfg := &Config{
		LibvirtURL:  "vpx://user@vcenter/dc/host/esxi?cacert=/opt/ca-bundle.crt",
		VmName:      "my-vm",
		Fingerprint: "aa:bb",
		CaBundle:    "/should/not/use",
	}
	in := BuildCopyInput(cfg, []string{"[ds] vm/a.vmdk"})
	if in.CaBundle != "/opt/ca-bundle.crt" {
		t.Fatalf("CaBundle = %q, want from libvirt URL", in.CaBundle)
	}
}

func TestBuildCopyInputCaBundleFromConfig(t *testing.T) {
	cfg := &Config{
		LibvirtURL:  "vpx://user@vcenter/dc/host/esxi",
		VmName:      "my-vm",
		Fingerprint: "aa:bb",
		CaBundle:    "/custom/bundle.pem",
	}
	in := BuildCopyInput(cfg, []string{"[ds] vm/a.vmdk"})
	if in.CaBundle != "/custom/bundle.pem" {
		t.Fatalf("CaBundle = %q, want cfg default", in.CaBundle)
	}
}

func TestParseLibvirtURL(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		host       string
		datacenter string
		insecure   bool
		caBundle   string
	}{
		{
			name:       "standard vCenter URL",
			url:        "vpx://user@vcenter/dc/host/esxi",
			host:       "vcenter",
			datacenter: "dc",
			insecure:   false,
		},
		{
			name:       "with no_verify",
			url:        "vpx://user@vcenter/dc/host/esxi?no_verify=1",
			host:       "vcenter",
			datacenter: "dc",
			insecure:   true,
			caBundle:   "",
		},
		{
			name:       "with cacert",
			url:        "vpx://user@vcenter/dc/host/esxi?cacert=/opt/ca-bundle.crt",
			host:       "vcenter",
			datacenter: "dc",
			insecure:   false,
			caBundle:   "/opt/ca-bundle.crt",
		},
		{
			name:       "with port",
			url:        "vpx://user@vcenter:8443/dc/host/esxi",
			host:       "vcenter:8443",
			datacenter: "dc",
			insecure:   false,
		},
		{
			name:       "ESXi direct no path",
			url:        "vpx://user@esxi-host",
			host:       "esxi-host",
			datacenter: "",
			insecure:   false,
		},
		{
			name:       "empty URL",
			url:        "",
			host:       "",
			datacenter: "",
			insecure:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, datacenter, insecure, caBundle := parseLibvirtURL(tt.url)
			if host != tt.host {
				t.Errorf("host = %q, want %q", host, tt.host)
			}
			if datacenter != tt.datacenter {
				t.Errorf("datacenter = %q, want %q", datacenter, tt.datacenter)
			}
			if insecure != tt.insecure {
				t.Errorf("insecure = %v, want %v", insecure, tt.insecure)
			}
			if caBundle != tt.caBundle {
				t.Errorf("caBundle = %q, want %q", caBundle, tt.caBundle)
			}
		})
	}
}
