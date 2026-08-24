//go:build unix

package qemu

import (
	"log/slog"
)

// UnmountFilesystems unmounts guest filesystems but keeps LUKS mappers, LVM
// volumes, and the appliance running (for post-fsck in finalize).
func (b *Backend) UnmountFilesystems() error {
	var firstErr error
	if b.probeActive != "" {
		if err := b.ProbeUnmount(b.probeActive); err != nil {
			firstErr = err
		}
	}
	if b.session != nil && b.session.client != nil {
		// Flush guest writes to the images before unmounting so nothing is lost
		// while the appliance keeps running for a later stage.
		if err := b.Sync(); err != nil && firstErr == nil {
			firstErr = err
		}
		if err := b.UnmountAll(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// ReleaseDevices closes LUKS mappings, deactivates LVM, and shuts down the
// appliance (owned session) or drops the connection (shared session).
func (b *Backend) ReleaseDevices() error {
	var firstErr error
	if b.session != nil && b.session.client != nil {
		for i := len(b.cryptMaps) - 1; i >= 0; i-- {
			if err := b.CloseCrypt(b.cryptMaps[i]); err != nil {
				slog.Debug("qemu cryptsetup close", "mapper", b.cryptMaps[i], "error", err)
				if firstErr == nil {
					firstErr = err
				}
			}
		}
		b.cryptMaps = nil
		b.removeLDM()
		if _, err := b.session.client.run("vgchange", "-an"); err != nil {
			slog.Debug("qemu vgchange -an", "error", err)
		}
	}
	if b.session != nil {
		if err := b.session.close(); err != nil && firstErr == nil {
			firstErr = err
		}
		b.session = nil
	}
	return firstErr
}

// Teardown fully unmounts and releases devices (finalize path).
func (b *Backend) Teardown() error {
	firstErr := b.UnmountFilesystems()
	if err := b.ReleaseDevices(); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

// TeardownDiscard is identical to Teardown for the qemu backend: guest edits go
// through qcow2 overlays managed by the pipeline, which discards them on failure
// by not committing.
func (b *Backend) TeardownDiscard() error {
	return b.Teardown()
}

// TeardownMountRoot best-effort cleans the host-side mount root scaffolding.
func TeardownMountRoot(mountRoot string) error {
	return clearDir(mountRoot)
}
