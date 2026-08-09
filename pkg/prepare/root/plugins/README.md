# root plugins

`RootSelector` interface — apply `PrepareInput.options.root` policy after discovery.

Before the prepare pipeline can mount and inspect the guest, it must determine
which partition is the OS root — especially on multi-boot systems or disks with
multiple OS installations. The root discovery step scans all partitions for
bootable OS markers (e.g. `/etc/os-release` for Linux, `Windows/System32` for
Windows). Root selection then applies a policy to choose among the discovered
candidates. The policy is set via `PrepareInput.options.root`: omitting it uses
the `first` selector (default), setting it to `single` requires exactly one
candidate, and providing a device path (e.g. `/dev/sda2`) uses the `device`
selector.

| Key | Package | Description |
|-----|---------|-------------|
| `first` | first/ | Pick the first discovered root (**default** when `options.root` is omitted) |
| `single` | single/ | Fail if multiple OS roots found (explicit only) |
| `device` | device/ | Pick root on a given block device path |

## first

**What it does:** Unconditionally selects the first discovered OS root
candidate, regardless of how many exist.

**How it works:** Returns `candidates[0]` directly. Errors only if the
candidate list is empty (no bootable OS root found on any partition). This is
the default policy when no `options.root` preference is specified.

## single

**What it does:** Requires that exactly one OS root candidate exists and fails
on multi-boot systems.

**How it works:** Returns `candidates[0]` only when the candidate list has
exactly one entry. If zero candidates are found, returns an error indicating
no OS root was detected. If more than one candidate is found, returns a
`MultiBootError` that lists all candidates with their device paths and product
names, instructing the user to use `first` or specify a device path instead.

## device

**What it does:** Selects the root candidate on a specific block device path
chosen by the user.

**How it works:** Iterates the candidate list comparing each `DevicePath`
against the user-supplied `choice` string (e.g. `/dev/sda2`). Returns the
matching candidate. If no candidate matches, returns an error listing all
available candidates with their device paths and product names so the user
can correct their selection.
