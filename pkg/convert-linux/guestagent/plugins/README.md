# guestagent plugins

Two plugin registries in this block:

## GuestAgent

| Key | Package | Description |
|-----|---------|-------------|
| `qemu-ga` | agent/qemuga/ | Detect / remove existing qemu-ga; firstboot installs via local RPM or network |

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
