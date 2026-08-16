//go:build linux

package networkd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/convert-linux/network/networkd"
)

func TestDetectAmazonLinux2023OSRelease(t *testing.T) {
	root := t.TempDir()
	usrLib := filepath.Join(root, "usr", "lib")
	if err := os.MkdirAll(usrLib, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(usrLib, "os-release"),
		[]byte("ID=amzn\nVERSION_ID=2023\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !networkd.Detect(root) {
		t.Error("Detect = false, want true for ID=amzn VERSION_ID=2023")
	}
}

func TestDetectAmazonLinux2023QuotedOSRelease(t *testing.T) {
	root := t.TempDir()
	usrLib := filepath.Join(root, "usr", "lib")
	if err := os.MkdirAll(usrLib, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(usrLib, "os-release"),
		[]byte(`ID="amzn"`+"\n"+`VERSION_ID="2023"`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !networkd.Detect(root) {
		t.Error("Detect = false, want true for quoted ID=amzn VERSION_ID=2023")
	}
}

func TestDetectAmazonLinux2NotUnconditional(t *testing.T) {
	root := t.TempDir()
	usrLib := filepath.Join(root, "usr", "lib")
	if err := os.MkdirAll(usrLib, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(usrLib, "os-release"),
		[]byte("ID=amzn\nVERSION_ID=2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if networkd.Detect(root) {
		t.Error("Detect = true, want false for Amazon Linux 2 without active systemd-networkd")
	}
}

func TestDetectAmazonLinux2FallsThroughToNetworkdPrimary(t *testing.T) {
	root := t.TempDir()
	setupNetworkdPrimary(t, root, true, true)

	if err := os.WriteFile(filepath.Join(root, "etc", "os-release"),
		[]byte("ID=amzn\nVERSION_ID=2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !networkd.Detect(root) {
		t.Error("Detect = false, want true for Amazon Linux 2 with active systemd-networkd")
	}
}

func TestDetectEC2NetworkFile(t *testing.T) {
	root := t.TempDir()
	netDir := filepath.Join(root, "usr", "lib", "systemd", "network")
	if err := os.MkdirAll(netDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(netDir, "80-ec2.network"), []byte("[Network]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !networkd.Detect(root) {
		t.Error("Detect = false, want true for 80-ec2.network")
	}
}

func TestDetectNegativeRHEL(t *testing.T) {
	root := t.TempDir()
	etc := filepath.Join(root, "etc")
	if err := os.MkdirAll(etc, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(etc, "os-release"),
		[]byte("ID=rhel\nVERSION_ID=9.4\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if networkd.Detect(root) {
		t.Error("Detect = true, want false for RHEL guest without networkd primary")
	}
}

func TestDetectNetworkdPrimaryMaskedNM(t *testing.T) {
	root := t.TempDir()
	setupNetworkdPrimary(t, root, true, true)

	if err := os.WriteFile(filepath.Join(root, "etc", "os-release"),
		[]byte("ID=fedora\nVERSION_ID=42\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !networkd.Detect(root) {
		t.Error("Detect = false, want true for networkd primary with masked NM")
	}
}

func TestDetectNetworkdWithNMEnabled(t *testing.T) {
	root := t.TempDir()
	setupNetworkdPrimary(t, root, true, false)
	enableUnitWants(t, root, "NetworkManager.service")

	if networkd.Detect(root) {
		t.Error("Detect = true, want false when NetworkManager is also enabled")
	}
}

func TestDetectAmazonLinux2023AmbiguousStack(t *testing.T) {
	root := t.TempDir()
	usrLib := filepath.Join(root, "usr", "lib")
	if err := os.MkdirAll(usrLib, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(usrLib, "os-release"),
		[]byte("ID=amzn\nVERSION_ID=2023\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	setupNetworkdPrimary(t, root, true, false)
	enableUnitWants(t, root, "NetworkManager.service")

	if networkd.Detect(root) {
		t.Error("Detect = true, want false for AL2023 when networkd and NM both enabled")
	}
}

func setupNetworkdPrimary(t *testing.T, root string, networkdEnabled, maskNM bool) {
	t.Helper()
	if networkdEnabled {
		enableUnitWants(t, root, "systemd-networkd.service")
	}
	if maskNM {
		maskUnit(t, root, "NetworkManager.service")
	}
}

func enableUnitWants(t *testing.T, root, unit string) {
	t.Helper()
	wantsDir := filepath.Join(root, "etc", "systemd", "system", "multi-user.target.wants")
	if err := os.MkdirAll(wantsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wantsDir, unit), []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func maskUnit(t *testing.T, root, unit string) {
	t.Helper()
	unitDir := filepath.Join(root, "etc", "systemd", "system")
	if err := os.MkdirAll(unitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	maskPath := filepath.Join(unitDir, unit)
	if err := os.Remove(maskPath); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if err := os.Symlink("/dev/null", maskPath); err != nil {
		t.Fatal(err)
	}
}

func TestInstallKubeVirtNetworking(t *testing.T) {
	root := t.TempDir()
	if err := networkd.InstallKubeVirtNetworking(root); err != nil {
		t.Fatalf("InstallKubeVirtNetworking: %v", err)
	}
	virtioNet := filepath.Join(root, "etc", "systemd", "network", "10-kc-virtio.network")
	if _, err := os.Stat(virtioNet); err != nil {
		t.Fatalf("virtio network file not created: %v", err)
	}
	dropIn := filepath.Join(root, "etc", "systemd", "system",
		"systemd-networkd-wait-online.service.d", "kc-timeout.conf")
	if _, err := os.Stat(dropIn); err != nil {
		t.Fatalf("wait-online drop-in not created: %v", err)
	}
}

func TestInstallDHCP(t *testing.T) {
	root := t.TempDir()
	if err := networkd.InstallDHCP(root); err != nil {
		t.Fatalf("InstallDHCP: %v", err)
	}
	path := filepath.Join(root, "etc", "systemd", "network", "10-kc-virtio.network")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading virtio network file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "enp*") {
		t.Errorf("missing enp* match: %s", content)
	}
	if !strings.Contains(content, "eth*") {
		t.Errorf("missing eth* match: %s", content)
	}
	if !strings.Contains(content, "Driver=virtio_net") {
		t.Errorf("missing virtio_net driver: %s", content)
	}
	if !strings.Contains(content, "DHCP=yes") {
		t.Errorf("missing DHCP=yes: %s", content)
	}
}

func TestInstallDHCPIdempotent(t *testing.T) {
	root := t.TempDir()
	if err := networkd.InstallDHCP(root); err != nil {
		t.Fatal(err)
	}
	if err := networkd.InstallDHCP(root); err != nil {
		t.Fatalf("second InstallDHCP: %v", err)
	}
}

func TestWriteStaticNetworks(t *testing.T) {
	root := t.TempDir()
	ips := []types.StaticIP{{
		MAC:     "0A:58:0A:83:00:48",
		IP:      "10.131.0.72",
		Gateway: "10.0.0.1",
		Netmask: "255.255.255.0",
		DNS:     []string{"8.8.8.8", "8.8.4.4"},
	}}
	if err := networkd.WriteStaticNetworks(root, ips); err != nil {
		t.Fatalf("WriteStaticNetworks: %v", err)
	}
	path := filepath.Join(root, "etc", "systemd", "network",
		"10-kc-static-0a-58-0a-83-00-48.network")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading static network file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "MACAddress=0a:58:0a:83:00:48") {
		t.Errorf("MAC not normalized: %s", content)
	}
	if !strings.Contains(content, "Address=10.131.0.72/24") {
		t.Errorf("missing address: %s", content)
	}
	if !strings.Contains(content, "Gateway=10.0.0.1") {
		t.Errorf("missing gateway: %s", content)
	}
	if !strings.Contains(content, "DNS=8.8.8.8 8.8.4.4") {
		t.Errorf("missing DNS: %s", content)
	}
}

func TestWriteStaticNetworksInvalidNetmask(t *testing.T) {
	root := t.TempDir()
	ips := []types.StaticIP{{
		MAC:     "0A:58:0A:83:00:48",
		IP:      "10.131.0.72",
		Gateway: "10.0.0.1",
		Netmask: "255.0.255.0",
	}}
	if err := networkd.WriteStaticNetworks(root, ips); err == nil {
		t.Fatal("WriteStaticNetworks = nil error, want error for invalid netmask")
	}
	entries, err := os.ReadDir(filepath.Join(root, "etc", "systemd", "network"))
	if err != nil {
		t.Fatalf("reading network dir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected no profiles written for invalid netmask, found %d", len(entries))
	}
}

func TestWriteStaticNetworksEmpty(t *testing.T) {
	root := t.TempDir()
	if err := networkd.WriteStaticNetworks(root, nil); err != nil {
		t.Fatalf("WriteStaticNetworks nil: %v", err)
	}
}

func TestInstallWaitOnlineDropIn(t *testing.T) {
	root := t.TempDir()
	if err := networkd.InstallWaitOnlineDropIn(root); err != nil {
		t.Fatalf("InstallWaitOnlineDropIn: %v", err)
	}
	path := filepath.Join(root, "etc", "systemd", "system",
		"systemd-networkd-wait-online.service.d", "kc-timeout.conf")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading drop-in: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "--timeout=30") {
		t.Errorf("missing timeout: %s", content)
	}
	if !strings.Contains(content, "--any") {
		t.Errorf("missing --any: %s", content)
	}
}
