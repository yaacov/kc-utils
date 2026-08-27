# Finalize converted disks

Unmount guest filesystems, trim, post-fsck, and write TargetMeta. CLI
contract: [`kc-finalize.md`](../apps/kc-finalize.md).

Keep the held qemu process until the debug checks below succeed, then stop it.

## Run

```sh
# from the repo root (make build → bin/)
export PATH="$PWD/bin:$PATH"
export GOVC_VM=yzamir-d-5g-linux
export WORKDIR=/tmp/kc-debug
export IMGDIR=${IMGDIR:-$WORKDIR/$GOVC_VM}
# KC_QEMU_* still set from start-appliance.md

# Stale hold: if qemu is gone, drop the env so finalize remounts instead of
# failing adopt, and so we do not later kill a reused PID.
if [ -n "${KC_QEMU_PID-}" ] && ! kill -0 "$KC_QEMU_PID" 2>/dev/null; then
  unset KC_QEMU_PID KC_QEMU_SOCK KC_QEMU_DEBUG_SOCK
fi

kc-finalize \
  --backend qemu \
  --input "$IMGDIR/pipeline.json" \
  --output "$IMGDIR/pipeline.json" \
  --mount-root /tmp/kc-guest \
  --log-level info
```

Same adopt path as prepare/convert. If the held qemu has already exited, the
block above unsets `KC_QEMU_*` so finalize boots a fresh appliance and remounts
from `pipeline.json`, then tears down. Leave `KC_QEMU_PID` unset in that case.

## Verify with the debug socket

Attach ([README.md](README.md#attach-the-debug-shell)) after finalize
unmounts (or watch during the run):

```sh
findmnt | grep /mnt/guest || echo "guest mounts gone (ok)"
ls /mnt/guest 2>/dev/null || echo "mount root empty or absent (ok)"
sync
```

Success: `/mnt/guest` is not holding the converted root. `sync` flushes
virtio-blk writes to the raw images before you kill qemu.

On the host:

```sh
jq '{firmware: .target.target_firmware, guestcaps: .target.guestcaps, warnings: .target.warnings}' \
  "$IMGDIR/pipeline.json"
```

`target_firmware` is `bios` or `uefi` — you need that for
[boot-guest-qemu-x86.md](boot-guest-qemu-x86.md).

## Stop the appliance

```sh
if [ -n "${KC_QEMU_PID-}" ] && kill -0 "$KC_QEMU_PID" 2>/dev/null; then
  comm=$(ps -o comm= -p "$KC_QEMU_PID" 2>/dev/null || true)
  comm=${comm##*/}
  case "$comm" in
    qemu-system-*)
      kill "$KC_QEMU_PID"
      # wait; SIGTERM is enough. SIGKILL only if it hangs.
      ;;
    *)
      echo "KC_QEMU_PID=$KC_QEMU_PID is not a qemu-system process; skipping kill" >&2
      ;;
  esac
fi
unset KC_QEMU_SOCK KC_QEMU_PID KC_QEMU_DEBUG_SOCK
```

The converted disks are still `$IMGDIR/diskN.img` (raw, in place). There is
no overlay commit in this cookbook.
