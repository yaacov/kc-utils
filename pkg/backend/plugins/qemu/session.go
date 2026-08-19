//go:build unix

package qemu

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/yaacov/kc-utils/pkg/common/types"
)

// Environment variables for cross-stage VM sharing. kc-v2v boots one appliance
// and exports these so stage subprocesses adopt it instead of booting their own.
const (
	EnvQEMUSock = "KC_QEMU_SOCK"
	EnvQEMUPID  = "KC_QEMU_PID"
)

// vmSession owns a running appliance: the qemu-system process and the agent
// client speaking to it over the unix socket. When ownedExternally is set the
// process belongs to a parent (kc-v2v) and must not be killed on close.
type vmSession struct {
	cmd             *exec.Cmd
	socketPath      string
	client          *agentClient
	drives          []driveSpec
	console         *boundedBuffer
	network         bool          // user-mode networking (Clevis/NBDE); preserved across restart
	exited          chan struct{} // closed by the sole cmd.Wait waiter when the process exits
	ownedExternally bool
}

// envPositiveInt returns the first positive integer among the named env vars,
// or def. Mirrors the guestfs backend's V2V_* resolution shape.
func envPositiveInt(def int, names ...string) int {
	for _, name := range names {
		if v := os.Getenv(name); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				return n
			}
		}
	}
	return def
}

func resolveMemSize() int {
	return envPositiveInt(2048, "V2V_memSize", "LIBGUESTFS_MEMSIZE")
}

func resolveSMP() int {
	smp := runtime.NumCPU()
	smp = min(smp, 8)
	smp = max(smp, 1)
	return envPositiveInt(smp, "V2V_smp", "LIBGUESTFS_SMP")
}

// toDriveSpecs converts prepared disk specs into virtio drives, resolving the
// image format so QEMU never has to auto-probe (a security/robustness hazard).
func toDriveSpecs(disks []types.DiskSpec) []driveSpec {
	out := make([]driveSpec, 0, len(disks))
	for _, d := range disks {
		format := d.Format
		if format == "" {
			format = detectImageFormat(d.Path)
		}
		out = append(out, driveSpec{Path: d.Path, Format: format})
	}
	return out
}

// detectImageFormat sniffs the qcow2 magic, defaulting to raw. Overlays are
// qcow2; original disk images are usually raw.
func detectImageFormat(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return "raw"
	}
	defer f.Close()
	var magic [4]byte
	if _, err := io.ReadFull(f, magic[:]); err != nil {
		return "raw"
	}
	if bytes.Equal(magic[:], []byte{'Q', 'F', 'I', 0xfb}) {
		return "qcow2"
	}
	return "raw"
}

// newVMSession boots a fresh appliance with the given drives attached.
func newVMSession(drives []driveSpec, network bool) (*vmSession, error) {
	s := &vmSession{drives: drives}
	if err := s.boot(network); err != nil {
		return nil, err
	}
	return s, nil
}

// adoptVMSession connects to an appliance already booted by a parent process,
// identified by the KC_QEMU_SOCK / KC_QEMU_PID environment variables. Returns
// (nil, false, nil) when no shared session is advertised.
func adoptVMSession(drives []driveSpec) (*vmSession, bool, error) {
	sock := os.Getenv(EnvQEMUSock)
	if sock == "" {
		return nil, false, nil
	}
	client, err := waitAgentReady(sock, nil, 30*time.Second)
	if err != nil {
		return nil, false, fmt.Errorf("adopt shared appliance (%s): %w", sock, err)
	}
	slog.Info("qemu adopting shared appliance", "socket", sock, "pid", os.Getenv(EnvQEMUPID))
	return &vmSession{
		socketPath:      sock,
		client:          client,
		drives:          drives,
		ownedExternally: true,
	}, true, nil
}

func bootTimeout(accel string) time.Duration {
	if accel == "tcg" {
		return 5 * time.Minute
	}
	return 2 * time.Minute
}

func (s *vmSession) boot(network bool) error {
	s.network = network
	arch := applianceArch()
	kernel, initrd, err := appliancePaths(arch)
	if err != nil {
		return err
	}
	accel := accelFor(runtime.GOOS, hostAccelAvailable())
	socketPath, err := newSocketPath()
	if err != nil {
		return err
	}
	s.socketPath = socketPath

	cfg := launchConfig{
		Binary:     qemuBinary(arch),
		Machine:    machineFor(arch),
		CPU:        cpuFor(arch, accel),
		Accel:      accel,
		MemMiB:     resolveMemSize(),
		SMP:        resolveSMP(),
		Kernel:     kernel,
		Initrd:     initrd,
		Cmdline:    kernelCmdline(arch),
		SocketPath: socketPath,
		Drives:     s.drives,
		Network:    network,
	}
	args := qemuArgs(&cfg)

	s.console = newBoundedBuffer(64 << 10)
	cmd := exec.Command(cfg.Binary, args...)
	cmd.Stdout = s.console
	cmd.Stderr = s.console
	slog.Info("qemu launching appliance",
		"binary", cfg.Binary,
		"accel", accel,
		"machine", cfg.Machine,
		"mem", cfg.MemMiB,
		"smp", cfg.SMP,
		"drives", len(cfg.Drives),
		"socket", socketPath,
	)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", cfg.Binary, err)
	}
	s.cmd = cmd
	// A single goroutine owns cmd.Wait(); it signals exit by closing s.exited.
	// processExited/kill observe the channel instead of touching ProcessState
	// concurrently, which would be a data race.
	s.exited = make(chan struct{})
	exited := s.exited
	go func() {
		_ = cmd.Wait()
		close(exited)
	}()

	client, err := waitAgentReady(socketPath, s.processExited, bootTimeout(accel))
	if err != nil {
		s.kill()
		return fmt.Errorf("qemu appliance boot: %w\n%s", err, s.console.String())
	}
	s.client = client
	slog.Info("qemu appliance ready", "socket", socketPath, "pid", cmd.Process.Pid)
	return nil
}

// processExited reports whether the qemu process has exited, with the captured
// console output for diagnostics. Used to fail boot fast on an early crash.
func (s *vmSession) processExited() (bool, string) {
	if s.exited == nil {
		return false, ""
	}
	select {
	case <-s.exited:
		return true, s.console.String()
	default:
		return false, ""
	}
}

// waitAgentReady dials the socket and pings until the agent answers or the
// timeout elapses. exited (may be nil) lets the caller abort early on crash.
// Each attempt uses a fresh connection so a timed-out ping never corrupts the
// framing of the returned client.
func waitAgentReady(socketPath string, exited func() (bool, string), timeout time.Duration) (*agentClient, error) {
	deadline := time.Now().Add(timeout)
	for {
		if exited != nil {
			if done, out := exited(); done {
				return nil, fmt.Errorf("appliance process exited during boot:\n%s", out)
			}
		}
		if conn, err := net.Dial("unix", socketPath); err == nil {
			c := &agentClient{conn: conn}
			if perr := c.pingReady(3 * time.Second); perr == nil {
				return c, nil
			}
			_ = c.close()
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("agent not ready on %s after %s", socketPath, timeout)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// newSocketPath returns a unix socket path inside a fresh private (0700)
// directory. QEMU creates the socket itself, so only the directory is
// pre-created; kill removes the whole directory. It stays under /tmp and keeps
// the name short to respect the ~104-byte sun_path limit on macOS.
func newSocketPath() (string, error) {
	dir, err := os.MkdirTemp("/tmp", "kc-qemu-*")
	if err != nil {
		return "", fmt.Errorf("allocate agent socket dir: %w", err)
	}
	return filepath.Join(dir, "agent.sock"), nil
}

// alive reports whether the appliance is still usable.
func (s *vmSession) alive() bool {
	if s == nil || s.client == nil {
		return false
	}
	if s.ownedExternally {
		return s.client.ping() == nil
	}
	if done, _ := s.processExited(); done {
		return false
	}
	return s.client.ping() == nil
}

// restart boots a fresh appliance in place of a dead one. The caller must
// re-mount guest filesystems afterwards.
func (s *vmSession) restart() error {
	if s.ownedExternally {
		return fmt.Errorf("cannot restart externally-owned appliance")
	}
	slog.Warn("qemu appliance restart")
	s.kill()
	// Preserve the networking setting so a Clevis/NBDE appliance keeps its
	// netdev after a crash restart.
	return s.boot(s.network)
}

// kill terminates the qemu process and removes the socket file. No-op for an
// externally-owned session (the parent owns the process lifecycle).
func (s *vmSession) kill() {
	if s.client != nil {
		_ = s.client.close()
		s.client = nil
	}
	if s.ownedExternally {
		return
	}
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Signal(syscall.SIGTERM)
		// The boot-time goroutine owns cmd.Wait(); observe its exit signal
		// instead of calling Wait() a second time (which races it).
		select {
		case <-s.exited:
		case <-time.After(5 * time.Second):
			_ = s.cmd.Process.Kill()
			<-s.exited // let the waiter reap the killed process
		}
	}
	if s.socketPath != "" {
		// Remove the socket and its private parent directory (see newSocketPath).
		_ = os.RemoveAll(filepath.Dir(s.socketPath))
	}
}

// close shuts the session down. For an owned session it powers off the VM; for
// an adopted one it only drops the client connection.
func (s *vmSession) close() error {
	if s == nil {
		return nil
	}
	if s.ownedExternally {
		if s.client != nil {
			err := s.client.close()
			s.client = nil
			return err
		}
		return nil
	}
	s.kill()
	return nil
}

// sharedEnv returns the env assignments a parent exports so child stages adopt
// this appliance.
func (s *vmSession) sharedEnv() []string {
	pid := 0
	if s.cmd != nil && s.cmd.Process != nil {
		pid = s.cmd.Process.Pid
	}
	return []string{
		EnvQEMUSock + "=" + s.socketPath,
		EnvQEMUPID + "=" + strconv.Itoa(pid),
		"V2V_memSize=" + strconv.Itoa(resolveMemSize()),
		"V2V_smp=" + strconv.Itoa(resolveSMP()),
	}
}

// boundedBuffer is a concurrency-safe rolling buffer that keeps only the last
// max bytes written, used to retain recent console output for diagnostics.
type boundedBuffer struct {
	mu  sync.Mutex
	max int
	buf bytes.Buffer
}

func newBoundedBuffer(max int) *boundedBuffer {
	return &boundedBuffer{max: max}
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := len(p)
	b.buf.Write(p)
	if b.buf.Len() > b.max {
		tail := b.buf.Bytes()[b.buf.Len()-b.max:]
		next := bytes.NewBuffer(append([]byte(nil), tail...))
		b.buf = *next
	}
	return n, nil
}

func (b *boundedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// applianceMountPath rebases a host mount path (under mountRoot) to its
// appliance-side location under applianceMountRoot.
func applianceMountPath(mountRoot, hostMountPoint string) string {
	if hostMountPoint == mountRoot || mountRoot == "" {
		return applianceMountRoot
	}
	rel, err := filepath.Rel(mountRoot, hostMountPoint)
	if err != nil || rel == "." {
		return applianceMountRoot
	}
	return filepath.Join(applianceMountRoot, filepath.ToSlash(rel))
}

// guestToAppliance maps a guest-absolute path (e.g. /etc/fstab) to its
// appliance-side path under the mount root. The guest path is first anchored at
// "/" and cleaned, so any leading or embedded ".." cannot resolve above the
// mount root: filepath.Clean("/../x") == "/x", keeping the result under
// applianceMountRoot.
func guestToAppliance(guestPath string) string {
	confined := filepath.Clean("/" + filepath.FromSlash(guestPath))
	return filepath.Join(applianceMountRoot, confined)
}
