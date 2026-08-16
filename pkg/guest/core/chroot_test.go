//go:build linux

package core

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/yaacov/kc-utils/pkg/guest/runtime/local"
)

var lddLibRE = regexp.MustCompile(`(/[^ ]+)`)

// TestRunCommandChroot exercises core.RunCommand over the host-local runtime,
// which is exactly the direct backend's execution path.
func TestRunCommandChroot(t *testing.T) {
	root := t.TempDir()
	installTestShell(t, root)

	b := New(local.New(), true)
	b.SetGuestRoot(root)

	out, err := b.RunCommand(root, []string{"/bin/sh", "-c", "echo guest-ok"})
	if err != nil {
		if guestRootExecUnavailable(out) {
			t.Skipf("guest root execution unavailable in this environment: %v\n%s", err, out)
		}
		t.Fatalf("RunCommand: %v\n%s", err, out)
	}
	if string(out) != "guest-ok\n" {
		t.Fatalf("output=%q", out)
	}
}

func installTestShell(t *testing.T, root string) {
	t.Helper()
	shReal, err := exec.LookPath("sh")
	if err != nil {
		t.Fatal(err)
	}
	if shReal, err = filepath.EvalSymlinks(shReal); err != nil {
		t.Fatal(err)
	}
	shDest := filepath.Join(root, "bin", "sh")
	if err := os.MkdirAll(filepath.Dir(shDest), 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(shReal)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(shDest, data, 0o755); err != nil {
		t.Fatal(err)
	}
	lddOut, err := exec.Command("ldd", shReal).CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	for _, lib := range lddLibRE.FindAllString(string(lddOut), -1) {
		if !strings.HasPrefix(lib, "/") {
			continue
		}
		dest := filepath.Join(root, lib)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			t.Fatal(err)
		}
		libData, err := os.ReadFile(lib)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dest, libData, 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func guestRootExecUnavailable(out []byte) bool {
	return bytes.Contains(out, []byte("Operation not permitted")) ||
		bytes.Contains(out, []byte("uid_map"))
}
