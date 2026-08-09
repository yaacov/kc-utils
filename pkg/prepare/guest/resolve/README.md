# resolve -- UUID/LABEL to device resolution

Resolves fstab-style device specifiers (`UUID=`, `LABEL=`, or `/dev/` paths) to actual host block device paths. This is used during mount planning to translate guest fstab entries into devices the host can mount.

`NewCatalog` builds a lookup table by iterating over known block devices and querying each for its UUID and LABEL via the guest wrapper's `BlkidAttr` method. The resulting `Catalog` stores three maps: `ByPath`, `ByUUID`, and `ByLabel`. The `Resolve` method on `Catalog` parses the incoming specifier prefix to choose the correct map, performing case-insensitive matching for UUIDs. For bare `/dev/` paths it falls back to `os.Stat` when the path is not in the catalog.

## Key exports

| Symbol | Role |
|--------|------|
| `Catalog` | Holds ByPath, ByUUID, and ByLabel lookup maps for device resolution |
| `NewCatalog` | Builds a `Catalog` from a list of block devices via the guest wrapper |
| `Catalog.Resolve` | Maps a `UUID=`, `LABEL=`, or `/dev/` specifier to a host device path |
| `AllDevices` | Combines disk partition paths and LVM volume paths into a single slice |
