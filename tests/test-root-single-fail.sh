#!/bin/bash -
# Test: multiboot detection fails unless options.root is set.

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
  "options": {}
}
EOF

set +e
"$BIN_DIR/kc-prepare" \
    --backend direct \
    --input "$d/input.json" \
    --output "$d/prepare.json" \
    --mount-root "$MOUNT_ROOT" \
    --log-level error
rc=$?
set -e

if [ $rc -eq 0 ]; then
    echo "FAIL: expected multiboot error"
    exit 1
fi

test -f "$d/prepare.json"
grep -q 'multiple operating systems found' "$d/prepare.json"

echo "PASS: test-root-single-fail"
