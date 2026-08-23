//go:build linux

package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"testing"
	"time"
)

func TestServeChannelOnceMissingPort(t *testing.T) {
	err := serveChannelOnce("org.kc-utils.does-not-exist", []string{"/bin/true"})
	if err == nil {
		t.Fatal("expected error for missing virtio-serial port")
	}
}

func TestRunOnChannelOutput(t *testing.T) {
	requirePTY(t)

	a, b := net.Pipe()
	defer b.Close()
	errc := make(chan error, 1)
	go func() {
		errc <- runOnChannel(a, []string{"/bin/sh", "-c", "echo channel-ok"})
	}()

	_ = b.SetReadDeadline(time.Now().Add(5 * time.Second))
	got, err := io.ReadAll(b)
	if err != nil {
		t.Fatalf("read channel: %v (got %q)", err, got)
	}
	if !bytes.Contains(got, []byte("channel-ok")) {
		t.Fatalf("output %q, want channel-ok", got)
	}

	select {
	case err := <-errc:
		if err != nil {
			t.Errorf("runOnChannel: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("runOnChannel did not return")
	}
}

func TestRunOnChannelReadsInput(t *testing.T) {
	requirePTY(t)

	a, b := net.Pipe()
	defer b.Close()
	errc := make(chan error, 1)
	go func() {
		errc <- runOnChannel(a, []string{"/bin/sh", "-c", `IFS= read -r line; printf 'got:%s\n' "$line"`})
	}()

	// Give the PTY/shell a moment to block in read before we write.
	time.Sleep(100 * time.Millisecond)
	if _, err := b.Write([]byte("ping\n")); err != nil {
		t.Fatalf("write: %v", err)
	}

	_ = b.SetReadDeadline(time.Now().Add(5 * time.Second))
	var buf bytes.Buffer
	tmp := make([]byte, 256)
	for {
		n, err := b.Read(tmp)
		if n > 0 {
			buf.Write(tmp[:n])
			if strings.Contains(buf.String(), "got:ping") {
				break
			}
		}
		if err != nil {
			t.Fatalf("read: %v (got %q)", err, buf.Bytes())
		}
	}

	select {
	case <-errc:
	case <-time.After(5 * time.Second):
		t.Fatal("runOnChannel did not return")
	}
}

func TestRunOnChannelDelayedConsumption(t *testing.T) {
	requirePTY(t)

	const lines = 200
	a, b := net.Pipe()
	defer b.Close()
	errc := make(chan error, 1)
	go func() {
		errc <- runOnChannel(a, []string{"/bin/sh", "-c",
			"i=1; while [ $i -le 200 ]; do printf 'line-%03d\n' $i; i=$((i+1)); done"})
	}()

	// Delay consumption well past the child's exit (but under drainTimeout):
	// the drain must keep the full session in the PTY instead of closing it
	// early and dropping output. The 200 short lines stay within the PTY
	// buffer, so the child never blocks.
	time.Sleep(500 * time.Millisecond)

	_ = b.SetReadDeadline(time.Now().Add(5 * time.Second))
	var buf bytes.Buffer
	tmp := make([]byte, 4096)
	for {
		n, err := b.Read(tmp)
		if n > 0 {
			buf.Write(tmp[:n])
		}
		if err != nil {
			break
		}
	}
	got := buf.String()
	for i := 1; i <= lines; i++ {
		if !strings.Contains(got, fmt.Sprintf("line-%03d", i)) {
			t.Fatalf("line-%03d missing from %d bytes of output: %q", i, len(got), got)
		}
	}

	select {
	case err := <-errc:
		if err != nil {
			t.Errorf("runOnChannel: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("runOnChannel did not return")
	}
}

func requirePTY(t *testing.T) {
	t.Helper()
	if _, err := os.Stat("/dev/ptmx"); err != nil {
		t.Skip("no /dev/ptmx")
	}
	if _, err := os.Stat("/dev/pts"); err != nil {
		t.Skip("no /dev/pts")
	}
}
