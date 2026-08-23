# Fetch VMware disks to the local machine

Copy a vSphere VM's disks to sparse raw `diskN.img` files with
[`kc-copy`](../apps/kc-copy.md). Runs on Mac or Linux. No VDDK, nbdkit, or
libguestfs.

## Credentials

`kc-copy` reads vSphere user/password from files, not flags:

```sh
sudo mkdir -p /etc/secret
printf '%s' "$GOVC_USERNAME" | sudo tee /etc/secret/accessKeyId >/dev/null
printf '%s' "$GOVC_PASSWORD" | sudo tee /etc/secret/secretKey >/dev/null
sudo chmod 600 /etc/secret/accessKeyId /etc/secret/secretKey
```

## vCenter fingerprint

SHA-1 thumbprint of the vCenter TLS certificate (colon-separated hex):

```sh
echo | openssl s_client -connect vcenter.example.com:443 2>/dev/null \
  | openssl x509 -noout -fingerprint -sha1
```

## Copy

From the repo root, after `make build` (or `make build-kc-copy`):

```sh
export WORKDIR=~/kc-debug/my-vm
mkdir -p "$WORKDIR"

kc-copy \
  --host vcenter.example.com \
  --datacenter mydatacenter \
  --insecure \
  --vm-name my-vm \
  --fingerprint "AB:CD:EF:..." \
  --target-dir "$WORKDIR" \
  --output "$WORKDIR/copy-progress.json" \
  --log-level info
```

`--insecure` is for a lab. Prefer `--ca-cert /path/to/ca.pem` (or omit both
`--insecure` and `--ca-cert` to use the host trust store) plus the fingerprint
as a thumbprint fallback. See [kc-copy.md](../apps/kc-copy.md) for TLS modes
and `--disk-path` to select a subset of VMDKs.

Result:

```text
$WORKDIR/disk0.img
$WORKDIR/disk1.img   # if the VM had more than one disk
$WORKDIR/copy-progress.json
```

Images are **raw** (stream-optimized VMDK decompressed on the fly), typically
sparse. `jq . "$WORKDIR/copy-progress.json"` should list each disk with
`"status": "complete"` (or the equivalent completed status in that file).

## Verify with the debug socket

Host `ls -lh` / `qemu-img info` only prove a file exists. Confirm the copy is
a real disk by attaching it to the kc appliance and inspecting block devices.

1. Follow [start-appliance.md](start-appliance.md) with `$WORKDIR/disk0.img`
   (and further `diskN.img` in the same order).
2. Attach the debug shell ([README.md](README.md#attach-the-debug-shell)).
3. In the appliance:

```sh
lsblk -f
blkid
```

Success: `/dev/vda` (then `/dev/vdb`, …) shows partitions and filesystem
types (ext4, xfs, ntfs, vfat, LVM, …), not an empty disk. `lsblk -J` is
what prepare will parse next.

Then continue with [prepare.md](prepare.md) **without** stopping qemu.
