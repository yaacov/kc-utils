# bootloader plugins

`BootloaderHandler` interface — detect and edit boot configuration. BLS is tried before grub2.

During Linux conversion the bootloader must be identified so that later blocks
can set the default kernel, inject virtio-related kernel arguments (serial
console, virtio video), and rewrite device names. Each plugin probes the
mounted guest filesystem for its configuration format and, if detected, returns
a handler that can read entries, set the default boot kernel, and modify kernel
command-line parameters. Plugins are tried in priority order — BLS first, then
grub2 — and the first match wins.

| Key | Package | Description |
|-----|---------|-------------|
| `bls` | bls/ | Boot Loader Specification entries under `/boot/loader/entries/` |
| `grub2` | grub2/ | Grub2 cfg paths and menuentry parsing |

## bls

**What it does:** Handles distributions that use the Boot Loader Specification
(BLS), where each kernel has a `.conf` drop-in file under
`/boot/loader/entries/`.

**How it works:** `Detect` checks for the presence of the `loader/entries/`
directory under `/boot`. When found, the handler parses each `.conf` file to
enumerate installed kernels and their command-line options. `SetDefaultKernel`
updates the BLS `default` entry (via `machine-id` or saved-entry) to point to
the selected kernel. Kernel arguments are modified by editing the `options`
line in the matching entry file using the `configedit/bls` package.

## grub2

**What it does:** Handles distributions that use traditional GRUB2
configuration files (`grub.cfg` / `grub2.cfg`).

**How it works:** `Detect` searches standard paths (`/boot/grub2/grub.cfg`,
`/boot/grub/grub.cfg`, `/boot/efi/EFI/*/grub.cfg`) for a GRUB configuration
file. The handler parses `menuentry` blocks to enumerate kernels and extract
command-line arguments. `SetDefaultKernel` writes the desired kernel version
into `/etc/default/grub` (`GRUB_DEFAULT`) via the `configedit/grub` package.
Kernel argument changes update `GRUB_CMDLINE_LINUX` in the defaults file.
