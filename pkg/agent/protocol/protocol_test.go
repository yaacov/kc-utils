package protocol_test

import (
	"bytes"
	"testing"

	"github.com/yaacov/kc-utils/pkg/agent/protocol"
)

func TestFrameRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	req := protocol.Request{ID: 7, Op: protocol.OpPing}
	if err := protocol.WriteFrame(&buf, req); err != nil {
		t.Fatal(err)
	}
	var got protocol.Request
	if err := protocol.ReadFrame(&buf, &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != 7 || got.Op != protocol.OpPing {
		t.Fatalf("got %+v", got)
	}
}

func TestBlobRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	want := []byte("hello-agent")
	if err := protocol.WriteBlob(&buf, want); err != nil {
		t.Fatal(err)
	}
	got, err := protocol.ReadBlob(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("got %q", got)
	}
}

func TestShellSock(t *testing.T) {
	got := protocol.ShellSock("/tmp/kc-qemu-xxx/agent.sock")
	if got != "/tmp/kc-qemu-xxx/shell.sock" {
		t.Fatalf("got %q", got)
	}
	if protocol.ShellSock("") != "" {
		t.Fatal("empty agent sock should yield empty shell sock")
	}
}

func TestShellConfigCommand(t *testing.T) {
	if got := (protocol.ShellConfig{}).Command(); len(got) != 1 || got[0] != "/bin/bash" {
		t.Fatalf("default %v", got)
	}
	got := protocol.ShellConfig{Chroot: "/mnt/guest", Argv: []string{"ls", "/"}}.Command()
	want := []string{"chroot", "/mnt/guest", "ls", "/"}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestReadBlobEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := protocol.WriteBlob(&buf, nil); err != nil {
		t.Fatal(err)
	}
	got, err := protocol.ReadBlob(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("got %d bytes", len(got))
	}
}
