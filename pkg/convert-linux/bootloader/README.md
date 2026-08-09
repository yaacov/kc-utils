# bootloader -- bootloader detection and management interface

Defines the `BootloaderHandler` interface for detecting and managing bootloader configuration inside a mounted guest filesystem. Concrete implementations (e.g. BLS, GRUB2) register themselves into a global plugin registry and are probed at conversion time.

`DetectFirst` iterates a preferred detection order (BLS before GRUB2), then falls back to any remaining registered handler, returning the first one whose `Detect` method matches the guest. The interface methods cover the operations needed during conversion: reading/setting the default kernel, adding/removing kernel arguments, and regenerating the bootloader config.

## Key exports

| Symbol | Role |
|--------|------|
| `BootloaderHandler` | Interface for detecting and managing a bootloader (Detect, Get/SetDefaultKernel, Add/RemoveKernelArg, RegenerateConfig) |
| `Handlers` | Global plugin registry of `BootloaderHandler` implementations |
| `PreferredOrder` | Detection order slice: `["bls", "grub2"]` |
| `DetectFirst` | Returns the first matching bootloader handler for a guest root |
