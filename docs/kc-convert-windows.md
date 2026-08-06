# kc-convert-windows Pipeline

Converts a Windows guest to run on KVM/virtio. Installs virtio-win drivers,
registers them in the Windows registry, disables hypervisor-specific services,
and generates firstboot scripts for post-reboot setup.

Requires Linux (`//go:build linux`).

## Entry Point

`cmd/kc-convert-windows/main.go` — orchestrator in `internal/convert-windows/pipeline.go`.

## CLI Flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--prepare-data` | yes | | Path to PrepareOutput JSON (from kc-prepare) |
| `--output` | no | `convert-out.json` | Path to write ConverterOutput JSON |
| `--mount-root` | no | `/tmp/kc-guest` | Host directory where guest filesystems are mounted |
| `--offline` | no | `false` | Skip network-only firstboot operations while still scheduling local guest-agent/driver setup |
| `--log-level` | no | `info` | Log level (`debug`, `info`, `warn`, `error`) |

VirtIO drivers are located from the pre-extracted virtio-win tree via the
`directory` `DriverSource` plugin at `/usr/share/virtio-win/drivers/by-os`.

## Pipeline Blocks

| # | Block | Type | Package | Description |
|---|-------|------|---------|-------------|
| 1 | Version | pluggable: `VersionHandler` | `pkg/convert-windows/version/` | Classify Windows era (win2008, win10, …) |
| 2 | Driver Source | pluggable: `DriverSource` | `pkg/convert-windows/driversource/` | Find virtio-win drivers from pre-extracted RPM tree |
| 2b | Antivirus Detection | strict | `pkg/convert-windows/inspect/` | Detect antivirus products (warnings) |
| 3 | RTC Mode | strict | `pkg/convert-windows/inspect/` | Detect RTC UTC/local mode |
| 4 | Hypervisor Remove | pluggable: `WindowsRemove` | `pkg/convert-windows/hypervisor/` | Remove hypervisor-specific software |
| 5 | Driver Copy | strict | `pkg/convert-windows/drivers/` | Copy virtio driver files into the guest |
| 6 | Driver Register | pluggable: `DriverRegistrar` | `pkg/convert-windows/drivers/` | Register drivers in Windows registry |
| 7 | DevicePath | strict | `pkg/convert-windows/drivers/` | Update DevicePath registry key |
| 9 | Hypervisor Services | pluggable: `WindowsServices` | `pkg/convert-windows/hypervisor/` | Disable hypervisor services via registry |
| 9 | Crash Control | strict | `pkg/convert-windows/crashcontrol/` | Disable auto-reboot on BSOD |
| 10-12 | Firstboot | strict | `pkg/convert-windows/firstboot/` | Generate version-appropriate firstboot scripts |
| 13 | NTFS Fix | strict | `pkg/convert-windows/ntfsfix/` | Patch NTFS boot sector for pre-Vista Windows |
| 14 | UEFI | pluggable: `UEFIEditor` | `pkg/common/uefi/` | Update UEFI boot entries on ESP partitions |
| 15 | Output | strict | `pkg/convert-windows/output/` | Build GuestCaps and fix permissions |
| 17 | Post-Convert | strict | `pkg/convert-windows/output/` | Post-convert permission fixup |

Block numbers match the pipeline comments in `internal/convert-windows/pipeline.go`.

## Input

- `PrepareOutput` JSON (from kc-prepare): OS info, disk layout, mount paths
- Mounted guest filesystem at `--mount-root`

VirtIO drivers are read from the **conversion host**, not from JSON or CLI flags.
Install the Linux `virtio-win` package before running the converter locally.

### Linux packages (Fedora / RHEL) — supported

```bash
sudo dnf install -y virtio-win
```

The `virtio-win` RPM installs drivers under `/usr/share/virtio-win/drivers/by-os/`.
The kc-v2v container image ships this tree directly. The image build also
**best-effort** stages per-version virtio-win **1.9.12**-era OS directories
(`2k8`, `2k3`, `xp`, `vista`) when a vendor artifact is present — see
[`build/kc-v2v/vendor/README.md`](../build/kc-v2v/vendor/README.md). Without
it, pre–Win 8 conversion fails at runtime with a clear error.

| Plugin | Path | Notes |
|--------|------|-------|
| `directory` | `/usr/share/virtio-win/drivers/by-os` | Match guest arch and Windows version; qemu-ga MSIs from `/usr/share/virtio-win/guest-agent/` when [`CollectGuestAgentMSI`](../pkg/convert-windows/version/guestagent.go) allows (omitted for XP, 2003, Server 2008, Vista) |

## Version classification

Windows guests are classified into version handlers (`win2008`, `win10`, …)
at pipeline block 1. Handlers drive virtio-win OS directory selection and
firstboot script variants.

See [guest-os-handlers.md](guest-os-handlers.md) for the full handler matrix,
code locations, and archived driver merge details. Summary tables also appear
below under **Driver source** and **Firstboot scripts**.

Optional upstream repo:

```bash
wget -qO- https://fedorapeople.org/groups/virt/virtio-win/virtio-win.repo \
  | sudo tee /etc/yum.repos.d/virtio-win.repo >/dev/null
```

There is no JSON field for driver location — install the host package before
conversion (or use the kc-v2v image, which includes it).

See [pkg/convert-windows/driversource/plugins/README.md](../pkg/convert-windows/driversource/plugins/README.md) for
plugin details. Linux guest offline packages (`qemu-guest-agent` RPM/DEB) are a
separate mechanism in `kc-convert-linux`, not VirtIO-Win.

Prepare input with static IPs: [examples/prepare-input-windows.json](examples/prepare-input-windows.json).

## Output

`ConverterOutput` JSON containing:

- `guestcaps` -- capabilities derived from copied VirtIO driver names:
  - `block_bus` -- `virtio` when viostor/vioscsi was copied, else `ide`
  - `net_bus` -- `virtio` when netkvm was copied, else `e1000`
  - feature flags (`virtio_rng`, balloon, socket) from matching drivers
  - `rtc_utc` -- whether the guest uses UTC for the hardware clock
- `warnings` -- non-fatal messages (for example detected antivirus products)

Example: [examples/convert-output-windows.json](examples/convert-output-windows.json).

## Plugin Implementations

| Interface | Implementations |
|-----------|----------------|
| `VersionHandler` | `win11`, `win10`, `win81`, `win8`, `win7`, `win2008r2`, `win2008`, `winvista`, `win2003`, `winxp`, `winunknown` |
| `DriverSource` | `directory` |
| `WindowsRemove` | `vmware`, `nutanix`, `awspv`, `ec2launch`, `ec2`, `virtualbox` |
| `DriverRegistrar` | `criticaldb` (legacy Windows), `driverdb` (Windows 8+) |
| `WindowsServices` | `vmware`, `nutanix`, `virtualbox` |
| `UEFIEditor` | `bcd` |

## Registry Access

Windows registry reads use the Go parser in `pkg/common/registry/`. Writes are
flushed through `hivexregedit` when the hive is saved.

Key registry operations:

- **CriticalDeviceDB** (legacy) -- `PCI#VEN_…&DEV_…&REV_…` keys so Windows loads
  viostor/vioscsi at boot before PnP runs.
- **DriverDatabase** (modern) -- `DriverInfFiles` / `DeviceIds` / packages plus
  Services `ImagePath` under `system32\drivers\`.
- **Services** -- disables hypervisor services by setting `Start=4`.
- **DevicePath** -- appends `\Drivers\VirtIO` so Windows finds INF files.

## Firstboot Scripts

Firstboot scripts live under `C:\Program Files\Guestfs\Firstboot\scripts\`
(`.ps1` and/or `.bat` depending on version handler), launched by
`firstboot.bat` via a RunOnce registry entry.

| Contributor | win2008 / 2003 / XP | win7 / 2008 R2 | win8+ |
|-------------|---------------------|----------------|-------|
| `pnputil` | PS1 or `.bat` | PS1 | PS1 |
| `staticipfb` | WMI+netsh or registry `.bat` | registry PS | `Get-NetAdapter` PS |
| `diskonliner` | skipped (2008/XP) or WMI+diskpart | WMI+diskpart | `Get-Disk` |
| `routecleanup` | skipped (WMI static IP) | full PS | full PS |
| `qemuga` | skipped when no MSI / unsupported | when MSI staged | when MSI staged |
| `vmwarecleanup` | `.bat` service cleanup | `.bat` or PS PnP | PS PnP |
| `signal` | unchanged (COM1) | unchanged | unchanged |

Legacy launchers set PowerShell 1.0 execution policy via `reg add` before
running `.ps1` scripts. Batch-only guests (XP/2003) run `.bat` contributors only.

| Script | Purpose |
|--------|---------|
| `2000-install-virtio-drivers` | Run `pnputil` for VirtIO INF files |
| `2500-static-ip` | Configure static IPs after netkvm is available (skipped with `--offline`) |
| `2600-remove-duplicate-routes` | Remove stale persistent routes |
| `2700-preserve-complementary-ips` | Add secondary IPs to NICs |
| `3000-install-qemu-ga` | Install QEMU guest agent MSI when `qemu-ga` was collected into `DriverFiles` and copied to the guest (skipped when MSI not collected, e.g. XP/2003/2008/Vista handlers) |
| `4000-disk-onliner` | Bring offline VirtIO disks online |
| `9100-cleanup-vmware` | Disable VMware PnP devices (conditional on VMware source) |
| `99999-signal-conversion-done` | Write `CONVERSION_DONE` to COM1 (conditional) |

After all scripts complete, `firstboot.bat` removes the
`C:\Program Files\Guestfs\Firstboot` directory and its contents.
