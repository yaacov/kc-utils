//go:build unix

package v2v

import (
	"strings"
	"testing"

	"github.com/yaacov/kc-utils/pkg/backend"
	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/guest"
)

func TestEnsureSharedListenerPreservesNetworkOnRestart(t *testing.T) {
	prev := startSharedListener
	t.Cleanup(func() { startSharedListener = prev })

	startSharedListener = func(_ string, _ []types.DiskSpec) (guest.SharedListener, error) {
		return &stubSharedListener{pid: 4242}, nil
	}

	// PID 0 is not alive; Close is a no-op — exercises the restart path only.
	var listener guest.SharedListener = &stubSharedListener{pid: 0}
	stageEnv := []string{guest.EnvGuestfsNetwork + "=1"}

	if err := ensureSharedListener(&listener, &stageEnv, "prepare", backend.NameGuestfs, nil); err != nil {
		t.Fatalf("ensureSharedListener: %v", err)
	}
	if !listener.Alive() {
		t.Fatalf("listener not alive after restart")
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
