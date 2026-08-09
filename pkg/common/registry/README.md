# registry -- Windows registry hive editing

Shared across all convert-windows blocks (drivers, hypervisor cleanup, crash control, firstboot, inspect, static IP) and the UEFI BCD editor. Lives in `common/` because nearly every Windows conversion block reads or writes registry hives, and the prepare stage also reads hives during OS inspection — centralizing the `Editor`/`Hive` interface avoids duplicating the hivex wrapper and mock infrastructure.

Defines interfaces for reading and writing Windows registry hive files offline. During Windows VM conversion, the pipeline opens hive files (SYSTEM, SOFTWARE) from the mounted guest disk and modifies registry keys to install virtio drivers, adjust boot configuration, and reconfigure network settings.

The `Editor` interface provides `OpenHive`, which opens a registry hive file and returns a `Hive` handle. `Hive` exposes full read/write operations: key enumeration and existence checks, typed getters (`GetString`, `GetDWORD`, `GetMultiString`, `GetValue`), value enumeration, key/value creation and deletion, typed setters, and `Save`/`Close` for persistence. Implementations register themselves in the global `Editors` registry. Registry value type constants (`REG_SZ`, `REG_DWORD`, etc.) are provided for interpreting raw value data.

## Key exports

| Symbol | Role |
|--------|------|
| `Editor` | Interface with `OpenHive(hivePath string) (Hive, error)` |
| `Hive` | Interface for read/write access to an opened registry hive |
| `ValueEntry` | Struct representing a registry value (Name, Type, Data) |
| `REG_SZ` | Constant for string registry value type (1) |
| `REG_EXPAND_SZ` | Constant for expandable string type (2) |
| `REG_BINARY` | Constant for binary data type (3) |
| `REG_DWORD` | Constant for 32-bit integer type (4) |
| `REG_MULTI_SZ` | Constant for multi-string type (7) |
| `Editors` | Global `plugin.Registry[string, Editor]` for registered implementations |
