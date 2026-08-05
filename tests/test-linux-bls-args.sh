#!/bin/bash -
# Test: BLS (Boot Loader Specification) kernel argument modification.
# Verifies that kc-convert-linux adds console=ttyS0 and video=virtio
# to BLS entry options and removes rhgb, quiet, vga=, video=cirrus.

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

# BLS detection: needs boot/loader/entries/*.conf (no etc/default/grub to avoid grub2)
mkdir -p "$d/boot/loader/entries"
cat > "$d/boot/loader/entries/5.14.0-284.conf" <<'EOF'
title Red Hat Enterprise Linux (5.14.0-284.el9.x86_64) 9.2 (Plow)
version 5.14.0-284.el9.x86_64
linux /vmlinuz-5.14.0-284
initrd /initramfs-5.14.0-284.img
options rhgb quiet vga=normal video=cirrus root=/dev/sda1
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

# SetDefaultKernel renames the entry to sort first (prefix "0-").
bls_entry="$d/boot/loader/entries/0-5.14.0-284.conf"
if [ ! -f "$bls_entry" ]; then
    bls_entry="$d/boot/loader/entries/5.14.0-284.conf"
fi

# Verify console arg added
grep -q 'console=ttyS0' "$bls_entry"

# Verify display arg added
grep -q 'video=virtio' "$bls_entry"

# Verify removed args are gone
! grep -q 'rhgb' "$bls_entry"
! grep -q ' quiet' "$bls_entry"
! grep -qE 'vga=' "$bls_entry"
! grep -q 'video=cirrus' "$bls_entry"

# Verify the root device was remapped alongside the other kernel args.
grep -q 'root=/dev/vda1' "$bls_entry"
! grep -q 'root=/dev/sda1' "$bls_entry"

echo "PASS: test-linux-bls-args"
