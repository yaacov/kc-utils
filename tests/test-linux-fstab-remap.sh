#!/bin/bash -
# Test: Focused fstab device remapping.
# Verifies sd->vd and hd->vd remapping, UUID lines left unchanged.

source "$(cd "$(dirname "$0")" && pwd)/functions.sh"
set -e
set -x

skip_if_skipped
requires_linux  # binaries are Linux ELF; cannot execute on other platforms

requires_jq
ensure_built

# Create phony guest with mixed fstab entries
d=$(mktemp -d)
cleanup_fn rm -rf "$d"

mkdir -p "$d/etc/modprobe.d"
cat > "$d/etc/fstab" <<'EOF'
/dev/sda1 / ext4 defaults 0 1
/dev/hdb1 /data ext4 defaults 0 2
UUID=abc-123 /boot ext4 defaults 0 2
EOF

# Generate prepare data
prepare_json=$(mktemp)
cleanup_fn rm -f "$prepare_json"
make_linux_prepare_json "$d" "rhel" 9 0 "x86_64" "Test VM" "bios" > "$prepare_json"

# Run converter
output_json=$(mktemp)
cleanup_fn rm -f "$output_json"

"$BIN_DIR/kc-convert-linux" \
    --prepare-data "$prepare_json" \
    --output "$output_json" \
    --mount-root "$d" \
    --offline \
    --log-level debug

# Verify fstab remapping
grep -q '/dev/vda1' "$d/etc/fstab"
grep -q '/dev/vdb1' "$d/etc/fstab"
grep -q 'UUID=abc-123' "$d/etc/fstab"

# Verify old device names are gone
! grep -q '/dev/sda' "$d/etc/fstab"
! grep -q '/dev/hdb' "$d/etc/fstab"

echo "PASS: test-linux-fstab-remap"
