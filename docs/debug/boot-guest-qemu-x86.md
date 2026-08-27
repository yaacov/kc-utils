# Boot converted disks with qemu-system-x86_64

A **second** qemu: the converted **guest** OS, not the kc appliance. There is
no `debug.sock` here. Confirm virtio and bootloader from the appliance debug
shell in [finalize.md](finalize.md) **before** killing that VM; after this
launch, use the guest VGA window.

On Apple Silicon this is TCG and slow. Success is a kernel/login or Windows
boot screen, not performance.

## Firmware from TargetMeta

```sh
export GOVC_VM=yzamir-d-5g-linux
export WORKDIR=/tmp/kc-debug
export IMGDIR=${IMGDIR:-$WORKDIR/$GOVC_VM}
jq -r '.target.target_firmware' "$IMGDIR/pipeline.json"
# bios | uefi
```

## BIOS guest

```sh
qemu-system-x86_64 \
  -machine q35,accel=tcg \
  -cpu max \
  -m 4096 \
  -smp 4 \
  -drive if=virtio,file="$IMGDIR/disk0.img",format=raw \
  -netdev user,id=net0 \
  -device virtio-net-pci,netdev=net0
```

Add another `-drive if=virtio,file=...,format=raw` per extra disk.

## UEFI guest

Homebrew (Mac):

```sh
OVMF="$(brew --prefix qemu)/share/qemu/edk2-x86_64-code.fd"
```

Linux (path varies by distro):

```sh
OVMF=/usr/share/OVMF/OVMF_CODE.fd
# or: /usr/share/edk2/x64/OVMF_CODE.fd
```

```sh
qemu-system-x86_64 \
  -machine q35,accel=tcg \
  -cpu max \
  -m 4096 \
  -smp 4 \
  -drive if=pflash,format=raw,readonly=on,file="$OVMF" \
  -drive if=virtio,file="$IMGDIR/disk0.img",format=raw \
  -netdev user,id=net0 \
  -device virtio-net-pci,netdev=net0
```

## If it does not boot

Do not guess from this qemu alone. Re-hold the appliance **read-only** if you
want to preserve the images:

```sh
# same start-appliance.md command, but add ,readonly=on to each -drive
```

Attach `debug.sock` ([README.md](README.md#attach-the-debug-shell)), mount
read-only if you remount by hand, and re-check virtio in the initramfs /
BLS / `Windows/System32/drivers` as in [convert.md](convert.md).
