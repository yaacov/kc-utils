# guest -- guest disk preparation helpers

Container package that groups sub-packages for preparing guest disk images before conversion. Each sub-package handles one aspect of guest disk setup: scanning for LUKS encryption key files, creating qcow2 overlay files for safe in-place modification, and resolving fstab-style device specifiers to host block device paths.

## Sub-packages

| Package | Role |
|---------|------|
| [luks/](luks/) | Scans a directory for LUKS key files and builds a device-to-keyfile map |
| [overlay/](overlay/) | Creates, commits, and discards qcow2 overlay files over raw backing disks |
| [resolve/](resolve/) | Resolves UUID=, LABEL=, and /dev/ device specifiers to block device paths |
