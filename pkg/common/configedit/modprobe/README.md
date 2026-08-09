# modprobe -- modprobe.d configuration editor

Used by the convert-linux guest cleanup block and device remap block. Lives in `common/configedit/` because both the cleanup block (adding virtio aliases) and the remap block (rewriting module aliases for renamed devices) edit modprobe config files.

Parses and edits `modprobe.d` configuration files, which control kernel module loading behavior. During conversion, the pipeline uses this to add virtio module aliases, set module options, or blacklist hypervisor-specific drivers.

`Parse` splits the content into lines for round-trip preservation. `Aliases` scans for lines starting with `alias` and returns them as a map from pattern to module name. `AddAlias` replaces an existing alias for the same pattern or appends a new one. `AddOption` appends an `options` directive, and `AddBlacklist` appends a `blacklist` directive. `String` joins the lines back for writing.

## Key exports

| Symbol | Role |
|--------|------|
| `Config` | Represents a parsed modprobe.d configuration file |
| `Parse(content string)` | Parse modprobe config content into a `Config` |
| `(*Config).Aliases()` | Return all alias directives as a `map[string]string` |
| `(*Config).AddAlias(pattern, module)` | Add or replace a module alias |
| `(*Config).AddOption(module, options)` | Append a module options directive |
| `(*Config).AddBlacklist(module)` | Append a blacklist directive |
| `(*Config).String()` | Serialize back to modprobe.d format |
