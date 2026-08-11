# hypervisor -- WindowsRemove and WindowsServices interfaces

Defines the plugin interfaces for detecting and removing source-hypervisor software from Windows guests during conversion. This covers two complementary cleanup phases: uninstalling hypervisor packages/files and disabling hypervisor-specific Windows services.

`WindowsRemove` implementations detect and remove hypervisor software (e.g. VMware Tools, EC2 drivers) by inspecting the guest filesystem and registry uninstall keys. They receive both the SYSTEM and SOFTWARE hives because uninstall metadata lives under SOFTWARE while some cleanups (e.g. EC2 services) require edits to SYSTEM. `WindowsServices` implementations detect and disable hypervisor services by examining SYSTEM hive service keys under the current control set; they operate even when the hypervisor's install directory has already been removed by an earlier pipeline stage.

## Key exports

| Symbol | Role |
|--------|------|
| `WindowsRemove` | Interface with `Detect` and `Remove` for offline hypervisor software removal |
| `WindowsServices` | Interface with `Detect`, `ServiceNames`, and `DisableServices` for disabling hypervisor services |
| `WindowsRemoves` | Plugin registry of `WindowsRemove` implementations keyed by name |
| `WindowsServiceDisablers` | Plugin registry of `WindowsServices` implementations keyed by name |
| `CurrentControlSet` | Resolves active `ControlSetNNN` from the SYSTEM hive |
| `RemoveFilter` | Removes a name from a class `UpperFilters` / `LowerFilters` REG_MULTI_SZ; deletes the value when the last entry is removed |
| `DisableService` | Sets service `Start=4` when the service key exists |
