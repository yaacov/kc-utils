# Guest OS handlers reference

kc-utils classifies Linux and Windows guests early in each conversion pipeline,
then uses that classification to pick version-specific drivers, packages, boot
settings, and firstboot scripts. Both sides use the same plugin pattern:
handlers register in `init()`, and the pipeline picks the first match.

```text
kc-prepare  →  InspectData (distro / major / minor / product_name)
                    │
        ┌───────────┴───────────┐
        v                       v
 kc-convert-linux         kc-convert-windows
 Block 1: distro.Handlers  Block 1: version.Classify
```

| OS | Registry | Classifier | Orchestrator |
|----|----------|------------|--------------|
| Linux | [`pkg/convert-linux/distro/`](../pkg/convert-linux/distro/) | First matching `DistroHandler.Matches` | [`internal/convert-linux/pipeline.go`](../internal/convert-linux/pipeline.go) |
| Windows | [`pkg/convert-windows/version/`](../pkg/convert-windows/version/) | First matching handler in registration order | [`internal/convert-windows/pipeline.go`](../internal/convert-windows/pipeline.go) |

---

## Linux distros

### Handler plugins

| Handler | Source file | Matches `inspect.distro` |
|---------|-------------|--------------------------|
| `rhel` | [`pkg/convert-linux/distro/plugins/rhel/rhel.go`](../pkg/convert-linux/distro/plugins/rhel/rhel.go) | `rhel`, `centos`, `rocky`, `almalinux`, `ol`, `fedora`, `amzn` |
| `debian` | [`pkg/convert-linux/distro/plugins/debian/debian.go`](../pkg/convert-linux/distro/plugins/debian/debian.go) | `debian`, `ubuntu` |
| `suse` | [`pkg/convert-linux/distro/plugins/suse/suse.go`](../pkg/convert-linux/distro/plugins/suse/suse.go) | `sles`, `opensuse-leap`, `opensuse-tumbleweed` |

If no handler matches, the pipeline logs a warning and continues with generic
defaults (console `ttyS0`, RPM/dnf assumptions below).

Registration is via blank import in [`cmd/kc-convert-linux/main.go`](../cmd/kc-convert-linux/main.go).

### What each handler controls today

| Handler | `DefaultConsole()` | `DefaultKernelArgs()` | Used in pipeline? |
|---------|-------------------|----------------------|-------------------|
| `rhel` | `ttyS0` | `console=ttyS0`, `crashkernel=auto` | Console only |
| `debian` | `ttyS0` | `console=ttyS0` | Console only |
| `suse` | `ttyS0` | `console=ttyS0` | Console only |

**Console** — [`pkg/convert-linux/bootconfig/console.go`](../pkg/convert-linux/bootconfig/console.go)
(`ConfigureConsole`, pipeline block 9):

- Removes `rhgb` and `quiet` from the bootloader config.
- Adds `console=<DefaultConsole()>` (always `ttyS0` for all three handlers today).

`DefaultKernelArgs()` is part of the interface and tested per plugin, but is
**not yet applied** by the pipeline; kernel args are adjusted only via console
setup and bootloader plugins.

### Package format and package manager (parallel lookup)

Separate from `DistroHandler`, [`distro.Format`](../pkg/convert-linux/distro/distro.go)
and [`distro.Name`](../pkg/convert-linux/distro/distro.go) map the raw
`inspect.distro` string (pipeline blocks 2–3):

| `inspect.distro` | Format (`Format`) | Package manager (`Name`) |
|------------------|-------------------|--------------------------|
| `debian`, `ubuntu` | `deb` | `apt` |
| `sles`, `opensuse-leap`, `opensuse-tumbleweed` | `rpm` | `zypper` |
| `rhel`, `centos`, `rocky`, `alma`, `ol`, `fedora` | `rpm` | `dnf` |
| anything else | `rpm` (warn) | `dnf` |

These drive:

| Concern | Code | Behavior |
|---------|------|----------|
| Kernel scan | [`internal/convert-linux/pipeline.go`](../internal/convert-linux/pipeline.go) `scanKernels` | Tries `rpm` scanner first, then `deb` |
| qemu-guest-agent install | [`pkg/convert-linux/guestagent/install.go`](../pkg/convert-linux/guestagent/install.go) | Local copy + firstboot `rpm -ivh` / `dpkg -i`, or network `dnf`/`apt`/`zypper` |
| Offline QGA package pick | [`pkg/convert-linux/guestagent/plugins/packagesource/directory/`](../pkg/convert-linux/guestagent/plugins/packagesource/directory/) | RPM: versioned `rpm/el{N}/{arch}/` (exact `el{major}` or nearest lower); DEB: flat `deb/{arch}/` |

RHEL-family offline packages are staged in the kc-v2v image under
`/usr/share/kc-packages/rpm/el{8,9,10}/x86_64/` by
[`build/kc-v2v/stage-linux-packages.sh`](../build/kc-v2v/stage-linux-packages.sh).
Selection uses `inspect.distro` + `inspect.major_version`, not the distro
handler name.

### Adding a Linux handler

1. Implement `DistroHandler` in `pkg/convert-linux/distro/plugins/<name>/`.
2. Register in `init()` with `distro.Handlers.Register`.
3. Blank-import the plugin from `cmd/kc-convert-linux/main.go`.
4. Extend `distro.Format` / `distro.Name` if the new family needs a non-default
   package format or manager.

---

## Windows versions

### Handler plugins and classification order

Handlers register most-specific-first in
[`pkg/convert-windows/version/register.go`](../pkg/convert-windows/version/register.go).
[`version.Classify`](../pkg/convert-windows/version/version.go) returns the
first `Matches()` hit, or `winunknown`.

Product names are normalized before substring checks (strip NUL, `(R)`, `(TM)`,
collapse whitespace) in
[`pkg/convert-windows/version/match.go`](../pkg/convert-windows/version/match.go).
The same normalization exists for driver alias expansion in
[`pkg/convert-windows/driversource/osversion.go`](../pkg/convert-windows/driversource/osversion.go).

| Handler | Match rule (primary) | Example guests |
|---------|----------------------|----------------|
| `win11` | NT ≥ 10 and product contains Windows 11 / Server 2022 / 2025 | Windows 11, Server 2022 |
| `win10` | NT ≥ 10, not Win 11 | Windows 10, Server 2016–2019 |
| `win81` | 6.3 | Windows 8.1, Server 2012 R2 |
| `win8` | 6.2 | Windows 8, Server 2012 |
| `win7` | 6.1 client, or product contains “windows 7” | Windows 7 |
| `win2008r2` | 6.1 server, or product contains “2008 r2” | Server 2008 R2 |
| `win2008` | 6.0 server, or “server 2008” (not R2) | Server 2008 |
| `winvista` | 6.0 client, or product contains “vista” | Windows Vista |
| `win2003` | 5.2, or product contains “server 2003” | Server 2003 |
| `winxp` | 5.1, or product contains “windows xp” | Windows XP |
| `winunknown` | fallback | Unrecognized product / missing inspect |

Implementation: [`pkg/convert-windows/version/handlers.go`](../pkg/convert-windows/version/handlers.go).

### Per-version behavior matrix

Columns describe what the handler returns; pipeline and firstboot plugins
consume these flags.

**Virtio-win OS dir** — expected `by-os/<arch>/<dir>/` under
`/usr/share/virtio-win/drivers/`. Dirs marked **†** are not in the default
1.9.40 RPM; conversion fails unless the kc-v2v image was built with legacy
vendor staging ([`build/kc-v2v/vendor/README.md`](../build/kc-v2v/vendor/README.md)).

**QEMU-GA** — whether qemu-ga is collected and installed. Handlers
`win2008`, `winvista`, `win2003`, and `winxp` omit GA MSIs during
[`CollectDrivers`](../pkg/convert-windows/driversource/collect.go) (see
[`CollectGuestAgentMSI`](../pkg/convert-windows/version/guestagent.go)).
**yes‡** means the handler collects GA when a matching `qemu-ga*.msi` exists
under `/usr/share/virtio-win/guest-agent/`; the `qemuga` firstboot contributor
runs only when `qemu-ga` appears in `DriverFiles`. **—** means GA is not
collected for that handler.

| Handler | Virtio-win OS dir | Firstboot launcher | Static IP | Disk online | VMware cleanup | QEMU-GA |
|---------|-------------------|--------------------|-----------|-------------|----------------|---------|
| `win11` | `w11` | Modern PS | Net cmdlets | `Get-Disk` | PS PnP | yes‡ |
| `win10` | `w10` | Modern PS | Net cmdlets | `Get-Disk` | PS PnP | yes‡ |
| `win81` | `w8.1` | Modern PS | Net cmdlets | `Get-Disk` | PS PnP | yes‡ |
| `win8` | `w8` | Modern PS | Net cmdlets | `Get-Disk` | PS PnP | yes‡ |
| `win7` | `w7` | PS 1.0 (reg execution policy) | Registry PS | WMI + diskpart | `.bat` | yes‡ |
| `win2008r2` | `2k8r2` | PS 1.0 | Registry PS | WMI + diskpart | `.bat` | yes‡ |
| `win2008` | `2k8` (fallback `2k8R2`) | PS 1.0 | WMI + netsh | **skipped** | `.bat` | — |
| `winvista` | `vista` † | PS 1.0 | WMI + netsh | WMI + diskpart | `.bat` | — |
| `win2003` | `2k3` † | Batch only | Registry `.bat` | WMI + diskpart | `.bat` | — |
| `winxp` | `xp` † | Batch only | Registry `.bat` | **skipped** | `.bat` | — |
| `winunknown` | generic alias match | Modern PS | Net cmdlets | `Get-Disk` | PS PnP | yes‡ |

**Launcher kinds** ([`pkg/convert-windows/firstboot/firstboot.go`](../pkg/convert-windows/firstboot/firstboot.go)):

- **Modern PS** — run `.bat` contributors, then `powershell -ExecutionPolicy Bypass -File` for `.ps1`.
- **PS 1.0** — `reg add` execution policy, then `powershell -File` (no `-ExecutionPolicy` flag).
- **Batch only** — `.bat` contributors only (XP / 2003).

### Where the Windows handler is consumed

| Stage | Code | How handler is used |
|-------|------|---------------------|
| Driver lookup | [`driversource.CollectDrivers`](../pkg/convert-windows/driversource/collect.go) → [`FindBestOSDirWithPrefs`](../pkg/convert-windows/driversource/osdir.go) | Passes `DriverOSPreferences()` and optional `DriverOSFallbacks()` (e.g. `win2008` → `2k8R2` when `2k8` is absent); omits qemu-ga MSIs when [`CollectGuestAgentMSI`](../pkg/convert-windows/version/guestagent.go) is false |
| Firstboot launcher | [`firstboot.Configure`](../pkg/convert-windows/firstboot/firstboot.go) | `Version.FirstbootLauncher()` selects `firstboot.bat` template |
| Firstboot contributors | [`pkg/convert-windows/firstboot/plugins/*`](../pkg/convert-windows/firstboot/plugins/) | Each contributor reads `ContributorConfig.Version` — e.g. `pnputil` emits `.bat` when `!SupportsPowerShell()`, `diskonliner` skips when `DiskOnlineSkip`, `qemuga` runs when `qemu-ga` is in `DriverFiles` |
| Static IP scripts | [`staticip`](../pkg/convert-windows/staticip/staticip.go) + [`staticipfb`](../pkg/convert-windows/firstboot/plugins/staticipfb/) | `StaticIPNetCmdlet`, `StaticIPRegistry`, or `StaticIPWMINetsh` |
| VMware cleanup | [`vmwarecleanup`](../pkg/convert-windows/firstboot/plugins/vmwarecleanup/) | PS PnP vs [`DevconVMwareCleanupBat`](../pkg/convert-windows/staticip/staticip.go) |

### Version-specific logic outside the handler (NT major/minor)

These paths still use raw `inspect.major_version` / `minor_version` rather than
the classified handler:

| Concern | Threshold | Code | Behavior |
|---------|-----------|------|----------|
| Boot-time driver registration | NT &lt; 6.2 → `criticaldb`; ≥ 6.2 → `driverdb` | [`internal/convert-windows/pipeline.go`](../internal/convert-windows/pipeline.go) `registerDrivers` | Pre-Win8 guests need CriticalDeviceDatabase entries so viostor/vioscsi load before PnP |
| NTFS boot sector heads | NT major &lt; 6 (pre-Vista) | [`pkg/convert-windows/ntfsfix/ntfsfix.go`](../pkg/convert-windows/ntfsfix/ntfsfix.go) | Patch `$NumberOfHeads` in the NTFS boot sector for virt-v2v parity |

### Pre–Win 8 virtio-win drivers

Server 2003, XP, and Vista need SHA-1-era drivers that are not in the modern
virtio-win 1.9.40 tree. **Server 2008** prefers `2k8` but falls back to
`2k8R2` when only the modern RPM is staged (same behavior as virtio-win 1.9.40
images without legacy vendor merge). The kc-v2v image build **best-effort**
stages each missing by-os directory when a vendor artifact is present (like
`rpm/el8`, `el9`, `el10` for Linux QGA). **Image build succeeds without
legacy vendor files**; XP/2003/Vista conversion fails at runtime with a hint.

There is no open/free public source for legacy `2k8`/`2k3`/`xp`/`vista` dirs —
public el8+ virtio-win RPMs strip them at build time. The known-good artifact is
virtio-win **1.9.12-4.el7** (RHEL supplementary; entitlement required).

- Script: [`build/kc-v2v/stage-windows-virtio-drivers.sh`](../build/kc-v2v/stage-windows-virtio-drivers.sh)
- Optional vendor prep: [`prepare-windows-virtio-drivers.sh`](../build/kc-v2v/prepare-windows-virtio-drivers.sh) — see [`build/kc-v2v/vendor/README.md`](../build/kc-v2v/vendor/README.md)
- Dirs staged per arch: `2k8`, `2k3`, `xp`, `vista`

At runtime, each version handler uses `DriverOSPreferences()` for its primary
by-os dir. `win2008` alone may use `DriverOSFallbacks()` (`2k8R2`) when `2k8`
is missing. Other pre–Win 8 handlers with no fallback produce an error naming
the handler, required dir, and vendor README hint. There is no virtio-win ISO
or `VIRTIO_WIN` env path — lookup is directory-only under
`/usr/share/virtio-win/drivers/by-os/`.

### Adding a Windows version handler

1. Add a type implementing `VersionHandler` in
   [`pkg/convert-windows/version/handlers.go`](../pkg/convert-windows/version/handlers.go).
2. Register it in
   [`pkg/convert-windows/version/register.go`](../pkg/convert-windows/version/register.go)
   **before** less-specific handlers (broader matches go last).
3. Blank-import `pkg/convert-windows/version` from
   [`cmd/kc-convert-windows/main.go`](../cmd/kc-convert-windows/main.go) (triggers
   `register.go` `init`).
4. Add classification tests in
   [`pkg/convert-windows/version/version_test.go`](../pkg/convert-windows/version/version_test.go).
5. If driver dirs are new, extend
   [`CanonicalOSVersions`](../pkg/convert-windows/driversource/osversion.go) and
   ensure the virtio-win tree (or archived merge script) contains matching
   `by-os/<arch>/<dir>/` paths.

---

## Related docs

- [kc-convert-linux.md](kc-convert-linux.md) — full Linux pipeline blocks
- [kc-convert-windows.md](kc-convert-windows.md) — full Windows pipeline blocks
- [build/kc-v2v/README.md](../build/kc-v2v/README.md) — image-staged guest packages and archived virtio-win dirs
