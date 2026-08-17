//go:build linux

package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/yaacov/kc-utils/pkg/agent/protocol"
)

const (
	shellHeaderWait  = 50 * time.Millisecond
	shellHeaderRest  = 500 * time.Millisecond
	shellPoll        = 200 * time.Millisecond
	shellPortWaitMax = 50
)

// ServeShell attaches /bin/bash (or a requested command) to the debug
// virtio-serial port. It is best-effort: a missing port (old QEMU cmdline)
// returns so RPC serving can continue.
func ServeShell() error {
	for {
		port, err := openShellPort()
		if err != nil {
			return err
		}
		if err := waitHostConnected(true); err != nil {
			port.Close()
			return err
		}
		_ = runShellSession(port)
		port.Close()
		_ = waitHostConnected(false)
	}
}

func openShellPort() (*os.File, error) {
	path := "/dev/virtio-ports/" + protocol.ShellPortName
	for range shellPortWaitMax {
		f, err := os.OpenFile(path, os.O_RDWR, 0)
		if err == nil {
			return f, nil
		}
		if _, waitErr := os.Stat(path); waitErr == nil {
			return nil, err
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil, fmt.Errorf("shell port %s not found", path)
}

func runShellSession(port *os.File) error {
	cfg, leftover := readShellConfig(port)
	argv := cfg.Command()
	env := shellEnv(cfg.TERM)

	pty, cmd, err := startPTY(argv, env, cfg.Rows, cfg.Cols)
	if err != nil {
		return err
	}
	defer pty.Close()
	if len(leftover) > 0 {
		_, _ = pty.Write(leftover)
	}
	copyShell(port, pty, cmd)
	return nil
}

func shellEnv(term string) []string {
	if term == "" {
		term = "xterm"
	}
	env := os.Environ()
	out := make([]string, 0, len(env)+2)
	for _, e := range env {
		if strings.HasPrefix(e, "TERM=") {
			continue
		}
		out = append(out, e)
	}
	return append(out, "TERM="+term, "HOME=/tmp")
}

func readShellConfig(port *os.File) (protocol.ShellConfig, []byte) {
	_ = port.SetReadDeadline(time.Now().Add(shellHeaderWait))
	defer func() { _ = port.SetReadDeadline(time.Time{}) }()

	first := make([]byte, 1)
	n, err := port.Read(first)
	if n == 0 || err != nil {
		return protocol.ShellConfig{}, nil
	}
	if first[0] != '{' {
		return protocol.ShellConfig{}, first[:n]
	}

	_ = port.SetReadDeadline(time.Now().Add(shellHeaderRest))
	buf := []byte{'{'}
	tmp := make([]byte, 1)
	for {
		n, err := port.Read(tmp)
		if n == 1 {
			if tmp[0] == '\n' {
				break
			}
			buf = append(buf, tmp[0])
			if len(buf) > 4096 {
				return protocol.ShellConfig{}, buf
			}
			continue
		}
		if err != nil {
			return protocol.ShellConfig{}, buf
		}
	}
	var cfg protocol.ShellConfig
	if json.Unmarshal(buf, &cfg) != nil {
		return protocol.ShellConfig{}, buf
	}
	return cfg, nil
}

func copyShell(port, pty *os.File, cmd *exec.Cmd) {
	var once sync.Once
	stop := make(chan struct{})
	done := func() { once.Do(func() { close(stop) }) }

	waitCh := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(waitCh)
		done()
	}()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			select {
			case <-stop:
				return
			default:
			}
			_ = port.SetReadDeadline(time.Now().Add(shellPoll))
			n, err := port.Read(buf)
			if n > 0 {
				if _, werr := pty.Write(buf[:n]); werr != nil {
					done()
					return
				}
			}
			if err == nil {
				continue
			}
			if errors.Is(err, os.ErrDeadlineExceeded) {
				ok, _ := hostConnected()
				if !ok {
					done()
					return
				}
				continue
			}
			done()
			return
		}
	}()
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			select {
			case <-stop:
				return
			default:
			}
			_ = pty.SetReadDeadline(time.Now().Add(shellPoll))
			n, err := pty.Read(buf)
			if n > 0 {
				_ = port.SetWriteDeadline(time.Now().Add(2 * time.Second))
				if _, werr := port.Write(buf[:n]); werr != nil {
					done()
					return
				}
			}
			if err == nil || errors.Is(err, os.ErrDeadlineExceeded) {
				continue
			}
			done()
			return
		}
	}()

	<-stop
	select {
	case <-waitCh:
	default:
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		<-waitCh
	}
	wg.Wait()
}

func waitHostConnected(want bool) error {
	for {
		ok, err := hostConnected()
		if err != nil {
			return err
		}
		if ok == want {
			return nil
		}
		time.Sleep(shellPoll)
	}
}

const virtioPortsSysfs = "/sys/class/virtio-ports"

func hostConnected() (bool, error) {
	entries, err := os.ReadDir(virtioPortsSysfs)
	if err != nil {
		return false, err
	}
	for _, e := range entries {
		dir := virtioPortsSysfs + "/" + e.Name()
		name, err := os.ReadFile(filepath.Join(dir, "name"))
		if err != nil {
			continue
		}
		if strings.TrimSpace(string(name)) != protocol.ShellPortName {
			continue
		}
		hc, err := os.ReadFile(filepath.Join(dir, "host_connected"))
		if err != nil {
			return false, err
		}
		return strings.TrimSpace(string(hc)) == "1", nil
	}
	return false, fmt.Errorf("virtio port %s not found in sysfs", protocol.ShellPortName)
}
