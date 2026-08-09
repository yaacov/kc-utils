# pkg/common — shared libraries

Cross-utility helper packages used by all pipeline stages. Each package has its
own README with detailed exports and usage; this file is an index.

| Package | Used by | Description |
|---------|---------|-------------|
| [`types/`](types/) | All binaries | JSON handoff structs between pipeline stages |
| [`logger/`](logger/) | All cmd entry points | Structured logging setup |
| [`plugin/`](plugin/) | Pluggable blocks | Generic `Registry[K,V]` with `sync.RWMutex` |
| [`configedit/`](configedit/) | Converters | Pure-Go editors for guest config files (fstab, grub, bls, modprobe, keyvalue) |
| [`compression/`](compression/) | Initramfs | gzip/xz/zstd/lz4 stream detection and (de)compression |
| [`registry/`](registry/) | Windows converter | Windows registry hive read/write |
| [`uefi/`](uefi/) | Converters | Shared UEFI editor interface and `ConvertAllESPs` helper |
| [`firstboot/`](firstboot/) | Converters, finalize | Shared firstboot handler interface and registry |

Import path prefix: `github.com/yaacov/kc-utils/pkg/common/…`
