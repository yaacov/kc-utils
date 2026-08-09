# configedit -- Linux configuration file editors

Shared across convert-linux blocks (bootloader, remap, guest cleanup) and prepare (fstab-based mount planning). Lives in `common/` because multiple blocks in different stages edit the same guest config files — for example, `fstab/` is used by both the prepare mount planner and the convert-linux remap block. Centralizing the parsers avoids divergent round-trip serialization logic.

Container package grouping parse-edit-serialize libraries for common Linux configuration file formats. Each sub-package follows the same pattern: parse a config file from a string, manipulate it through typed accessors, and serialize it back to a string for writing.

## Sub-packages

| Package | Description |
|---------|-------------|
| [bls/](bls/) | Boot Loader Specification entry parser/editor |
| [fstab/](fstab/) | `/etc/fstab` and similar tabular file parser/editor |
| [grub/](grub/) | `/etc/default/grub` parser/editor with kernel argument helpers |
| [keyvalue/](keyvalue/) | Generic `key=value` INI-style config file editor |
| [modprobe/](modprobe/) | `modprobe.d` config file editor for aliases, options, and blacklists |
