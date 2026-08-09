# crashcontrol -- disable crash auto-reboot

Prevents Windows from automatically rebooting after a blue-screen crash (BSOD) during or after conversion. This is important because an unexpected reboot during firstboot driver installation could leave the guest in an unrecoverable state.

The package writes a single registry DWORD (`AutoReboot = 0`) under the `CrashControl` key in the SYSTEM hive's current control set. If the key does not already exist it is created first.

## Key exports

| Symbol | Role |
|--------|------|
| `Disable` | Sets `AutoReboot` to 0 in `<CCS>\Control\CrashControl` to suppress automatic reboot on BSOD |
