#!/bin/bash -
# Test: Full kc-v2v.sh chain on a real Windows disk image.
# Creates a phony Windows Server 2022 disk image, runs the full
# prepare → convert → finalize chain, verifies completion.

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

# Create phony Windows disk image.
"$TESTS_DIR/make-disk-windows.sh" "$d/windows.img"
rc=$?
if [ $rc -eq 77 ]; then
    echo "$0: no NTFS support, skipping"
    exit 77
fi

# Run full chain.
"$TESTS_DIR/kc-v2v.sh" \
    --disk "$d/windows.img" \
    --offline \
    --work-dir "$d" \
    --log-level debug

# Verify all 3 JSON outputs exist.
test -f "$d/prepare-out.json"
test -f "$d/convert-out.json"
test -f "$d/target-meta.json"

# Verify prepare completed.
check_json_field "$d/prepare-out.json" '.prepare.status' 'complete'

# Verify guest is unmounted after finalize.
if mountpoint -q /tmp/kc-guest 2>/dev/null; then
    echo "FAIL: /tmp/kc-guest is still mounted after chain"
    exit 1
fi

echo "PASS: test-disk-windows-chain"
