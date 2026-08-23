# Start the kc appliance and attach the debug shell

Boot our appliance (`vmlinuz` + `initramfs.img`) with the imported disks as
virtio-blk, then log in through `debug.sock`. This is **not** booting the
converted guest — that is [boot-guest-qemu-x86.md](boot-guest-qemu-x86.md).

The in-appliance binary is [`kc-guest-agent`](../apps/kc-guest-agent.md)
(`/init`): it serves the agent protocol and binds `/bin/bash -i` to
`org.kc-utils.debug`.

Keep this qemu command aligned with
[`qemuArgs` in launch.go](../../pkg/backend/plugins/qemu/launch.go).

## Build the appliance

```sh
cd /path/to/kc-utils
export KC_APPLIANCE_DIR=$PWD/bin/appliance

# x86 VMware guests (required on Apple Silicon / arm64 hosts):
export KC_APPLIANCE_ARCH=amd64
ARCHES=amd64 make build-appliance
```

Artifacts: `$KC_APPLIANCE_DIR/amd64/{vmlinuz,initramfs.img}`.

## Launch (held)

QEMU creates the unix sockets; only the directory is created first. Redirect
serial to a file so the process can sit in the background across prepare /
convert / finalize.

```sh
export WORKDIR=~/kc-debug/my-vm
mkdir -p "$WORKDIR"
SOCKDIR=$(mktemp -d /tmp/kc-qemu-XXXXXX)

# Linux x86 with KVM: accel=kvm and -cpu host
# Apple Silicon / no KVM: accel=tcg and -cpu max (below)

qemu-system-x86_64 \
  -machine q35,accel=tcg \
  -cpu max \
  -m 2048 \
  -smp 4 \
  -kernel "$KC_APPLIANCE_DIR/amd64/vmlinuz" \
  -initrd "$KC_APPLIANCE_DIR/amd64/initramfs.img" \
  -append 'console=ttyS0 panic=1 loglevel=4' \
  -nodefaults \
  -no-reboot \
  -display none \
  -serial "file:$WORKDIR/appliance-console.log" \
  -chardev "socket,id=kcagent,path=$SOCKDIR/agent.sock,server=on,wait=off" \
  -device virtio-serial \
  -device virtserialport,chardev=kcagent,name=org.kc-utils.agent \
  -chardev "socket,id=kcdebug,path=$SOCKDIR/debug.sock,server=on,wait=off" \
  -device virtserialport,chardev=kcdebug,name=org.kc-utils.debug \
  -drive "if=none,id=disk0,file=$WORKDIR/disk0.img,format=raw,cache=writeback" \
  -device virtio-blk-pci,drive=disk0 \
  &
# add another -drive/-device pair per extra diskN.img (disk1 → /dev/vdb, …)

export KC_QEMU_PID=$!
export KC_QEMU_SOCK=$SOCKDIR/agent.sock
export KC_QEMU_DEBUG_SOCK=$SOCKDIR/debug.sock

# Agent Ping: length-prefixed JSON {"op":"ping"} (same framing as pkg/qemuagent/proto).
agent_ping() {
  python3 -c '
import json, socket, struct, sys
s = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
s.settimeout(2)
s.connect(sys.argv[1])
body = b"{\"op\":\"ping\"}"
s.sendall(struct.pack(">I", len(body)) + body)
hdr = s.recv(4)
if len(hdr) != 4:
    raise SystemExit(1)
n = struct.unpack(">I", hdr)[0]
resp = json.loads(s.recv(n) or b"{}")
raise SystemExit(1 if resp.get("err") else 0)
' "$1"
}

# Wait until the agent answers Ping (not only debug.sock). TCG boot can take minutes.
deadline=$((SECONDS + 300))
while true; do
  if ! kill -0 "$KC_QEMU_PID" 2>/dev/null; then
    echo "qemu exited before agent ready; see $WORKDIR/appliance-console.log" >&2
    exit 1
  fi
  if [ -S "$KC_QEMU_SOCK" ] && agent_ping "$KC_QEMU_SOCK"; then
    break
  fi
  if [ "$SECONDS" -ge "$deadline" ]; then
    echo "appliance not ready after 300s; see $WORKDIR/appliance-console.log" >&2
    exit 1
  fi
  sleep 1
done
```

A comma in a disk path must be doubled in the `-drive` value (QEMU option
separator). Extra disks: `id=disk1`, `file=$WORKDIR/disk1.img`, then
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

Detach with `exit`. Bash respawns for the next connect. Kernel messages are
in `$WORKDIR/appliance-console.log`, not on this channel.

Continue with [prepare.md](prepare.md) from another terminal, same env.
