# uefi -- UEFI ESP conversion to virtio

Shared across convert-linux and convert-windows. Lives in `common/` because both Linux and Windows UEFI guests need ESP boot entries rewritten for virtio — the BCD editor (Windows) and GRUB fallback editor (Linux) both implement the same `UEFIEditor` interface so the pipeline calls `ConvertAllESPs` identically regardless of OS.

Defines the `UEFIEditor` interface and provides `ConvertAllESPs`, which iterates over all registered UEFI editors and ESP devices to convert EFI System Partitions for virtio boot. This is used during VM conversion to ensure UEFI guests can boot under KVM/virtio.

Implementations register themselves in the `Editors` registry (a `plugin.Registry[string, UEFIEditor]`) during `init()`. When the pipeline calls `ConvertAllESPs`, it iterates over every registered editor and every ESP device path, calling `ConvertToVirtio` on each combination. Failures are collected as `BlockError` values rather than aborting, so the pipeline can report partial success. Progress and errors are logged via `slog`.

## Key exports

| Symbol | Role |
|--------|------|
| `UEFIEditor` | Interface with `ConvertToVirtio(guestRoot, espPath string) error` |
| `Editors` | Global `plugin.Registry[string, UEFIEditor]` for registered implementations |
| `ConvertAllESPs(mountRoot, espDevices)` | Run all registered editors on all ESP devices; return collected `BlockError` values |
