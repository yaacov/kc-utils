#!/usr/bin/bash
# Merge archived virtio-win by-os directories for pre-Win 8 guests into the main tree.
# Copies SHA-1-era driver dirs (2k8, 2k3, xp, vista) without overwriting modern dirs.
set -euo pipefail

dest="${1:-/usr/share/virtio-win/drivers/by-os}"
archived_rpm_ver="1.9.12-4"
archived_base="https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/archive-virtio/virtio-win-${archived_rpm_ver}"
work="$(mktemp -d)"
cleanup() { rm -rf "$work"; }
trap cleanup EXIT

echo "Downloading archived virtio-win ${archived_rpm_ver} RPM..."
rpm_url="${archived_base}/virtio-win-${archived_rpm_ver}.el7.noarch.rpm"
curl -fsSL "$rpm_url" -o "$work/virtio-win-archive.rpm"

echo "Extracting archived virtio-win RPM..."
( cd "$work" && rpm2cpio virtio-win-archive.rpm | cpio -idmv '*/drivers/by-os/*' 2>/dev/null )

src_root="$(find "$work" -type d -path '*/drivers/by-os' | head -1)"
if [ -z "$src_root" ]; then
    echo "ERROR: archived RPM has no drivers/by-os tree" >&2
    exit 1
fi

legacy_dirs=(2k8 2k3 xp vista)
arch_dirs=(amd64 x86)

mkdir -p "$dest"
for arch in "${arch_dirs[@]}"; do
    src_arch="${src_root}/${arch}"
    [ -d "$src_arch" ] || continue
    mkdir -p "$dest/${arch}"
    for osdir in "${legacy_dirs[@]}"; do
        if [ ! -d "${src_arch}/${osdir}" ]; then
            echo "skip missing archived dir: ${arch}/${osdir}"
            continue
        fi
        if [ -d "${dest}/${arch}/${osdir}" ]; then
            echo "skip existing modern dir: ${arch}/${osdir}"
            continue
        fi
        echo "merge archived dir: ${arch}/${osdir}"
        cp -a "${src_arch}/${osdir}" "${dest}/${arch}/${osdir}"
    done
done

echo "Archived virtio-win OS dirs staged under ${dest}"
