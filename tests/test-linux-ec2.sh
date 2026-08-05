#!/bin/bash -
# Test: EC2/AWS unconfiguration.
# Verifies EC2 service symlinks are removed and cloud-init datasource
# is disabled after conversion.

source "$(cd "$(dirname "$0")" && pwd)/functions.sh"
set -e
set -x

skip_if_skipped
requires_linux  # binaries are Linux ELF; cannot execute on other platforms

requires_jq
ensure_built

# Create phony guest with EC2 indicators
d=$(mktemp -d)
cleanup_fn rm -rf "$d"

mkdir -p "$d/etc/modprobe.d"
mkdir -p "$d/etc"
cat > "$d/etc/fstab" <<'EOF'
/dev/sda1 / ext4 defaults 0 1
EOF

# EC2 indicators
mkdir -p "$d/usr/bin"
touch "$d/usr/bin/amazon-ssm-agent"

mkdir -p "$d/etc/cloud/cloud.cfg.d"
cat > "$d/etc/cloud/cloud.cfg" <<'EOF'
datasource_list: [Ec2, None]
EOF

# EC2 service symlinks
mkdir -p "$d/etc/systemd/system/multi-user.target.wants"
touch "$d/etc/systemd/system/multi-user.target.wants/amazon-ssm-agent.service"
touch "$d/etc/systemd/system/multi-user.target.wants/amazon-cloudwatch-agent.service"

# Verify EC2 indicators exist before conversion
test -f "$d/usr/bin/amazon-ssm-agent"
test -f "$d/etc/cloud/cloud.cfg"
test -f "$d/etc/systemd/system/multi-user.target.wants/amazon-ssm-agent.service"
test -f "$d/etc/systemd/system/multi-user.target.wants/amazon-cloudwatch-agent.service"

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

# Verify EC2 service symlinks were removed
test ! -f "$d/etc/systemd/system/multi-user.target.wants/amazon-ssm-agent.service"
test ! -f "$d/etc/systemd/system/multi-user.target.wants/amazon-cloudwatch-agent.service"

# Verify cloud-init EC2 datasource was disabled
test -f "$d/etc/cloud/cloud.cfg.d/99-kc-disable-ec2.cfg"
grep -q 'datasource_list: \[None\]' "$d/etc/cloud/cloud.cfg.d/99-kc-disable-ec2.cfg"

# Verify the converter ran successfully
check_json_field "$output_json" '.guestcaps.block_bus' 'virtio'

echo "PASS: test-linux-ec2"
