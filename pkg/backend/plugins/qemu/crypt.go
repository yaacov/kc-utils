//go:build unix

package qemu

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// EnvClevisNetwork enables appliance user-mode networking before boot, required
// for Clevis/NBDE (Tang) unlock. It shares the name used by the guestfs backend
// (KC_GUESTFS_NETWORK) so orchestrators set a single variable.
const EnvClevisNetwork = "KC_GUESTFS_NETWORK"

// clevisNetworkRequested reports whether appliance networking should be enabled
// for Clevis/NBDE (Tang) unlock.
func clevisNetworkRequested() bool {
	v := strings.TrimSpace(os.Getenv(EnvClevisNetwork))
	return v == "1" || strings.EqualFold(v, "true")
}

// Decrypt opens a LUKS mapping with a key file. The key is uploaded to a temp
// path in the appliance, then cryptsetup opens the mapping there.
func (b *Backend) Decrypt(device, keyFile, mapperName string) (string, error) {
	keyData, err := os.ReadFile(keyFile)
	if err != nil {
		return "", fmt.Errorf("read LUKS keyfile %s: %w", keyFile, err)
	}
	keyData = bytes.TrimRight(keyData, "\r\n")
	keyData = append(keyData, '\n')

	applianceKey := "/tmp/.kc-luks-key-" + mapperName
	if err := b.session.client.writeFile(applianceKey, keyData, 0o600); err != nil {
		return "", fmt.Errorf("upload LUKS keyfile: %w", err)
	}
	defer func() { _ = b.session.client.remove(applianceKey, false) }()

	if _, err := b.session.client.run("cryptsetup", "luksOpen", "--key-file", applianceKey, device, mapperName); err != nil {
		return "", fmt.Errorf("cryptsetup open %s: %w", device, err)
	}
	mapped := "/dev/mapper/" + mapperName
	if !b.mapperPresent(mapped) {
		return "", fmt.Errorf("cryptsetup open %s: mapper %s not created", device, mapped)
	}
	b.cryptMaps = append(b.cryptMaps, mapperName)
	slog.Info("qemu LUKS decrypted with keyfile", "device", device, "mapper", mapped)
	return mapped, nil
}

// UnlockClevis unlocks a Clevis-bound LUKS volume (NBDE / Tang). Requires
// appliance networking (see clevisNetworkRequested).
func (b *Backend) UnlockClevis(device, mapperName string) (string, error) {
	if _, err := b.session.client.run("clevis", "luks", "unlock", "-d", device, "-n", mapperName); err != nil {
		return "", fmt.Errorf("clevis unlock %s: %w", device, err)
	}
	mapped := "/dev/mapper/" + mapperName
	if !b.mapperPresent(mapped) {
		return "", fmt.Errorf("clevis unlock %s: mapper %s not created", device, mapped)
	}
	b.cryptMaps = append(b.cryptMaps, mapperName)
	slog.Info("qemu Clevis LUKS unlocked", "device", device, "mapper", mapped)
	return mapped, nil
}

// CloseCrypt closes a LUKS mapping.
func (b *Backend) CloseCrypt(mapperName string) error {
	name := strings.TrimPrefix(mapperName, "/dev/mapper/")
	_, err := b.session.client.run("cryptsetup", "close", name)
	return err
}

// RescanBlock re-activates LVM after LUKS unlock so LVs on unlocked devices
// appear in LVPaths.
func (b *Backend) RescanBlock() error {
	lvs, err := b.discoverLVs()
	if err != nil {
		return fmt.Errorf("rescan LVs after decrypt: %w", err)
	}
	b.lvPaths = lvs
	slog.Info("qemu rescan after decrypt", "lvs", len(lvs), "paths", lvs)
	return nil
}

// mapperPresent reports whether a device-mapper node exists in the appliance.
func (b *Backend) mapperPresent(mapped string) bool {
	st, err := b.session.client.stat(mapped)
	return err == nil && st.Exists
}
