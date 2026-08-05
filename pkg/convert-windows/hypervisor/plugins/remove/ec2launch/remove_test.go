//go:build linux

package ec2launch

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yaacov/kc-utils/pkg/common/registry/mock"
)

func TestDetectEC2Launch(t *testing.T) {
	h := mock.NewMockHive()
	h.CreateKey(`Microsoft\Windows\CurrentVersion\Uninstall\EC2Launch`)

	u := &Remove{}
	if !u.Detect("/fake", nil, h) {
		t.Error("Detect returned false, want true when EC2Launch key exists")
	}
}

func TestDetectEc2Config(t *testing.T) {
	h := mock.NewMockHive()
	h.CreateKey(`Microsoft\Windows\CurrentVersion\Uninstall\Ec2Config`)

	u := &Remove{}
	if !u.Detect("/fake", nil, h) {
		t.Error("Detect returned false, want true when Ec2Config key exists")
	}
}

func TestDetectViaDirectory(t *testing.T) {
	guestRoot := t.TempDir()
	ec2Dir := filepath.Join(guestRoot, "Program Files", "Amazon", "EC2Launch")
	if err := os.MkdirAll(ec2Dir, 0o755); err != nil {
		t.Fatal(err)
	}

	h := mock.NewMockHive()

	u := &Remove{}
	if !u.Detect(guestRoot, nil, h) {
		t.Error("Detect returned false, want true when EC2Launch directory exists (no registry key)")
	}
}

func TestDetectAbsent(t *testing.T) {
	guestRoot := t.TempDir()
	h := mock.NewMockHive()

	u := &Remove{}
	if u.Detect(guestRoot, nil, h) {
		t.Error("Detect returned true, want false when neither directory nor registry key exists")
	}
}

func TestRemove(t *testing.T) {
	guestRoot := t.TempDir()

	ec2LaunchDir := filepath.Join(guestRoot, "Program Files", "Amazon", "EC2Launch")
	if err := os.MkdirAll(ec2LaunchDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ec2LaunchDir, "EC2Launch.exe"), []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	h := mock.NewMockHive()
	h.CreateKey(`Microsoft\Windows\CurrentVersion\Uninstall\EC2Launch`)
	h.CreateKey(`Microsoft\Windows\CurrentVersion\Uninstall\Ec2Config`)

	u := &Remove{}
	if err := u.Remove(guestRoot, nil, h); err != nil {
		t.Fatalf("Remove returned error: %v", err)
	}

	if h.KeyExists(`Microsoft\Windows\CurrentVersion\Uninstall\EC2Launch`) {
		t.Error("EC2Launch uninstall key still exists")
	}
	if h.KeyExists(`Microsoft\Windows\CurrentVersion\Uninstall\Ec2Config`) {
		t.Error("Ec2Config uninstall key still exists")
	}

	if _, err := os.Stat(ec2LaunchDir); !os.IsNotExist(err) {
		t.Error("EC2Launch directory still exists after Remove")
	}
}
