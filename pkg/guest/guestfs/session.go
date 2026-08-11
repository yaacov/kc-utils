//go:build linux

package guestfs

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	EnvGuestfishPID   = "GUESTFISH_PID"
	EnvKCGuestfishPID = "KC_GUESTFISH_PID"
	// EnvGuestfsNetwork enables QEMU user networking in the appliance before
	// launch. Set to "1" or "true" when Clevis/NBDE unlock is required
	// (Forklift V2V_NBDE_CLEVIS). Must be set before ensureLaunched / run.
	EnvGuestfsNetwork = "KC_GUESTFS_NETWORK"
)

func guestfsNetworkEnabled() bool {
	v := strings.TrimSpace(os.Getenv(EnvGuestfsNetwork))
	return v == "1" || strings.EqualFold(v, "true")
}

var guestfishPIDRE = regexp.MustCompile(`GUESTFISH_PID=(\d+)`)

type guestfishSession struct {
	pid             int
	ownedExternally bool
	launched        bool
}

// SharedListener is a guestfish --listen process owned by the caller (kc-v2v).
// Stages adopt it via GUESTFISH_PID / KC_GUESTFISH_PID.
type SharedListener struct {
	PID int
}

const guestfishHome = "/var/tmp"

// StartSharedListener starts guestfish --listen and returns a handle. The
// caller must Close it.
func StartSharedListener() (*SharedListener, error) {
	if err := ensureGuestfishEnv(); err != nil {
		return nil, err
	}
	pid, err := startGuestfishListen()
	if err != nil {
		return nil, err
	}
	slog.Info("guestfish shared listener started", "pid", pid)
	return &SharedListener{PID: pid}, nil
}

// envPositiveInt returns the first positive integer found among the given env
// var names, or def if none is set / valid.  Forklift sets V2V_memSize and
// V2V_smp; LIBGUESTFS_MEMSIZE / LIBGUESTFS_SMP are the underlying libguestfs
// knobs.  Checking the V2V_ name first keeps the canonical interface while
// still honouring a direct libguestfs override.
func envPositiveInt(def int, names ...string) int {
	for _, name := range names {
		v := os.Getenv(name)
		if v == "" {
			continue
		}
		n, err := strconv.Atoi(v)
		if err == nil && n > 0 {
			return n
		}
	}
	return def
}

func resolveMemSize() int {
	return envPositiveInt(2048, "V2V_memSize", "LIBGUESTFS_MEMSIZE")
}

func resolveSMP() int {
	smp := runtime.NumCPU()
	if smp > 8 {
		smp = 8
	}
	if smp < 1 {
		smp = 1
	}
	return envPositiveInt(smp, "V2V_smp", "LIBGUESTFS_SMP")
}

// Env returns environment assignments for stage subprocesses.
func (l *SharedListener) Env() []string {
	if l == nil || l.PID <= 0 {
		return nil
	}
	return []string{
		fmt.Sprintf("%s=%d", EnvGuestfishPID, l.PID),
		fmt.Sprintf("%s=%d", EnvKCGuestfishPID, l.PID),
		"LIBGUESTFS_BACKEND=direct",
		"LIBGUESTFS_MEMSIZE=" + strconv.Itoa(resolveMemSize()),
		"LIBGUESTFS_SMP=" + strconv.Itoa(resolveSMP()),
		"HOME=" + guestfishHome,
	}
}

func ensureGuestfishEnv() error {
	if err := os.Setenv("LIBGUESTFS_BACKEND", "direct"); err != nil {
		return fmt.Errorf("setting LIBGUESTFS_BACKEND: %w", err)
	}
	if err := os.Setenv("LIBGUESTFS_MEMSIZE", strconv.Itoa(resolveMemSize())); err != nil {
		return fmt.Errorf("setting LIBGUESTFS_MEMSIZE: %w", err)
	}
	if err := os.Setenv("LIBGUESTFS_SMP", strconv.Itoa(resolveSMP())); err != nil {
		return fmt.Errorf("setting LIBGUESTFS_SMP: %w", err)
	}
	home := os.Getenv("HOME")
	if home == "" || home == "/" {
		if err := os.Setenv("HOME", guestfishHome); err != nil {
			return fmt.Errorf("setting HOME: %w", err)
		}
		slog.Info("guestfish HOME adjusted for writable history",
			"from", home,
			"to", guestfishHome,
			"uid", os.Getuid(),
		)
	}
	return nil
}

// Alive reports whether the listener's guestfish process is still running.
func (l *SharedListener) Alive() bool {
	return l != nil && l.PID > 0 && sessionAlive(l.PID)
}

// Close stops the listener via guestfish --remote exit.
func (l *SharedListener) Close() error {
	if l == nil || l.PID <= 0 {
		return nil
	}
	pid := l.PID
	l.PID = 0
	slog.Info("guestfish shared listener exit", "pid", pid)
	_, err := runGuestfsCmd(guestfishBinary(), fmt.Sprintf("--remote=%d", pid), "--", "exit")
	if err != nil {
		_ = syscall.Kill(pid, syscall.SIGTERM)
	}
	return err
}

func openGuestfishSession() (*guestfishSession, error) {
	if err := ensureGuestfishEnv(); err != nil {
		return nil, err
	}

	if pid, ok := envGuestfishPID(); ok {
		if !sessionAlive(pid) {
			return nil, fmt.Errorf("guestfish shared session pid %d is not running (socket %s)", pid, guestfishSocketPath(pid))
		}
		slog.Info("guestfish adopting shared session",
			"pid", pid,
			"guestfishPID", pid,
			"shared", true,
			"HOME", os.Getenv("HOME"),
			"socket", guestfishSocketPath(pid),
		)
		return &guestfishSession{pid: pid, ownedExternally: true}, nil
	}

	pid, err := startGuestfishListen()
	if err != nil {
		return nil, err
	}
	slog.Info("guestfish local listener started", "pid", pid, "HOME", os.Getenv("HOME"))
	return &guestfishSession{pid: pid, ownedExternally: false}, nil
}

func envGuestfishPID() (int, bool) {
	for _, key := range []string{EnvKCGuestfishPID, EnvGuestfishPID} {
		v := strings.TrimSpace(os.Getenv(key))
		if v == "" {
			continue
		}
		pid, err := strconv.Atoi(v)
		if err != nil || pid <= 0 {
			continue
		}
		return pid, true
	}
	return 0, false
}

const guestfishListenTimeout = 30 * time.Second

func startGuestfishListen() (int, error) {
	bin := guestfishBinary()
	cmd := exec.Command(bin, "--listen")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 0, fmt.Errorf("guestfish --listen stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return 0, fmt.Errorf("guestfish --listen stderr pipe: %w", err)
	}

	// Drain stderr into a rolling buffer for startup failure messages.
	var stderrMu sync.Mutex
	var stderrBuf bytes.Buffer
	go drainGuestfishListenStderr(stderr, &stderrMu, &stderrBuf)

	slog.Info("guestfish listen starting", "bin", bin)
	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("guestfish --listen start: %w", err)
	}
	go func() { _, _ = cmd.Process.Wait() }()

	pidCh := make(chan int, 1)
	errCh := make(chan error, 1)
	go func() {
		pid, err := drainGuestfishListenStdout(stdout, pidCh)
		if err != nil {
			errCh <- err
			return
		}
		if pid == 0 {
			errCh <- fmt.Errorf("no GUESTFISH_PID in stdout")
		}
	}()

	stderrSnapshot := func() string {
		stderrMu.Lock()
		defer stderrMu.Unlock()
		return strings.TrimSpace(stderrBuf.String())
	}

	ctx, cancel := context.WithTimeout(context.Background(), guestfishListenTimeout)
	defer cancel()

	var pid int
	select {
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		msg := stderrSnapshot()
		if msg != "" {
			return 0, fmt.Errorf("guestfish --listen: timed out waiting for GUESTFISH_PID after %s\n%s", guestfishListenTimeout, msg)
		}
		return 0, fmt.Errorf("guestfish --listen: timed out waiting for GUESTFISH_PID after %s", guestfishListenTimeout)
	case err := <-errCh:
		_ = cmd.Process.Kill()
		msg := stderrSnapshot()
		if msg != "" {
			return 0, fmt.Errorf("guestfish --listen: %w\n%s", err, msg)
		}
		return 0, fmt.Errorf("guestfish --listen: %w", err)
	case pid = <-pidCh:
	}

	deadline := time.Now().Add(5 * time.Second)
	for {
		if sessionAlive(pid) {
			slog.Info("guestfish listen socket ready", "pid", pid, "socket", guestfishSocketPath(pid), "bin", bin)
			return pid, nil
		}
		if time.Now().After(deadline) {
			_ = syscall.Kill(pid, syscall.SIGTERM)
			msg := stderrSnapshot()
			if msg != "" {
				return 0, fmt.Errorf("guestfish --listen: PID %d socket not ready at %s\n%s", pid, guestfishSocketPath(pid), msg)
			}
			return 0, fmt.Errorf("guestfish --listen: PID %d socket not ready at %s", pid, guestfishSocketPath(pid))
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// drainGuestfishListenStderr keeps a rolling buffer for startup error messages.
func drainGuestfishListenStderr(r io.Reader, mu *sync.Mutex, buf *bytes.Buffer) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if mu == nil || buf == nil {
			continue
		}
		mu.Lock()
		buf.WriteString(line)
		buf.WriteByte('\n')
		if buf.Len() > 64*1024 {
			b := buf.Bytes()
			buf.Reset()
			buf.Write(b[len(b)-32*1024:])
		}
		mu.Unlock()
	}
	if err := sc.Err(); err != nil {
		slog.Debug("guestfish listen stderr drain ended", "error", err)
	}
}

func parseListenOutput(out string) (int, error) {
	m := guestfishPIDRE.FindStringSubmatch(out)
	if len(m) < 2 {
		return 0, fmt.Errorf("guestfish --listen: no GUESTFISH_PID in output %q", strings.TrimSpace(out))
	}
	pid, err := strconv.Atoi(m[1])
	if err != nil || pid <= 0 {
		return 0, fmt.Errorf("guestfish --listen: invalid PID %q", m[1])
	}
	return pid, nil
}

func drainGuestfishListenStdout(r io.Reader, pidCh chan<- int) (int, error) {
	buf := make([]byte, 0, 256)
	tmp := make([]byte, 256)
	sent := false
	var pid int
	for {
		n, err := r.Read(tmp)
		if n > 0 {
			if !sent {
				buf = append(buf, tmp[:n]...)
				if p, perr := parseListenOutput(string(buf)); perr == nil {
					pid = p
					sent = true
					pidCh <- p
					buf = nil
				}
			}
		}
		if err != nil {
			if err == io.EOF {
				if !sent {
					return 0, fmt.Errorf("no GUESTFISH_PID in output %q", strings.TrimSpace(string(buf)))
				}
				return pid, nil
			}
			if !sent {
				return 0, err
			}
			return pid, nil
		}
	}
}

func guestfishSocketPath(pid int) string {
	return fmt.Sprintf("/tmp/.guestfish-%d/socket-%d", os.Getuid(), pid)
}

func sessionAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	if err := syscall.Kill(pid, 0); err != nil {
		return false
	}
	_, err := os.Stat(guestfishSocketPath(pid))
	return err == nil
}

func (s *guestfishSession) remoteFlag() string {
	return fmt.Sprintf("--remote=%d", s.pid)
}

func (s *guestfishSession) ensureLaunched(diskPaths []string) error {
	if s == nil {
		return fmt.Errorf("guestfish session is nil")
	}
	if s.launched {
		return nil
	}

	slog.Info("guestfish ensureLaunched",
		"pid", s.pid,
		"disks", len(diskPaths),
		"paths", diskPaths,
		"HOME", os.Getenv("HOME"),
		"LIBGUESTFS_BACKEND", os.Getenv("LIBGUESTFS_BACKEND"),
		"socket", guestfishSocketPath(s.pid),
		"sessionAlive", sessionAlive(s.pid),
	)

	if ready, probeOut := s.probeReady(); ready {
		s.launched = true
		slog.Info("guestfish session already launched", "pid", s.pid, "devices", strings.TrimSpace(probeOut))
		return nil
	}

	var script strings.Builder
	network := guestfsNetworkEnabled()
	if network {
		script.WriteString("set-network true\n")
		slog.Info("guestfish enabling appliance network for Clevis/NBDE", "pid", s.pid)
	}
	for _, dp := range diskPaths {
		script.WriteString("-add-drive-opts ")
		script.WriteString(quoteGuestfish(dp))
		script.WriteString(" readonly:false\n")
	}
	script.WriteString("-run\n")
	slog.Info("guestfish launching appliance", "pid", s.pid, "disks", len(diskPaths), "network", network)
	out, err := s.remoteScript(script.String())
	launchOut := strings.TrimSpace(string(out))
	if err != nil {
		return fmt.Errorf("guestfish launch: %w", err)
	}
	if launchOut != "" {
		slog.Info("guestfish launch output", "output", launchOut)
	} else {
		slog.Info("guestfish launch script finished", "pid", s.pid, "sessionAlive", sessionAlive(s.pid))
	}

	ready, probeOut := s.probeReady()
	if !ready {
		alive := sessionAlive(s.pid)
		slog.Error("guestfish appliance not ready after run",
			"pid", s.pid,
			"sessionAlive", alive,
			"socket", guestfishSocketPath(s.pid),
			"launchOutput", launchOut,
			"probeOutput", strings.TrimSpace(probeOut),
			"HOME", os.Getenv("HOME"),
		)
		if !alive {
			return fmt.Errorf("guestfish launch: listen server pid %d died during launch", s.pid)
		}
		if probeOut != "" {
			return fmt.Errorf("guestfish launch: appliance not ready after run (probe=%q)", strings.TrimSpace(probeOut))
		}
		return fmt.Errorf("guestfish launch: appliance not ready after run")
	}
	s.launched = true
	slog.Info("guestfish appliance launched",
		"pid", s.pid,
		"disks", len(diskPaths),
		"devices", strings.TrimSpace(probeOut),
	)
	return nil
}

func (s *guestfishSession) probeReady() (bool, string) {
	out, err := s.remoteScript("-list-devices\n")
	msg := string(out)
	if err != nil {
		slog.Info("guestfish ready probe failed", "pid", s.pid, "error", err, "output", strings.TrimSpace(msg))
		return false, msg
	}
	ready := strings.Contains(msg, "/dev/")
	slog.Info("guestfish ready probe",
		"pid", s.pid,
		"ready", ready,
		"devices", strings.TrimSpace(msg),
	)
	return ready, msg
}

func (s *guestfishSession) remote(args ...string) ([]byte, error) {
	if err := s.checkAlive(); err != nil {
		return nil, err
	}
	all := make([]string, 0, 2+len(args))
	all = append(all, s.remoteFlag(), "--")
	all = append(all, args...)
	return runGuestfsCmd(guestfishBinary(), all...)
}

func (s *guestfishSession) remoteScript(script string) ([]byte, error) {
	if err := s.checkAlive(); err != nil {
		return nil, err
	}
	return runGuestfishScript([]string{s.remoteFlag()}, script)
}

// remoteScriptSoft runs a guestfish script without failing on libguestfs
// error lines from '-'-prefixed commands (listener stays alive either way).
func (s *guestfishSession) remoteScriptSoft(script string) ([]byte, error) {
	if err := s.checkAlive(); err != nil {
		return nil, err
	}
	return runGuestfishScriptSoft([]string{s.remoteFlag()}, script)
}

// checkAlive returns an error if the guestfish session process is no longer
// running. This catches a crashed appliance early, before attempting a
// command against a dead socket.
func (s *guestfishSession) checkAlive() error {
	if s == nil || s.pid <= 0 {
		return fmt.Errorf("guestfish session is nil or has no PID")
	}
	if !sessionAlive(s.pid) {
		return fmt.Errorf("guestfish session pid %d is not running (socket %s)", s.pid, guestfishSocketPath(s.pid))
	}
	return nil
}

// restart starts a fresh guestfish --listen process, replacing the current
// (presumably dead) session. The caller must re-launch the appliance and
// re-mount filesystems after restart.
func (s *guestfishSession) restart(diskPaths []string) error {
	oldPid := s.pid
	slog.Warn("guestfish session restart", "oldPid", oldPid)
	pid, err := startGuestfishListen()
	if err != nil {
		return fmt.Errorf("guestfish restart: %w", err)
	}
	s.pid = pid
	s.launched = false
	s.ownedExternally = false
	slog.Info("guestfish session restarted", "oldPid", oldPid, "newPid", pid)
	return s.ensureLaunched(diskPaths)
}

func (s *guestfishSession) close() error {
	if s == nil || s.pid <= 0 {
		return nil
	}
	if s.ownedExternally {
		slog.Debug("guestfish session externally owned; skip exit", "pid", s.pid)
		s.pid = 0
		return nil
	}
	pid := s.pid
	s.pid = 0
	s.launched = false
	slog.Info("guestfish local listener exit", "pid", pid)
	_, err := runGuestfsCmd(guestfishBinary(), fmt.Sprintf("--remote=%d", pid), "--", "exit")
	return err
}
