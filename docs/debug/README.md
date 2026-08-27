# Local qemu debug cookbook

Hands-on conversion on a Mac or Linux workstation with `--backend qemu`.
Each page is a copy-paste how-to and ends with **debug-socket checks** that
prove that step worked.

This is not the CLI contract. Stage flags and JSON live under
[`docs/apps/`](../apps/). Design of the appliance and protocol:
[`docs/architecture/backends.md`](../architecture/backends.md).
The in-appliance binary: [`docs/apps/kc-guest-agent.md`](../apps/kc-guest-agent.md).

```text
1. fetch-vmware-disks.md    kc-copy → diskN.img; boot guest with IDE+e1000
2. start-appliance.md       hold qemu + attach debug.sock
3. prepare.md               kc-prepare --backend qemu (adopts)
4. convert.md               kc-convert-* (same appliance)
5. finalize.md              kc-finalize, then stop qemu
6. boot-guest-qemu-x86.md   qemu-system-x86_64 boots the converted guest (virtio)
```

Do **not** use `direct` or `guestfs` here (Linux-only, root / libguestfs).
Do **not** enable kc-v2v qcow2 overlays; convert in place on the raw
`diskN.img` files so step 6 boots the same images.

## Prerequisites

| Tool | macOS | Linux |
|------|-------|-------|
| `qemu-system-aarch64` / `qemu-system-x86_64` | `brew install qemu` | distro `qemu-system-*` |
| `jq` | `brew install jq` | distro `jq` |
| attach client | `brew install socat` | distro `socat` |
| kc binaries | `make build` → `bin/` (`export PATH="$PWD/bin:$PATH"` from the repo root) | same |
| appliance | `make build-appliance` | same |

The appliance matches the **host** CPU. On Apple Silicon that is **arm64**
(HVF). Convert chroots into the disk and runs foreign guest binaries via
**binfmt / qemu-user**.

```sh
cd /path/to/kc-utils
export PATH="$PWD/bin:$PATH"
export KC_APPLIANCE_DIR=$PWD/bin/appliance
# Host architecture: arm64 on Apple Silicon / arm64 Linux; amd64 on x86_64 Linux.
export KC_APPLIANCE_ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
make build
ARCHES=$KC_APPLIANCE_ARCH make build-appliance
```

## Workdir

```sh
export GOVC_VM=yzamir-d-5g-linux
export WORKDIR=/tmp/kc-debug
export IMGDIR=$WORKDIR/$GOVC_VM
mkdir -p "$IMGDIR"
# diskN.img, prepare-input.json, pipeline.json live in $IMGDIR
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
prepare, guest filesystems appear under `/mnt/guest`. Guest `exit` / Ctrl-D
only ends that bash; QEMU keeps `debug.sock` open so the agent starts a new
shell and `socat` stays connected. Leave with **Ctrl-]** (`escape=0x1d`).
Bash starts again on the next attach.

```sh
# If unset: ls /tmp/kc-qemu-*/debug.sock and pick one
# (start-appliance.md exports KC_QEMU_DEBUG_SOCK).
sock=$KC_QEMU_DEBUG_SOCK
if [ -z "$sock" ]; then
  sock=$(ls /tmp/kc-qemu-*/debug.sock 2>/dev/null | awk 'NR==1')
fi
# Ctrl-] leaves socat (exit / Ctrl-D only respawn bash)
socat UNIX-CONNECT:"$sock" STDIO,raw,echo=0,escape=0x1d
```

Keep flags in [start-appliance.md](start-appliance.md) aligned with
[`pkg/backend/plugins/qemu/launch.go`](../../pkg/backend/plugins/qemu/launch.go)
(`qemuArgs`).
