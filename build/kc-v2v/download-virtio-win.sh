#!/usr/bin/bash
# Download and extract virtio-win drivers from public Fedora People ISOs.
#
# Two ISOs are fetched:
#   - Modern (0.1.285): Win7+ / 2008R2+ drivers and qemu-ga MSIs
#   - Legacy (0.1.160): pre-Win 8 drivers (2k8, 2k3, xp, vista)
#
# All URLs are configurable via environment variables for version bumps or mirrors.
# Set VIRTIO_WIN_SKIP_DOWNLOAD=1 to skip download (for pre-populated trees or airgap).
# Set VIRTIO_WIN_CACHE_DIR to a local directory to cache downloaded ISOs across builds.
set -euo pipefail

DEST="${VIRTIO_WIN_DEST:-/usr/share/virtio-win}"
MODERN_URL="${VIRTIO_WIN_MODERN_URL:-https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/archive-virtio/virtio-win-0.1.285-1/virtio-win-0.1.285.iso}"
LEGACY_URL="${VIRTIO_WIN_LEGACY_URL:-https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/archive-virtio/virtio-win-0.1.160-1/virtio-win-0.1.160.iso}"
CACHE_DIR="${VIRTIO_WIN_CACHE_DIR:-}"

LEGACY_OS_DIRS=(2k8 2k3 xp vista)

if [ "${VIRTIO_WIN_SKIP_DOWNLOAD:-0}" = "1" ]; then
    echo "VIRTIO_WIN_SKIP_DOWNLOAD=1; skipping download (expecting pre-populated tree at ${DEST})"
    exit 0
fi

work="$(mktemp -d)"
cleanup() { rm -rf "$work"; }
trap cleanup EXIT

download() {
    local url="$1" dest="$2" filename
    filename="$(basename "$url")"
    if [ -n "$CACHE_DIR" ] && [ -f "$CACHE_DIR/$filename" ]; then
        echo "Using cached $filename from $CACHE_DIR"
        cp "$CACHE_DIR/$filename" "$dest"
        return 0
    fi
    echo "Downloading $filename..."
    curl -fL --retry 3 --retry-delay 5 -o "$dest" "$url"
    if [ -n "$CACHE_DIR" ]; then
        mkdir -p "$CACHE_DIR"
        cp "$dest" "$CACHE_DIR/$filename"
    fi
}

mkdir -p "$DEST/drivers/by-os" "$DEST/guest-agent"

# --- Modern ISO -----------------------------------------------------------
modern_iso="$work/modern.iso"
download "$MODERN_URL" "$modern_iso"

echo "Extracting modern ISO drivers..."
modern_root="$work/modern"
mkdir -p "$modern_root"
bsdtar -C "$modern_root" -xf "$modern_iso"

# Copy by-os driver tree (layout varies: some ISOs use a top-level by-os/, others nest under drivers/)
for candidate in "$modern_root/by-os" "$modern_root/drivers/by-os"; do
    if [ -d "$candidate" ]; then
        cp -a "$candidate"/. "$DEST/drivers/by-os/"
        echo "Staged modern by-os tree from $(basename "$(dirname "$candidate")")/by-os"
        break
    fi
done

# Copy guest-agent MSIs
for ga_dir in "$modern_root/guest-agent" "$modern_root/drivers/guest-agent"; do
    if [ -d "$ga_dir" ]; then
        cp -a "$ga_dir"/. "$DEST/guest-agent/"
        echo "Staged guest-agent MSIs"
        break
    fi
done

rm -f "$modern_iso"

# --- Legacy ISO (only missing pre-Win 8 dirs) -----------------------------
legacy_iso="$work/legacy.iso"
download "$LEGACY_URL" "$legacy_iso"

echo "Extracting legacy ISO for pre-Win 8 dirs..."
legacy_root="$work/legacy"
mkdir -p "$legacy_root"
bsdtar -C "$legacy_root" -xf "$legacy_iso"

declare -A arch_map=( [amd64]=amd64 [x86]=i386 [i386]=i386 )

for iso_arch in amd64 x86 i386; do
    dest_arch="${arch_map[$iso_arch]}"
    for osdir in "${LEGACY_OS_DIRS[@]}"; do
        out="$DEST/drivers/by-os/${dest_arch}/${osdir}"
        [ -d "$out" ] && continue

        # Try common ISO layouts
        for candidate in \
            "$legacy_root/by-os/${iso_arch}/${osdir}" \
            "$legacy_root/drivers/by-os/${iso_arch}/${osdir}" \
            "$legacy_root/${iso_arch}/${osdir}"; do
            if [ -d "$candidate" ]; then
                shopt -s nullglob
                infs=("$candidate"/*.inf)
                shopt -u nullglob
                [ "${#infs[@]}" -gt 0 ] || continue
                mkdir -p "$out"
                cp -a "$candidate"/. "$out/"
                echo "Staged legacy dir: ${dest_arch}/${osdir}"
                break
            fi
        done
    done
done

rm -f "$legacy_iso"

echo "virtio-win drivers staged under ${DEST}"
echo "  Modern: $(basename "$MODERN_URL")"
echo "  Legacy: $(basename "$LEGACY_URL")"
