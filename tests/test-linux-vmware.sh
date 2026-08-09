#!/bin/bash -
# Test: VMware unconfiguration.
# Verifies VMware tools service symlinks are removed after conversion.

source "$(cd "$(dirname "$0")" && pwd)/functions.sh"
set -e
set -x

skip_if_skipped
requires_linux  # binaries are Linux ELF; cannot execute on other platforms

requires_jq
ensure_built

# Create phony guest with VMware indicators
d=$(mktemp -d)
cleanup_fn rm -rf "$d"

mkdir -p "$d/etc/modprobe.d"
mkdir -p "$d/etc"
cat > "$d/etc/fstab" <<'EOF'
/dev/sda1 / ext4 defaults 0 1
EOF

# VMware indicators
mkdir -p "$d/etc/vmware-tools"
mkdir -p "$d/usr/lib/vmware-tools"

# VMware service symlinks
mkdir -p "$d/etc/systemd/system/multi-user.target.wants"
touch "$d/etc/systemd/system/multi-user.target.wants/vmtoolsd.service"

# Verify VMware indicators and service exist before conversion
test -d "$d/etc/vmware-tools"
test -d "$d/usr/lib/vmware-tools"
test -f "$d/etc/systemd/system/multi-user.target.wants/vmtoolsd.service"

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
    --input "$pipeline_json" \
    --output "$output_json" \
    --mount-root "$d" \
    --offline \
    --log-level debug

# Verify vmtoolsd.service symlink was removed
test ! -f "$d/etc/systemd/system/multi-user.target.wants/vmtoolsd.service"

# Verify the converter ran successfully
check_json_field "$output_json" '.convert.guestcaps.block_bus' 'virtio'

echo "PASS: test-linux-vmware"
