# driversource plugins

`DriverSource` interface — locate VirtIO-Win driver files on the conversion host.

Windows guests need VirtIO drivers (storage, network, display, balloon, serial)
to boot and operate under KVM. These drivers are not part of Windows — they
must be injected during conversion from a pre-extracted driver tree on the
conversion host. Each driver source plugin knows where to find these files and
how to filter them for the guest's architecture and Windows version.

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

### directory

**What it does:** Locates VirtIO-Win driver files from a pre-extracted
directory tree on the conversion host, filtering by guest architecture (x86,
amd64) and Windows version.

**How it works:** `Available()` checks whether the driver directory exists.
`FindDrivers` walks the `by-os/` tree, matching directories against the guest's
OS version string (with version-specific preferences and fallbacks provided by
the `VersionHandler`). For each matching directory, collects `.sys`, `.inf`,
`.cat`, and `.msi` files into `DriverFile` structs that include the host path,
driver name, and metadata needed by the copy and registration blocks.

## Driver tree population

The kc-v2v container image downloads virtio-win ISOs from the public
[Fedora People archive](https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/archive-virtio/)
at build time (see [`build/kc-v2v/stage-virtio-win.sh`](../../../../build/kc-v2v/stage-virtio-win.sh)).
Drivers are extracted under `/usr/share/virtio-win/drivers/by-os/` and qemu-ga
MSIs under `/usr/share/virtio-win/guest-agent/`.

For local development, install the `virtio-win` package on Fedora/RHEL or
extract a virtio-win ISO into the same path.

[`CollectDrivers`](../collect.go) reads guest-agent MSIs from the directory
plugin, then omits `qemu-ga` entries when
[`CollectGuestAgentMSI`](../../version/guestagent.go) is false for the
classified handler (XP, 2003, Server 2008, Vista). It then runs
[`FilterComplete`](../complete.go) so only packages with all offline
payload files are kept. Kept packages have `Files` set for staging; the
`qemuga` firstboot contributor installs the exact selected MSI basename.
