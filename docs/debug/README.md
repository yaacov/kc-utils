# Local qemu debug cookbook

Hands-on conversion on a Mac or Linux workstation with `--backend qemu`.
Each page is a copy-paste how-to and ends with **debug-socket checks** that
prove that step worked.

This is not the CLI contract. Stage flags and JSON live under
[`docs/apps/`](../apps/). Design of the appliance and protocol:
[`docs/architecture/backends.md`](../architecture/backends.md).
The in-appliance binary: [`docs/apps/kc-guest-agent.md`](../apps/kc-guest-agent.md).

```text
1. fetch-vmware-disks.md    kc-copy → diskN.img
2. start-appliance.md       hold qemu + attach debug.sock
3. prepare.md               kc-prepare --backend qemu (adopts)
4. convert.md               kc-convert-* (same appliance)
5. finalize.md              kc-finalize, then stop qemu
6. boot-guest-qemu-x86.md   qemu-system-x86_64 boots the guest
```

Do **not** use `direct` or `guestfs` here (Linux-only, root / libguestfs).
Do **not** enable kc-v2v qcow2 overlays; convert in place on the raw
`diskN.img` files so step 6 boots the same images.

## Prerequisites

| Tool | macOS | Linux |
|------|-------|-------|
| `qemu-system-x86_64` | `brew install qemu` | distro `qemu-system-x86` (or equivalent) |
| `jq` | `brew install jq` | distro `jq` |
| `python3` | Xcode CLT or `brew install python` | distro `python3` (agent Ping wait in [start-appliance.md](start-appliance.md)) |
| attach client | BSD `nc` (stock) | `socat` |
| kc binaries | `make build` → `bin/` | same |
| appliance | `make build-appliance` | same |

VMware x86 guests on Apple Silicon (or any arm64 host) need an **amd64**
appliance so convert can chroot into guest-arch binaries. That appliance runs
under TCG (slow, expected):

```sh
export KC_APPLIANCE_ARCH=amd64
ARCHES=amd64 make build-appliance
```

Native x86 Linux uses KVM. An arm64 Mac converting an arm64 guest can use HVF
and `KC_APPLIANCE_ARCH=arm64`; this cookbook assumes x86 VMware VMs.

```sh
cd /path/to/kc-utils
export KC_APPLIANCE_DIR=$PWD/bin/appliance
export PATH="$PWD/bin:$PATH"
```

## Workdir

```sh
export WORKDIR=~/kc-debug/my-vm
mkdir -p "$WORKDIR"
# diskN.img, prepare-input.json, pipeline.json live here
```

## Held appliance (main path)

Launch qemu yourself ([start-appliance.md](start-appliance.md)) with disks
attached and keep that process alive until after finalize. Export:

```sh
export KC_QEMU_SOCK=$SOCKDIR/agent.sock
export KC_QEMU_PID=<qemu-pid>
export KC_QEMU_DEBUG_SOCK=$SOCKDIR/debug.sock
```

Every stage then **adopts** the VM (`ownedExternally`) and must not kill it.
Mounts established in prepare persist into convert and finalize.

### Fallback (qemu died)

If the held process exits after prepare, convert and finalize still work: they
boot a fresh appliance and remount from `pipeline.json`
(`remountFromDiskInfos`). Re-export is not required; attach a new `debug.sock`
from the new `/tmp/kc-qemu-*/` directory. You lose live mounts from the
previous VM.

## Attach the debug shell

You are in the **appliance** (initramfs), not the converted guest. After
prepare, guest filesystems appear under `/mnt/guest`. Detach with `exit` or by
closing the client; bash respawns.

```sh
if [ -n "${KC_QEMU_DEBUG_SOCK-}" ] && [ -S "$KC_QEMU_DEBUG_SOCK" ]; then
    sock=$KC_QEMU_DEBUG_SOCK
else
    set -- /tmp/kc-qemu-*/debug.sock
    if [ "$#" -ne 1 ] || [ ! -S "$1" ]; then
        echo "pick a debug socket and set sock=/tmp/kc-qemu-XXX/debug.sock:" >&2
        [ "$#" -gt 0 ] && ls -1 "$@"
        exit 1
    fi
    sock=$1
fi

# Linux (socat)
socat UNIX-CONNECT:"$sock" STDIO,raw,echo=0

# macOS (BSD nc); restore the tty after
stty raw -echo
nc -U "$sock"
stty sane
```

Keep flags in [start-appliance.md](start-appliance.md) aligned with
[`pkg/backend/plugins/qemu/launch.go`](../../pkg/backend/plugins/qemu/launch.go)
(`qemuArgs`).
