#!/bin/bash -
# Test: kc-prepare on a real Linux disk image.
# Creates a phony RHEL disk image via guestfish, runs kc-prepare to
# attach and mount it, verifies the prepare output and mounted guest,
# then runs kc-finalize to clean up.

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

# Create phony Linux disk image.
"$TESTS_DIR/make-disk-linux.sh" "$d/linux.img"

# Write prepare input JSON.
cat > "$d/input.json" <<EOF
{
  "disks": [{"path": "$d/linux.img", "format": "raw"}],
  "source": {"name": "test-rhel", "type": "disk"},
  "options": {}
}
EOF

# Run kc-prepare.
"$BIN_DIR/kc-prepare" \
    --input "$d/input.json" \
    --output "$d/prepare.json" \
    --mount-root "$MOUNT_ROOT" \
    --log-level debug

# Verify prepare output exists and completed.
test -f "$d/prepare.json"
check_json_field "$d/prepare.json" '.prepare.status' 'complete'

# Verify guest is mounted — check for files we placed in the image.
test -f "$MOUNT_ROOT/etc/os-release" || {
    echo "FAIL: guest /etc/os-release not found at $MOUNT_ROOT"
    exit 1
}
test -d "$MOUNT_ROOT/boot/grub2" || {
    echo "FAIL: guest /boot/grub2 not found at $MOUNT_ROOT"
    exit 1
}

# Clean up with kc-finalize.
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

jq --slurpfile c "$d/convert.json" '. + {convert: $c[0]}' "$d/prepare.json" > "$d/pipeline.json"

"$BIN_DIR/kc-finalize" \
    --input "$d/pipeline.json" \
    --output "$d/finalize.json" \
    --mount-root "$MOUNT_ROOT" \
    --log-level debug

# Verify finalize output exists.
test -f "$d/finalize.json"

# Verify guest is unmounted after finalize.
if mountpoint -q "$MOUNT_ROOT" 2>/dev/null; then
    echo "FAIL: $MOUNT_ROOT is still mounted after finalize"
    exit 1
fi

echo "PASS: test-disk-linux-prepare"
