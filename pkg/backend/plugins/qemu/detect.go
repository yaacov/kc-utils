//go:build unix

package qemu

import (
	"log/slog"
	"strings"

	"github.com/yaacov/kc-utils/pkg/backend"
	"github.com/yaacov/kc-utils/pkg/backend/windowsvol"
)

func (b *Backend) checkWindowsVolumes() error {
	issues := windowsvol.ScanDiskInfos(b.diskInfos, b.qemuPartProbe)
	for _, issue := range issues {
		if issue.Kind == windowsvol.KindLDM {
			slog.Info("LDM metadata detected", "device", issue.Device)
		}
	}
	if issue := windowsvol.FirstUnsupported(issues, backend.NameQEMU); issue != nil {
		return windowsvol.UnsupportedError(issue.Kind, issue.Device, backend.NameQEMU)
	}
	return nil
}

func (b *Backend) qemuPartProbe(device string) (partType, fsType string) {
	if b.session != nil && b.session.client != nil {
		if out, err := b.session.client.run("lsblk", "-no", "PARTTYPE", device); err == nil {
			partType = strings.TrimSpace(string(out))
		}
	}
	fsType = b.blkidType(device)
	return partType, fsType
}
