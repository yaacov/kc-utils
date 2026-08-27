# Start the kc appliance and attach the debug shell

Boot our appliance (`vmlinuz` + `initramfs.img`) with the imported disks as
virtio-blk, then log in through `debug.sock`. This is **not** booting the
converted guest — that is [boot-guest-qemu-x86.md](boot-guest-qemu-x86.md).

The in-appliance binary is [`kc-guest-agent`](../apps/kc-guest-agent.md)
(`/init`): it serves the agent protocol and starts `/bin/bash -i` on
`org.kc-utils.debug` when a host client connects.

Keep this qemu command aligned with
[`qemuArgs` in launch.go](../../pkg/backend/plugins/qemu/launch.go).

## Build the appliance

```sh
cd /path/to/kc-utils
export KC_APPLIANCE_DIR=$PWD/bin/appliance
export KC_APPLIANCE_ARCH=arm64

# arm64 + HVF on Apple Silicon (x86 guests run tools via binfmt/qemu-user).
ARCHES=arm64 make build-appliance
```

Artifacts: `$KC_APPLIANCE_DIR/arm64/{vmlinuz,initramfs.img}`. Keep flags aligned
with [`qemuArgs` in launch.go](../../pkg/backend/plugins/qemu/launch.go).

## Launch (held)

QEMU creates the unix sockets; only the directory is created first. Redirect
serial to a file so the process can sit in the background across prepare /
convert / finalize.

Apple Silicon / arm64 Linux — default path. HVF on macOS, KVM on Linux:

```sh
export GOVC_VM=yzamir-d-5g-linux
export WORKDIR=/tmp/kc-debug
export IMGDIR=${IMGDIR:-$WORKDIR/$GOVC_VM}
export KC_APPLIANCE_DIR=${KC_APPLIANCE_DIR:-$PWD/bin/appliance}
export KC_APPLIANCE_ARCH=arm64
mkdir -p "$IMGDIR"
SOCKDIR=$(mktemp -d /tmp/kc-qemu-XXXXXX)

if [ ! -s "$KC_APPLIANCE_DIR/arm64/vmlinuz" ]; then
  echo "missing $KC_APPLIANCE_DIR/arm64/vmlinuz — from the repo: ARCHES=arm64 make build-appliance" >&2
  exit 1
fi

case "$(uname -s)" in
  Darwin) ACCEL=hvf ;;
  *)      ACCEL=kvm ;;
esac

qemu-system-aarch64 \
  -machine virt,accel=$ACCEL \
  -cpu host \
  -m 2048 \
  -smp 4 \
  -kernel "$KC_APPLIANCE_DIR/arm64/vmlinuz" \
  -initrd "$KC_APPLIANCE_DIR/arm64/initramfs.img" \
  -append 'console=ttyAMA0 panic=1 loglevel=4' \
  -nodefaults \
  -no-reboot \
  -display none \
  -serial "file:$IMGDIR/appliance-console.log" \
  -chardev "socket,id=kcagent,path=$SOCKDIR/agent.sock,server=on,wait=off" \
  -device virtio-serial \
  -device virtserialport,chardev=kcagent,name=org.kc-utils.agent \
  -chardev "socket,id=kcdebug,path=$SOCKDIR/debug.sock,server=on,wait=off" \
  -device virtserialport,chardev=kcdebug,name=org.kc-utils.debug \
  -drive "if=none,id=disk0,file=$IMGDIR/disk0.img,format=raw,cache=writeback" \
  -device virtio-blk-pci,drive=disk0 \
  &
# add another -drive/-device pair per extra diskN.img (disk1 → /dev/vdb, …)

# arm64 Linux uses accel=kvm (selected above); macOS uses HVF.
# x86 Linux / TCG fallback: qemu-system-x86_64 -machine q35,accel=kvm|tcg
#   -cpu host|max, $KC_APPLIANCE_DIR/amd64/{vmlinuz,initramfs.img}, console=ttyS0.

export KC_QEMU_PID=$!
export KC_QEMU_SOCK=$SOCKDIR/agent.sock
export KC_QEMU_DEBUG_SOCK=$SOCKDIR/debug.sock
```

First boot of a new initramfs can take a minute even with HVF; if attach or
prepare fails immediately, wait and retry, or check
`$IMGDIR/appliance-console.log`.

A comma in a disk path must be doubled in the `-drive` value (QEMU option
separator). Extra disks: `id=disk1`, `file=$IMGDIR/disk1.img`, then
`-device virtio-blk-pci,drive=disk1`. Order is `/dev/vda`, `/dev/vdb`, ….

Leave this process running. Stages adopt it via `KC_QEMU_*` and must not
kill it. Stop it only after [finalize.md](finalize.md).

## Attach

Use the snippet in [README.md](README.md#attach-the-debug-shell).

You are in the appliance initramfs. PID 1 is the agent; the debug PTY is a
child bash.

```sh
cat /proc/1/cmdline | tr '\0' ' '; echo
# .../kc-guest-agent or /init
```

## Verify with the debug socket

```sh
ls /dev/vd*
lsblk -J
cat /sys/class/virtio-ports/*/name
```

Success:

- `/dev/vda` exists (and `/dev/vdb` if you attached a second disk)
- `lsblk -J` lists partitions
- port names include `org.kc-utils.agent` and `org.kc-utils.debug`

Guest `exit` / Ctrl-D only respawns bash; leave socat with **Ctrl-]**
(`escape=0x1d` in the attach snippet).
Kernel messages are in `$IMGDIR/appliance-console.log`, not on this channel.

Continue with [prepare.md](prepare.md) from another terminal, same env.
