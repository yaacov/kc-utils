//go:build linux

package agent

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"golang.org/x/sys/unix"
)

func openPTY() (pty, tty *os.File, err error) {
	pty, err = os.OpenFile("/dev/ptmx", os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		return nil, nil, err
	}
	if err := unix.IoctlSetPointerInt(int(pty.Fd()), unix.TIOCSPTLCK, 0); err != nil {
		pty.Close()
		return nil, nil, err
	}
	n, err := unix.IoctlGetInt(int(pty.Fd()), unix.TIOCGPTN)
	if err != nil {
		pty.Close()
		return nil, nil, err
	}
	tty, err = os.OpenFile(fmt.Sprintf("/dev/pts/%d", n), os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		pty.Close()
		return nil, nil, err
	}
	return pty, tty, nil
}

func setWinsize(pty *os.File, rows, cols uint16) {
	if rows == 0 {
		rows = 24
	}
	if cols == 0 {
		cols = 80
	}
	_ = unix.IoctlSetWinsize(int(pty.Fd()), unix.TIOCSWINSZ, &unix.Winsize{Row: rows, Col: cols})
}

func startPTY(argv []string, env []string, rows, cols uint16) (*os.File, *exec.Cmd, error) {
	if len(argv) == 0 {
		return nil, nil, fmt.Errorf("empty argv")
	}
	pty, tty, err := openPTY()
	if err != nil {
		return nil, nil, err
	}
	setWinsize(pty, rows, cols)

	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdin = tty
	cmd.Stdout = tty
	cmd.Stderr = tty
	cmd.Env = env
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid:  true,
		Setctty: true,
		Ctty:    0,
	}
	if err := cmd.Start(); err != nil {
		tty.Close()
		pty.Close()
		return nil, nil, err
	}
	tty.Close()
	return pty, cmd, nil
}
