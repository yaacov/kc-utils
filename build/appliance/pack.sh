#!/usr/bin/bash
# Pack vmlinuz + initramfs.img for the QEMU appliance.
# Virtio-win and qemu-ga RPMs are host-side; they are not packed here.
set -euo pipefail

OUT=/out
ROOT=/initramfs
mkdir -p "$OUT" "$ROOT"

KVER="$(ls /usr/lib/modules | head -1)"
if [ -z "$KVER" ]; then
    echo "error: no kernel modules under /usr/lib/modules" >&2
    exit 1
fi

VMLINUZ=""
if [ -f "/usr/lib/modules/${KVER}/vmlinuz" ]; then
    VMLINUZ="/usr/lib/modules/${KVER}/vmlinuz"
else
    VMLINUZ="$(ls /boot/vmlinuz-* 2>/dev/null | head -1 || true)"
fi
if [ -z "$VMLINUZ" ] || [ ! -f "$VMLINUZ" ]; then
    echo "error: vmlinuz not found for ${KVER}" >&2
    exit 1
fi
cp "$VMLINUZ" "$OUT/vmlinuz"

echo "Building initramfs root (kernel ${KVER})..."
dnf -y --installroot="$ROOT" --releasever=44 \
    --setopt=install_weak_deps=False --setopt=reposdir=/etc/yum.repos.d --nogpgcheck \
    install \
        bash coreutils util-linux kmod \
        lvm2 device-mapper cryptsetup clevis clevis-luks \
        e2fsprogs xfsprogs btrfs-progs ntfs-3g ntfsprogs \
        hivex perl-hivex \
        glibc

cp /kc-agent "$ROOT/kc-agent"
chmod +x "$ROOT/kc-agent"

mkdir -p "$ROOT/usr/lib/modules/${KVER}" "$ROOT/dev" "$ROOT/proc" "$ROOT/sys" "$ROOT/tmp" "$ROOT/mnt" "$ROOT/run"

copy_module_tree() {
    local dest="$ROOT/usr/lib/modules/${KVER}"
    local src="/usr/lib/modules/${KVER}"
    mkdir -p "$dest"
    local mods=(
        virtio virtio_ring virtio_pci virtio_pci_legacy_dev virtio_pci_modern_dev
        virtio_mmio virtio_blk virtio_console virtio_scsi virtio_net
        ext4 jbd2 mbcache crc32c_generic libcrc32c
        xfs
        btrfs xor raid6_pq zstd_compress
        fat vfat nls_cp437 nls_iso8859-1 nls_utf8
        ntfs3
        dm_mod dm-mod dm_crypt dm-crypt
        sha256_generic aes_generic xts essiv
    )
    local name line f rel
    copy_one() {
        f="$1"
        [ -f "$f" ] || return 0
        rel="${f#"$src"/}"
        mkdir -p "$dest/$(dirname "$rel")"
        cp -a "$f" "$dest/$rel"
    }
    for name in "${mods[@]}"; do
        while IFS= read -r line; do
            case "$line" in
                insmod\ *) copy_one "${line#insmod }" ;;
            esac
        done < <(modprobe -S "$KVER" --show-depends "$name" 2>/dev/null || true)
        while IFS= read -r f; do
            copy_one "$f"
        done < <(find "$src" \( -name "${name}.ko" -o -name "${name}.ko.xz" -o -name "${name}.ko.zst" -o -name "${name}.ko.gz" \) -print 2>/dev/null || true)
    done
    for meta in modules.dep modules.dep.bin modules.alias modules.alias.bin \
        modules.builtin modules.builtin.alias.bin modules.builtin.bin \
        modules.builtin.modinfo modules.order modules.symbols modules.symbols.bin \
        modules.devname; do
        if [ -e "$src/$meta" ]; then
            cp -a "$src/$meta" "$dest/$meta"
        fi
    done
    if command -v depmod >/dev/null; then
        depmod -b "$ROOT" "$KVER" || true
    fi
}

copy_module_tree

mkdir -p "$ROOT/etc"
touch "$ROOT/etc/fstab"

echo "Stripping docs, locales, and caches..."
rm -rf \
    "$ROOT/usr/share/doc" \
    "$ROOT/usr/share/man" \
    "$ROOT/usr/share/info" \
    "$ROOT/usr/share/licenses" \
    "$ROOT/usr/share/locale" \
    "$ROOT/usr/share/i18n" \
    "$ROOT/usr/lib/locale" \
    "$ROOT/var/cache" \
    "$ROOT/var/log" \
    "$ROOT/var/lib/dnf" \
    "$ROOT/usr/lib/sysimage" \
    "$ROOT/boot" \
    "$ROOT/home" \
    "$ROOT/root" \
    "$ROOT/usr/src" 2>/dev/null || true
find "$ROOT" -name '*.a' -delete 2>/dev/null || true
# Keep perl for hivexregedit; drop dnf/rpm from the ramdisk if present.
rm -rf "$ROOT/usr/bin/dnf"* "$ROOT/usr/bin/yum"* "$ROOT/etc/dnf" "$ROOT/etc/yum.repos.d" 2>/dev/null || true

echo "Packing initramfs.img (xz)..."
( cd "$ROOT" && find . -print0 | cpio --null -o -H newc ) | xz -T0 -9 --check=crc32 > "$OUT/initramfs.img"

echo "Appliance artifacts:"
ls -lh "$OUT"
