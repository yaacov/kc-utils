# Convert on an already prepared appliance

Run the converter selected by prepare **without** stopping qemu. Mounts from
prepare are still live under `/mnt/guest`. CLI contracts:
[`kc-convert-linux.md`](../apps/kc-convert-linux.md),
[`kc-convert-windows.md`](../apps/kc-convert-windows.md).

## Run

```sh
export WORKDIR=~/kc-debug/my-vm
# KC_QEMU_* still set from start-appliance.md

CONVERTER=$(jq -r '.prepare.converter' "$WORKDIR/pipeline.json")

"$CONVERTER" \
  --backend qemu \
  --input "$WORKDIR/pipeline.json" \
  --output "$WORKDIR/pipeline.json" \
  --mount-root /tmp/kc-guest \
  --offline \
  --log-level info
```

`--offline` skips network firstboot package installs. Use it on a laptop
unless you staged local packages:

- Linux RHEL-family: `/usr/share/kc-packages/rpm/el{8,9,10}/x86_64/`
  ([kc-convert-linux.md](../apps/kc-convert-linux.md#qemu-guest-agent-installation))
- Windows: virtio-win tree on the **host** at
  `/usr/share/virtio-win/drivers/by-os/`
  ([CONTRIBUTING.md](../../community/CONTRIBUTING.md)). Convert copies those
  files into the guest through the appliance; `--offline` does not remove that
  requirement.

Linux virtio injection chroots into the guest's own tools (`dracut`,
`update-initramfs`, …) inside the appliance. On Apple Silicon that is TCG and
can take a long time. Keep the debug shell attached if you want to watch
`/mnt/guest`.

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
jq '.convert' "$WORKDIR/pipeline.json"
```

Continue with [finalize.md](finalize.md).
