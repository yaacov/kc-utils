# driversource plugins

`DriverSource` interface — locate VirtIO-Win driver files on the conversion host.

```go
type DriverSource interface {
    Available() bool
    FindDrivers(arch, osVersion string) ([]DriverFile, error)
}
```

Conversion uses the **directory** source only (`CollectDrivers`).

| Key | Package | Default path | Description |
|-----|---------|--------------|-------------|
| `directory` | directory/ | `/usr/share/virtio-win/drivers/by-os` | Read pre-extracted RPM driver tree. Filter by guest arch and Windows version |

## Linux distro packages

```bash
sudo dnf install -y virtio-win
```

The `virtio-win` RPM installs drivers under `/usr/share/virtio-win/drivers/by-os/`
and qemu-ga MSIs under `/usr/share/virtio-win/guest-agent/`.

The kc-v2v container image ships this tree directly (no ISO).

Guest agent MSIs are staged into the guest's `Windows\Drivers\VirtIO\` during
driver copy; firstboot installs them via `qemu-ga*.msi`.
