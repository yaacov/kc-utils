# guestcleanup -- cache and modprobe alias cleanup

Removes stale caches and reconfigures kernel module aliases after a VM conversion so the guest boots cleanly with virtio drivers instead of leftover hypervisor-specific module bindings.

`Run` is the entry point and calls `Clean` followed by `Configure`. `Clean` deletes stale blkid/LVM caches that reference old device names and removes RPM DB lock files that could block firstboot package installs. `Configure` scans all `.conf` files in `/etc/modprobe.d/` and strips any `alias`, `install`, `options`, or `blacklist` directives that reference hypervisor modules (VMware, Hyper-V, Xen, VirtualBox), then writes a `kc-virtio.conf` file that aliases SCSI host adapters and the primary NIC to their virtio equivalents.

## File layout

| File | Purpose |
|------|---------|
| `guestcleanup.go` | `Run` entry point that calls `Clean` and `Configure` |
| `cache.go` | `Clean` removes stale blkid/LVM caches and RPM DB lock files |
| `modalias.go` | `Configure` writes virtio modprobe aliases and removes stale hypervisor aliases |

## Key exports

| Symbol | Role |
|--------|------|
| `Run` | Entry point: cleans caches and reconfigures modprobe aliases |
| `Clean` | Removes stale blkid/LVM caches and RPM DB lock files from the guest |
| `Configure` | Writes virtio module aliases to `kc-virtio.conf` and strips stale hypervisor aliases |
