#!/bin/bash -
# Test: UEFI disk image — prepare and finalize on GPT + ESP layout.
# Creates a UEFI RHEL disk image (GPT with vfat ESP partition + ext4
# root), runs kc-prepare, verifies the pipeline completes and the
# guest is mounted, then runs kc-finalize to clean up.

source "$(cd "$(dirname "$0")" && pwd)/functions.sh"
set -e
set -x

skip_if_skipped
requires guestfish --version
requires_jq
ensure_built
root_test
requires_loop_partitions

d=$(mktemp -d)
cleanup_fn rm -rf "$d"

MOUNT_ROOT="$d/mnt"
mkdir -p "$MOUNT_ROOT"

# Create UEFI Linux disk image.
"$TESTS_DIR/make-disk-linux-uefi.sh" "$d/uefi.img"

# Write prepare input JSON.
cat > "$d/input.json" <<EOF
{
  "disks": [{"path": "$d/uefi.img", "format": "raw"}],
  "source": {"name": "test-uefi", "type": "disk"},
  "options": {}
}
EOF

# Run kc-prepare.
"$BIN_DIR/kc-prepare" \
    --input "$d/input.json" \
    --output "$d/prepare.json" \
    --mount-root "$MOUNT_ROOT" \
    --log-level debug

# Verify prepare completed.
test -f "$d/prepare.json"
check_json_field "$d/prepare.json" '.status' 'complete'

# Verify guest is mounted.
test -f "$MOUNT_ROOT/etc/os-release" || {
    echo "FAIL: guest /etc/os-release not found at $MOUNT_ROOT"
    exit 1
}

# Clean up mounts.
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
    "rtc_utc": true
  }
}
EOF

"$BIN_DIR/kc-finalize" \
    --prepare-data "$d/prepare.json" \
    --convert-data "$d/convert.json" \
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

echo "PASS: test-disk-uefi"
