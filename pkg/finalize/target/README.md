# target -- Firmware resolution and bus/NIC assignment

Resolves the target VM firmware type and maps source disks and NICs to their target bus slots. These mappings feed into the final pipeline metadata that the migration controller uses to create the KubeVirt VirtualMachine spec.

`Target` normalizes the firmware string, defaulting empty values to BIOS. `Buses` iterates source disks and assigns each to a bus slot (virtio, SCSI, or IDE) based on the requested bus type. `NICs` converts source NIC metadata into target NIC entries with the requested network bus model; when no source NICs are available it produces a single default NIC with a zero MAC address.

## File layout

| File | Purpose |
|------|---------|
| `fwresolve.go` | `Target` -- firmware string normalization (defaults empty to BIOS) |
| `busassign.go` | `Buses` and `NICs` -- disk bus slot mapping and NIC assignment |

## Key exports

| Symbol | Role |
|--------|------|
| `Target` | Returns the firmware string, defaulting empty to `"bios"` |
| `Buses` | Maps a slice of `DiskInfo` to `TargetBuses` under the given bus type (virtio/scsi/ide) |
| `NICs` | Converts source `NICSpec` entries to `TargetNIC` entries with the specified network bus model |
