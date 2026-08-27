#!/usr/bin/env bash
# Download and stage virtio-win drivers from public Fedora People ISOs.
#
# Usage: stage-virtio-win.sh [DEST]
#   DEST  virtio-win tree (default: /usr/share/virtio-win)
#         drivers → $DEST/drivers/by-os/  qemu-ga → $DEST/guest-agent/
#
# Two ISOs are fetched:
#   - Modern (0.1.285): Win7+ / 2008R2+ drivers and qemu-ga MSIs
#   - Legacy (0.1.160): pre-Win 8 drivers (2k8, 2k3, xp, vista)
#
# Optional environment:
#   VIRTIO_WIN_MODERN_URL    override modern ISO URL (version bumps / mirrors)
#   VIRTIO_WIN_LEGACY_URL    override legacy ISO URL
#   VIRTIO_WIN_CACHE_DIR     cache dir for downloaded ISOs (skip curl if present)
#   VIRTIO_WIN_SKIP_DOWNLOAD set to 1 to skip all downloads (pre-populated tree)
set -euo pipefail

DEST="${1:-/usr/share/virtio-win}"
MODERN_URL="${VIRTIO_WIN_MODERN_URL:-https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/archive-virtio/virtio-win-0.1.285-1/virtio-win-0.1.285.iso}"
LEGACY_URL="${VIRTIO_WIN_LEGACY_URL:-https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/archive-virtio/virtio-win-0.1.160-1/virtio-win-0.1.160.iso}"
CACHE_DIR="${VIRTIO_WIN_CACHE_DIR:-}"
LEGACY_OS_DIRS=(2k8 2k3 xp vista)

if [ "${VIRTIO_WIN_SKIP_DOWNLOAD:-0}" = "1" ]; then
    echo "VIRTIO_WIN_SKIP_DOWNLOAD=1; skipping (expecting pre-populated tree at ${DEST})"
    exit 0
fi

work="$(mktemp -d)"
cleanup() { rm -rf "$work"; }
trap cleanup EXIT

# ── Phase 1: Fetch ──────────────────────────────────────────────────
# Download ISOs (or use cache). Pure I/O, no logic.

fetch() {
    local url="$1" dest="$2"
    local filename
    filename="$(basename "$url")"

    if [ -n "$CACHE_DIR" ] && [ -f "$CACHE_DIR/$filename" ]; then
        echo "  cached: $filename"
        cp "$CACHE_DIR/$filename" "$dest"
        return 0
    fi

    echo "  downloading: $filename"
    curl -fL --retry 3 --retry-delay 5 -o "$dest" "$url"

    if [ -n "$CACHE_DIR" ]; then
        mkdir -p "$CACHE_DIR"
        cp "$dest" "$CACHE_DIR/$filename"
    fi
}

echo "Phase 1: Fetch ISOs"
modern_iso="$work/modern.iso"
legacy_iso="$work/legacy.iso"
fetch "$MODERN_URL" "$modern_iso"
fetch "$LEGACY_URL" "$legacy_iso"

# ── Phase 2: Extract ────────────────────────────────────────────────
# Unpack ISOs into temp dirs. No layout decisions here.

echo "Phase 2: Extract ISOs"
modern_root="$work/modern"
legacy_root="$work/legacy"
mkdir -p "$modern_root" "$legacy_root"
bsdtar -C "$modern_root" -xf "$modern_iso"
bsdtar -C "$legacy_root" -xf "$legacy_iso"
rm -f "$modern_iso" "$legacy_iso"

# ── Phase 3: Stage ──────────────────────────────────────────────────
# Reorganize from upstream layout into runtime-expected layout.
#
# Upstream ISO layout (per-driver):
#   <Driver>/<os-version>/<arch>/<files>
#
# Runtime layout (per-OS, what directory.go expects):
#   by-os/<arch>/<os-version>/<files>

echo "Phase 3: Stage drivers"
mkdir -p "$DEST/drivers/by-os" "$DEST/guest-agent"

stage_drivers() {
    local root="$1" dest_base="$2" filter="$3"
    local count=0

    # Shortcut: if ISO already has a by-os/ tree, use it directly
    for candidate in "$root/by-os" "$root/drivers/by-os"; do
        if [ -d "$candidate" ]; then
            echo "  found pre-built by-os tree"
            cp -a "$candidate"/. "$dest_base/"
            return 0
        fi
    done

    # Transpose per-driver layout into per-OS layout
    for driver_dir in "$root"/*/; do
        [ -d "$driver_dir" ] || continue
        local dname
        dname="$(basename "$driver_dir")"
        case "$dname" in
            guest-agent|cert|docs|installer|tools|NetKVM_Documentation) continue ;;
        esac

        for os_dir in "$driver_dir"*/; do
            [ -d "$os_dir" ] || continue
            local os_name
            os_name="$(basename "$os_dir")"

            for arch_dir in "$os_dir"*/; do
                [ -d "$arch_dir" ] || continue
                local arch_name
                arch_name="$(basename "$arch_dir")"

                # Normalize arch name
                local dest_arch="$arch_name"
                case "$arch_name" in
                    x86) dest_arch="i386" ;;
                esac

                # Apply OS filter (for legacy ISO: only stage listed dirs)
                if [ -n "$filter" ]; then
                    local match=0
                    for f in $filter; do
                        [ "$os_name" = "$f" ] && match=1 && break
                    done
                    [ "$match" -eq 0 ] && continue
                fi

                # Skip if already staged (legacy doesn't overwrite modern)
                local out="$dest_base/${dest_arch}/${os_name}"
                if [ -d "$out" ] && [ -n "$filter" ]; then
                    continue
                fi

                # Only stage dirs that contain driver files
                shopt -s nullglob
                local infs=("$arch_dir"/*.inf)
                shopt -u nullglob
                [ "${#infs[@]}" -gt 0 ] || continue

                mkdir -p "$out"
                cp -a "$arch_dir"/. "$out/"
                count=$((count + 1))
            done
        done
    done

    echo "  staged $count arch/os dirs"
    [ "$count" -gt 0 ]
}

# Stage modern drivers (all OS versions)
echo "  modern ISO → by-os tree"
stage_drivers "$modern_root" "$DEST/drivers/by-os" "" || \
    echo "  WARNING: no drivers found in modern ISO"

# Stage legacy drivers (only pre-Win 8 dirs missing from modern)
echo "  legacy ISO → pre-Win 8 dirs only"
stage_drivers "$legacy_root" "$DEST/drivers/by-os" "${LEGACY_OS_DIRS[*]}" || \
    echo "  WARNING: no pre-Win 8 dirs found in legacy ISO"

# Stage guest-agent MSIs from modern ISO
if [ -d "$modern_root/guest-agent" ]; then
    cp -a "$modern_root/guest-agent"/. "$DEST/guest-agent/"
    echo "  staged guest-agent MSIs"
fi

# ── Summary ─────────────────────────────────────────────────────────
echo ""
echo "Done. Drivers staged under ${DEST}/drivers/by-os/"
echo "  Modern: $(basename "$MODERN_URL")"
echo "  Legacy: $(basename "$LEGACY_URL")"
echo "  OS dirs: $(find "$DEST/drivers/by-os" -mindepth 2 -maxdepth 2 -type d | wc -l)"
