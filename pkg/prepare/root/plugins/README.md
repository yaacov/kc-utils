# root plugins

`RootSelector` interface — apply `PrepareInput.options.root` policy after discovery.

| Key | Package | Description |
|-----|---------|-------------|
| `first` | first/ | Pick the first discovered root (**default** when `options.root` is omitted) |
| `single` | single/ | Fail if multiple OS roots found (explicit only) |
| `device` | device/ | Pick root on a given block device path |
