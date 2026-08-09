# firstboot -- first-boot script installation

Shared across convert-linux (guest agent install), convert-windows (driver/network firstboot), and finalize (dynamic customization scripts). Lives in `common/` because both Linux and Windows converters install first-boot commands through the same interface — the only difference is the plugin (systemd unit vs batch script).

Defines the `FirstBootHandler` interface and a global registry for implementations that install first-boot scripts into a guest filesystem. After conversion, the pipeline may need the guest to run commands on its first reboot (e.g., regenerating initramfs or relabeling SELinux); this package provides the abstraction for that.

Implementations register themselves in the `Handlers` registry (a `plugin.Registry[string, FirstBootHandler]`) during `init()`. The pipeline looks up a handler by name, then calls `Install` with the guest root mount path and a list of shell commands. The handler is responsible for writing the commands into the guest's first-boot mechanism (e.g., a systemd unit or rc.local script).

## Key exports

| Symbol | Role |
|--------|------|
| `FirstBootHandler` | Interface with `Install(guestRoot string, commands []string) error` |
| `Handlers` | Global `plugin.Registry[string, FirstBootHandler]` for registered implementations |
