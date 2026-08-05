# driversource plugins

`DriverSource` interface — locate VirtIO-Win driver files on the conversion host.

```go
type DriverSource interface {
    Available() bool
    FindDrivers(arch, osVersion string) ([]DriverFile, error)
}
```

Conversion uses the **ISO** source only (`CollectDrivers`). The extract tree is
kept until after `drivers.Copy`, then cleaned up via `Cleaner`.

| Key | Package | Default path | Description |
|-----|---------|--------------|-------------|
| `iso` | iso/ | `/usr/share/virtio-win/virtio-win.iso` | Extract with `bsdtar` (no loop device). Filter by guest arch and Windows version |

## Linux distro packages

```bash
sudo dnf install -y virtio-win
```

Optional repo:

```bash
wget -qO- https://fedorapeople.org/groups/virt/virtio-win/virtio-win.repo \
  | sudo tee /etc/yum.repos.d/virtio-win.repo >/dev/null
```

On Debian/Ubuntu, download
[virtio-win.iso](https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/stable-virtio/virtio-win.iso)
to `/usr/share/virtio-win/virtio-win.iso`.

Guest agent MSIs under `/usr/share/guest-agent/` are not read by kc-convert-windows;
firstboot looks for `qemu-ga*.msi` under the guest's `Windows\Drivers\VirtIO\` after
driver copy.
