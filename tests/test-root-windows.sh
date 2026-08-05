#!/bin/bash -
# Test: Windows NTFS root mounts at / and root_device is set.

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

"$TESTS_DIR/make-disk-windows.sh" "$d/windows.img"
rc=$?
if [ $rc -eq 77 ]; then
    echo "$0: no NTFS support, skipping"
    exit 77
fi

cat > "$d/input.json" <<EOF
{
  "disks": [{"path": "$d/windows.img", "format": "raw"}],
  "source": {"name": "test-win2k22", "type": "disk"},
  "options": {}
}
EOF

"$BIN_DIR/kc-prepare" \
    --input "$d/input.json" \
    --output "$d/prepare.json" \
    --mount-root "$MOUNT_ROOT" \
    --log-level error

check_json_field "$d/prepare.json" '.status' 'complete'
check_json_field "$d/prepare.json" '.inspect.type' 'windows'
test -n "$(jq -r '.root_device' "$d/prepare.json")"
test -d "$MOUNT_ROOT/Windows/System32/Config"

echo "PASS: test-root-windows"
