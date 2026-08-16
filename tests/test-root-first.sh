#!/bin/bash -
# Test: options.root first selects first OS on multiboot disk.

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
test -f "$MOUNT_ROOT/etc/os-release"
grep -q 'Red Hat' "$MOUNT_ROOT/etc/os-release"

echo "PASS: test-root-first"
