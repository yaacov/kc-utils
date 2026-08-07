# uefi plugins

`UEFIEditor` interface — read and modify UEFI boot configuration.
`uefi.ConvertAllESPs` runs every registered editor against each ESP.

| Key | Package | Editor |
|-----|---------|--------|
| `bcd` | bcdeditor/ | Windows BCD store editor via hivexregedit |
| `grub-fallback` | ../../convert-linux/uefi/plugins/grubfallback/ | Linux shim/grub fallback bootloader on ESP |
