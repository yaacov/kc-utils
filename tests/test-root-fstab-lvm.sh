#!/bin/bash -
# Test: root on LVM with UUID-based fstab; /boot mounted from fstab.

source "$(cd "$(dirname "$0")" && pwd)/functions.sh"
set -e

skip_if_skipped
requires_linux
requires_jq
ensure_built
root_test
requires_loop_partitions
requires guestfish --version
requires vgchange --version
requires lvscan --version

d=$(mktemp -d)
cleanup_fn rm -rf "$d"
cleanup_fn vgchange -an kclvm 2>/dev/null || true

MOUNT_ROOT="$d/mnt"
mkdir -p "$MOUNT_ROOT"

img="$d/lvm.img"
make_lvm_linux_img "$img"

cat > "$d/input.json" <<EOF
{
  "disks": [{"path": "$img", "format": "raw"}],
  "source": {"name": "lvm-root", "type": "disk"},
  "options": {}
}
EOF

"$BIN_DIR/kc-prepare" \
    --input "$d/input.json" \
    --output "$d/prepare.json" \
    --mount-root "$MOUNT_ROOT" \
    --log-level error

check_json_field "$d/prepare.json" '.prepare.status' 'complete'
grep -q 'RHEL 9 LVM' "$MOUNT_ROOT/etc/os-release"
test -f "$MOUNT_ROOT/boot/grub2/grub.cfg"

root_dev=$(jq -r '.prepare.root_device' "$d/prepare.json")
case "$root_dev" in
    /dev/mapper/kclvm-root|/dev/kclvm/root) ;;
    *)
        echo "FAIL: unexpected LVM root device: $root_dev"
        exit 1
        ;;
esac

echo "PASS: test-root-fstab-lvm"
