//go:build unix

package agentsh

import (
	"testing"

	"github.com/yaacov/kc-utils/pkg/guest/backend"
)

func TestResolveSockExplicitShell(t *testing.T) {
	got, err := ResolveSock("/tmp/kc/shell.sock")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/tmp/kc/shell.sock" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveSockExplicitAgent(t *testing.T) {
	got, err := ResolveSock("/tmp/kc/agent.sock")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/tmp/kc/shell.sock" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveSockFromEnv(t *testing.T) {
	t.Setenv(backend.EnvAgentSock, "/var/tmp/kc-qemu-x/agent.sock")
	got, err := ResolveSock("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/var/tmp/kc-qemu-x/shell.sock" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveSockMissing(t *testing.T) {
	t.Setenv(backend.EnvAgentSock, "")
	if _, err := ResolveSock(""); err == nil {
		t.Fatal("expected error")
	}
}
