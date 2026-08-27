# Prepare imported disks for conversion

Inspect the guest, run pre-fsck, and mount filesystems **inside** the held
appliance. CLI contract: [`kc-prepare.md`](../apps/kc-prepare.md).

`--mount-root` is a **path key**. The qemu backend rebases it to `/mnt/guest`
in the appliance. It is not a host FUSE tree; inspect files via the debug
shell.

## Input

Use `disk_dir` so every `diskN.img` from [fetch-vmware-disks.md](fetch-vmware-disks.md)
is picked up (same shape as
[prepare-input-disk-dir.json](../apps/examples/prepare-input-disk-dir.json)):

```sh
export GOVC_VM=yzamir-d-5g-linux
export WORKDIR=/tmp/kc-debug
export IMGDIR=${IMGDIR:-$WORKDIR/$GOVC_VM}
# KC_QEMU_SOCK / KC_QEMU_PID / KC_QEMU_DEBUG_SOCK already set
# (see start-appliance.md)

cat > "$IMGDIR/prepare-input.json" <<EOF
{
  "disk_dir": "$IMGDIR",
  "source": {
    "name": "$GOVC_VM",
    "type": "vmware",
    "firmware_hint": "bios"
  },
  "options": {
    "tmp_dir": "$IMGDIR/tmp"
  }
}
EOF
mkdir -p "$IMGDIR/tmp"
```

Set `firmware_hint` to `bios` or `uefi` if you know it; prepare also detects
firmware from the disk. Explicit `disks` list instead of `disk_dir`:

```json
"disks": [{ "path": "/absolute/path/disk0.img", "format": "raw" }]
```

## Run (adopts the held appliance)

```sh
# from the repo root (make build → bin/)
export PATH="$PWD/bin:$PATH"

kc-prepare \
  --backend qemu \
  --input "$IMGDIR/prepare-input.json" \
  --output "$IMGDIR/pipeline.json" \
  --mount-root /tmp/kc-guest \
  --log-level info
```

With `KC_QEMU_SOCK` set, prepare connects to the existing agent instead of
booting qemu. Logs should mention adopting the shared appliance, not a fresh
launch. Leave qemu running.

## Verify with the debug socket

Attach ([README.md](README.md#attach-the-debug-shell)) after prepare
completes (or while it is still running, once mounts exist):

```sh
findmnt
ls /mnt/guest
# Linux:
cat /mnt/guest/etc/os-release
# Windows:
ls /mnt/guest/Windows
```

Success: guest root is under `/mnt/guest` (`os-release` or `Windows`). Nested
mounts (`/boot`, `/boot/efi`) appear under `/mnt/guest/boot` as well.

On the host:

```sh
jq '{converter: .prepare.converter, inspect: .prepare.inspect, firmware: .prepare.firmware, root_device: .prepare.root_device}' \
  "$IMGDIR/pipeline.json"
```

`.prepare.converter` is `kc-convert-linux` or `kc-convert-windows`. Continue
with [convert.md](convert.md) against the **same** qemu process.
