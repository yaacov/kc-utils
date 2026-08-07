//go:build linux

package v2v

import (
	"strings"
	"testing"

	"github.com/yaacov/kc-utils/pkg/guest"
)

func TestEnsureSharedListenerPreservesNetworkOnRestart(t *testing.T) {
	prev := startSharedListener
	t.Cleanup(func() { startSharedListener = prev })

	startSharedListener = func() (*guest.SharedListener, error) {
		return &guest.SharedListener{PID: 4242}, nil
	}

	// PID 0 is not alive; Close is a no-op — exercises the restart path only.
	listener := &guest.SharedListener{PID: 0}
	stageEnv := []string{guest.EnvGuestfsNetwork + "=1"}

	if err := ensureSharedListener(listener, &stageEnv, "prepare"); err != nil {
		t.Fatalf("ensureSharedListener: %v", err)
	}
	if listener.PID != 4242 {
		t.Fatalf("listener PID=%d want 4242", listener.PID)
	}

	found := false
	for _, e := range stageEnv {
		if e == guest.EnvGuestfsNetwork+"=1" || strings.HasPrefix(e, guest.EnvGuestfsNetwork+"=") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("stageEnv missing %s after restart: %v", guest.EnvGuestfsNetwork, stageEnv)
	}
}
