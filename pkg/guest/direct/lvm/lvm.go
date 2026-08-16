//go:build unix

package lvm

import (
	"fmt"
	"os/exec"
	"strings"
)

// ScanAndActivate scans for LVM physical volumes on the given devices
// and activates any volume groups found.
func ScanAndActivate(devices []string) ([]string, error) {
	for _, dev := range devices {
		if err := exec.Command("pvscan", "--cache", dev).Run(); err != nil {
			continue
		}
	}

	if err := exec.Command("vgscan").Run(); err != nil {
		return nil, fmt.Errorf("vgscan: %w", err)
	}

	if err := exec.Command("vgchange", "-ay").Run(); err != nil {
		return nil, fmt.Errorf("vgchange -ay: %w", err)
	}

	out, err := exec.Command("lvscan").Output()
	if err != nil {
		return nil, fmt.Errorf("lvscan: %w", err)
	}

	var lvs []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "ACTIVE") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				lv := strings.Trim(parts[1], "'")
				lvs = append(lvs, lv)
			}
		}
	}
	return lvs, nil
}

// Deactivate deactivates a volume group.
func Deactivate(vgName string) error {
	return exec.Command("vgchange", "-an", vgName).Run()
}
