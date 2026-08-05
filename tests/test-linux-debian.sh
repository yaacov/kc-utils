#!/bin/bash -
# Test: Debian 12 conversion pipeline.
# Verifies fstab remapping, virtio caps for Debian.

source "$(cd "$(dirname "$0")" && pwd)/functions.sh"
set -e
set -x

skip_if_skipped
requires_linux  # binaries are Linux ELF; cannot execute on other platforms

requires_jq
ensure_built

# Create phony Debian 12 guest
d=$(mktemp -d)
cleanup_fn rm -rf "$d"

mkdir -p "$d/etc"
cat > "$d/etc/os-release" <<'EOF'
PRETTY_NAME="Debian GNU/Linux 12 (bookworm)"
NAME="Debian GNU/Linux"
VERSION_ID="12"
VERSION="12 (bookworm)"
ID=debian
EOF

echo "12.0" > "$d/etc/debian_version"

cat > "$d/etc/fstab" <<'EOF'
/dev/sda1 / ext4 defaults 0 1
/dev/sda2 /home ext4 defaults 0 2
EOF

mkdir -p "$d/boot"
touch "$d/boot/vmlinuz-6.1.0-18-amd64"
touch "$d/boot/initrd.img-6.1.0-18-amd64"

mkdir -p "$d/usr/lib/modules/6.1.0-18-amd64/kernel/drivers/virtio"
touch "$d/usr/lib/modules/6.1.0-18-amd64/kernel/drivers/virtio/virtio_blk.ko"
touch "$d/usr/lib/modules/6.1.0-18-amd64/kernel/drivers/virtio/virtio_net.ko"
touch "$d/usr/lib/modules/6.1.0-18-amd64/kernel/drivers/virtio/virtio_scsi.ko"
touch "$d/usr/lib/modules/6.1.0-18-amd64/kernel/drivers/virtio/virtio_pci.ko"

mkdir -p "$d/etc/modprobe.d"
install_stub_dracut "$d"

# Generate prepare data
prepare_json=$(mktemp)
cleanup_fn rm -f "$prepare_json"
make_linux_prepare_json "$d" "debian" 12 0 "x86_64" "Debian GNU/Linux 12" "bios" > "$prepare_json"

# Run converter
output_json=$(mktemp)
cleanup_fn rm -f "$output_json"

"$BIN_DIR/kc-convert-linux" \
    --prepare-data "$prepare_json" \
    --output "$output_json" \
    --mount-root "$d" \
    --offline \
    --log-level debug

# Verify output JSON
check_json_field "$output_json" '.guestcaps.block_bus' 'virtio'
check_json_field "$output_json" '.guestcaps.net_bus' 'virtio'
check_json_field "$output_json" '.guestcaps.arch' 'x86_64'

# Verify fstab remapped to virtio
grep -q '/dev/vda1' "$d/etc/fstab"
grep -q '/dev/vda2' "$d/etc/fstab"
! grep -q '/dev/sda' "$d/etc/fstab"

echo "PASS: test-linux-debian"
