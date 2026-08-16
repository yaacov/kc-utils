//go:build linux

package v2v

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/yaacov/kc-utils/pkg/guest"
	"github.com/yaacov/kc-utils/pkg/v2v/env"

	_ "github.com/yaacov/kc-utils/pkg/guest/direct"
	_ "github.com/yaacov/kc-utils/pkg/guest/guestfs"
)

type stubSession struct {
	pid   int
	alive bool
}

func (s *stubSession) Close() error { return nil }
func (s *stubSession) Alive() bool  { return s.alive && s.pid > 0 }
func (s *stubSession) Env() []string {
	if s.pid <= 0 {
		return nil
	}
	return []string{fmt.Sprintf("%s=%d", guest.EnvGuestfishPID, s.pid)}
}

func TestStageCommonArgs(t *testing.T) {
	cfg := &env.Config{
		MountRoot: "/mnt/guest",
		LogLevel:  "debug",
		Backend:   "direct",
	}
	got := stageCommonArgs(cfg, "/in.json", "/out.json")
	want := []string{
		"--input", "/in.json",
		"--output", "/out.json",
		"--mount-root", "/mnt/guest",
		"--log-level", "debug",
		"--backend", "direct",
	}
	if len(got) != len(want) {
		t.Fatalf("args=%v want=%v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("args[%d]=%q want %q", i, got[i], want[i])
		}
	}

	cfg.Backend = "guestfs"
	got = stageCommonArgs(cfg, "/in.json", "/out.json")
	if got[len(got)-1] != "guestfs" || got[len(got)-2] != "--backend" {
		t.Fatalf("expected trailing --backend guestfs, got %v", got)
	}
}

func TestSetupSharedListenerDisabled(t *testing.T) {
	listener, stageEnv, err := setupSharedListener(&env.Config{Backend: "direct"})
	if err != nil {
		t.Fatalf("setupSharedListener: %v", err)
	}
	if listener != nil || stageEnv != nil {
		t.Fatalf("want nil listener/env, got listener=%v env=%v", listener, stageEnv)
	}
}

func TestSetupSharedListenerClevisNetwork(t *testing.T) {
	prev := startSharedSession
	t.Cleanup(func() { startSharedSession = prev })

	startSharedSession = func(backend string) (guest.SharedSession, error) {
		if backend != "guestfs" {
			t.Fatalf("backend=%q", backend)
		}
		return &stubSession{pid: 1111, alive: true}, nil
	}

	listener, stageEnv, err := setupSharedListener(&env.Config{
		Backend:    "guestfs",
		NbdeClevis: true,
	})
	if err != nil {
		t.Fatalf("setupSharedListener: %v", err)
	}
	if listener == nil {
		t.Fatal("expected listener")
	}
	found := false
	for _, e := range stageEnv {
		if e == guest.EnvGuestfsNetwork+"=1" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("stageEnv missing %s=1: %v", guest.EnvGuestfsNetwork, stageEnv)
	}
}

func TestSetupSharedListenerError(t *testing.T) {
	prev := startSharedSession
	t.Cleanup(func() { startSharedSession = prev })

	startSharedSession = func(string) (guest.SharedSession, error) {
		return nil, errors.New("boom")
	}

	_, _, err := setupSharedListener(&env.Config{Backend: "guestfs"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "shared backend session") {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("error should wrap boom: %v", err)
	}
}

func TestEnsureSharedListenerNilNoop(t *testing.T) {
	stageEnv := []string{"FOO=1"}
	if err := ensureSharedListener(nil, &stageEnv, "prepare", "direct"); err != nil {
		t.Fatalf("nil listener: %v", err)
	}
	if len(stageEnv) != 1 || stageEnv[0] != "FOO=1" {
		t.Fatalf("stageEnv changed: %v", stageEnv)
	}
}
