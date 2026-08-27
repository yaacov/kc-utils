# Convert on an already prepared appliance

Run the converter selected by prepare **without** stopping qemu. Mounts from
prepare are still live under `/mnt/guest`. CLI contracts:
[`kc-convert-linux.md`](../apps/kc-convert-linux.md),
[`kc-convert-windows.md`](../apps/kc-convert-windows.md).

## Stage local packages

Defaults live under `/usr/share/` (kc-v2v image). On a laptop, stage into
`$IMGDIR` and pass that path to the converter.

Linux qemu-guest-agent RPMs (this demo VM):

```sh
export GOVC_VM=yzamir-d-5g-linux
export WORKDIR=/tmp/kc-debug
export IMGDIR=${IMGDIR:-$WORKDIR/$GOVC_VM}

# layout: $IMGDIR/kc-packages/rpm/el{8,9,10}/x86_64/qemu-guest-agent-*.rpm
build/kc-v2v/stage-linux-packages.sh "$IMGDIR/kc-packages"
```

Windows virtio-win (only if prepare picked `kc-convert-windows`). Needs
`bsdtar` (`brew install libarchive` on macOS):

```sh
# drivers: $IMGDIR/virtio-win/drivers/by-os/
# qemu-ga: $IMGDIR/virtio-win/guest-agent/
# optional ISO cache: VIRTIO_WIN_CACHE_DIR=$WORKDIR/virtio-win-cache
build/kc-v2v/stage-virtio-win.sh "$IMGDIR/virtio-win"
```

`--offline` skips network firstboot package installs. Local files are still
used when present. Virtio-win copy into the guest does not need the network;
without a staged tree Windows convert cannot find drivers.

## Run

```sh
# from the repo root (make build → bin/)
export PATH="$PWD/bin:$PATH"
export GOVC_VM=yzamir-d-5g-linux
export WORKDIR=/tmp/kc-debug
export IMGDIR=${IMGDIR:-$WORKDIR/$GOVC_VM}
# KC_QEMU_* still set from start-appliance.md

CONVERTER=$(jq -r '.prepare.converter' "$IMGDIR/pipeline.json")
echo "running $CONVERTER"

"$CONVERTER" \
  --backend qemu \
  --input "$IMGDIR/pipeline.json" \
  --output "$IMGDIR/pipeline.json" \
  --mount-root /tmp/kc-guest \
  --offline \
  --packages-dir "$IMGDIR/kc-packages" \
  --virtio-win-dir "$IMGDIR/virtio-win" \
  --log-level info
```

Linux virtio injection chroots into the guest's own tools (`dracut`,
`update-initramfs`, …) inside the appliance. On Apple Silicon the appliance is
arm64 (`KC_APPLIANCE_ARCH=arm64`, HVF); x86 guest tools run via binfmt/qemu-user.
Keep the debug shell attached if you want to watch `/mnt/guest`.

If qemu died after prepare, omit `KC_QEMU_*` and run the same command:
convert boots a new appliance and remounts from `pipeline.json`
([README.md](README.md#fallback-qemu-died)).

## Verify with the debug socket

Attach ([README.md](README.md#attach-the-debug-shell)).

### Linux

```sh
ls /mnt/guest/etc/systemd/system/kc-firstboot.service
ls /mnt/guest/usr/local/bin/kc-firstboot.sh
# VMware tools should be gone or repos disabled:
ls /mnt/guest/etc/vmware-tools 2>/dev/null || echo "vmware-tools dir absent (ok)"
grep -l virtio /mnt/guest/boot/loader/entries/* 2>/dev/null
grep virtio /mnt/guest/etc/default/grub 2>/dev/null
```

Success: firstboot unit present when the converter scheduled work; VMware
tools paths gone or yum/dnf repos with `vmware.com` have `enabled=0`; virtio
shows up in BLS entries or grub defaults after remap / initramfs rebuild.

### Windows

```sh
ls "/mnt/guest/Windows/System32/drivers" | grep -i vio
ls "/mnt/guest/Program Files/Guestfs/Firstboot"
```

Success: virtio driver files (`viostor`, `netkvm`, …) under `System32/drivers`
and firstboot scripts under `Program Files/Guestfs/Firstboot`.

On the host:

```sh
jq '.convert' "$IMGDIR/pipeline.json"
```

Continue with [finalize.md](finalize.md).
