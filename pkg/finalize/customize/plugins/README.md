# customize plugins

`Customizer` interface — run post-mount customization. Errors become warnings.

The customize block runs after the guest filesystem is mounted and conversion
is complete, just before the finalize pipeline trims and unmounts. Each plugin
can modify the guest filesystem to apply final adjustments — setting the
hostname, timezone, scheduling firstboot scripts, or running user-provided
customization scripts. Errors from customizers are recorded as warnings but
do not fail the pipeline, since they are non-critical enhancements.

| Key | Package | Description |
|-----|---------|-------------|
| `native` | native/ | Direct file injection/firstboot without external binary |
| `dynamicscripts` | dynamicscripts/ | Run external customization scripts |

Firstboot handlers for `dynamicscripts`: [`firstboot/plugins/README.md`](../firstboot/plugins/README.md).

## native

**What it does:** Applies basic guest customizations by writing directly to the
mounted guest filesystem: hostname, timezone, and SELinux auto-relabel.

**How it works:** Processes three option keys from the finalize configuration:

- `hostname` — writes the desired hostname to `/etc/hostname` in the guest.
- `timezone` — replaces `/etc/localtime` with a symlink pointing to the
  requested timezone under `/usr/share/zoneinfo/`.
- `selinux_relabeled` — if the Linux converter already performed an offline
  SELinux relabel (this option is `"true"`), skips the autorelabel marker.
  Otherwise, when `/etc/selinux/` exists, creates `/.autorelabel` so the guest
  triggers a full relabel on its next boot.

## dynamicscripts

**What it does:** Discovers and applies user-provided customization scripts
from a host directory (default `/mnt/dynamic_scripts`), allowing operators to
inject arbitrary pre-boot or firstboot logic into the converted guest.

**How it works:** Scans the scripts directory for files matching a naming
convention that encodes priority, target OS, and action type:

- `<priority>_linux_run_<name>.sh` — uploaded into the guest's `/tmp/` and
  executed immediately via `/bin/bash` inside the mounted filesystem.
- `<priority>_linux_firstboot_<name>.sh` — installed as a firstboot command
  using the systemd firstboot handler, so it runs on the guest's first boot.
- `<priority>_win_firstboot_<name>.ps1` — copied into the Windows
  `Program Files/Guestfs/Firstboot/scripts/` directory for execution on
  the guest's first Windows boot.

Scripts are sorted by the numeric priority prefix (lower runs first). Failures
on individual scripts are logged as warnings but do not halt processing of
remaining scripts.
