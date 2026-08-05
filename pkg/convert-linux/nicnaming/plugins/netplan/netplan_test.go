//go:build linux

package netplan

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yaacov/kc-utils/pkg/convert-linux/nicnaming"
)

func TestDetect(t *testing.T) {
	p := &Plugin{}
	root := t.TempDir()
	if p.Detect(root) {
		t.Error("Detect true without netplan dir")
	}
	if err := os.MkdirAll(filepath.Join(root, "etc", "netplan"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !p.Detect(root) {
		t.Error("Detect false with netplan dir")
	}
}

func TestResolveNames(t *testing.T) {
	p := &Plugin{}
	root := t.TempDir()
	netplanDir := filepath.Join(root, "etc", "netplan")
	if err := os.MkdirAll(netplanDir, 0o755); err != nil {
		t.Fatal(err)
	}
	yaml := `network:
  version: 2
  ethernets:
    ens192:
      addresses:
        - 10.0.0.5/24
      gateway4: 10.0.0.1
`
	if err := os.WriteFile(filepath.Join(netplanDir, "01-netcfg.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	rules, err := p.ResolveNames(root, []nicnaming.MacIPEntry{
		{MAC: "00:11:22:33:44:55", IP: "10.0.0.5"},
		{MAC: "aa:bb:cc:dd:ee:ff", IP: "192.168.1.10"},
	})
	if err != nil {
		t.Fatalf("ResolveNames: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("rules = %+v, want 1", rules)
	}
	if rules[0].MAC != "00:11:22:33:44:55" || rules[0].Device != "ens192" {
		t.Errorf("rule = %+v", rules[0])
	}
}

func TestResolveNamesNoFiles(t *testing.T) {
	p := &Plugin{}
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "etc", "netplan"), 0o755); err != nil {
		t.Fatal(err)
	}
	rules, err := p.ResolveNames(root, []nicnaming.MacIPEntry{{MAC: "00:11:22:33:44:55", IP: "10.0.0.5"}})
	if err != nil {
		t.Fatalf("ResolveNames: %v", err)
	}
	if rules != nil {
		t.Errorf("expected nil rules, got %+v", rules)
	}
}
