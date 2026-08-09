# fstrim -- Trimmer plugin interface

Defines the extension point for filesystem trim operations during the finalize phase. Trimmers discard unused blocks on a mounted filesystem, reducing the effective size of the target disk image.

Implementations register themselves with the package-level `Trimmers` registry. At finalize time the runner calls `Trim` on each registered trimmer, passing the mountpoint path of the guest filesystem partition.

## Key exports

| Symbol | Role |
|--------|------|
| `Trimmer` | Interface with `Trim(mountpoint string) error` |
| `Trimmers` | Global `plugin.Registry[string, Trimmer]` for registering/looking up trimmer implementations |
