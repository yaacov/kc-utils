//go:build unix

package agentsh

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/yaacov/kc-utils/pkg/agent/protocol"
	"github.com/yaacov/kc-utils/pkg/guest/backend"
	"golang.org/x/term"
)

const dialTimeout = 10 * time.Second

// Config is the kc-agent-sh CLI contract.
type Config struct {
	Sock   string
	Chroot string
	Argv   []string
}

// Run dials the appliance debug shell and copies the local TTY until exit.
func Run(cfg Config) error {
	sock, err := ResolveSock(cfg.Sock)
	if err != nil {
		return err
	}
	if _, err := os.Stat(sock); err != nil {
		return fmt.Errorf("shell socket %s: %w (is the qemu appliance running? rebuild with make appliance after upgrading kc-agent)", sock, err)
	}
	conn, err := dialShell(sock, dialTimeout)
	if err != nil {
		return err
	}
	defer conn.Close()

	header, err := json.Marshal(shellConfig(cfg))
	if err != nil {
		return err
	}
	if _, err := conn.Write(append(header, '\n')); err != nil {
		return fmt.Errorf("writing shell header: %w", err)
	}

	in := int(os.Stdin.Fd())
	if term.IsTerminal(in) {
		old, err := term.MakeRaw(in)
		if err != nil {
			return fmt.Errorf("terminal raw mode: %w", err)
		}
		defer func() { _ = term.Restore(in, old) }()
	}

	errCh := make(chan error, 2)
	go func() {
		_, err := io.Copy(conn, os.Stdin)
		errCh <- err
	}()
	go func() {
		_, err := io.Copy(os.Stdout, conn)
		errCh <- err
	}()
	if err := <-errCh; err != nil && err != io.EOF {
		return err
	}
	return nil
}

func shellConfig(cfg Config) protocol.ShellConfig {
	sc := protocol.ShellConfig{
		Chroot: cfg.Chroot,
		Argv:   cfg.Argv,
		TERM:   os.Getenv("TERM"),
	}
	if term.IsTerminal(int(os.Stdin.Fd())) {
		if w, h, err := term.GetSize(int(os.Stdin.Fd())); err == nil {
			sc.Cols = uint16(w)
			sc.Rows = uint16(h)
		}
	}
	return sc
}

func dialShell(sock string, timeout time.Duration) (net.Conn, error) {
	deadline := time.Now().Add(timeout)
	var last error
	for time.Now().Before(deadline) {
		c, err := net.Dial("unix", sock)
		if err == nil {
			return c, nil
		}
		last = err
		time.Sleep(100 * time.Millisecond)
	}
	return nil, fmt.Errorf("dialing shell socket %s: %w (qemu backend not running, or appliance not rebuilt)", sock, last)
}

// ResolveSock maps --sock or KC_AGENT_SOCK to the debug-shell Unix socket.
// Passing the agent RPC socket (basename agent.sock) derives the sibling
// shell.sock; any other explicit path is used as-is.
func ResolveSock(explicit string) (string, error) {
	if explicit != "" {
		if filepath.Base(explicit) == "agent.sock" {
			return protocol.ShellSock(explicit), nil
		}
		return explicit, nil
	}
	env := os.Getenv(backend.EnvAgentSock)
	if env == "" {
		return "", fmt.Errorf("no shell socket: set --sock or %s (qemu backend not running, or appliance not rebuilt)", backend.EnvAgentSock)
	}
	sock := protocol.ShellSock(env)
	if sock == "" {
		return "", fmt.Errorf("no shell socket: set --sock or %s (qemu backend not running, or appliance not rebuilt)", backend.EnvAgentSock)
	}
	return sock, nil
}
