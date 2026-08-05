#!/bin/bash -
# Test: Modprobe kc-virtio.conf creation and idempotency.
# Verifies the correct virtio aliases are written and re-running is safe.

source "$(cd "$(dirname "$0")" && pwd)/functions.sh"
set -e
set -x

skip_if_skipped
requires_linux  # binaries are Linux ELF; cannot execute on other platforms

requires_jq
ensure_built

# Create minimal phony guest
d=$(mktemp -d)
cleanup_fn rm -rf "$d"

mkdir -p "$d/etc/modprobe.d"
mkdir -p "$d/etc"
cat > "$d/etc/fstab" <<'EOF'
/dev/sda1 / ext4 defaults 0 1
EOF

# Generate prepare data
prepare_json=$(mktemp)
cleanup_fn rm -f "$prepare_json"
make_linux_prepare_json "$d" "rhel" 9 0 "x86_64" "Test VM" "bios" > "$prepare_json"

# Run converter (first pass)
output_json=$(mktemp)
cleanup_fn rm -f "$output_json"

"$BIN_DIR/kc-convert-linux" \
    --prepare-data "$prepare_json" \
    --output "$output_json" \
    --mount-root "$d" \
    --offline \
    --log-level debug

# Verify kc-virtio.conf exists with expected aliases
conf="$d/etc/modprobe.d/kc-virtio.conf"
test -f "$conf"
grep -q 'alias scsi_hostadapter virtio_blk' "$conf"
grep -q 'alias scsi_hostadapter1 virtio_scsi' "$conf"
grep -q 'alias eth0 virtio_net' "$conf"

# Save content for comparison
first_pass=$(cat "$conf")

# Run converter again (idempotency test)
output_json2=$(mktemp)
cleanup_fn rm -f "$output_json2"

# Re-generate prepare data (fstab was already remapped, that is fine)
prepare_json2=$(mktemp)
cleanup_fn rm -f "$prepare_json2"
make_linux_prepare_json "$d" "rhel" 9 0 "x86_64" "Test VM" "bios" > "$prepare_json2"

"$BIN_DIR/kc-convert-linux" \
    --prepare-data "$prepare_json2" \
    --output "$output_json2" \
    --mount-root "$d" \
    --offline \
    --log-level debug

# Verify config is still valid after second run
second_pass=$(cat "$conf")
grep -q 'alias scsi_hostadapter virtio_blk' "$conf"
grep -q 'alias scsi_hostadapter1 virtio_scsi' "$conf"
grep -q 'alias eth0 virtio_net' "$conf"

# Verify no duplicate entries (count alias lines)
alias_count=$(grep -c '^alias ' "$conf")
if [ "$alias_count" -ne 3 ]; then
    echo "FAIL: expected 3 alias lines after idempotent run, got $alias_count"
    exit 1
fi

echo "PASS: test-linux-modprobe"
