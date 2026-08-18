//go:build unix

package nicnaming_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/convert-linux/nicnaming"

	_ "github.com/yaacov/kc-utils/pkg/convert-linux/nicnaming/plugins/ifcfg"
	_ "github.com/yaacov/kc-utils/pkg/convert-linux/nicnaming/plugins/nm"
)

func TestApplyWithNMConnection(t *testing.T) {
	root := t.TempDir()

	connDir := filepath.Join(root, "etc", "NetworkManager", "system-connections")
	if err := os.MkdirAll(connDir, 0o755); err != nil {
		t.Fatal(err)
	}
	nmConn := "[connection]\ntype=ethernet\ninterface-name=ens192\n\n[ipv4]\nmethod=manual\naddress1=10.0.0.5/24,10.0.0.1\n"
	if err := os.WriteFile(filepath.Join(connDir, "ens192.nmconnection"), []byte(nmConn), 0o644); err != nil {
		t.Fatal(err)
	}

	staticIPs := []types.StaticIP{{MAC: "00:11:22:33:44:55", IP: "10.0.0.5"}}
	if err := nicnaming.Apply(root, staticIPs); err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	// Check udev rules
	rulesPath := filepath.Join(root, "etc", "udev", "rules.d", "70-persistent-net.rules")
	data, err := os.ReadFile(rulesPath)
	if err != nil {
		t.Fatalf("expected udev rules file: %v", err)
	}
	if !strings.Contains(string(data), "00:11:22:33:44:55") {
		t.Error("udev rules should contain MAC address")
	}
	if !strings.Contains(string(data), `NAME="ens192"`) {
		t.Error("udev rules should contain device name ens192")
	}

	// Check systemd .link file
	linkPath := filepath.Join(root, "etc", "systemd", "network", "10-v2v-ens192.link")
	linkData, err := os.ReadFile(linkPath)
	if err != nil {
		t.Fatalf("expected systemd .link file: %v", err)
	}
	if !strings.Contains(string(linkData), "MACAddress=00:11:22:33:44:55") {
		t.Error(".link file should contain MAC")
	}
	if !strings.Contains(string(linkData), "Name=ens192") {
		t.Error(".link file should contain interface name")
	}
}

func TestApplyNoStaticIPs(t *testing.T) {
	root := t.TempDir()
	if err := nicnaming.Apply(root, nil); err != nil {
		t.Fatalf("Apply with no IPs should succeed: %v", err)
	}
	rulesPath := filepath.Join(root, "etc", "udev", "rules.d", "70-persistent-net.rules")
	if _, err := os.Stat(rulesPath); !os.IsNotExist(err) {
		t.Error("should not create udev rules when no static IPs")
	}
}

func TestApplyWithIfcfg(t *testing.T) {
	root := t.TempDir()

	scriptsDir := filepath.Join(root, "etc", "sysconfig", "network-scripts")
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ifcfg := "DEVICE=eth0\nIPADDR=192.168.1.100\nNETMASK=255.255.255.0\n"
	if err := os.WriteFile(filepath.Join(scriptsDir, "ifcfg-eth0"), []byte(ifcfg), 0o644); err != nil {
		t.Fatal(err)
	}

	staticIPs := []types.StaticIP{{MAC: "aa:bb:cc:dd:ee:ff", IP: "192.168.1.100"}}
	if err := nicnaming.Apply(root, staticIPs); err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	rulesPath := filepath.Join(root, "etc", "udev", "rules.d", "70-persistent-net.rules")
	data, err := os.ReadFile(rulesPath)
	if err != nil {
		t.Fatalf("expected udev rules: %v", err)
	}
	if !strings.Contains(string(data), `NAME="eth0"`) {
		t.Errorf("expected eth0 in rules, got: %s", string(data))
	}
}
