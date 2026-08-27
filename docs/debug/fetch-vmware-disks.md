# Fetch VMware disks to the local machine

Copy a vSphere VM's disks to sparse raw `diskN.img` files with
[`kc-copy`](../apps/kc-copy.md). Runs on Mac or Linux. No libguestfs.

This page uses the **eco** vCenter (`ECO_VSPHERE_*` in the shell).

## Credentials

Pass vSphere user/password as flags (`$ECO_VSPHERE_USERNAME` /
`$ECO_VSPHERE_PASSWORD`). `--password-file` is optional when you do not want
the password on argv. If a flag is empty, `kc-copy` reads
`/etc/secret/accessKeyId` and `/etc/secret/secretKey` (Forklift secret
mount).

`--host` is `$ECO_VSPHERE_URL` (hostname only, no `https://`).

## vCenter fingerprint

SHA-1 thumbprint of the vCenter TLS certificate (colon-separated hex):

```sh
export ECO_VSPHERE_FINGERPRINT=$(echo | openssl s_client -connect "${ECO_VSPHERE:-${ECO_VSPHERE_URL}:${ECO_VSPHERE_PORT:-443}}" 2>/dev/null \
  | openssl x509 -noout -fingerprint -sha1 | cut -d= -f2)
```

## Copy

From the repo root, after `GOOS=darwin make build-kc-copy` (or `make build` on
Linux):

```sh
export PATH="$PWD/bin:$PATH"
export ECO_VSPHERE_VM=yzamir-d-5g-linux
export GOVC_VM=$ECO_VSPHERE_VM
export ECO_VSPHERE_DATACENTER=Eco-Datacenter
export WORKDIR=/tmp/kc-debug
export IMGDIR=$WORKDIR/$GOVC_VM
mkdir -p "$IMGDIR"

kc-copy \
  --host "$ECO_VSPHERE_URL" \
  --username "$ECO_VSPHERE_USERNAME" \
  --password "$ECO_VSPHERE_PASSWORD" \
  --datacenter "$ECO_VSPHERE_DATACENTER" \
  --vm-name "$ECO_VSPHERE_VM" \
  --fingerprint "$ECO_VSPHERE_FINGERPRINT" \
  --target-dir "$IMGDIR" \
  --output "$IMGDIR/copy-progress.json" \
  --log-level info
```

Prefer `--ca-cert /path/to/ca.pem` (or omit both `--insecure` and `--ca-cert`
to use the host trust store) plus the fingerprint as a thumbprint fallback.
See [kc-copy.md](../apps/kc-copy.md) for TLS modes and `--disk-path` to select
a subset of VMDKs.

Lab only — skip TLS verification:

```sh
kc-copy \
  --host "$ECO_VSPHERE_URL" \
  --username "$ECO_VSPHERE_USERNAME" \
  --password "$ECO_VSPHERE_PASSWORD" \
  --datacenter "$ECO_VSPHERE_DATACENTER" \
  --insecure \
  --vm-name "$ECO_VSPHERE_VM" \
  --fingerprint "$ECO_VSPHERE_FINGERPRINT" \
  --target-dir "$IMGDIR" \
  --output "$IMGDIR/copy-progress.json"
```

Result:

```text
$IMGDIR/disk0.img
$IMGDIR/copy-progress.json
```

Images are **raw** (stream-optimized VMDK decompressed on the fly), typically
sparse. `jq . "$IMGDIR/copy-progress.json"` should list each disk with
`"status": "complete"` (or the equivalent completed status in that file).

## Boot the unconverted guest (legacy IDE + e1000)

The copy is still a VMware guest: no virtio. Attach disks as **IDE** and the
NIC as **e1000** so in-tree drivers can boot it. This is not the kc
appliance and not [boot-guest-qemu-x86.md](boot-guest-qemu-x86.md) (virtio
after convert). SeaBIOS is qemu's default (BIOS guest).

```sh
export ECO_VSPHERE_VM=yzamir-d-5g-linux
export GOVC_VM=$ECO_VSPHERE_VM
export WORKDIR=/tmp/kc-debug
export IMGDIR=$WORKDIR/$GOVC_VM

disks=()
i=0
while [ -e "$IMGDIR/disk$i.img" ]; do
  disks+=(-drive "file=$IMGDIR/disk$i.img,format=raw,if=ide")
  i=$((i+1))
done

qemu-system-x86_64 \
  -machine pc,accel=tcg \
  -cpu max \
  -m 4096 \
  -smp 2 \
  "${disks[@]}" \
  -netdev user,id=net0 \
  -device e1000,netdev=net0 \
  -vga std
```

The loop attaches every `$IMGDIR/diskN.img` as IDE (boot disk first). Do not use
`-nographic`: an unconverted guest rarely has a serial console. On Apple
Silicon this is TCG and slow; success is GRUB or a login prompt.

Quit qemu (`Ctrl-C` in that terminal, or the window close) **before**
[start-appliance.md](start-appliance.md). The same `diskN.img` cannot be
opened by two qemu processes.

Then continue with [start-appliance.md](start-appliance.md) and
[prepare.md](prepare.md) to convert.
