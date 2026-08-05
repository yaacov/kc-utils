#!/usr/bin/bash
# Stage per-Windows-version virtio-win by-os dirs missing from the modern RPM.
# Each entry in PINNED_OS_DIRS is merged individually (like rpm/el8, el9, el10).
# Expects an optional local RPM or ISO from prepare-windows-virtio-drivers.sh.
# When vendor/ is empty, exits 0 with a warning (image builds without pre–Win 8 dirs).
set -euo pipefail

dest="${1:-/usr/share/virtio-win/drivers/by-os}"
vendor_input="${2:-/tmp/virtio-win-vendor}"

# Pre–Win 8 guest handlers map 1:1 to these by-os directory names.
PINNED_OS_DIRS=(2k8 2k3 xp vista)

work="$(mktemp -d)"
cleanup() { rm -rf "$work"; }
trap cleanup EXIT

find_vendor_artifact() {
    local root="$1"
    if [ -f "$root" ]; then
        echo "$root"
        return 0
    fi
    if [ ! -d "$root" ]; then
        return 1
    fi
    local f
    for f in "$root"/*.rpm "$root"/*.iso; do
        [ -f "$f" ] || continue
        echo "$f"
        return 0
    done
    return 1
}

if ! artifact="$(find_vendor_artifact "$vendor_input")"; then
    cat >&2 <<'EOF'
WARNING: skipping pre–Win 8 virtio-win by-os dirs (2k8, 2k3, xp, vista); no .rpm or .iso under vendor/.
         Image will not support win2008, win2003, winxp, or winvista conversion.
         See build/kc-v2v/vendor/README.md to stage virtio-win 1.9.12-4.el7 at build time.
EOF
    exit 0
fi
echo "Staging per-version Windows virtio-win dirs from $(basename "$artifact")..."

iso="$work/virtio-win.iso"
case "$artifact" in
    *.iso)
        cp -a "$artifact" "$iso"
        ;;
    *.rpm)
        echo "Extracting virtio-win.iso from RPM..."
        ( cd "$work" && rpm2cpio "$(readlink -f "$artifact" || echo "$artifact")" | cpio -idmv './usr/share/virtio-win/virtio-win.iso' 2>/dev/null )
        if [ ! -f "$work/usr/share/virtio-win/virtio-win.iso" ]; then
            echo "ERROR: RPM does not contain usr/share/virtio-win/virtio-win.iso" >&2
            exit 1
        fi
        cp -a "$work/usr/share/virtio-win/virtio-win.iso" "$iso"
        ;;
    *)
        echo "ERROR: unsupported virtio-win artifact: $artifact" >&2
        exit 1
        ;;
esac

echo "Extracting virtio-win ISO..."
iso_root="$work/iso"
mkdir -p "$iso_root"
bsdtar -C "$iso_root" -xf "$iso"

declare -A by_os_arch_for_iso=(
    [amd64]=amd64
    [x86]=i386
    [i386]=i386
)

mkdir -p "$dest"

copy_tree_if_missing() {
    local src="$1"
    local rel="$2"
    local out="${dest}/${rel}"
    if [ -d "$out" ]; then
        echo "skip existing dir: ${rel}"
        return 0
    fi
    if [ ! -d "$src" ]; then
        echo "skip missing dir: ${rel}"
        return 0
    fi
    shopt -s nullglob
    local infs=("$src"/*.inf)
    shopt -u nullglob
    if [ "${#infs[@]}" -eq 0 ]; then
        echo "skip empty dir: ${rel}"
        return 0
    fi
    echo "stage dir: ${rel}"
    mkdir -p "$out"
    cp -a "$src"/. "$out/"
}

merge_from_by_os() {
    local root="$1"
    local found=0
    for iso_arch in amd64 x86 i386; do
        local by_os_arch="${by_os_arch_for_iso[$iso_arch]}"
        for osdir in "${PINNED_OS_DIRS[@]}"; do
            for candidate in \
                "${root}/by-os/${iso_arch}/${osdir}" \
                "${root}/by-os/${by_os_arch}/${osdir}" \
                "${root}/${iso_arch}/${osdir}"; do
                if [ -d "$candidate" ]; then
                    copy_tree_if_missing "$candidate" "${by_os_arch}/${osdir}"
                    found=1
                fi
            done
        done
    done
    return "$((1 - found))"
}

merge_from_by_driver() {
    local root="$1"
    local found=0
    for iso_arch in amd64 x86 i386; do
        local by_os_arch="${by_os_arch_for_iso[$iso_arch]}"
        for osdir in "${PINNED_OS_DIRS[@]}"; do
            local out="${dest}/${by_os_arch}/${osdir}"
            if [ -d "$out" ]; then
                continue
            fi
            local merged=0
            local hits=()
            while IFS= read -r -d '' hit; do
                hits+=("$hit")
            done < <(find "$root" -type d \( -path "*/${osdir}/${iso_arch}" -o -path "*/${iso_arch}/${osdir}" \) -print0 2>/dev/null)

            [ "${#hits[@]}" -gt 0 ] || continue
            mkdir -p "$out"
            for hit in "${hits[@]}"; do
                shopt -s nullglob
                local infs=("$hit"/*.inf)
                shopt -u nullglob
                [ "${#infs[@]}" -gt 0 ] || continue
                echo "stage ${hit#"$root"/} -> ${by_os_arch}/${osdir}"
                cp -a "$hit"/. "$out/"
                merged=1
                found=1
            done
            if [ "$merged" -eq 0 ]; then
                rmdir "$out" 2>/dev/null || true
            fi
        done
    done
    return "$((1 - found))"
}

for drivers_root in \
    "$iso_root/drivers" \
    "$iso_root"; do
    [ -d "$drivers_root" ] || continue
    if merge_from_by_os "$drivers_root"; then
        break
    fi
    merge_from_by_driver "$drivers_root" || true
done

echo "Per-version Windows virtio-win OS dirs staged under ${dest}"
