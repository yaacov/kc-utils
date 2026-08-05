#!/bin/bash -
# Test: Full kc-v2v.sh chain on a real Linux disk image using --guestfs mode.
# Identical to test-disk-linux-chain.sh but uses the libguestfs appliance
# (shared guestfish --listen + live FS RPC) instead of privileged mounts —
# does NOT require root, --privileged, or /dev/fuse.

source "$(cd "$(dirname "$0")" && pwd)/functions.sh"
set -e
set -x

export LIBGUESTFS_BACKEND=direct

skip_if_skipped
requires guestfish --version
requires virt-filesystems --version
requires_jq
ensure_built

d=$(mktemp -d)
cleanup_fn rm -rf "$d"

# Create phony Linux disk image.
"$TESTS_DIR/make-disk-linux.sh" "$d/linux.img"

# Run full chain with --guestfs (unprivileged appliance mode).
"$TESTS_DIR/kc-v2v.sh" \
    --disk "$d/linux.img" \
    --offline \
    --guestfs \
    --work-dir "$d" \
    --log-level debug \
    2>&1 | tee "$d/chain.log"

# Verify all 3 JSON outputs exist.
test -f "$d/prepare-out.json"
test -f "$d/convert-out.json"
test -f "$d/target-meta.json"

# Verify prepare completed.
check_json_field "$d/prepare-out.json" '.status' 'complete'

# Verify converter output has guestcaps.
check_json_field "$d/convert-out.json" '.guestcaps.block_bus' 'virtio'
check_json_field "$d/convert-out.json" '.guestcaps.net_bus' 'virtio'

# Verify guest is unmounted after finalize (direct mode leftover).
if mountpoint -q /tmp/kc-guest 2>/dev/null; then
    echo "FAIL: /tmp/kc-guest is still mounted after chain"
    exit 1
fi

# Shared listener: prepare/convert/finalize must all adopt the same guestfish PID.
pid_list=$(grep -oE 'guestfishPID=[0-9]+' "$d/chain.log" | cut -d= -f2 | sort -u)
pid_count=$(printf '%s\n' "$pid_list" | grep -cE '^[0-9]+$' || true)
if [ "$pid_count" -ne 1 ]; then
    echo "FAIL: expected exactly one shared guestfishPID across stages, got: ${pid_list:-none}"
    grep -E 'guestfishPID=|adopting shared session|shared=' "$d/chain.log" || true
    exit 1
fi
shared_pid=$pid_list
adopt_count=$(grep -cE "guestfishPID=${shared_pid}" "$d/chain.log" || true)
if [ "$adopt_count" -lt 3 ]; then
    echo "FAIL: expected guestfishPID=$shared_pid logged at least 3 times (prepare/convert/finalize), got $adopt_count"
    exit 1
fi
# slog key order may vary; require shared=true on a line that also has this PID.
if ! grep -E "guestfishPID=${shared_pid}" "$d/chain.log" | grep -q 'shared=true'; then
    echo "FAIL: guestfishPID=$shared_pid was not logged with shared=true"
    exit 1
fi

echo "PASS: test-disk-linux-guestfs (shared guestfishPID=$shared_pid)"
