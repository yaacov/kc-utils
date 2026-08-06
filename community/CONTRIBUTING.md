# Contributing to kc-utils

## Before you start

- Read [architecture.md](architecture.md) before changing code structure, packages, plugins, or guest disk access.
- All binaries target Linux (`GOOS=linux` / `//go:build linux`).

## Directory layout

Path pattern: `<layer>/<utility>/<semantic-name>/` — utility matches the binary;
semantic name describes what the block does (block numbers live in READMEs, not paths).

```
kc-utils/
  cmd/                  Thin CLI entry points (one per binary)
  pkg/
    common/             Cross-utility helpers (types, plugin, configedit, registry, fschecker, ...)
    prepare/            kc-prepare blocks (strict + pluggable with plugins/ subdirs)
    convert-linux/      kc-convert-linux blocks
    convert-windows/    kc-convert-windows blocks
    finalize/           kc-finalize blocks
    v2v/                kc-v2v orchestration (config, env, copy, vsphere, inspection)
  internal/             Thin orchestrators only
    prepare/pipeline.go
    convert-linux/pipeline.go
    convert-windows/pipeline.go
    finalize/pipeline.go
    v2v/server/         HTTP warning server
  community/            Contributor and architecture guidance
  docs/                 Pipeline block tables per utility
  tests/                Shell-based e2e tests and fixtures
```

See [pkg/README.md](../pkg/README.md) and [internal/README.md](../internal/README.md) for block maps.

## External dependencies

These are standard RHEL/Fedora packages, invoked as CLI tools at runtime.

### Required

| Tool | Package | Used For |
|------|---------|----------|
| `hivexregedit` | perl-hivex | Windows registry writes |
| `e2fsck` | e2fsprogs | ext2/3/4 filesystem check |
| `xfs_repair` | xfsprogs | XFS filesystem check |
| `btrfs` | btrfs-progs | Btrfs filesystem check |
| `ntfsfix` | ntfs-3g | NTFS filesystem check |
| `fstrim` | util-linux | Filesystem trim (discard unused blocks) |
| `losetup`, `partx` | util-linux | Loop device setup for disk images |
| `lsblk` | util-linux | Partition layout inspection |
| `lvm`, `vgscan`, `pvscan`, `lvscan` | lvm2 | LVM volume activation |
| `cryptsetup` | cryptsetup | LUKS partition decryption |
| `virtio-win` | virtio-win | Windows VirtIO drivers on conversion host (see below) |

### Optional

| Tool | Package | Used For |
|------|---------|----------|
| `clevis` | clevis | Tang/TPM-bound LUKS unlock |
| `guestfish` | guestfs-tools (Fedora: `libguestfs-tools`) | Guestfs mode (`--guestfs` / `V2V_guestfs=true`) |

For Windows conversions, also install VirtIO-Win drivers on the host (RPM places
files under `/usr/share/virtio-win/` for `kc-convert-windows`):

```bash
sudo dnf install -y virtio-win
```

Linux conversions take virtio modules from the guest kernel; no host VirtIO
package is required. For offline qemu-guest-agent install on RHEL-family guests,
stage RPMs under `/usr/share/kc-packages/rpm/el{8,9,10}/x86_64/` (baked into the
kc-v2v image; used with `--offline`). See
[docs/kc-convert-linux.md](../docs/kc-convert-linux.md).

### Dev dependencies

For testing and development (`make test-e2e`, `make test-e2e-disk`, `make lint`):

```bash
sudo dnf install -y \
  golang make jq perl-hivex hivex guestfs-tools ntfs-3g ntfsprogs
```

(`perl-hivex` provides `hivexregedit`; `hivex` provides `hivexget`.
On Fedora, `libguestfs-tools` still provides `guestfish` if `guestfs-tools`
is unavailable.)

Disk-image e2e tests need a privileged container (`make test-e2e-disk`); guestfs
disk e2e also needs `guestfs-tools` (`make test-e2e-disk-guestfs`).

## Build

```bash
make build              # all six binaries into bin/
make build-converters   # kc-convert-linux and kc-convert-windows only
make build-kc-v2v       # kc-v2v only
make build-kc-copy      # kc-copy only
make build-kc-v2v-image # container image (build/kc-v2v/Containerfile)
```

Or build directly (Linux only):

```bash
GOOS=linux go build ./cmd/...
```

Cross-compile for multiple Linux architectures:

```bash
make cross-all      # amd64, arm64, ppc64le, s390x
```

## Test

```bash
make test                 # unit tests
make lint                 # golangci-lint (auto-installs pinned golangci-lint)
make check                # fmt + vet + lint + unit tests
make test-e2e             # shell e2e tests (see below)
make test-e2e-container   # e2e in a UBI 10 container (all test-e2e scripts)
make test-e2e-disk        # disk-image tests (privileged container)
make test-e2e-disk-guestfs # disk-image tests via guestfs (no --privileged)
```

### `make test-e2e`

Runs all shell scripts matching `tests/test-{linux,windows,root,kc,dynamicscripts}-*.sh`.
Each script exits **0** (pass), **77** (skip), or another code (fail). Skips do
not fail the target; only real failures do. Logs for a run are in
`tests/<script-name>.log`.

| Group | Scripts | Runs by default? | Skip when |
|-------|---------|------------------|-----------|
| Linux | `test-linux-*.sh` | Yes | Not on Linux, or `jq` missing |
| Integration | `test-kc-*.sh`, `test-dynamicscripts-*.sh` | Yes | Not on Linux |
| Windows | `test-windows-*.sh` | Usually | `hivexregedit`, or fixture files missing; or cannot write `/usr/share/virtio-win` (needs root) |
| Windows registry | `test-windows-registry.sh` | Often skipped alone | Also needs `hivexget` (`hivex` on Fedora; `libhivex-bin` on Debian) |
| Root selection | `test-root-*.sh` | Usually skipped | Not run as root, `guestfish` missing, or `losetup --partscan` cannot create `/dev/loopNp1` nodes |

**Linux tests** build fake guest trees under `/tmp` and exercise converters
offline. No root, real disks, or libguestfs required.

**Windows tests** build registry hives with `hivexregedit`, stage a fake
virtio-win ISO under `/usr/share/virtio-win`, then run `kc-convert-windows`.
Install deps from [Dev dependencies](#dev-dependencies) and run as root:

```bash
sudo make test-e2e
```

`test-windows-registry.sh` reads hive values back with `hivexget` and is the
only Windows e2e script that needs the `hivex` package in addition to
`perl-hivex`.

**Root-selection tests** create disk images with `guestfish`, attach loop
devices, and verify `kc-prepare` root picking. They require root, working loop
partition nodes (`/dev/loopNp1` after `losetup --partscan`), and
`guestfs-tools`. If `losetup` fails (common inside nested containers or
restricted VMs), all six `test-root-*.sh` scripts skip with
`losetup failed, skipping`:

```bash
sudo dnf install -y guestfs-tools
sudo make test-e2e
```

On hosts where loop partitions are unavailable, use disk e2e in a privileged
container instead: `make test-e2e-disk`.

To run the full suite without installing host packages, use the UBI 10 test
container (built from `tests/Containerfile`, with CentOS Stream 10 + EPEL for
guestfs/hivex/ntfs packages). The container runs **privileged** so loop
devices are available when Podman/Docker is rootful; **root-selection tests
still skip under rootless Podman** (`losetup: Permission denied`):

```bash
make test-e2e-container
```

For root-selection and disk-image tests on rootless hosts, use
`make test-e2e-disk` (privileged container with loop setup).

Individual scripts can be skipped explicitly, e.g.
`SKIP_TEST_LINUX_RHEL_SH=1 make test-e2e` (see `skip_if_skipped` in
`tests/functions.sh`).

Per-script logs: `tests/<script-name>.log`.

## Architecture expectations

- Keep pipeline stages isolated: no cross-stage imports under `pkg/`.
- All guest disk access goes through `pkg/guest/`; never call mount/guestfish tools from block code.
- Prefer self-contained blocks and plugins over shorter coupled code.
- Full rules: [architecture.md](architecture.md).

## Guest conversion conventions

These patterns apply when adding or modifying guest filesystem changes in
`convert-linux` or `convert-windows`.

### Symlinks must use guest-absolute paths

When creating symlinks inside the guest (e.g. systemd unit enablement), the
**target** must be a guest-absolute path like `/etc/systemd/system/foo.service`,
not a host-absolute path that includes the mount root. The mount root prefix is
only valid on the conversion host — inside the booted guest it produces a
dangling symlink.

### Firstboot scripts must be append-safe

Multiple pipeline blocks may contribute firstboot commands. The Linux `systemd`
firstboot handler's `Install()` appends new commands before the self-cleanup
tail rather than overwriting the script. Callers should call `Install()` for
each set of commands; they do not need to manage the script format directly.

For Windows, `firstboot.bat` iterates `scripts/*.ps1` in sorted order and
self-cleans the `Guestfs\Firstboot` directory when done. Add new `.ps1` files
with an appropriate numeric prefix.

### Hypervisor cleanup uses shared helpers

Linux hypervisor plugins must use `hypervisor.DisableSystemdUnit()` to disable
and mask services consistently across all three directories (`multi-user.target.wants`,
`sockets.target.wants`, `graphical.target.wants`). Do not remove symlinks
manually.

Windows cleanup plugins that delete driver files should match by prefix +
`.sys` suffix (e.g. `xen*.sys`, `vbox*.sys`) using `guest.FileReadDir`.

### Prefer guest-native tools via RunInGuest

When the guest has standard tools for a task, use `guest.RunInGuest()` instead
of reimplementing the logic in Go. Example: initramfs rebuilds use the guest's
own `dracut` / `update-initramfs` / `mkinitramfs` rather than custom CPIO
manipulation. This gets correct module compression, dependency resolution, and
firmware inclusion for free.

### Offline removal vs firstboot

Prefer offline removal (deleting files, disabling services) over scheduling
cleanup at firstboot. Use firstboot only for operations that require a running
OS (package managers, PnP, network configuration). Offline changes are
deterministic and do not depend on the guest booting successfully.

## PR checklist

- [ ] `make check` passes
- [ ] Run relevant e2e tests when changing convert, guest, or pipeline behavior
- [ ] Keep the change focused
- [ ] Follow [architecture.md](architecture.md) for structural changes

## Docs

Update `docs/` and package READMEs when behavior or pipeline contracts change.
Product and pipeline documentation lives under [docs/](../docs/).
