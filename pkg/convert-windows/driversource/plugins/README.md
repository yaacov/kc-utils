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
| `directory` | directory/ | `/usr/share/virtio-win/drivers/by-os` | Read pre-extracted driver tree. Filter by guest arch and Windows version |

## Driver tree population

The kc-v2v container image downloads virtio-win ISOs from the public
[Fedora People archive](https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/archive-virtio/)
at build time (see [`build/kc-v2v/download-virtio-win.sh`](../../../../build/kc-v2v/download-virtio-win.sh)).
Drivers are extracted under `/usr/share/virtio-win/drivers/by-os/` and qemu-ga
MSIs under `/usr/share/virtio-win/guest-agent/`.

For local development, install the `virtio-win` package on Fedora/RHEL or
extract a virtio-win ISO into the same path.

[`CollectDrivers`](../../collect.go) reads guest-agent MSIs from the directory
plugin, then omits `qemu-ga` entries when
[`CollectGuestAgentMSI`](../../version/guestagent.go) is false for the
classified handler (XP, 2003, Server 2008, Vista). Guest agent MSIs that
remain in `DriverFiles` are staged into the guest's `Windows\Drivers\VirtIO\`
during driver copy; the `qemuga` firstboot contributor installs them via
`qemu-ga*.msi`.
