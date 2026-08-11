# systemd — shared guest systemd unit helpers

Stage-local helpers for masking/disabling systemd units during Linux conversion. Not a pipeline block — imported by hypervisor cleanup plugins and `network/networkd` detection.

| Function | Role |
|----------|------|
| `DisableSystemdUnit` | Remove wants symlinks and mask unit to `/dev/null` |
| `SystemdUnitMaskPath` | Host path to the mask symlink for a unit |
| `VendorWantsPath` | Host path to vendor preset wants symlink |
| `UnitWantsEnabled` | True when unit has a wants symlink under standard targets |
| `UnitIsMasked` | True when unit is masked to `/dev/null` |
| `DisableEC2NetHooks` | Mask EC2 IMDS hostname and policy-route template units |
| `RemovePaths` | Delete guest paths via `guest.FileRemoveAll` |

EC2 hypervisor cleanup calls `DisableEC2NetHooks`. Other hypervisor plugins use `DisableSystemdUnit` and `RemovePaths`.
