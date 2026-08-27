//go:build linux

package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"
	"unsafe"
)

// debugPortName is the virtio-serial port QEMU bridges to debug.sock. Keep in
// sync with pkg/backend/plugins/qemu.debugPortName.
const debugPortName = "org.kc-utils.debug"

// Linux ioctl numbers for /dev/ptmx (same encoding on amd64 and arm64).
const (
	ioctlTIOCGPTN   = 0x80045430 // _IOR('T', 0x30, unsigned int)
	ioctlTIOCSPTLCK = 0x40045431 // _IOW('T', 0x31, int)

	// poll(2) events (linux/poll.h). The syscall package only exports EPOLL*.
	pollIn  = 0x1
	pollOut = 0x4
	pollHup = 0x10

	// drainTimeout bounds the post-exit PTY drain in runOnChannel. Normally
	// the last slave close unblocks the drain within milliseconds; the bound
	// protects against a slow consumer or a stray slave holder stalling the
	// channel indefinitely.
	drainTimeout = 5 * time.Second
)

// serveChannel binds a virtio-serial port to an interactive app (getty-style):
// wait until a host client is connected, run argv on a PTY, and loop when the
// process exits or the host disconnects. Never returns; intended to run as a
// PID-1 goroutine. Bash is not started while debug.sock has no client (an
// unconnected virtio-serial read is EOF and would otherwise spawn/kill a
// shell every 200ms).
func serveChannel(portName string, argv []string) {
	for {
		if err := serveChannelOnce(portName, argv); err != nil {
			fmt.Fprintf(os.Stderr, "kc-guest-agent: channel %s: %v\n", portName, err)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func serveChannelOnce(portName string, argv []string) error {
	dev, err := resolvePort("/dev/virtio-ports/" + portName)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(dev, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("open %s: %w", dev, err)
	}
	if err := waitHostConnected(f); err != nil {
		_ = f.Close()
		return err
	}
	return runOnChannel(f, argv)
}

// waitHostConnected blocks until the virtio-serial host is attached.
// Linux virtio_console sets POLLHUP while host_connected is false; QEMU's
// unix chardev (server=on,wait=off) stays in that state until nc/socat
// connects to debug.sock.
func waitHostConnected(f *os.File) error {
	fd := int(f.Fd())
	for {
		revents, err := pollFD(fd, pollIn|pollOut|pollHup, 1000)
		if err != nil {
			if err == syscall.EINTR {
				continue
			}
			return fmt.Errorf("poll debug port: %w", err)
		}
		if hostConnected(revents) {
			return nil
		}
	}
}

func hostConnected(revents int16) bool {
	return revents != 0 && revents&pollHup == 0
}

type pollFd struct {
	Fd      int32
	Events  int16
	Revents int16
}

func pollFD(fd int, events int16, timeoutMs int) (int16, error) {
	pfd := pollFd{Fd: int32(fd), Events: events}
	ts := syscall.Timespec{Sec: int64(timeoutMs / 1000), Nsec: int64(timeoutMs%1000) * 1e6}
	_, _, errno := syscall.Syscall6(
		syscall.SYS_PPOLL,
		uintptr(unsafe.Pointer(&pfd)),
		1,
		uintptr(unsafe.Pointer(&ts)),
		0,
		0,
		0,
	)
	if errno != 0 {
		return 0, errno
	}
	return pfd.Revents, nil
}

// runOnChannel takes ownership of rw, runs argv with a PTY whose master is
// copied to/from rw, and closes rw before returning.
func runOnChannel(rw io.ReadWriteCloser, argv []string) error {
	defer rw.Close()
	if len(argv) == 0 {
		return fmt.Errorf("empty argv")
	}

	master, slave, err := openPTY()
	if err != nil {
		return err
	}

	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdin = slave
	cmd.Stdout = slave
	cmd.Stderr = slave
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		// ANSI colors pass through socat raw; bash -i has no /root/.bashrc here.
		`PS1=\[\e[01;32m\]\u@\h\[\e[00m\]:\[\e[01;34m\]\w\[\e[00m\]# `,
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid:  true,
		Setctty: true,
		Ctty:    0, // child's stdin is the PTY slave
	}
	if err := cmd.Start(); err != nil {
		_ = slave.Close()
		return fmt.Errorf("start %s: %w", argv[0], err)
	}
	_ = slave.Close() // child holds the slave; parent only copies via master

	inDone := make(chan struct{})
	go func() {
		defer close(inDone)
		_, _ = io.Copy(master, rw)
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}()
	outDone := make(chan struct{})
	go func() {
		defer close(outDone)
		_, _ = io.Copy(rw, master)
	}()

	waitErr := cmd.Wait()
	// The child is gone, so its copy of the PTY slave is closed too and the
	// pending output drains quickly; the bound below only covers a slow
	// consumer or a descendant still holding a slave. Until the drain
	// finishes (or times out), the PTY keeps every byte of the session.
	select {
	case <-outDone:
	case <-time.After(drainTimeout):
	}
	_ = rw.Close()
	_ = master.Close()
	select {
	case <-inDone:
	case <-time.After(drainTimeout):
	}
	return waitErr
}

func openPTY() (master, slave *os.File, err error) {
	master, err = os.OpenFile("/dev/ptmx", os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("open /dev/ptmx: %w", err)
	}
	var unlock int32
	if err := ioctl(master.Fd(), ioctlTIOCSPTLCK, uintptr(unsafe.Pointer(&unlock))); err != nil {
		_ = master.Close()
		return nil, nil, fmt.Errorf("unlock ptmx: %w", err)
	}
	var n uint32
	if err := ioctl(master.Fd(), ioctlTIOCGPTN, uintptr(unsafe.Pointer(&n))); err != nil {
		_ = master.Close()
		return nil, nil, fmt.Errorf("ptmx pty number: %w", err)
	}
	slavePath := fmt.Sprintf("/dev/pts/%d", n)
	slave, err = os.OpenFile(slavePath, os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		_ = master.Close()
		return nil, nil, fmt.Errorf("open %s: %w", slavePath, err)
	}
	return master, slave, nil
}

func ioctl(fd, req, arg uintptr) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, req, arg)
	if errno != 0 {
		return errno
	}
	return nil
}
