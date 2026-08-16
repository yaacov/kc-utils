#!/bin/bash -
# Test: Full RHEL 9 conversion pipeline.
# Verifies fstab remapping, virtio caps, modprobe config.

source "$(cd "$(dirname "$0")" && pwd)/functions.sh"
set -e
set -x

skip_if_skipped
requires_linux  # binaries are Linux ELF; cannot execute on other platforms

requires_jq
ensure_built

# Create phony RHEL 9 guest
d=$(mktemp -d)
cleanup_fn rm -rf "$d"

mkdir -p "$d/etc"
cat > "$d/etc/os-release" <<'EOF'
NAME="Red Hat Enterprise Linux"
VERSION="9.2 (Plow)"
ID="rhel"
ID_LIKE="fedora"
VERSION_ID="9.2"
PRETTY_NAME="Red Hat Enterprise Linux 9.2 (Plow)"
EOF

cat > "$d/etc/fstab" <<'EOF'
/dev/sda1 / ext4 defaults 0 1
/dev/sda2 /home ext4 defaults 0 2
EOF

mkdir -p "$d/etc/default"
mkdir -p "$d/boot/grub2"
cat > "$d/boot/grub2/grub.cfg" <<'EOF'
menuentry 'Red Hat Enterprise Linux (5.14.0-284.el9.x86_64) 9.2 (Plow)' {
    linux /vmlinuz-5.14.0-284 root=/dev/sda1
    initrd /initramfs-5.14.0-284.img
}
EOF

cat > "$d/etc/default/grub" <<'EOF'
GRUB_TIMEOUT=5
GRUB_CMDLINE_LINUX="rhgb quiet vga=normal video=cirrus root=/dev/sda1"
GRUB_DISABLE_SUBMENU=true
EOF

touch "$d/boot/vmlinuz-5.14.0-284"
touch "$d/boot/initramfs-5.14.0-284.img"

mkdir -p "$d/usr/lib/modules/5.14.0-284/kernel/drivers/virtio"
touch "$d/usr/lib/modules/5.14.0-284/kernel/drivers/virtio/virtio_blk.ko"
touch "$d/usr/lib/modules/5.14.0-284/kernel/drivers/virtio/virtio_net.ko"
touch "$d/usr/lib/modules/5.14.0-284/kernel/drivers/virtio/virtio_scsi.ko"
touch "$d/usr/lib/modules/5.14.0-284/kernel/drivers/virtio/virtio_pci.ko"

mkdir -p "$d/etc/modprobe.d"
install_stub_dracut "$d"

# Generate prepare data
prepare_json=$(mktemp)
cleanup_fn rm -f "$prepare_json"
make_linux_prepare_json "$d" "rhel" 9 2 "x86_64" "Red Hat Enterprise Linux 9.2" "bios" > "$prepare_json"

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

# Verify output JSON
check_json_field "$output_json" '.convert.guestcaps.block_bus' 'virtio'
check_json_field "$output_json" '.convert.guestcaps.net_bus' 'virtio'
check_json_field "$output_json" '.convert.guestcaps.arch' 'x86_64'
check_json_field "$output_json" '.convert.guestcaps.rtc_utc' 'true'

# Verify fstab remapped to virtio
grep -q '/dev/vda1' "$d/etc/fstab"
grep -q '/dev/vda2' "$d/etc/fstab"
! grep -q '/dev/sda' "$d/etc/fstab"

# Verify modprobe config exists with virtio aliases
test -f "$d/etc/modprobe.d/kc-virtio.conf"
grep -q 'alias scsi_hostadapter virtio_blk' "$d/etc/modprobe.d/kc-virtio.conf"
grep -q 'alias eth0 virtio_net' "$d/etc/modprobe.d/kc-virtio.conf"

# Verify kernel args were modified in etc/default/grub
grep -q 'console=ttyS0' "$d/etc/default/grub"
grep -q 'video=virtio' "$d/etc/default/grub"
! grep -q 'rhgb' "$d/etc/default/grub"
! grep -q 'quiet' "$d/etc/default/grub"
! grep -qE 'vga=' "$d/etc/default/grub"
! grep -q 'video=cirrus' "$d/etc/default/grub"

echo "PASS: test-linux-rhel"
