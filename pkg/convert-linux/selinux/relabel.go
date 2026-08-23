//go:build unix

package selinux

import (
	"bufio"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/yaacov/kc-utils/pkg/guest"
)

// Relabel performs an offline SELinux relabel of the guest filesystem using
// setfiles. This avoids the slow boot-time relabel and automatic reboot that
// /.autorelabel triggers.
//
// On success it removes any existing /.autorelabel. On failure the caller
// should fall back to creating /.autorelabel.
func Relabel(guestRoot string, mountPoints []string) (bool, error) {
	selinuxDir := filepath.Join(guestRoot, "etc", "selinux")
	if !guest.FileExists(selinuxDir) {
		slog.Debug("no /etc/selinux, skipping SELinux relabel")
		return false, nil
	}

	policy, disabled, err := readSELinuxConfig(guestRoot)
	if err != nil {
		return false, fmt.Errorf("reading SELinux config: %w", err)
	}
	if disabled {
		slog.Info("SELinux is disabled in guest, skipping relabel")
		return false, nil
	}

	specFile := fmt.Sprintf("/etc/selinux/%s/contexts/files/file_contexts", policy)
	specHostPath := filepath.Join(guestRoot, specFile)
	if !guest.FileExists(specHostPath) {
		return false, fmt.Errorf("SELinux spec file not found: %s", specFile)
	}

	setfilesPath := findSetfiles(guestRoot)
	if setfilesPath == "" {
		return false, fmt.Errorf("setfiles binary not found in guest")
	}

	// setfiles does not cross filesystem boundaries, so run it against
	// each mountpoint.
	targets := mountPointsForSetfiles(mountPoints)
	slog.Info("running offline SELinux relabel", "policy", policy, "targets", len(targets))

	args := []string{setfilesPath, "-r", "/", specFile}
	args = append(args, targets...)
	out, err := guest.RunInGuest(guestRoot, args)
	if err != nil {
		return false, fmt.Errorf("setfiles failed: %w (output: %s)", err, string(out))
	}

	slog.Info("offline SELinux relabel complete", "policy", policy)

	// Remove /.autorelabel so the guest doesn't redo the relabel at boot.
	autorelabel := filepath.Join(guestRoot, ".autorelabel")
	_ = guest.FileRemove(autorelabel)

	return true, nil
}

// readSELinuxConfig parses /etc/selinux/config and returns the policy type
// (e.g., "targeted") and whether SELinux is disabled.
func readSELinuxConfig(guestRoot string) (policy string, disabled bool, err error) {
	configPath := filepath.Join(guestRoot, "etc", "selinux", "config")
	data, err := guest.FileRead(configPath)
	if err != nil {
		return "targeted", false, err
	}

	policy = "targeted"
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		switch key {
		case "SELINUX":
			if val == "disabled" {
				disabled = true
			}
		case "SELINUXTYPE":
			if val != "" {
				policy = val
			}
		}
	}

	return policy, disabled, nil
}

// findSetfiles checks common locations for the setfiles binary in the guest.
func findSetfiles(guestRoot string) string {
	for _, p := range []string{"/usr/sbin/setfiles", "/sbin/setfiles"} {
		if guest.FileExists(filepath.Join(guestRoot, p)) {
			return p
		}
	}
	return ""
}

// mountPointsForSetfiles returns guest-relative mount paths for setfiles.
// If no mountpoints are provided, defaults to "/".
func mountPointsForSetfiles(mountPoints []string) []string {
	if len(mountPoints) == 0 {
		return []string{"/"}
	}

	seen := make(map[string]bool)
	var result []string
	for _, mp := range mountPoints {
		if mp == "" {
			mp = "/"
		}
		if !strings.HasPrefix(mp, "/") {
			mp = "/" + mp
		}
		if seen[mp] {
			continue
		}
		seen[mp] = true
		result = append(result, mp)
	}
	if !seen["/"] {
		result = append([]string{"/"}, result...)
	}
	return result
}
