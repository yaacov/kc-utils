//go:build linux

package systemd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/yaacov/kc-utils/pkg/common/firstboot"
	"github.com/yaacov/kc-utils/pkg/guest"
)

type SystemdFirstBoot struct{}

func init() {
	firstboot.Handlers.Register("systemd", &SystemdFirstBoot{})
}

const scriptHeader = "#!/bin/bash\nset -o pipefail\nRETRIES=2\nDELAY=5\n\nrun_with_retry() {\n  local cmd=\"$1\"\n  for attempt in $(seq 1 $RETRIES); do\n    if eval \"$cmd\"; then return 0; fi\n    logger -t kc-firstboot \"Command failed (attempt $attempt/$RETRIES): $cmd\"\n    [ \"$attempt\" -lt \"$RETRIES\" ] && sleep $DELAY\n  done\n  logger -t kc-firstboot \"Command failed after $RETRIES attempts: $cmd\"\n  return 1\n}\n\n"

const scriptTail = "\nsystemctl disable kc-firstboot.service\nrm -f /etc/systemd/system/kc-firstboot.service\nrm -f /usr/local/bin/kc-firstboot.sh\n"

func (s *SystemdFirstBoot) Install(guestRoot string, commands []string) error {
	scriptDir := filepath.Join(guestRoot, "usr", "local", "bin")
	if err := guest.FileMkdirAll(scriptDir, 0o755); err != nil {
		return err
	}

	scriptPath := filepath.Join(scriptDir, "kc-firstboot.sh")

	if existing, err := guest.FileRead(scriptPath); err == nil && len(existing) > 0 {
		return appendCommands(scriptPath, string(existing), commands)
	}

	var script strings.Builder
	script.WriteString(scriptHeader)
	for _, cmd := range commands {
		fmt.Fprintf(&script, "run_with_retry %q\n", cmd)
	}
	script.WriteString(scriptTail)
	if err := guest.FileWrite(scriptPath, []byte(script.String()), 0o755); err != nil {
		return err
	}

	return s.installUnit(guestRoot)
}

func appendCommands(scriptPath, existing string, commands []string) error {
	marker := "\nsystemctl disable kc-firstboot.service"
	idx := strings.Index(existing, marker)
	if idx < 0 {
		var buf strings.Builder
		buf.WriteString(existing)
		for _, cmd := range commands {
			fmt.Fprintf(&buf, "run_with_retry %q\n", cmd)
		}
		return guest.FileWrite(scriptPath, []byte(buf.String()), 0o755)
	}
	var buf strings.Builder
	buf.WriteString(existing[:idx])
	buf.WriteByte('\n')
	for _, cmd := range commands {
		fmt.Fprintf(&buf, "run_with_retry %q\n", cmd)
	}
	buf.WriteString(existing[idx:])
	return guest.FileWrite(scriptPath, []byte(buf.String()), 0o755)
}

func (s *SystemdFirstBoot) installUnit(guestRoot string) error {
	unit := `[Unit]
Description=KC Utils First Boot
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
ExecStart=/usr/local/bin/kc-firstboot.sh
RemainAfterExit=true
TimeoutStartSec=120

[Install]
WantedBy=multi-user.target
`
	unitPath := filepath.Join(guestRoot, "etc", "systemd", "system", "kc-firstboot.service")
	if err := guest.FileWrite(unitPath, []byte(unit), 0o644); err != nil {
		return err
	}
	symlinkDir := filepath.Join(guestRoot, "etc", "systemd", "system", "multi-user.target.wants")
	if err := guest.FileMkdirAll(symlinkDir, 0o755); err != nil {
		return err
	}
	return guest.FileSymlink("/etc/systemd/system/kc-firstboot.service",
		filepath.Join(symlinkDir, "kc-firstboot.service"))
}
