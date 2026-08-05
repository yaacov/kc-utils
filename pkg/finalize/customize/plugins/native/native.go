//go:build linux

package native

import (
	"log/slog"
	"path/filepath"

	"github.com/yaacov/kc-utils/pkg/finalize/customize"
	"github.com/yaacov/kc-utils/pkg/guest"
)

type NativeCustomizer struct{}

func init() {
	customize.Customizers.Register("native", &NativeCustomizer{})
}

func (n *NativeCustomizer) Apply(guestRoot string, options map[string]string) error {
	if hostname, ok := options["hostname"]; ok && hostname != "" {
		hostnamePath := filepath.Join(guestRoot, "etc", "hostname")
		if err := guest.FileMkdirAll(filepath.Dir(hostnamePath), 0o755); err != nil {
			return err
		}
		if err := guest.FileWrite(hostnamePath, []byte(hostname+"\n"), 0o644); err != nil {
			return err
		}
		slog.Info("set hostname", "hostname", hostname)
	}

	if tz, ok := options["timezone"]; ok && tz != "" {
		localtimePath := filepath.Join(guestRoot, "etc", "localtime")
		_ = guest.FileRemove(localtimePath)
		if err := guest.FileSymlink("/usr/share/zoneinfo/"+tz, localtimePath); err != nil {
			return err
		}
		slog.Info("set timezone", "timezone", tz)
	}

	// Create /.autorelabel only as a fallback when the converter did not
	// perform an offline setfiles relabel. The offline relabel (matching
	// virt-v2v) avoids the slow boot-time relabel + automatic reboot.
	if options["selinux_relabeled"] != "true" {
		selinuxDir := filepath.Join(guestRoot, "etc", "selinux")
		if guest.FileExists(selinuxDir) {
			autorelabel := filepath.Join(guestRoot, ".autorelabel")
			if err := guest.FileWrite(autorelabel, nil, 0o644); err != nil {
				slog.Warn("creating .autorelabel failed", "error", err)
			} else {
				slog.Info("created /.autorelabel for SELinux relabel on first boot (offline relabel was not performed)")
			}
		}
	} else {
		slog.Info("skipping /.autorelabel, offline SELinux relabel was performed during conversion")
	}

	return nil
}
