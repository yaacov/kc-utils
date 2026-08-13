# guestagent plugins

Two plugin registries in this block for installing the QEMU Guest Agent on
Linux guests. The guest agent enables the KubeVirt host to communicate with the
running VM for graceful shutdown, filesystem freeze/thaw (for consistent
snapshots), and network information reporting. Installation is handled by a
combination of agent detection and package sourcing: the `GuestAgent` plugin
detects and manages the agent lifecycle, while the `PackageSource` plugin
locates the agent package on the conversion host.

## GuestAgent

| Key | Package | Description |
|-----|---------|-------------|
| `qemu-ga` | agent/qemuga/ | Detect / remove existing qemu-ga; firstboot installs via local RPM or network |

### qemu-ga

**What it does:** Detects an existing QEMU guest agent binary and ensures the
systemd unit is enabled for next boot. When the binary is absent, schedules a
firstboot installation from either a local package or the guest's package
manager.

**How it works:** During conversion, checks for the `qemu-ga` binary on the
guest. When present and `qemu-guest-agent.service` is already enabled and not
masked, no further work is done. When present but disabled or masked,
conversion enables the unit offline via `systemd.EnableSystemdUnit`. When the
binary is absent, locates a suitable package via the `PackageSource` registry
(the `directory` source). If a local RPM/DEB is available, copies it into the
guest and generates firstboot commands to install it via `rpm -ivh` /
`dpkg -i`. If no local package is found and the guest is online, generates
firstboot commands to install via the guest's package manager
(`dnf`/`yum`/`apt`/`zypper`). The firstboot commands are installed via the
shared `pkg/common/firstboot` handler (systemd oneshot service). The plugin
also exposes `Remove()` for tests; the conversion pipeline does not call it.

## PackageSource

| Key | Package | Description |
|-----|---------|-------------|
| `directory` | packagesource/directory/ | Local packages under `/usr/share/kc-packages` |

### Host layout (`directory`)

Preferred (RHEL family, used by the kc-v2v image):

```text
/usr/share/kc-packages/rpm/el8/x86_64/qemu-guest-agent-*.rpm
/usr/share/kc-packages/rpm/el9/x86_64/qemu-guest-agent-*.rpm
/usr/share/kc-packages/rpm/el10/x86_64/qemu-guest-agent-*.rpm
```

Lookup picks a **single** best match: exact `el{major}` from inspect, else nearest lower. Legacy flat `$base/rpm/$arch/` (and `deb/$arch`) still works for custom mounts.

Pins and download script: [`build/kc-v2v/stage-linux-packages.sh`](../../../../build/kc-v2v/stage-linux-packages.sh).

Note: Linux offline packages here are unrelated to VirtIO-Win drivers for Windows
(see [`pkg/convert-windows/driversource/plugins/`](../../convert-windows/driversource/plugins/)).

Firstboot support is provided by `pkg/common/firstboot/` (shared across converters).
