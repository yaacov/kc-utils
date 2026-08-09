# customize -- Customizer plugin interface

Defines the extension point for guest-OS customization during the finalize phase. Customizers apply post-conversion tweaks (hostname, timezone, custom scripts, SELinux relabeling) to a mounted guest filesystem.

Implementations register themselves with the package-level `Customizers` registry. At finalize time the runner iterates registered customizers and calls `Apply` on each, passing the guest root mount path and an options map built from pipeline metadata.

## Key exports

| Symbol | Role |
|--------|------|
| `Customizer` | Interface with `Apply(guestRoot string, options map[string]string) error` |
| `Customizers` | Global `plugin.Registry[string, Customizer]` for registering/looking up customizer implementations |
