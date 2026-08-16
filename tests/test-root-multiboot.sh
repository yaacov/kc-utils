#!/bin/bash -
# Test: only the chosen OS root is mounted on multiboot disks.

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

cat > "$d/input.json" <<EOF
{
  "disks": [{"path": "$img", "format": "raw"}],
  "source": {"name": "multiboot", "type": "disk"},
  "options": {"root": "first"}
}
EOF

"$BIN_DIR/kc-prepare" \
    --backend direct \
    --input "$d/input.json" \
    --output "$d/prepare.json" \
    --mount-root "$MOUNT_ROOT" \
    --log-level error

check_json_field "$d/prepare.json" '.prepare.status' 'complete'
grep -q 'Red Hat' "$MOUNT_ROOT/etc/os-release"
if grep -q 'Debian' "$MOUNT_ROOT/etc/os-release"; then
    echo "FAIL: second OS content visible at mount root"
    exit 1
fi

# Second root partition must not be mounted at guest /.
mounted=$(jq -r '.prepare.disks[0].partitions[] | select(.mount_point=="/") | .index' "$d/prepare.json")
if [ "$mounted" != "1" ]; then
    echo "FAIL: expected partition 1 mounted at /, got index $mounted"
    exit 1
fi

echo "PASS: test-root-multiboot"
