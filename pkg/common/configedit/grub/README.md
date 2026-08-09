# grub -- /etc/default/grub parser and editor

Used by the convert-linux bootloader grub2 plugin and the boot config block. Lives in `common/configedit/` because both the grub2 bootloader plugin (detection and regeneration) and the bootconfig block (serial console and virtio video args) modify the same grub defaults file.

Parses and edits `/etc/default/grub`, the GRUB2 defaults configuration file. During conversion, the pipeline uses this to modify kernel command-line arguments -- for example, removing hypervisor-specific console settings or adding virtio-related parameters.

`Parse` splits the file into lines and extracts `KEY=VALUE` pairs (stripping surrounding quotes) into a lookup map while preserving the original line ordering for round-trip fidelity. `Set` updates an existing line in place or appends a new one, always quoting the value. `GetKernelArgs` and `AddKernelArg`/`RemoveKernelArg` provide specialized access to the `GRUB_CMDLINE_LINUX` value, splitting it into individual arguments and matching by key prefix (for `key=value` style args) or exact string.

## Key exports

| Symbol | Role |
|--------|------|
| `Config` | Represents a parsed `/etc/default/grub` file |
| `Parse(content string)` | Parse grub defaults content into a `Config` |
| `(*Config).Get(key)` | Return the value for a configuration key |
| `(*Config).Set(key, value)` | Set or add a configuration key-value pair |
| `(*Config).GetKernelArgs()` | Return `GRUB_CMDLINE_LINUX` split into individual arguments |
| `(*Config).AddKernelArg(arg)` | Add an argument to `GRUB_CMDLINE_LINUX` if not already present |
| `(*Config).RemoveKernelArg(prefix)` | Remove arguments matching a prefix from `GRUB_CMDLINE_LINUX` |
| `(*Config).String()` | Serialize back to `/etc/default/grub` format |
