# bootconfig -- console/display kernel argument configuration

Adjusts bootloader kernel arguments and X.org display settings so that a converted guest uses virtio-compatible video output and a serial console. This is the post-conversion step that replaces source-hypervisor display arguments (e.g. `vga=`, `video=cirrus`) with their KubeVirt equivalents and removes splash-screen flags like `rhgb` and `quiet`.

`ConfigureConsole` strips silent-boot arguments and adds a `console=<dev>` argument whose device name comes from the distro handler (defaulting to `ttyS0`). `ConfigureDisplay` replaces legacy VGA/cirrus video arguments with `video=virtio`. `ConfigureXorgDriver` rewrites the X.org config file's Device section, switching the driver to `modesetting` and removing vendor/board lines left by VMware tools. All three functions are no-ops when the relevant handler or config file is absent.

## File layout

| File | Purpose |
|------|---------|
| `console.go` | Removes `rhgb`/`quiet` and adds a serial console kernel arg |
| `display.go` | Replaces `vga=`/`video=cirrus` with `video=virtio` |
| `xorg.go` | Rewrites X.org Device sections to use the `modesetting` driver |

## Key exports

| Symbol | Role |
|--------|------|
| `ConfigureConsole` | Adds a serial console kernel arg and removes silent-boot flags |
| `ConfigureDisplay` | Switches video kernel args from legacy VGA/cirrus to virtio |
| `ConfigureXorgDriver` | Patches xorg.conf Device sections to use the modesetting driver |
