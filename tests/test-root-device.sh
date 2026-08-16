#!/bin/bash -
# Test: options.root with a device path picks the correct OS.

source "$(cd "$(dirname "$0")" && pwd)/functions.sh"
set -e

skip_if_skipped
requires_linux
requires_jq
ensure_built
root_test
requires_loop_partitions
requires guestfish --version

d=$(mktemp -d)
cleanup_fn rm -rf "$d"

MOUNT_ROOT="$d/mnt"
mkdir -p "$MOUNT_ROOT"

img="$d/multiboot.img"
make_multiboot_linux_img "$img"

loop=$(losetup --partscan --find --show "$img")
cleanup_fn losetup -d "$loop" 2>/dev/null || true
root_dev="${loop}p2"

cat > "$d/input.json" <<EOF
{
  "disks": [{"path": "$loop", "format": "raw"}],
  "source": {"name": "multiboot", "type": "disk"},
  "options": {"root": "$root_dev"}
}
EOF

"$BIN_DIR/kc-prepare" \
    --backend direct \
    --input "$d/input.json" \
    --output "$d/prepare.json" \
    --mount-root "$MOUNT_ROOT" \
    --log-level error

check_json_field "$d/prepare.json" '.prepare.status' 'complete'
check_json_field "$d/prepare.json" '.prepare.root_device' "$root_dev"
grep -q 'Debian' "$MOUNT_ROOT/etc/os-release"

echo "PASS: test-root-device"
