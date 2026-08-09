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
| `dynamicscriptslinux` | dynamicscriptslinux/ | Run external Linux customization scripts |
| `dynamicscriptswindows` | dynamicscriptswindows/ | Run external Windows customization scripts |

Firstboot handlers for `dynamicscriptslinux`: [`firstboot/plugins/README.md`](../../../common/firstboot/plugins/README.md).

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

## dynamicscriptslinux

**What it does:** Discovers and applies user-provided Linux customization
scripts from a host directory (default `/mnt/dynamic_scripts`).

**How it works:** No-ops unless `os_type` is `linux`. Scans the scripts
directory for files matching:

- `<priority>_linux_run_<name>.sh` — uploaded into the guest's `/tmp/` and
  executed immediately via `/bin/bash` inside the mounted filesystem.
- `<priority>_linux_firstboot_<name>.sh` — installed as a firstboot command
  using the systemd firstboot handler, so it runs on the guest's first boot.

Scripts are sorted by the numeric priority prefix (lower runs first). Failures
on individual scripts are logged as warnings but do not halt processing of
remaining scripts.

## dynamicscriptswindows

**What it does:** Discovers and applies user-provided Windows customization
scripts from a host directory (default `/mnt/dynamic_scripts`).

**How it works:** No-ops unless `os_type` is `windows`. Scans the scripts
directory for files matching:

- `<priority>_win_firstboot_<name>.ps1` — copied into the Windows
  `Program Files/Guestfs/Firstboot/scripts/` directory for execution on
  the guest's first Windows boot.

Scripts are sorted by the numeric priority prefix (lower runs first). Failures
on individual scripts are logged as warnings but do not halt processing of
remaining scripts.
