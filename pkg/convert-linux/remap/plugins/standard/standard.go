//go:build linux

package standard

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/configedit/bls"
	"github.com/yaacov/kc-utils/pkg/common/configedit/fstab"
	cfggrub "github.com/yaacov/kc-utils/pkg/common/configedit/grub"
	"github.com/yaacov/kc-utils/pkg/convert-linux/remap"
	"github.com/yaacov/kc-utils/pkg/guest"
)

type Remapper struct{}

func init() {
	remap.Remappers.Register("standard", &Remapper{})
}

func (r *Remapper) Name() string { return "standard" }

func (r *Remapper) Detect(guestRoot string) bool {
	fstabPath := filepath.Join(guestRoot, "etc", "fstab")
	return guest.FileExists(fstabPath)
}

func (r *Remapper) Remap(guestRoot string) error {
	remapPrefixes := [][2]string{
		{"/dev/hd", "/dev/vd"},
		{"/dev/sd", "/dev/vd"},
		{"/dev/xvd", "/dev/vd"},
		{"/dev/nvme0n1p", "/dev/vda"},
		{"/dev/cciss/c0d0p", "/dev/vda"},
	}

	if err := remapFile(guestRoot, "etc/fstab", remapPrefixes); err != nil {
		return err
	}
	if err := remapCrypttab(guestRoot, remapPrefixes); err != nil {
		return err
	}
	if err := remapGrubDefaults(guestRoot, remapPrefixes); err != nil {
		return err
	}
	if err := remapBLSEntries(guestRoot, remapPrefixes); err != nil {
		return err
	}
	return nil
}

func remapFile(guestRoot, relPath string, prefixes [][2]string) error {
	filePath := filepath.Join(guestRoot, relPath)
	data, err := guest.FileRead(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	f := fstab.Parse(string(data))
	for _, p := range prefixes {
		f.RemapDevice(p[0], p[1])
	}
	if err := guest.FileWrite(filePath, []byte(f.String()), 0o644); err != nil {
		return err
	}
	slog.Info("remapped block devices to virtio", "file", relPath)
	return nil
}

// remapCrypttab prefers UUID= for /dev/sd* devices (virt-v2v behavior) when
// blkid can resolve the device on the host; otherwise falls back to vd* remap.
func remapCrypttab(guestRoot string, prefixes [][2]string) error {
	filePath := filepath.Join(guestRoot, "etc", "crypttab")
	data, err := guest.FileRead(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	f := fstab.Parse(string(data))
	for i := range f.Entries {
		e := &f.Entries[i]
		if e.Comment != "" || e.MountPoint == "" {
			continue
		}
		// crypttab: name device keyfile options — device is MountPoint in our parser.
		dev := e.MountPoint
		if strings.HasPrefix(dev, "/dev/sd") {
			if uuid := blkidUUID(dev); uuid != "" {
				e.MountPoint = "UUID=" + uuid
				continue
			}
		}
		for _, p := range prefixes {
			if strings.HasPrefix(dev, p[0]) {
				e.MountPoint = p[1] + dev[len(p[0]):]
				break
			}
		}
	}
	if err := guest.FileWrite(filePath, []byte(f.String()), 0o644); err != nil {
		return err
	}
	slog.Info("remapped block devices to virtio", "file", "etc/crypttab")
	return nil
}

func blkidUUID(device string) string {
	return guest.BlkidUUID(device)
}

func remapGrubDefaults(guestRoot string, prefixes [][2]string) error {
	filePath := filepath.Join(guestRoot, "etc", "default", "grub")
	data, err := guest.FileRead(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	cfg := cfggrub.Parse(string(data))
	changed := false
	for _, key := range []string{"GRUB_CMDLINE_LINUX", "GRUB_CMDLINE_LINUX_DEFAULT"} {
		args := strings.Fields(cfg.Get(key))
		if len(args) == 0 {
			continue
		}
		remappedArgs, remapped := remapKernelArgs(args, prefixes)
		if remapped {
			cfg.Set(key, strings.Join(remappedArgs, " "))
			changed = true
		}
	}
	if !changed {
		return nil
	}

	if err := guest.FileWrite(filePath, []byte(cfg.String()), 0o644); err != nil {
		return err
	}
	slog.Info("remapped block devices to virtio", "file", "etc/default/grub")
	return nil
}

func remapBLSEntries(guestRoot string, prefixes [][2]string) error {
	entriesDir := filepath.Join(guestRoot, "boot", "loader", "entries")
	entries, err := guest.FileReadDir(entriesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir || !strings.HasSuffix(entry.Name, ".conf") {
			continue
		}
		path := filepath.Join(entriesDir, entry.Name)
		data, err := guest.FileRead(path)
		if err != nil {
			return err
		}

		parsed := bls.Parse(string(data))
		args := strings.Fields(parsed.Get("options"))
		if len(args) == 0 {
			continue
		}
		remappedArgs, changed := remapKernelArgs(args, prefixes)
		if !changed {
			continue
		}
		parsed.Set("options", strings.Join(remappedArgs, " "))
		if err := guest.FileWrite(path, []byte(parsed.String()), 0o644); err != nil {
			return err
		}
		slog.Info("remapped block devices to virtio", "file", filepath.Join("boot", "loader", "entries", entry.Name))
	}
	return nil
}

func remapKernelArgs(args []string, prefixes [][2]string) ([]string, bool) {
	out := make([]string, 0, len(args))
	changed := false
	for _, arg := range args {
		remapped := remapKernelArg(arg, prefixes)
		if remapped != arg {
			changed = true
		}
		out = append(out, remapped)
	}
	return out, changed
}

func remapKernelArg(arg string, prefixes [][2]string) string {
	switch {
	case strings.HasPrefix(arg, "resume="):
		return remapArgValue(arg, "resume=", prefixes)
	case strings.HasPrefix(arg, "root="):
		return remapArgValue(arg, "root=", prefixes)
	default:
		return arg
	}
}

func remapArgValue(arg, prefix string, remapPrefixes [][2]string) string {
	value := strings.TrimPrefix(arg, prefix)
	for _, p := range remapPrefixes {
		if strings.HasPrefix(value, p[0]) {
			return prefix + p[1] + value[len(p[0]):]
		}
	}
	return arg
}
