# overlay -- qcow2 overlay creation

Creates temporary qcow2 overlay files backed by raw disk images so that in-place conversion can modify guest filesystems without altering the original disk data. If the conversion succeeds the overlays are committed back; on failure they are discarded, leaving the backing disks untouched.

`CreateOverlays` iterates over a slice of `Disk` structs, runs `qemu-img create` for each to produce a qcow2 overlay in the working directory, and redirects `disk.Path` to point at the overlay. `CommitOverlays` merges the changes back into the backing images with `qemu-img commit` and restores the original paths. `DiscardOverlays` removes the overlay files without committing. `RunWithOverlay` wraps an arbitrary function with create/commit/discard semantics as a convenience.

## Key exports

| Symbol | Role |
|--------|------|
| `Disk` | Struct describing a disk with backing, current, and overlay paths |
| `Overlay` | Struct tracking one qcow2 overlay and its associated `Disk` |
| `CreateOverlays` | Creates qcow2 overlays in a workdir and redirects disk paths to them |
| `CommitOverlays` | Merges overlays into backing disks via `qemu-img commit` |
| `DiscardOverlays` | Removes overlay files without committing changes |
| `RunWithOverlay` | Wraps a function with overlay create/commit/discard lifecycle |
