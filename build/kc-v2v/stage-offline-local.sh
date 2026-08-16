#!/usr/bin/env bash
# Stage virtio-win + qemu-ga RPMs into a gitignored local tree for host-side
# testing (KC_VIRTIO_WIN / KC_PACKAGES). Uses a Fedora container so the host
# does not need dnf, virtio-win, or bsdtar.
#
# Same layout as the kc-v2v image:
#   $DEST/virtio-win/drivers/by-os/<arch>/<os>/
#   $DEST/virtio-win/guest-agent/
#   $DEST/kc-packages/rpm/el{8,9,10}/x86_64/qemu-guest-agent-*.rpm
#
# Usage (from repository root):
#   make stage-offline
#   ./build/kc-v2v/stage-offline-local.sh
#
# Then:
#   export KC_VIRTIO_WIN=$PWD/build/offline/virtio-win
#   export KC_PACKAGES=$PWD/build/offline/kc-packages
#
# Environment:
#   KC_OFFLINE_DIR       output root (default: <repo>/build/offline)
#   KC_OFFLINE_IMAGE     Fedora image (default: quay.io/fedora/fedora:44)
#   CONTAINER_RUNTIME    docker or podman (auto-detected)
#   FORCE=1              restage even if the tree already looks populated
#   VIRTIO_WIN_*         passed through to stage-virtio-win.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
DEST="${KC_OFFLINE_DIR:-$ROOT/build/offline}"
CACHE="$ROOT/build/kc-v2v/cache"
IMAGE="${KC_OFFLINE_IMAGE:-quay.io/fedora/fedora:44}"
VIRTIO="$DEST/virtio-win"
PACKAGES="$DEST/kc-packages"

if [ -n "${CONTAINER_RUNTIME:-}" ]; then
    CTR="$(command -v "$CONTAINER_RUNTIME" || true)"
else
    CTR="$(command -v docker || true)"
    if [ -z "$CTR" ]; then
        CTR="$(command -v podman || true)"
    fi
fi
if [ -z "$CTR" ] || [ ! -x "$CTR" ]; then
    echo "error: docker or podman required (set CONTAINER_RUNTIME)" >&2
    exit 1
fi

populated() {
    local byos="$VIRTIO/drivers/by-os"
    local rpms
    rpms="$(find "$PACKAGES" -type f -name 'qemu-guest-agent*.rpm' 2>/dev/null | head -1 || true)"
    [ -d "$byos" ] && [ -n "$(find "$byos" -mindepth 2 -maxdepth 2 -type d 2>/dev/null | head -1)" ] && [ -n "$rpms" ]
}

print_exports() {
    echo ""
    echo "Offline assets: $DEST"
    echo "  export KC_VIRTIO_WIN=$VIRTIO"
    echo "  export KC_PACKAGES=$PACKAGES"
}

if [ "${FORCE:-0}" != "1" ] && populated; then
    echo "Already staged under $DEST (set FORCE=1 to restage)"
    print_exports
    exit 0
fi

mkdir -p "$VIRTIO" "$PACKAGES" "$CACHE"

echo "Staging offline drivers via $CTR ($IMAGE)..."
"$CTR" run --rm \
    -e VIRTIO_WIN_MODERN_URL="${VIRTIO_WIN_MODERN_URL:-}" \
    -e VIRTIO_WIN_LEGACY_URL="${VIRTIO_WIN_LEGACY_URL:-}" \
    -e VIRTIO_WIN_CACHE_DIR=/cache \
    -e VIRTIO_WIN_DEST=/out/virtio-win \
    -e HOST_UID="$(id -u)" \
    -e HOST_GID="$(id -g)" \
    -v "$ROOT/build/kc-v2v/stage-virtio-win.sh:/tmp/stage-virtio-win.sh:ro,Z" \
    -v "$ROOT/build/kc-v2v/stage-linux-packages.sh:/tmp/stage-linux-packages.sh:ro,Z" \
    -v "$CACHE:/cache:Z" \
    -v "$DEST:/out:Z" \
    "$IMAGE" \
    bash -c '
        set -euo pipefail
        dnf install -y --setopt=install_weak_deps=False curl bsdtar
        chmod +x /tmp/stage-virtio-win.sh /tmp/stage-linux-packages.sh
        # Empty override from the host must not wipe stage-virtio-win.sh defaults.
        [ -n "${VIRTIO_WIN_MODERN_URL:-}" ] || unset VIRTIO_WIN_MODERN_URL
        [ -n "${VIRTIO_WIN_LEGACY_URL:-}" ] || unset VIRTIO_WIN_LEGACY_URL
        /tmp/stage-virtio-win.sh
        /tmp/stage-linux-packages.sh /out/kc-packages
        chown -R "${HOST_UID}:${HOST_GID}" /out
    '

print_exports
