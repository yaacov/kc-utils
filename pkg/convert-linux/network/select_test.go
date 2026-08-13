//go:build linux

package network_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/convert-linux/network"

	_ "github.com/yaacov/kc-utils/pkg/convert-linux/network/handlers/default"
	_ "github.com/yaacov/kc-utils/pkg/convert-linux/network/handlers/networkd"
)

func TestSelectNetworkdPrimaryWithLeftoverNMProfiles(t *testing.T) {
	root := t.TempDir()
	setupNetworkdPrimary(t, root, true, true)
	connDir := filepath.Join(root, "etc", "NetworkManager", "system-connections")
	if err := os.MkdirAll(connDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(connDir, "eth0.nmconnection"), []byte("[connection]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := network.Select(root).Name(); got != "networkd" {
		t.Errorf("Select().Name() = %q, want networkd", got)
	}
}

func TestSelectNetworkdAndNMBothEnabled(t *testing.T) {
	root := t.TempDir()
	setupNetworkdPrimary(t, root, true, false)
	enableUnitWants(t, root, "NetworkManager.service")

	if got := network.Select(root).Name(); got != network.DefaultHandlerName {
		t.Errorf("Select().Name() = %q, want %q", got, network.DefaultHandlerName)
	}
}

func TestSelectRHELNMOnly(t *testing.T) {
	root := t.TempDir()
	etc := filepath.Join(root, "etc")
	if err := os.MkdirAll(etc, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(etc, "os-release"),
		[]byte("ID=rhel\nVERSION_ID=9.4\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	enableUnitWants(t, root, "NetworkManager.service")

	if got := network.Select(root).Name(); got != network.DefaultHandlerName {
		t.Errorf("Select().Name() = %q, want %q", got, network.DefaultHandlerName)
	}
}

func TestSelectAmazonLinux2023OSReleaseOnly(t *testing.T) {
	root := t.TempDir()
	usrLib := filepath.Join(root, "usr", "lib")
	if err := os.MkdirAll(usrLib, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(usrLib, "os-release"),
		[]byte("ID=amzn\nVERSION_ID=2023\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := network.Select(root).Name(); got != "networkd" {
		t.Errorf("Select().Name() = %q, want networkd", got)
	}
}

func TestSelectAmazonLinux2NetworkdPrimary(t *testing.T) {
	root := t.TempDir()
	setupNetworkdPrimary(t, root, true, true)
	if err := os.WriteFile(filepath.Join(root, "etc", "os-release"),
		[]byte("ID=amzn\nVERSION_ID=2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := network.Select(root).Name(); got != "networkd" {
		t.Errorf("Select().Name() = %q, want networkd", got)
	}
}

func TestSelectAmazonLinux2NoNetworkd(t *testing.T) {
	root := t.TempDir()
	usrLib := filepath.Join(root, "usr", "lib")
	if err := os.MkdirAll(usrLib, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(usrLib, "os-release"),
		[]byte("ID=amzn\nVERSION_ID=2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := network.Select(root).Name(); got != network.DefaultHandlerName {
		t.Errorf("Select().Name() = %q, want %q", got, network.DefaultHandlerName)
	}
}

func TestSelectEC2NetworkFile(t *testing.T) {
	root := t.TempDir()
	netDir := filepath.Join(root, "usr", "lib", "systemd", "network")
	if err := os.MkdirAll(netDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(netDir, "80-ec2.network"), []byte("[Network]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := network.Select(root).Name(); got != "networkd" {
		t.Errorf("Select().Name() = %q, want networkd", got)
	}
}

func TestSelectEmptyGuestRoot(t *testing.T) {
	root := t.TempDir()

	if got := network.Select(root).Name(); got != network.DefaultHandlerName {
		t.Errorf("Select().Name() = %q, want %q", got, network.DefaultHandlerName)
	}
}

func TestPrimaryLabel(t *testing.T) {
	root := t.TempDir()
	setupNetworkdPrimary(t, root, true, true)

	netdHandler := network.Select(root)
	if got := network.PrimaryLabel(netdHandler); got != types.NetworkPrimarySystemdNetworkd {
		t.Errorf("PrimaryLabel(networkd) = %q, want %q", got, types.NetworkPrimarySystemdNetworkd)
	}

	defaultHandler, ok := network.Handlers.Get(network.DefaultHandlerName)
	if !ok {
		t.Fatal("default handler not registered")
	}
	if got := network.PrimaryLabel(defaultHandler); got != types.NetworkPrimaryLegacy {
		t.Errorf("PrimaryLabel(default) = %q, want %q", got, types.NetworkPrimaryLegacy)
	}
	if got := network.PrimaryLabel(nil); got != types.NetworkPrimaryLegacy {
		t.Errorf("PrimaryLabel(nil) = %q, want %q", got, types.NetworkPrimaryLegacy)
	}
}

func TestSelectAmazonLinux2023AmbiguousStack(t *testing.T) {
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

	if got := network.Select(root).Name(); got != network.DefaultHandlerName {
		t.Errorf("Select().Name() = %q, want %q when networkd and NM both enabled", got, network.DefaultHandlerName)
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
