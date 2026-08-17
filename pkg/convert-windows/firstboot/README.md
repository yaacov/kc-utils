# firstboot -- Contributor interface and firstboot.bat generation

Generates the Windows firstboot script infrastructure that runs once after conversion, then self-deletes. The firstboot system installs VirtIO drivers via PnP, configures static IPs, onlines disks, removes hypervisor remnants, and performs other post-conversion fixups inside the running Windows guest.

The `Configure` function collects all registered `Contributor` plugins, filters them by `ShouldRun`, sorts by priority, and writes each contributor's generated script (PowerShell or batch) into a numbered scripts directory under `C:\Program Files\Guestfs\Firstboot\scripts\`. It then writes a `firstboot.bat` launcher selected by the Windows version handler (modern PowerShell, PSv1-compatible, or batch-only for pre-Vista). Finally it registers a Windows service (`kcfirstboot`) in the SYSTEM hive via `rhsrvany.exe` so the launcher runs at boot as SYSTEM — no interactive user login is required. The launcher iterates scripts in filename order (including the optional COM1 `CONVERSION_DONE` signal at priority 99999), uninstalls the service, schedules a guest reboot (`C:\Windows\System32\shutdown.exe /r /t 5 /f`), then cleans up the Guestfs directory. Reboot is scheduled before deleting Firstboot because cmd.exe stops running a `.bat` once its own file is removed. That order matches Forklift's wait-for-reboot watcher: signal first, then reboot.

## Boot-time execution (rhsrvany.exe)

Previous versions used a `RunOnce` registry entry which only fires after a user interactively logs in. The current implementation uses `rhsrvany.exe` (a lightweight Windows service wrapper from the [rhsrvany](https://github.com/rwmjones/rhsrvany) project, matching the upstream virt-v2v approach). The service is registered offline in the SYSTEM hive under `{CurrentControlSet}\Services\kcfirstboot` with `Start=2` (auto-start) and `ObjectName=LocalSystem`, ensuring firstboot.bat runs at machine startup regardless of whether anyone logs in.

The `KC_VIRT_TOOLS` environment variable overrides the host directory containing `rhsrvany.exe` (default: `/usr/share/virt-tools`).

## File layout

| File | Purpose |
|------|---------|
| `firstboot.go` | `Config` struct, `Configure` orchestrator, service registration, `WriteScript`/`WriteBatScript` helpers, and launcher templates |
| `contributor.go` | `Contributor` interface, `ContributorConfig` struct, and the global `Contributors` plugin registry |

## Key exports

| Symbol | Role |
|--------|------|
| `Config` | Parameters for firstboot generation: mount root, offline flag, driver files, static IPs, options, and version handler |
| `Configure` | Orchestrates contributor execution, writes numbered scripts, generates the launcher, copies rhsrvany.exe into the guest, and registers the firstboot Windows service |
| `WriteScript` | Writes a PowerShell firstboot script with a priority-based `NNNN-name.ps1` filename |
| `WriteBatScript` | Writes a batch firstboot script with a priority-based `NNNN-name.bat` filename |
| `ContributorConfig` | Context passed to each contributor: mount root, offline mode, driver files, static IPs, options, version |
| `Contributor` | Interface for firstboot script plugins with `Priority`, `ShouldRun`, `Generate`, `Name`, and `UsesBatch` methods |
| `Contributors` | Global plugin registry of `Contributor` implementations |
| `DefaultUsesBatch` | Helper that returns false, suitable for PowerShell-based contributors |
