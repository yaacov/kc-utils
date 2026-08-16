#!/bin/bash -
# Test: Cache file cleaning.
# Verifies blkid and LVM caches are removed after conversion.

source "$(cd "$(dirname "$0")" && pwd)/functions.sh"
set -e
set -x

skip_if_skipped
requires_linux  # binaries are Linux ELF; cannot execute on other platforms

requires_jq
ensure_built

# Create phony guest with cache files
d=$(mktemp -d)
cleanup_fn rm -rf "$d"

mkdir -p "$d/etc/modprobe.d"
mkdir -p "$d/etc"
cat > "$d/etc/fstab" <<'EOF'
/dev/sda1 / ext4 defaults 0 1
EOF

# Create stale cache files
echo "fake blkid cache" > "$d/etc/blkid.tab"

mkdir -p "$d/etc/lvm/cache"
echo "fake lvm cache" > "$d/etc/lvm/cache/.cache"

mkdir -p "$d/run/blkid"
echo "fake run blkid cache" > "$d/run/blkid/blkid.tab"

# Verify cache files exist before conversion
test -f "$d/etc/blkid.tab"
test -f "$d/etc/lvm/cache/.cache"
test -f "$d/run/blkid/blkid.tab"

# Generate prepare data
prepare_json=$(mktemp)
cleanup_fn rm -f "$prepare_json"
make_linux_prepare_json "$d" "rhel" 9 0 "x86_64" "Test VM" "bios" > "$prepare_json"

pipeline_json=$(mktemp)
cleanup_fn rm -f "$pipeline_json"
jq -n --slurpfile p "$prepare_json" '{prepare: $p[0]}' > "$pipeline_json"

# Run converter
output_json=$(mktemp)
cleanup_fn rm -f "$output_json"

"$BIN_DIR/kc-convert-linux" \
    --backend direct \
    --input "$pipeline_json" \
    --output "$output_json" \
    --mount-root "$d" \
    --offline \
    --log-level debug

# Verify all cache files are removed
test ! -f "$d/etc/blkid.tab"
test ! -f "$d/etc/lvm/cache/.cache"
test ! -f "$d/run/blkid/blkid.tab"

echo "PASS: test-linux-cache-clean"
