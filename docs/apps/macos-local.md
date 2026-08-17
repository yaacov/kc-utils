# Convert a vSphere Linux VM on macOS

Walkthrough for copying an x86_64 VMware guest to your Mac, converting it with
the [qemu backend](../backends/qemu.md), and booting the result under
`qemu-system-x86_64`.

This is a **local shell** flow (`kc-copy` → `kc-prepare` → `kc-convert-linux`
→ `kc-finalize` → QEMU). It is not the MTV conversion-pod path
([kc-v2v.md](kc-v2v.md)).

The examples use the benchmark RHEL guest **`mtv-func-rhel9-4-uefi`**
(x86_64, UEFI). Swap the name if you pinned a different `RHEL_VM`.

Two different QEMU processes are involved:

| Process | Binary | Why |
|---------|--------|-----|
| Conversion appliance | host-arch QEMU (`qemu-system-aarch64` on Apple Silicon, `qemu-system-x86_64` on Intel) | `kc-prepare` / convert / finalize talk to `kc-agent` inside this VM. It **mounts** the guest disks; it does not boot the guest OS. |
| Converted guest | `qemu-system-x86_64` | Boots the converted Linux VM after finalize. |

On Apple Silicon, the appliance uses HVF (fast). The x86_64 guest uses TCG
(software emulation, much slower). Do **not** build `GOARCH=amd64` binaries on
Apple Silicon: HVF cannot run an x86 appliance there.

## Prerequisites

Install:

```bash
brew install go qemu jq
```

You also need a container runtime (Docker Desktop or Podman) for
`make appliance-*` and `make stage-offline`.

Optional but useful for discovering VMDK paths: `govc` (same credentials as
`tests/scenarios/.env` — do not commit that file).

Confirm QEMU is on `PATH`:

```bash
qemu-system-x86_64 --version
# Apple Silicon also needs:
qemu-system-aarch64 --version
```

Firmware for the **guest** boot (UEFI RHEL):

```bash
ls "$(dirname "$(which qemu-system-x86_64)")/../share/qemu/edk2-x86_64-code.fd"
```

Homebrew usually installs that file at
`/opt/homebrew/share/qemu/edk2-x86_64-code.fd` (Apple Silicon) or
`/usr/local/share/qemu/edk2-x86_64-code.fd` (Intel).

## 1. Build host binaries and the appliance

From the repository root. `make` defaults to `GOOS=linux`; override it for a
Mac CLI:

```bash
cd /path/to/kc-utils

export GOOS=darwin
make build

# Appliance matches the Mac CPU, not the guest.
case "$(uname -m)" in
  arm64)  make appliance-arm64; export KC_APPLIANCE_DIR="$PWD/build/appliance/out/arm64" ;;
  x86_64) make appliance-amd64; export KC_APPLIANCE_DIR="$PWD/build/appliance/out/amd64" ;;
esac

# Offline qemu-guest-agent RPMs + rhsrvany.exe for RHEL-family guests (Fedora container).
make stage-offline
export KC_PACKAGES="$PWD/build/offline/kc-packages"
export KC_VIRT_TOOLS="$PWD/build/offline/virt-tools"

export PATH="$PWD/bin:$PATH"
file bin/kc-prepare   # must be Mach-O arm64 on Apple Silicon, x86_64 on Intel
```

`kc-agent` is Linux-only and is packed into the appliance; you do not run it
on the Mac. Details: [appliance.md](../backends/appliance.md).

## 2. Work directory

```bash
export WORK="$PWD/tmp/macos-v2v"
mkdir -p "$WORK/disks" "$WORK/copy"
```

vSphere credentials — two plain files in a directory you control:

```bash
mkdir -p "$WORK/secret"
printf '%s' "$GOVC_USERNAME" > "$WORK/secret/accessKeyId"
printf '%s' "$GOVC_PASSWORD" > "$WORK/secret/secretKey"
chmod 600 "$WORK/secret/accessKeyId" "$WORK/secret/secretKey"
```

Pass `--secret-dir "$WORK/secret"` to `kc-copy` (default is `/etc/secret`).

## 3. Identify the source VM

Power the VM **off** for a consistent cold copy:

```bash
export VM_NAME="${RHEL_VM:-mtv-func-rhel9-4-uefi}"
export VC_HOST="${GOVC_URL#https://}"
VC_HOST="${VC_HOST%%/*}"

govc vm.power -off "$VM_NAME"

# SHA-1 thumbprint with colons (required even with --insecure).
export VC_FINGERPRINT="$(
  openssl s_client -connect "${VC_HOST}:443" </dev/null 2>/dev/null \
    | openssl x509 -noout -fingerprint -sha1 \
    | sed 's/^.*=//'
)"

# Datastore VMDK paths, e.g. [ds] folder/disk.vmdk
govc vm.info -json "$VM_NAME" | jq -r '
  .VirtualMachines[0].Config.Hardware.Device[]
  | select((.DeviceInfo.Label // "") | test("Hard disk"; "i"))
  | .Backing.FileName
'
```

Copy those `FileName` values into `--disk-path` (comma-separated, same order
as `disk0`, `disk1`, …) if you want to select specific disks. If omitted,
all VM disks are copied.

Optional: MAC and firmware for `prepare-input.json`:

```bash
govc device.info -json -vm "$VM_NAME" ethernet-* \
  | jq -r '.. | .MacAddress? // empty' | head -n1

govc vm.info -json "$VM_NAME" \
  | jq -r '.VirtualMachines[0].Config.Firmware'
```

`mtv-func-rhel9-4-uefi` is UEFI.

## 4. kc-copy

Copies all VM disks as raw images into `--output-dir`. The files are named
`disk0.img`, `disk1.img`, … and can be passed directly to `kc-prepare` via the
input JSON.

```bash
kc-copy \
  --host "$VC_HOST" \
  --insecure \
  --vm-name "$VM_NAME" \
  --fingerprint "$VC_FINGERPRINT" \
  --secret-dir "$WORK/secret" \
  --output-dir "$WORK/disks" \
  --work-dir "$WORK/copy" \
  --output "$WORK/copy/copy-progress.json" \
  --log-level info

ls -lh "$WORK/disks/"
jq . "$WORK/copy/copy-progress.json"
```

See [kc-copy.md](kc-copy.md) for TLS modes (`--ca-cert` instead of
`--insecure`), `--datacenter`, and `--disk-path` to select specific disks.

## 5. Shared qemu appliance session

Convert and finalize **attach** to the appliance that prepare starts. They do
not spawn a second VM. Export a socket path **before** prepare, and do not
create the socket file yourself:

```bash
export KC_AGENT_SOCK="$WORK/agent.sock"
export KC_APPLIANCE_DIR="${KC_APPLIANCE_DIR:?set in step 1}"
# Optional: more RAM for dracut inside the appliance (default 2048).
export V2V_memSize=4096
```

If `KC_AGENT_SOCK` is unset, prepare starts a process-local QEMU and convert
fails with `qemu attach requires KC_AGENT_SOCK`.

`--mount-root` is only a path key in qemu mode (no host bind-mount):

```bash
export MOUNT_ROOT="$WORK/guest"
mkdir -p "$MOUNT_ROOT"
```

## 6. kc-prepare (`--backend qemu`)

Build a minimal input JSON — disks are auto-discovered from `disk_dir`:

```bash
cat > "$WORK/prepare-input.json" <<EOF
{
  "disk_dir": "$WORK/disks",
  "source": {
    "name": "$VM_NAME",
    "type": "vmware",
    "firmware_hint": "uefi"
  },
  "options": {
    "tmp_dir": "$WORK/tmp"
  }
}
EOF

mkdir -p "$WORK/tmp"

kc-prepare \
  --input "$WORK/prepare-input.json" \
  --output "$WORK/pipeline.json" \
  --mount-root "$MOUNT_ROOT" \
  --backend qemu \
  --log-level info
```

`disk_dir` scans the directory for `disk*.img` files (sorted by name, assumed
raw format). You can override it with `--disk-dir` on the command line, or list
disks explicitly in the `"disks"` array for full control.

Check inspect before converting:

```bash
jq '{converter, inspect: .inspect, firmware: .firmware}' "$WORK/pipeline.json"
```

For this Linux guest, `converter` should be `kc-convert-linux`. Leave the
appliance QEMU running.

Debug the live appliance with [kc-agent-sh](kc-agent-sh.md):

```bash
kc-agent-sh --sock "$KC_AGENT_SOCK" -- lsblk
```

## 7. kc-convert-linux

```bash
kc-convert-linux \
  --input "$WORK/pipeline.json" \
  --output "$WORK/pipeline.json" \
  --mount-root "$MOUNT_ROOT" \
  --backend qemu \
  --offline \
  --log-level info
```

`--offline` installs qemu-guest-agent from `$KC_PACKAGES` at firstboot when a
matching RHEL RPM exists, and does not fall back to `dnf` on the guest
network. See [kc-convert-linux.md](kc-convert-linux.md).

## 8. kc-finalize

```bash
kc-finalize \
  --input "$WORK/pipeline.json" \
  --output "$WORK/pipeline.json" \
  --mount-root "$MOUNT_ROOT" \
  --backend qemu \
  --log-level info

jq '.target | {target_firmware, guestcaps, target_buses, warnings}' \
  "$WORK/pipeline.json"
```

Finalize unmounts and fscks inside the appliance. It does **not** kill QEMU
when `KC_AGENT_SOCK` was pre-set (`owned=false`). Stop it yourself:

```bash
kill "$(cat "$WORK/qemu.pid")" 2>/dev/null || true
rm -f "$KC_AGENT_SOCK" "$WORK/qemu.pid"
```

## 9. Boot the converted guest (`qemu-system-x86_64`)

Read firmware and buses from `pipeline.json` (`target` section). For the
benchmark UEFI RHEL VM:

```bash
DISK0="$WORK/disks/disk0.img"
QEMU_SHARE="$(dirname "$(which qemu-system-x86_64)")/../share/qemu"
OVMF_CODE="$QEMU_SHARE/edk2-x86_64-code.fd"
OVMF_VARS_SRC="$QEMU_SHARE/edk2-i386-vars.fd"
cp "$OVMF_VARS_SRC" "$WORK/ovmf-vars.fd"

# Apple Silicon: TCG. Intel Mac: HVF.
case "$(uname -m)" in
  arm64)  ACCEL=tcg; CPU=qemu64 ;;
  x86_64) ACCEL=hvf; CPU=host ;;
esac

qemu-system-x86_64 \
  -machine q35,accel="$ACCEL" \
  -cpu "$CPU" \
  -m 4096 \
  -smp 2 \
  -drive if=pflash,format=raw,readonly=on,file="$OVMF_CODE" \
  -drive if=pflash,format=raw,file="$WORK/ovmf-vars.fd" \
  -drive file="$DISK0",if=none,id=hd0,format=raw,cache=none \
  -device virtio-blk-pci,drive=hd0 \
  -netdev user,id=net0,hostfwd=tcp::2222-:22 \
  -device virtio-net-pci,netdev=net0 \
  -nographic
```

Serial console is configured during conversion, so `-nographic` shows GRUB
and the kernel. SSH (if the guest has sshd and a user) is on
`localhost:2222`.

BIOS guests (`jq -r .target.target_firmware "$WORK/pipeline.json"` → `bios`):
drop the two `pflash` drives and let QEMU use SeaBIOS.

Extra disks: add `-drive file=$WORK/disks/diskN.img,if=none,id=hdN,format=raw`
and `-device virtio-blk-pci,drive=hdN` for each.

First boot can take several minutes (firstboot scripts, qemu-guest-agent RPM
install). TCG on Apple Silicon is slow; give it time.

Quit QEMU: `Ctrl-a x` (nographic), or `kill` the process.

## Cleanup

```bash
kill "$(cat "$WORK/qemu.pid")" 2>/dev/null || true
rm -rf "$WORK"
```

## Troubleshooting

| Symptom | What to check |
|---------|----------------|
| `read vSphere password` | `--secret-dir` must point to a directory with `accessKeyId` and `secretKey` files. |
| `appliance artifact … set KC_APPLIANCE_DIR` | Step 1: `make appliance-arm64` (Apple Silicon) or `appliance-amd64` (Intel). Directory must contain `vmlinuz` and `initramfs.img`. |
| `qemu attach requires KC_AGENT_SOCK` | Export `KC_AGENT_SOCK` **before** `kc-prepare`, and keep it for convert/finalize. |
| Appliance QEMU exits immediately | `qemu-system-aarch64` (Apple Silicon) / `qemu-system-x86_64` (Intel) on `PATH`; HVF available; `KC_APPLIANCE_DIR` matches `uname -m`. |
| Convert binary is `kc-convert-windows` | Wrong VM, or inspect failed. This guide assumes the Linux benchmark guest. |
| Guest hangs at firmware | UEFI vs BIOS mismatch; use `target.target_firmware` from `pipeline.json`. |
| Guest has no virtio disk | Conversion did not finish; confirm `target.guestcaps.block_bus` is `virtio`. |

## Related

- [README.md](README.md) — pipeline overview
- [../backends/qemu.md](../backends/qemu.md) — qemu backend, macOS notes, shared session
- [kc-copy.md](kc-copy.md), [kc-prepare.md](kc-prepare.md), [kc-convert-linux.md](kc-convert-linux.md), [kc-finalize.md](kc-finalize.md)
- [examples/](examples/README.md) — sample JSON
- [../../tests/scenarios/test-mtv-benchmark.md](../../tests/scenarios/test-mtv-benchmark.md) — cluster benchmark (`mtv-func*` VMs)
