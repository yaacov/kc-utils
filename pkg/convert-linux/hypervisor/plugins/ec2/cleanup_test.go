//go:build linux

package ec2

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yaacov/kc-utils/pkg/convert-linux/hypervisor/plugins/testassert"
)

func TestDetectSSMAgent(t *testing.T) {
	guestRoot := t.TempDir()
	ssmPath := filepath.Join(guestRoot, "usr", "bin", "amazon-ssm-agent")
	if err := os.MkdirAll(filepath.Dir(ssmPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ssmPath, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}

	u := &Cleanup{}
	if !u.Detect(guestRoot) {
		t.Error("Detect returned false, want true when SSM agent exists")
	}
}

func TestDetectCloudInitEc2(t *testing.T) {
	guestRoot := t.TempDir()
	cloudDir := filepath.Join(guestRoot, "etc", "cloud")
	if err := os.MkdirAll(cloudDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cloudDir, "cloud.cfg"),
		[]byte("datasource_list: [Ec2]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	u := &Cleanup{}
	if !u.Detect(guestRoot) {
		t.Error("Detect returned false, want true when cloud.cfg contains Ec2")
	}
}

func TestDetectAbsent(t *testing.T) {
	guestRoot := t.TempDir()

	u := &Cleanup{}
	if u.Detect(guestRoot) {
		t.Error("Detect returned true, want false with no EC2 indicators")
	}
}

func TestCleanup(t *testing.T) {
	guestRoot := t.TempDir()
	setupEC2ServiceSymlinks(t, guestRoot)

	cloudCfgDir := filepath.Join(guestRoot, "etc", "cloud", "cloud.cfg.d")
	if err := os.MkdirAll(cloudCfgDir, 0o755); err != nil {
		t.Fatal(err)
	}

	u := &Cleanup{}
	if err := u.Cleanup(guestRoot); err != nil {
		t.Fatalf("Cleanup returned error: %v", err)
	}

	assertEC2ServicesDisabled(t, guestRoot)

	disableCfg := filepath.Join(cloudCfgDir, "99-kc-disable-ec2.cfg")
	data, err := os.ReadFile(disableCfg)
	if err != nil {
		t.Fatalf("cloud-init disable config not created: %v", err)
	}
	if string(data) != "datasource_list: [None]\n" {
		t.Errorf("disable config content = %q, want datasource_list: [None]", data)
	}
}

func TestCleanupCreatesCloudCfgDir(t *testing.T) {
	guestRoot := t.TempDir()
	setupEC2ServiceSymlinks(t, guestRoot)

	u := &Cleanup{}
	if err := u.Cleanup(guestRoot); err != nil {
		t.Fatalf("Cleanup returned error: %v", err)
	}

	disableCfg := filepath.Join(guestRoot, "etc", "cloud", "cloud.cfg.d", "99-kc-disable-ec2.cfg")
	if _, err := os.Stat(disableCfg); err != nil {
		t.Fatalf("cloud-init disable config not created without pre-existing cloud.cfg.d: %v", err)
	}
}

func setupEC2ServiceSymlinks(t *testing.T, guestRoot string) {
	t.Helper()

	wantsDir := filepath.Join(guestRoot, "etc", "systemd", "system", "multi-user.target.wants")
	if err := os.MkdirAll(wantsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	vendorWantsDir := filepath.Join(guestRoot, "usr", "lib", "systemd", "system", "multi-user.target.wants")
	if err := os.MkdirAll(vendorWantsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, svc := range []string{
		"amazon-ssm-agent.service",
		"amazon-cloudwatch-agent.service",
		"ec2-instance-connect.service",
		"hibagent.service",
		"hibinit-agent.service",
	} {
		if err := os.WriteFile(filepath.Join(wantsDir, svc), []byte("fake"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(vendorWantsDir, svc), []byte("fake"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func assertEC2ServicesDisabled(t *testing.T, guestRoot string) {
	t.Helper()

	wantsDir := filepath.Join(guestRoot, "etc", "systemd", "system", "multi-user.target.wants")
	for _, svc := range []string{
		"amazon-ssm-agent.service",
		"amazon-cloudwatch-agent.service",
		"ec2-instance-connect.service",
		"hibagent.service",
		"hibinit-agent.service",
	} {
		if _, err := os.Stat(filepath.Join(wantsDir, svc)); !os.IsNotExist(err) {
			t.Errorf("service symlink %s still exists", svc)
		}
		testassert.UnitDisabled(t, guestRoot, svc)
	}
}
