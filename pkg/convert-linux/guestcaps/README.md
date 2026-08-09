# guestcaps -- guest capabilities builder

Populates a `GuestCaps` struct with the hardware capabilities the converted VM should advertise to its management layer. This determines the bus types, device features, and machine type that downstream tooling (e.g. KubeVirt) uses when defining the target VM.

`Build` starts by assuming full virtio support (virtio block/net buses, RNG, balloon, vsock, PV panic, virtio 1.0, UTC RTC). If a selected kernel is provided and positively lacks virtio support, it downgrades to legacy buses (IDE block, e1000 network) and disables virtio device features. The machine type is set from the guest architecture: `q35` for x86_64, `virt` for aarch64, `pseries` for ppc64le, and `s390-ccw-virtio` for s390x, defaulting to `q35` for unrecognized architectures.

## Key exports

| Symbol | Role |
|--------|------|
| `Build` | Fills a `GuestCaps` struct with bus types, device features, and machine type based on architecture and kernel virtio support |
