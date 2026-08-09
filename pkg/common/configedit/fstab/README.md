# fstab -- /etc/fstab parser and editor

Used by both prepare (fstab mount planner) and convert-linux (device remap block). Lives in `common/configedit/` because the prepare stage parses fstab to plan mounts, while the convert stage later rewrites device paths in the same file — sharing the parser ensures identical round-trip serialization.

Parses and edits `/etc/fstab` and similar tabular configuration files (e.g., `/etc/crypttab`). During conversion, device paths often change (e.g., `/dev/sda` to `/dev/vda`), so this package provides device remapping alongside standard parse-edit-serialize operations.

`Parse` splits content into lines and classifies each as a comment, blank line, or device entry with up to six whitespace-delimited fields (device, mount point, fs type, options, dump, pass). `RemapDevice` replaces device-path prefixes in the first column only, while `RemapAllFields` replaces prefixes across all text fields (useful for crypttab). `DeviceEntries` filters out comments and blanks. `String` serializes back to tab-separated fstab format, defaulting dump/pass to `0` and options to `defaults` when empty.

## Key exports

| Symbol | Role |
|--------|------|
| `File` | Represents a parsed fstab file (slice of `Entry` values) |
| `Entry` | A single fstab line: Device, MountPoint, FSType, Options, Dump, Pass, Comment |
| `Parse(content string)` | Parse fstab content into a `File` |
| `(*File).RemapDevice(oldPrefix, newPrefix)` | Replace device-path prefixes in column 1 |
| `(*File).RemapAllFields(oldPrefix, newPrefix)` | Replace device-path prefixes in all text fields |
| `(*File).DeviceEntries()` | Return non-comment, non-empty entries |
| `(*File).String()` | Serialize back to fstab format |
