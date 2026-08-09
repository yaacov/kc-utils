# bls -- Boot Loader Specification entry editor

Used by the convert-linux bootloader BLS plugin and remap block. Lives in `common/configedit/` because BLS entries are edited by both the bootloader detection block (to identify the boot config format) and the device remap block (to rewrite device paths in boot entries).

Parses and edits Boot Loader Specification (BLS) entry files, typically found under `/boot/loader/entries/`. During conversion, the pipeline may need to update kernel paths, initrd references, or boot arguments in BLS entries.

`Parse` splits the file content into key-value fields, skipping blank lines and comments. Each non-comment line is split on the first space into a key and value. `Get` performs a linear scan for the first matching key. `Set` updates an existing key in place or appends a new field. `String` serializes the entry back to BLS format with one `key value` pair per line.

## Key exports

| Symbol | Role |
|--------|------|
| `Entry` | Represents a parsed BLS entry file |
| `Parse(content string)` | Parse BLS file content into an `Entry` |
| `(*Entry).Get(key)` | Return the value for a key |
| `(*Entry).Set(key, value)` | Set or add a key-value pair |
| `(*Entry).String()` | Serialize back to BLS format |
