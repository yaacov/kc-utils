# Guest Filesystem Checks and Repair

kc-utils runs filesystem check/repair on guest block devices through
`Guest.FSCheck()` in [`pkg/guest/`](../../pkg/guest/). All fsck, fstrim, and
mount operations on guest disks must go through that package — orchestrators and
pipeline blocks never invoke `e2fsck`, `xfs_repair`, or `guestfish` fsck
commands directly. See [community/architecture.md](../../community/architecture.md).

This document is the single reference for **when** checks run, **which**
filesystem types are supported, **check vs repair** behavior per backend, and
how failures are handled.

## Pipeline timeline

Filesystem checks run twice during a normal conversion — once before any guest
filesystem is mounted, and once after conversion edits are flushed and all
mounts are torn down.

| Pass | Binary | Order in pipeline | Device state |
|------|--------|-------------------|--------------|
| Pre-fsck | `kc-prepare` | After LUKS decrypt, **before** root discovery and mount | Unmounted block devices |
| Post-fsck | `kc-finalize` | After `fstrim` and unmount; **before** LUKS close / LVM deactivate | Unmounted; LUKS mappers still open |
| *(not fsck)* | `kc-finalize --teardown-only` | Failure cleanup from `kc-v2v` | Teardown only — **no fsck** |
| *(not FSCheck)* | `kc-convert-windows` block 13 | During convert, while guest is mounted | NTFS **heads** boot-sector patch — see [Windows NTFS operations](#windows-ntfs-operations) |

```text
kc-prepare
  … decrypt LUKS …
  → FSCheck (pre-fsck, unmounted)
  → discover root, mount filesystems
  …

kc-convert-{linux,windows}
  … conversion while mounted …

kc-finalize
  → customize (mounted)
  → fstrim (mounted)
  → UnmountFilesystems (unmount; LUKS/LVM stay open)
  → FSCheck (post-fsck, unmounted)
  → ReleaseDevices (cryptsetup close, LVM off, loops)
  → target metadata
```

Code: pre-fsck in [`pkg/cmd/prepare/pipeline.go`](../../pkg/cmd/prepare/pipeline.go);
post-fsck in [`pkg/cmd/finalize/pipeline.go`](../../pkg/cmd/finalize/pipeline.go);
`--teardown-only` path in [`pkg/cmd/v2v/pipeline.go`](../../pkg/cmd/v2v/pipeline.go).

`kc-v2v` does not call `FSCheck` itself — it runs `kc-prepare` and
`kc-finalize` as subprocesses, which perform the checks above.

In guestfs mode (kc-v2v container default), finalize unmounts inside the
libguestfs appliance before post-fsck; LUKS mappers stay open until fsck
completes. See [../backends/guestfs.md](../backends/guestfs.md).

## LUKS and encryption

kc-utils **does not re-encrypt** guest disks. Prepare opens LUKS volumes
(`cryptsetup open` / Clevis unlock); finalize closes them (`cryptsetup close`).
The on-disk LUKS payload stays encrypted throughout conversion — closing the
mapper only drops the decrypted `/dev/mapper/...` view.

Post-fsck must run on **unmounted** filesystems while **mapper devices still
exist**. Finalize therefore splits teardown:

1. `Guest.UnmountFilesystems()` — unmount all guest filesystems
2. `Guest.FSCheck()` — fsck on block devices (including `/dev/mapper/...` paths)
3. `Guest.ReleaseDevices()` — close LUKS, deactivate LVM, detach loops

Pre-fsck in prepare runs after LUKS **open** and before mount, with the same
mapper-visible device paths.

`Guest.Teardown()` is `UnmountFilesystems()` followed by `ReleaseDevices()` —
used by deferred cleanup and `--teardown-only` paths (which skip fsck).

## Supported filesystems by backend

Behavior depends on the guest access backend. Implementation:

- Direct (host-mount): [`pkg/guest/direct/backend.go`](../../pkg/guest/direct/backend.go) `FSCheck`
- Guestfs (libguestfs appliance): [`pkg/guest/guestfs/backend.go`](../../pkg/guest/guestfs/backend.go) `fscheckCommand`
- QEMU appliance: [`pkg/guest/qemu/`](../../pkg/guest/qemu/) (tools run inside `kc-agent`)

| FS type | Guestfs (`V2V_backend=guestfs`, kc-v2v image) | Direct / QEMU appliance |
|---------|----------------------------------------------|-------------------------|
| ext2, ext3, ext4 | guestfish `e2fsck-f` | `e2fsck -f -y` |
| xfs | guestfish `xfs-repair` | `xfs_repair` |
| ntfs, ntfs3 | guestfish `ntfsfix` | `ntfsfix -d` (**ntfs3 only**; plain `ntfs` skipped) |
| btrfs | skipped | `btrfs check` (no `--repair`) |
| vfat, swap, unknown | skipped | skipped |

Both backends iterate **every partition** on every disk from the prepare
output. Partitions with an unsupported or empty `FSType` are skipped (`FSCheck`
returns without error).

## Check vs repair semantics

Checks are **not read-only globally**. Aggressiveness varies by filesystem type
and backend:

| FS | Behavior |
|----|----------|
| **ext2/3/4** | Direct mode auto-fixes with `e2fsck -f -y`. Guestfs runs `e2fsck-f` (force check via libguestfs); it is less aggressive than direct `-y` auto-repair. |
| **xfs** | Repair-oriented in both backends (`xfs_repair` / guestfish `xfs-repair`). |
| **ntfs / ntfs3** | `ntfsfix` clears the NTFS dirty flag. Direct mode passes `-d`. Distinct from the Windows [NTFS heads fix](#windows-ntfs-operations) during convert. |
| **btrfs** | Direct mode runs `btrfs check` without `--repair` (check only, no repair). Guestfs mode skips btrfs entirely. |
| **vfat / EFI** | Skipped — ESP and other vfat partitions are not fsck'd. |

## Failure policy

`FSCheck` errors are **non-fatal**:

- **kc-prepare:** logs `fscheck failed` at warn level; pipeline continues.
- **kc-finalize:** logs a warning and appends a string to `TargetMeta.warnings`.

A failed fsck does not abort conversion. Inspect pod logs and the warnings
HTTP endpoint (`GET /warnings` when `LOCAL_MIGRATION=true`) for details.

## Windows NTFS operations

Two separate NTFS mechanisms exist — do not conflate them:

| Mechanism | When | What it does |
|-----------|------|--------------|
| **`Guest.FSCheck()` / `ntfsfix`** | Pre-fsck (prepare) and post-fsck (finalize), on unmounted devices | Filesystem consistency / dirty-flag cleanup via `ntfsfix` |
| **NTFS heads fix** | `kc-convert-windows` block 13, during convert on mounted guest | Patches `$NumberOfHeads` in the NTFS boot sector for pre-Vista Windows ([`pkg/convert-windows/ntfsfix/ntfsfix.go`](../../pkg/convert-windows/ntfsfix/ntfsfix.go)) |

The heads fix is gated by `NeedsNTFSHeadsFix()` on the Windows version handler
and writes raw bytes to the block device — it is not an fsck pass.

## Related documentation

- Stage behavior: [kc-prepare.md](../apps/kc-prepare.md), [kc-finalize.md](../apps/kc-finalize.md)
- Orchestrator: [kc-v2v.md](../apps/kc-v2v.md)
- Backend trade-offs: [../backends/README.md](../backends/README.md)
- Windows convert pipeline: [kc-convert-windows.md](../apps/kc-convert-windows.md)
- Container / guestfs default: [build/kc-v2v/README.md](../../build/kc-v2v/README.md)
