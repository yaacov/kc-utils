# firstboot -- Contributor interface and firstboot.bat generation

Generates the Windows firstboot script infrastructure that runs once after conversion, then self-deletes. The firstboot system installs VirtIO drivers via PnP, configures static IPs, onlines disks, removes hypervisor remnants, and performs other post-conversion fixups inside the running Windows guest.

The `Configure` function collects all registered `Contributor` plugins, filters them by `ShouldRun`, sorts by priority, and writes each contributor's generated script (PowerShell or batch) into a numbered scripts directory under `C:\Program Files\Guestfs\Firstboot\scripts\`. It then writes a `firstboot.bat` launcher selected by the Windows version handler (modern PowerShell, PSv1-compatible, or batch-only for pre-Vista). Finally it sets a RunOnce registry entry in the SOFTWARE hive so Windows executes `firstboot.bat` on next login. The launcher iterates scripts in filename order and cleans up the Guestfs directory when done.

## File layout

| File | Purpose |
|------|---------|
| `firstboot.go` | `Config` struct, `Configure` orchestrator, `WriteScript`/`WriteBatScript` helpers, and launcher templates |
| `contributor.go` | `Contributor` interface, `ContributorConfig` struct, and the global `Contributors` plugin registry |

## Key exports

| Symbol | Role |
|--------|------|
| `Config` | Parameters for firstboot generation: mount root, offline flag, driver files, static IPs, options, and version handler |
| `Configure` | Orchestrates contributor execution, writes numbered scripts, generates the launcher, and sets the RunOnce registry key |
| `WriteScript` | Writes a PowerShell firstboot script with a priority-based `NNNN-name.ps1` filename |
| `WriteBatScript` | Writes a batch firstboot script with a priority-based `NNNN-name.bat` filename |
| `ContributorConfig` | Context passed to each contributor: mount root, offline mode, driver files, static IPs, options, version |
| `Contributor` | Interface for firstboot script plugins with `Priority`, `ShouldRun`, `Generate`, `Name`, and `UsesBatch` methods |
| `Contributors` | Global plugin registry of `Contributor` implementations |
| `DefaultUsesBatch` | Helper that returns false, suitable for PowerShell-based contributors |
