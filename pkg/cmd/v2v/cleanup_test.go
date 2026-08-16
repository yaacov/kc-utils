//go:build linux

package v2v

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yaacov/kc-utils/pkg/v2v/env"
)

func TestTeardownOnlyArgsIncludesPrepareWhenPresent(t *testing.T) {
	dir := t.TempDir()
	prepareOut := filepath.Join(dir, "prepare-out.json")
	if err := os.WriteFile(prepareOut, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &env.Config{
		MountRoot: "/tmp/kc-guest",
		LogLevel:  "info",
		Backend:   "guestfs",
	}
	args := teardownOnlyArgs(cfg, prepareOut)
	want := []string{
		"--teardown-only",
		"--mount-root", "/tmp/kc-guest",
		"--log-level", "info",
		"--backend", "guestfs",
		"--input", prepareOut,
	}
	if len(args) != len(want) {
		t.Fatalf("args=%v want=%v", args, want)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Fatalf("args[%d]=%q want %q (full args=%v)", i, args[i], want[i], args)
		}
	}
}

func TestTeardownOnlyArgsOmitsMissingPrepare(t *testing.T) {
	cfg := &env.Config{
		MountRoot: "/tmp/kc-guest",
		LogLevel:  "debug",
		Backend:   "direct",
	}
	args := teardownOnlyArgs(cfg, filepath.Join(t.TempDir(), "missing.json"))
	want := []string{
		"--teardown-only",
		"--mount-root", "/tmp/kc-guest",
		"--log-level", "debug",
		"--backend", "direct",
	}
	if len(args) != len(want) {
		t.Fatalf("args=%v want=%v", args, want)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Fatalf("args[%d]=%q want %q", i, args[i], want[i])
		}
	}
}
