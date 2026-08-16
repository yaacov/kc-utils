#!/bin/bash -
# Test: kc-prepare on a real Windows disk image.
# Creates a phony Windows Server 2022 disk image with NTFS + registry
# hives, runs kc-prepare to attach and mount it, verifies the guest
# is accessible, then runs kc-finalize to clean up.

source "$(cd "$(dirname "$0")" && pwd)/functions.sh"
set -e
set -x

skip_if_skipped
requires guestfish --version
requires command -v hivexregedit
requires_jq
ensure_built
root_test
requires_loop_partitions

d=$(mktemp -d)
cleanup_fn rm -rf "$d"

MOUNT_ROOT="$d/mnt"
mkdir -p "$MOUNT_ROOT"

# Create phony Windows disk image.
"$TESTS_DIR/make-disk-windows.sh" "$d/windows.img"
rc=$?
if [ $rc -eq 77 ]; then
    echo "$0: no NTFS support, skipping"
    exit 77
fi

# Write prepare input JSON.
cat > "$d/input.json" <<EOF
{
  "disks": [{"path": "$d/windows.img", "format": "raw"}],
  "source": {"name": "test-win2k22", "type": "disk"},
  "options": {}
}
EOF

# Run kc-prepare.
"$BIN_DIR/kc-prepare" \
    --backend direct \
    --input "$d/input.json" \
    --output "$d/prepare.json" \
    --mount-root "$MOUNT_ROOT" \
    --log-level debug

# Verify prepare completed.
test -f "$d/prepare.json"
check_json_field "$d/prepare.json" '.prepare.status' 'complete'

# Verify guest is mounted and Windows files are accessible at the root mount.
test -d "$MOUNT_ROOT/Windows/System32/Config" || {
    echo "FAIL: Windows registry files not found under $MOUNT_ROOT"
    exit 1
}

# Clean up mounts with kc-finalize.
cat > "$d/convert.json" <<EOF
{
  "guestcaps": {
    "block_bus": "virtio",
    "net_bus": "virtio",
    "virtio_rng": false,
    "virtio_balloon": false,
    "virtio_socket": false,
    "isa_pvpanic": false,
    "machine_type": "q35",
    "arch": "x86_64",
    "virtio_1_0": true,
    "rtc_utc": false
  }
}
EOF

jq --slurpfile c "$d/convert.json" '. + {convert: $c[0]}' "$d/prepare.json" > "$d/pipeline.json"

"$BIN_DIR/kc-finalize" \
    --backend direct \
    --input "$d/pipeline.json" \
    --output "$d/finalize.json" \
    --mount-root "$MOUNT_ROOT" \
    --log-level debug

# Verify finalize completed.
test -f "$d/finalize.json"

# Verify guest is unmounted.
if mountpoint -q "$MOUNT_ROOT" 2>/dev/null; then
    echo "FAIL: $MOUNT_ROOT is still mounted after finalize"
    exit 1
fi

echo "PASS: test-disk-windows-prepare"
