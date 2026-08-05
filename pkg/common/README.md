# pkg/common — shared libraries

Cross-utility helper packages (no plugin registries).

| Package | Used by | Description |
|---------|---------|-------------|
| [`types/`](types/) | All binaries | JSON handoff structs between pipeline stages |
| [`logger/`](logger/) | All cmd entry points | Structured logging setup |
| [`plugin/`](plugin/) | Pluggable blocks | Generic `Registry[K,V]` with `sync.RWMutex` |
| [`configedit/`](configedit/) | Converters | Augeas replacement for guest config files |
| [`compression/`](compression/) | Initramfs | gzip/xz/zstd/lz4 stream detection and (de)compression |
| [`registry/`](registry/) | Windows converter | Windows registry hive read/write |
| [`uefi/`](uefi/) | Converters | Shared UEFI editor interface and `ConvertAllESPs` helper |
| [`firstboot/`](firstboot/) | Converters, finalize | Shared firstboot handler interface and registry |

Import path prefix: `github.com/yaacov/kc-utils/pkg/common/…`

## types

Shared data structures flowing between the four kc-utils binaries:

- `PrepareInput` — disks, source hypervisor, LUKS config, `options.root`
- `PrepareOutput` — converter name, inspect, firmware, disks, root/boot device
- `RootCandidate` — discovered OS root (multiboot errors)
- `GuestCaps` — block/net bus, virtio flags, machine type
- `ConverterOutput` — `GuestCaps` + warnings
- `TargetMeta` — finalize output: buses, NICs, firmware, disk mappings

## logger

Global [`slog`](https://pkg.go.dev/log/slog) setup for all binaries:

| Function | Role |
|---|---|
| `Init(level)` | Configure text handler on stderr (`debug`, `info`, `warn`, `error`) |

Called from each `cmd/*/main.go` and from `kc-v2v` after config load.

## plugin

Generic thread-safe plugin registry used by all pluggable pipeline blocks:

| Type | Role |
|---|---|
| `Registry[K,V]` | `Register`, `Get`, `All` with `sync.RWMutex` |

Implementations self-register in `init()` under `pkg/<utility>/<block>/plugins/`.

## configedit

Pure-Go editors replacing Augeas for guest config files:

| Sub-package | Edits |
|-------------|-------|
| [`fstab/`](configedit/fstab/) | `/etc/fstab` parse/edit and device remapping |
| [`grub/`](configedit/grub/) | `grub.cfg` / `grub` default without re-running `grub2-mkconfig` |
| [`bls/`](configedit/bls/) | Boot Loader Specification entries |
| [`modprobe/`](configedit/modprobe/) | modprobe.d aliases and options |
| [`keyvalue/`](configedit/keyvalue/) | INI-style key=value files via `go-ini/ini` |

## registry

Windows registry hive read/write without CGo:

- **Read:** pure Go via `Velocidex/regparser`
- **Write:** batched `.reg` files applied with `hivexregedit --merge`

| Sub-package | Role |
|---|---|
| [`registry/hivex/`](registry/hivex/) | `hivexregedit --merge` wrapper for batched writes |
| [`registry/mock/`](registry/mock/) | Test double for registry operations |

## fschecker

Shared fsck wrappers used by prepare and finalize fschecker plugins:
`e2fsck`, `xfs_repair`, `btrfs check`, `ntfsfix`.

## compression

Magic-byte detection for gzip/xz/zstd/lz4 streams (pure Go).
