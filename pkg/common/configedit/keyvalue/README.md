# keyvalue -- generic key=value config file editor

Used by convert-linux NIC naming plugins (ifcfg, NetworkManager) and hypervisor cleanup plugins. Lives in `common/configedit/` because multiple unrelated blocks edit `/etc/sysconfig/` style key=value files — sharing one parser avoids duplicating quote-handling and round-trip logic.

Parses and edits simple `key=value` configuration files such as those found in `/etc/sysconfig/`. This is a general-purpose editor for INI-style files that use `=` as the separator without section headers.

`Parse` splits the content into lines and preserves them for round-trip serialization. `Get` performs a linear scan for a line starting with `key=`, strips surrounding quotes from the value, and returns it. `Set` updates the first matching line in place or appends a new quoted entry. `Delete` removes all lines matching the key. `String` joins the lines back with newlines.

## Key exports

| Symbol | Role |
|--------|------|
| `File` | Represents a parsed key=value configuration file |
| `Parse(content string)` | Parse content into a `File` |
| `(*File).Get(key)` | Return the unquoted value for a key |
| `(*File).Set(key, value)` | Set or add a key-value pair (value is quoted) |
| `(*File).Delete(key)` | Remove all lines matching the key |
| `(*File).String()` | Serialize back to the original format |
