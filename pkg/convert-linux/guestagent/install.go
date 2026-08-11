//go:build linux

package guestagent

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/firstboot"
	"github.com/yaacov/kc-utils/pkg/guest"
)

// Install ensures qemu-guest-agent is installed or scheduled via firstboot.
func Install(guestRoot string, pkgFormat, pkgManager, arch, distro string, majorVersion int, offline bool) {
	agentInstalled := false
	if qa, ok := Agents.Get("qemu-ga"); ok {
		agentInstalled = qa.Detect(guestRoot)
	}
	if agentInstalled {
		slog.Info("qemu-guest-agent already installed")
		return
	}

	// networkAvailable reports whether a firstboot network package-manager
	// install can ever succeed for this guest. Amazon Linux 2023 does not
	// ship qemu-guest-agent in its guest repos at all, so network install
	// is never an option there, regardless of the offline flag.
	networkAvailable := !offline
	if distro == "amzn" && majorVersion >= 2023 {
		networkAvailable = false
	}

	localPkgs := findLocalPackages(FindRequest{
		Name:         "qemu-guest-agent",
		Format:       pkgFormat,
		Arch:         arch,
		Distro:       distro,
		MajorVersion: localPackageMajorVersion(distro, majorVersion),
	})

	var firstbootCmds []string
	switch {
	case len(localPkgs) > 0:
		slog.Info("found local qemu-guest-agent package",
			"file", localPkgs[0].FileName, "el", localPkgs[0].ELTag)
		cmds, err := copyAndInstallLocal(guestRoot, localPkgs, pkgFormat)
		if err != nil {
			slog.Warn("local package prep failed", "error", err)
			if networkAvailable {
				firstbootCmds = networkInstallCommands(pkgManager)
			}
		} else {
			firstbootCmds = cmds
		}
	case networkAvailable:
		slog.Warn("no local packages found, firstboot will attempt network install")
		firstbootCmds = networkInstallCommands(pkgManager)
	default:
		slog.Warn("no local qemu-guest-agent package found and network install unavailable, skipping agent install")
	}
	if len(firstbootCmds) > 0 {
		if fbHandler, ok := firstboot.Handlers.Get("systemd"); ok {
			if err := fbHandler.Install(guestRoot, firstbootCmds); err != nil {
				slog.Warn("firstboot install failed", "error", err)
			}
		}
	}
}

func findLocalPackages(req FindRequest) []PackageFile {
	for _, src := range Sources.All() {
		if !src.Available() {
			continue
		}
		pkgs, err := src.FindPackages(req)
		if err != nil {
			continue
		}
		if len(pkgs) > 0 {
			// Never install more than one package file.
			return pkgs[:1]
		}
	}
	return nil
}

func copyAndInstallLocal(guestRoot string, pkgs []PackageFile, format string) ([]string, error) {
	guestPkgDir := filepath.Join(guestRoot, "var", "lib", "kc-packages")
	if err := guest.FileMkdirAll(guestPkgDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating guest package dir: %w", err)
	}

	var installPaths []string
	for _, pkg := range pkgs {
		dst := filepath.Join(guestPkgDir, pkg.FileName)
		if err := guest.FileUpload(pkg.HostPath, dst); err != nil {
			return nil, fmt.Errorf("uploading %s: %w", pkg.HostPath, err)
		}
		installPaths = append(installPaths, "/var/lib/kc-packages/"+pkg.FileName)
	}

	cmds := []string{selinuxDisableCmd}
	switch format {
	case "deb":
		cmds = append(cmds, "dpkg -i "+strings.Join(installPaths, " "))
	default:
		cmds = append(cmds, "rpm -ivh "+strings.Join(installPaths, " "))
	}
	cmds = append(cmds,
		"systemctl enable --now qemu-guest-agent",
		"rm -rf /var/lib/kc-packages",
		selinuxRestoreCmd,
	)
	return cmds, nil
}

func networkInstallCommands(pkgManager string) []string {
	cmds := []string{networkWaitCmd, selinuxDisableCmd}
	switch pkgManager {
	case "apt":
		cmds = append(cmds,
			"apt-get update -q",
			"apt-get install -y qemu-guest-agent",
		)
	case "zypper":
		cmds = append(cmds,
			"zypper --non-interactive install qemu-guest-agent",
		)
	default:
		cmds = append(cmds,
			"dnf install -y qemu-guest-agent || yum install -y qemu-guest-agent",
		)
	}
	cmds = append(cmds,
		"systemctl enable --now qemu-guest-agent",
		selinuxRestoreCmd,
	)
	return cmds
}

// networkWaitCmd waits for network connectivity at first boot. The systemd
// unit already has After=network-online.target, but NetworkManager can fire
// that target before actual connectivity is established.
const networkWaitCmd = `if conn=$(nmcli networking connectivity 2>/dev/null); then tries=0; while [ $tries -lt 30 ] && [ full != "$conn" ]; do sleep 1; tries=$((tries + 1)); conn=$(nmcli networking connectivity); done; elif systemctl -q is-active systemd-networkd 2>/dev/null; then /usr/lib/systemd/systemd-networkd-wait-online -q --timeout=30 || true; fi; true`

// selinuxDisableCmd temporarily disables SELinux enforcement before package
// installation. RPM scriptlets in a freshly converted guest can trigger
// SELinux denials (RHBZ#2028764). The KC_ENFORCING variable persists in
// the shell for selinuxRestoreCmd to use.
const selinuxDisableCmd = `KC_ENFORCING=0; if command -v getenforce >/dev/null && [ Enforcing = "$(getenforce)" ]; then KC_ENFORCING=1; setenforce 0; fi`

// selinuxRestoreCmd re-enables SELinux enforcement if it was previously
// disabled by selinuxDisableCmd.
const selinuxRestoreCmd = `if [ "${KC_ENFORCING:-0}" = 1 ]; then setenforce 1; fi; true`
