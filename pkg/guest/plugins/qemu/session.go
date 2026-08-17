//go:build unix

package qemu

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/yaacov/kc-utils/pkg/common/types"
	"github.com/yaacov/kc-utils/pkg/guest/backend"
)

const qemuPIDFile = "qemu.pid"

// SharedSession is a QEMU + kc-agent process owned by kc-v2v.
// StartSharedSession only reserves a socket path; prepare Setup launches QEMU
// with guest disks and writes qemu.pid next to the socket.
type SharedSession struct {
	Sock  string
	dir   string
	owned bool
}

func (s *SharedSession) pid() int {
	if s == nil {
		return 0
	}
	b, err := os.ReadFile(filepath.Join(filepath.Dir(s.Sock), qemuPIDFile))
	if err != nil {
		return 0
	}
	n, _ := strconv.Atoi(strings.TrimSpace(string(b)))
	return n
}

func (s *SharedSession) Close() error {
	if s == nil || !s.owned {
		return nil
	}
	pid := s.pid()
	if pid > 0 {
		proc, err := os.FindProcess(pid)
		if err == nil {
			_ = proc.Signal(syscall.SIGTERM)
			deadline := time.Now().Add(5 * time.Second)
			for time.Now().Before(deadline) {
				if proc.Signal(syscall.Signal(0)) != nil {
					break
				}
				time.Sleep(100 * time.Millisecond)
			}
			_ = proc.Kill()
		}
	}
	if s.Sock != "" {
		_ = os.Remove(s.Sock)
	}
	if s.dir != "" {
		_ = os.RemoveAll(s.dir)
	}
	return nil
}

func (s *SharedSession) Alive() bool {
	if s == nil {
		return false
	}
	pid := s.pid()
	if pid <= 0 {
		// Socket reserved; QEMU starts at prepare Setup.
		return true
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

func (s *SharedSession) Env() []string {
	if s == nil || s.Sock == "" {
		return nil
	}
	env := []string{
		fmt.Sprintf("%s=%s", backend.EnvAgentSock, s.Sock),
	}
	if pid := s.pid(); pid > 0 {
		env = append(env, fmt.Sprintf("%s=%d", backend.EnvQemuPID, pid))
	}
	return env
}

func startQEMU(disks []types.DiskSpec, sock string) (*exec.Cmd, error) {
	kernel, initrd, assets, err := resolveArtifacts()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(sock), 0o700); err != nil {
		return nil, err
	}
	cfg := LaunchConfig{
		Arch:     defaultArch(),
		Accel:    defaultAccel(),
		Kernel:   kernel,
		Initrd:   initrd,
		Assets:   assets,
		Socket:   sock,
		Disks:    disks,
		MemoryMB: resolveMemSize(),
		SMP:      resolveSMP(),
		Network:  guestfsNetworkEnabled(),
	}
	bin, args, err := BuildQEMUArgs(&cfg)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(bin, args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting %s: %w", bin, err)
	}
	pidPath := filepath.Join(filepath.Dir(sock), qemuPIDFile)
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(cmd.Process.Pid)), 0o600); err != nil {
		slog.Warn("writing qemu pidfile", "path", pidPath, "error", err)
	}
	slog.Info("qemu appliance started", "pid", cmd.Process.Pid, "bin", bin, "accel", cfg.Accel, "arch", cfg.Arch)
	return cmd, nil
}

func waitAgent(sock string, timeout time.Duration) (*client, error) {
	deadline := time.Now().Add(timeout)
	var last error
	for time.Now().Before(deadline) {
		cl, err := dialAgent(sock)
		if err == nil {
			return cl, nil
		}
		last = err
		time.Sleep(200 * time.Millisecond)
	}
	return nil, fmt.Errorf("waiting for kc-agent at %s: %w", sock, last)
}

func agentSockFromEnv() string {
	return os.Getenv(backend.EnvAgentSock)
}
