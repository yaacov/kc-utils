//go:build unix

package backend

import (
	"bufio"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// Probes holds runtime environment checks. Tests may replace individual funcs.
var Probes = struct {
	OnLinux      func() bool
	HasRoot      func() bool
	HasKVM       func() bool
	HasGuestfish func() bool
	HasQEMU      func() bool
	HasAccel     func() bool
}{
	OnLinux:      onLinux,
	HasRoot:      hasRoot,
	HasKVM:       hasKVM,
	HasGuestfish: hasGuestfish,
	HasQEMU:      hasQEMU,
	HasAccel:     hasAccel,
}

func onLinux() bool {
	return runtime.GOOS == "linux"
}

func hasRoot() bool {
	if os.Geteuid() == 0 {
		return true
	}
	return hasCapSysAdmin()
}

func hasCapSysAdmin() bool {
	f, err := os.Open("/proc/self/status")
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "CapEff:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return false
		}
		capEff, err := strconv.ParseUint(fields[1], 16, 64)
		if err != nil {
			return false
		}
		const capSysAdminBit = 1 << 21
		return capEff&capSysAdminBit != 0
	}
	return false
}

func hasKVM() bool {
	f, err := os.Open("/dev/kvm")
	if err != nil {
		return false
	}
	_ = f.Close()
	return true
}

func hasGuestfish() bool {
	if _, err := exec.LookPath("virt-guestfish"); err == nil {
		return true
	}
	_, err := exec.LookPath("guestfish")
	return err == nil
}

// qemuSystemBinary is the qemu-system binary name for the host architecture.
func qemuSystemBinary() string {
	switch runtime.GOARCH {
	case "arm64":
		return "qemu-system-aarch64"
	case "amd64":
		return "qemu-system-x86_64"
	default:
		return "qemu-system-" + runtime.GOARCH
	}
}

func hasQEMU() bool {
	if _, err := exec.LookPath(qemuSystemBinary()); err == nil {
		return true
	}
	_, err := exec.LookPath("qemu-kvm")
	return err == nil
}

// hasAccel reports whether hardware virtualization is available: KVM on Linux,
// HVF on macOS. Used to prefer acceleration at launch; the qemu backend still
// works via TCG when this is false.
func hasAccel() bool {
	switch runtime.GOOS {
	case "linux":
		f, err := os.Open("/dev/kvm")
		if err != nil {
			return false
		}
		_ = f.Close()
		return true
	case "darwin":
		out, err := exec.Command(qemuSystemBinary(), "-accel", "help").Output()
		if err != nil {
			return false
		}
		return strings.Contains(string(out), "hvf")
	default:
		return false
	}
}

func checkRequirements(req Requirements) (bool, string) {
	if req.Linux && !Probes.OnLinux() {
		return false, "requires Linux"
	}
	if req.Root && !Probes.HasRoot() {
		return false, "requires root or CAP_SYS_ADMIN"
	}
	if req.KVM && !Probes.HasKVM() {
		return false, "/dev/kvm not accessible"
	}
	if req.Guestfish && !Probes.HasGuestfish() {
		return false, "guestfish not found in PATH"
	}
	if req.QEMU && !Probes.HasQEMU() {
		return false, "qemu-system binary not found in PATH"
	}
	if req.Accel && !Probes.HasAccel() {
		return false, "hardware virtualization (KVM/HVF) not available"
	}
	return true, ""
}
