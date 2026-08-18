//go:build linux

package v2v

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/yaacov/kc-utils/pkg/backend"
	"github.com/yaacov/kc-utils/pkg/guest"
	"github.com/yaacov/kc-utils/pkg/v2v/env"
)

type stubSharedListener struct {
	pid int
}

func (s *stubSharedListener) Env() []string {
	return []string{fmt.Sprintf("%s=%d", guest.EnvGuestfishPID, s.pid)}
}

func (s *stubSharedListener) Alive() bool { return s.pid > 0 }

func (s *stubSharedListener) Close() error { return nil }

func TestStageCommonArgs(t *testing.T) {
	cfg := &env.Config{
		MountRoot: "/mnt/guest",
		LogLevel:  "debug",
		Backend:   backend.NameDirect,
	}
	got := stageCommonArgs(cfg, "/in.json", "/out.json")
	want := []string{
		"--input", "/in.json",
		"--output", "/out.json",
		"--mount-root", "/mnt/guest",
		"--log-level", "debug",
		"--backend", backend.NameDirect,
	}
	if len(got) != len(want) {
		t.Fatalf("args=%v want=%v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("args[%d]=%q want %q", i, got[i], want[i])
		}
	}

	cfg.Backend = backend.NameGuestfs
	got = stageCommonArgs(cfg, "/in.json", "/out.json")
	if got[len(got)-1] != backend.NameGuestfs || got[len(got)-2] != "--backend" {
		t.Fatalf("expected trailing --backend guestfs, got %v", got)
	}
}

func TestSetupSharedListenerDisabled(t *testing.T) {
	listener, stageEnv, err := setupSharedListener(&env.Config{Backend: backend.NameDirect})
	if err != nil {
		t.Fatalf("setupSharedListener: %v", err)
	}
	if listener != nil || stageEnv != nil {
		t.Fatalf("want nil listener/env, got listener=%v env=%v", listener, stageEnv)
	}
}

func TestSetupSharedListenerClevisNetwork(t *testing.T) {
	prev := startSharedListener
	t.Cleanup(func() { startSharedListener = prev })

	startSharedListener = func() (guest.SharedListener, error) {
		return &stubSharedListener{pid: 1111}, nil
	}

	listener, stageEnv, err := setupSharedListener(&env.Config{
		Backend:    backend.NameGuestfs,
		NbdeClevis: true,
	})
	if err != nil {
		t.Fatalf("setupSharedListener: %v", err)
	}
	if listener == nil || !listener.Alive() {
		t.Fatalf("listener=%v", listener)
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
	prev := startSharedListener
	t.Cleanup(func() { startSharedListener = prev })

	startSharedListener = func() (guest.SharedListener, error) {
		return nil, errors.New("boom")
	}

	_, _, err := setupSharedListener(&env.Config{Backend: backend.NameGuestfs})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "guestfish shared listener") {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("error should wrap boom: %v", err)
	}
}

func TestEnsureSharedListenerNilNoop(t *testing.T) {
	stageEnv := []string{"FOO=1"}
	if err := ensureSharedListener(nil, &stageEnv, "prepare"); err != nil {
		t.Fatalf("nil listener: %v", err)
	}
	if len(stageEnv) != 1 || stageEnv[0] != "FOO=1" {
		t.Fatalf("stageEnv changed: %v", stageEnv)
	}
}
