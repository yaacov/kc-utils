# luks -- LUKS key file scanning

Scans a directory (typically `/etc/luks`) for LUKS encryption key files and builds a map that can be passed to decryption routines. This allows the prepare pipeline to automatically discover and apply key files to encrypted guest volumes.

`ScanKeyFiles` lists all regular files in the given directory, skipping subdirectories and dotdot-prefixed entries (Kubernetes projected-volume metadata). `KeyFilesMap` wraps `ScanKeyFiles` and returns a `map[string]string` where each key is an `"all"` sentinel (or `"all-N"` when multiple files exist), matching virt-v2v's `all:file:` semantics so decryption tries every key on every encrypted volume.

## Key exports

| Symbol | Role |
|--------|------|
| `ScanKeyFiles` | Returns absolute paths of key files found in a directory |
| `KeyFilesMap` | Builds a device-to-keyfile map using `"all"` sentinels for broadcast matching |
