#!/bin/bash -
# Test: Grub2 console and display kernel argument modification.
# Verifies that kc-convert-linux adds console=ttyS0 and video=virtio
# to GRUB_CMDLINE_LINUX and removes rhgb, quiet, vga=, video=cirrus.

source "$(cd "$(dirname "$0")" && pwd)/functions.sh"
set -e
set -x

skip_if_skipped
requires_linux  # binaries are Linux ELF; cannot execute on other platforms
requires_jq
ensure_built

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

# Grub2 detection: needs etc/default/grub and one of the grub.cfg paths
mkdir -p "$d/etc/default"
mkdir -p "$d/boot/grub2"
cat > "$d/boot/grub2/grub.cfg" <<'EOF'
menuentry 'RHEL 9.2' {
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
mkdir -p "$d/etc/modprobe.d"
install_stub_dracut "$d"

prepare_json=$(mktemp)
cleanup_fn rm -f "$prepare_json"
make_linux_prepare_json "$d" "rhel" 9 2 "x86_64" "Red Hat Enterprise Linux 9.2" "bios" > "$prepare_json"

output_json=$(mktemp)
cleanup_fn rm -f "$output_json"

"$BIN_DIR/kc-convert-linux" \
    --prepare-data "$prepare_json" \
    --output "$output_json" \
    --mount-root "$d" \
    --offline \
    --log-level debug

# Verify console arg added
grep -q 'console=ttyS0' "$d/etc/default/grub"

# Verify display arg added
grep -q 'video=virtio' "$d/etc/default/grub"

# Verify removed args are gone
! grep -q 'rhgb' "$d/etc/default/grub"
! grep -q ' quiet' "$d/etc/default/grub"
! grep -qE 'vga=' "$d/etc/default/grub"
! grep -q 'video=cirrus' "$d/etc/default/grub"

# Verify unrelated settings are preserved while root= is remapped to virtio.
grep -q 'GRUB_TIMEOUT=5' "$d/etc/default/grub"
grep -q 'GRUB_DISABLE_SUBMENU=true' "$d/etc/default/grub"
grep -q 'root=/dev/vda1' "$d/etc/default/grub"
! grep -q 'root=/dev/sda1' "$d/etc/default/grub"

echo "PASS: test-linux-grub2-args"
