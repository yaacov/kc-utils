//go:build linux

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

const binfmtMiscDir = "/proc/sys/fs/binfmt_misc"

var binfmtConfDirs = []string{
	"/usr/lib/binfmt.d",
	"/etc/binfmt.d",
}

// registerBinfmt mounts binfmt_misc and registers F-flag qemu-user-static
// interpreters so chrooted foreign-ISA guest ELFs run via the appliance
// emulator. No-op when qemu-user-static is not packaged. Fatal when it is
// packaged but nothing could be registered — otherwise every later chroot
// exec is "Exec format error".
func registerBinfmt() error {
	interps, err := filepath.Glob("/usr/bin/qemu-*-static")
	if err != nil {
		return err
	}
	if len(interps) == 0 {
		return nil
	}

	if err := prepareBinfmtMount(); err != nil {
		return fmt.Errorf("mount binfmt_misc: %w", err)
	}

	lines, err := collectBinfmtRegisterLines(binfmtConfDirs, interps, runtime.GOARCH)
	if err != nil {
		return err
	}
	if len(lines) == 0 {
		return fmt.Errorf("qemu-user-static present (%v) but no binfmt lines to register", interps)
	}

	var errs []error
	n := 0
	for _, line := range lines {
		if err := writeBinfmtRegister(line); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", binfmtName(line), err))
			continue
		}
		n++
	}
	if n == 0 {
		return fmt.Errorf("binfmt register failed: %w", errors.Join(errs...))
	}
	fmt.Fprintf(os.Stderr, "kc-guest-agent: registered %d binfmt interpreter(s)\n", n)
	return nil
}

func prepareBinfmtMount() error {
	_ = exec.Command("modprobe", "binfmt_misc").Run()
	if err := os.MkdirAll(binfmtMiscDir, 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(binfmtMiscDir, "register")); err == nil {
		return nil
	}
	if err := syscall.Mount("binfmt_misc", binfmtMiscDir, "binfmt_misc", 0, ""); err != nil {
		if !errors.Is(err, syscall.EBUSY) {
			return err
		}
	}
	if _, err := os.Stat(filepath.Join(binfmtMiscDir, "register")); err != nil {
		return fmt.Errorf("%s/register missing after mount: %w", binfmtMiscDir, err)
	}
	return nil
}

func writeBinfmtRegister(line string) error {
	name := binfmtName(line)
	if name != "" {
		if _, err := os.Stat(filepath.Join(binfmtMiscDir, name)); err == nil {
			return nil
		}
	}
	f, err := os.OpenFile(filepath.Join(binfmtMiscDir, "register"), os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteString(line + "\n"); err != nil {
		// Already registered by a previous write or the host kernel.
		if errors.Is(err, syscall.EEXIST) || strings.Contains(err.Error(), "File exists") {
			return nil
		}
		return err
	}
	return nil
}
