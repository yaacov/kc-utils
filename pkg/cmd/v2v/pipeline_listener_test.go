//go:build linux

package v2v

import (
	"strings"
	"testing"

	"github.com/yaacov/kc-utils/pkg/guest/backend"
)

func TestEnsureSharedListenerPreservesNetworkOnRestart(t *testing.T) {
	prev := startSharedSession
	t.Cleanup(func() { startSharedSession = prev })

	var gotBackend string
	startSharedSession = func(backend string) (backend.SharedSession, error) {
		gotBackend = backend
		return &stubSession{pid: 4242, alive: true}, nil
	}

	var listener backend.SharedSession = &stubSession{pid: 0, alive: false}
	stageEnv := []string{backend.EnvGuestfsNetwork + "=1"}

	if err := ensureSharedListener(&listener, &stageEnv, "prepare", "guestfs"); err != nil {
		t.Fatalf("ensureSharedListener: %v", err)
	}
	if gotBackend != "guestfs" {
		t.Fatalf("restart backend=%q want guestfs", gotBackend)
	}
	stub, ok := listener.(*stubSession)
	if !ok || stub.pid != 4242 {
		t.Fatalf("listener=%v want stub pid 4242", listener)
	}

	found := false
	for _, e := range stageEnv {
		if e == backend.EnvGuestfsNetwork+"=1" || strings.HasPrefix(e, backend.EnvGuestfsNetwork+"=") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("stageEnv missing %s after restart: %v", backend.EnvGuestfsNetwork, stageEnv)
	}
}
