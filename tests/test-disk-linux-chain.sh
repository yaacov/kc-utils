#!/bin/bash -
# Test: Full kc-v2v.sh chain on a real Linux disk image.
# Creates a phony RHEL disk image, runs the full prepare → convert →
# finalize chain via kc-v2v.sh, and verifies all three output JSONs.

source "$(cd "$(dirname "$0")" && pwd)/functions.sh"
set -e
set -x

export LIBGUESTFS_BACKEND=direct

skip_if_skipped
requires guestfish --version
requires_jq
ensure_built
root_test
requires_loop_partitions

d=$(mktemp -d)
cleanup_fn rm -rf "$d"

# Create phony Linux disk image.
"$TESTS_DIR/make-disk-linux.sh" "$d/linux.img"

# Run full chain.
"$TESTS_DIR/kc-v2v.sh" \
    --disk "$d/linux.img" \
    --offline \
    --work-dir "$d" \
    --log-level debug

# Verify all 3 JSON outputs exist.
test -f "$d/prepare-out.json"
test -f "$d/convert-out.json"
test -f "$d/target-meta.json"

# Verify prepare completed.
check_json_field "$d/prepare-out.json" '.prepare.status' 'complete'

# Verify converter output has guestcaps.
check_json_field "$d/convert-out.json" '.convert.guestcaps.block_bus' 'virtio'
check_json_field "$d/convert-out.json" '.convert.guestcaps.net_bus' 'virtio'

# Verify guest is unmounted after finalize.
if mountpoint -q /tmp/kc-guest 2>/dev/null; then
    echo "FAIL: /tmp/kc-guest is still mounted after chain"
    exit 1
fi

echo "PASS: test-disk-linux-chain"
