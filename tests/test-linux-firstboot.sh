#!/bin/bash -
# Test: Guest agent firstboot service creation.
# Verifies that kc-convert-linux creates a systemd firstboot unit
# to install qemu-guest-agent when it is not pre-installed.
# Also verifies that --offline mode skips the firstboot creation.

source "$(cd "$(dirname "$0")" && pwd)/functions.sh"
set -e
set -x

skip_if_skipped
requires_linux  # binaries are Linux ELF; cannot execute on other platforms
requires_jq
ensure_built

# --- Test 1: Online mode creates firstboot service ---
d=$(mktemp -d)
cleanup_fn rm -rf "$d"

mkdir -p "$d/etc"
cat > "$d/etc/os-release" <<'EOF'
NAME="Red Hat Enterprise Linux"
VERSION="9.2 (Plow)"
ID="rhel"
ID_LIKE="fedora"
VERSION_ID="9.2"
EOF

cat > "$d/etc/fstab" <<'EOF'
/dev/sda1 / ext4 defaults 0 1
EOF

mkdir -p "$d/etc/default"
mkdir -p "$d/boot/grub2"
cat > "$d/boot/grub2/grub.cfg" <<'EOF'
menuentry 'RHEL' { linux /vmlinuz-5.14.0-284 root=/dev/sda1; }
EOF
cat > "$d/etc/default/grub" <<'EOF'
GRUB_CMDLINE_LINUX="root=/dev/sda1"
EOF

touch "$d/boot/vmlinuz-5.14.0-284"
mkdir -p "$d/usr/lib/modules/5.14.0-284/kernel/drivers/virtio"
touch "$d/usr/lib/modules/5.14.0-284/kernel/drivers/virtio/virtio_blk.ko"
mkdir -p "$d/etc/modprobe.d"
mkdir -p "$d/etc/systemd/system"
install_stub_dracut "$d"

prepare_json=$(mktemp)
cleanup_fn rm -f "$prepare_json"
make_linux_prepare_json "$d" "rhel" 9 2 "x86_64" "Red Hat Enterprise Linux 9.2" "bios" > "$prepare_json"

pipeline_json=$(mktemp)
cleanup_fn rm -f "$pipeline_json"
jq -n --slurpfile p "$prepare_json" '{prepare: $p[0]}' > "$pipeline_json"

output_json=$(mktemp)
cleanup_fn rm -f "$output_json"

"$BIN_DIR/kc-convert-linux" \
    --input "$pipeline_json" \
    --output "$output_json" \
    --mount-root "$d" \
    --log-level debug

# Verify firstboot service was created
test -f "$d/etc/systemd/system/kc-firstboot.service"
test -f "$d/usr/local/bin/kc-firstboot.sh"
test -L "$d/etc/systemd/system/multi-user.target.wants/kc-firstboot.service"

# Verify firstboot script contains dnf install command for RHEL
grep -q 'dnf install -y qemu-guest-agent' "$d/usr/local/bin/kc-firstboot.sh"
grep -q 'systemctl enable --now qemu-guest-agent' "$d/usr/local/bin/kc-firstboot.sh"

# --- Test 2: Offline mode skips firstboot ---
d2=$(mktemp -d)
cleanup_fn rm -rf "$d2"

mkdir -p "$d2/etc"
cat > "$d2/etc/os-release" <<'EOF'
NAME="Red Hat Enterprise Linux"
VERSION="9.2 (Plow)"
ID="rhel"
ID_LIKE="fedora"
VERSION_ID="9.2"
EOF

cat > "$d2/etc/fstab" <<'EOF'
/dev/sda1 / ext4 defaults 0 1
EOF

mkdir -p "$d2/etc/default"
mkdir -p "$d2/boot/grub2"
cat > "$d2/boot/grub2/grub.cfg" <<'EOF'
menuentry 'RHEL' { linux /vmlinuz-5.14.0-284 root=/dev/sda1; }
EOF
cat > "$d2/etc/default/grub" <<'EOF'
GRUB_CMDLINE_LINUX="root=/dev/sda1"
EOF

touch "$d2/boot/vmlinuz-5.14.0-284"
mkdir -p "$d2/usr/lib/modules/5.14.0-284/kernel/drivers/virtio"
touch "$d2/usr/lib/modules/5.14.0-284/kernel/drivers/virtio/virtio_blk.ko"
mkdir -p "$d2/etc/modprobe.d"
mkdir -p "$d2/etc/systemd/system"
install_stub_dracut "$d2"

prepare_json2=$(mktemp)
cleanup_fn rm -f "$prepare_json2"
make_linux_prepare_json "$d2" "rhel" 9 2 "x86_64" "Red Hat Enterprise Linux 9.2" "bios" > "$prepare_json2"

pipeline_json2=$(mktemp)
cleanup_fn rm -f "$pipeline_json2"
jq -n --slurpfile p "$prepare_json2" '{prepare: $p[0]}' > "$pipeline_json2"

output_json2=$(mktemp)
cleanup_fn rm -f "$output_json2"

"$BIN_DIR/kc-convert-linux" \
    --input "$pipeline_json2" \
    --output "$output_json2" \
    --mount-root "$d2" \
    --offline \
    --log-level debug

# Verify firstboot service was NOT created in offline mode
if [ -f "$d2/etc/systemd/system/kc-firstboot.service" ]; then
    echo "FAIL: firstboot service should not exist in offline mode"
    exit 1
fi

echo "PASS: test-linux-firstboot"
