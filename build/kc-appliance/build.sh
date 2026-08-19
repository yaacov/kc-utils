#!/usr/bin/env bash
# Build the kc-appliance kernel + initramfs for one or more architectures and
# extract them to bin/appliance/<arch>/{vmlinuz,initramfs.img} — the layout the
# qemu backend expects under KC_APPLIANCE_DIR (default /usr/lib/kc-utils/appliance).
#
# Usage:
#   build/kc-appliance/build.sh [arch ...]      # arch in: arm64 amd64 (Go naming)
#   ARCHES="arm64" build/kc-appliance/build.sh  # via env
#
# Env:
#   CONTAINER_RUNTIME   podman|docker (auto-detected)
#   OUT_DIR             output root (default: bin/appliance)
#
# Cross-arch builds need binfmt/qemu-user registered (podman machine on macOS has
# it; on Linux run `podman run --rm --privileged tonistiigi/binfmt --install all`).
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$repo_root"

runtime="${CONTAINER_RUNTIME:-}"
if [ -z "$runtime" ]; then
    if command -v podman >/dev/null 2>&1; then runtime=podman
    elif command -v docker >/dev/null 2>&1; then runtime=docker
    else echo "error: need podman or docker" >&2; exit 1
    fi
fi

out_dir="${OUT_DIR:-bin/appliance}"
containerfile="build/kc-appliance/Containerfile"

# Default arches: CLI args > $ARCHES > both.
if [ "$#" -gt 0 ]; then
    arches=("$@")
elif [ -n "${ARCHES:-}" ]; then
    # shellcheck disable=SC2206
    arches=(${ARCHES})
else
    arches=(arm64 amd64)
fi

# Map Go arch -> OCI platform arch (they happen to match for arm64/amd64).
platform_arch() { echo "$1"; }

for arch in "${arches[@]}"; do
    platform="linux/$(platform_arch "$arch")"
    dest="$out_dir/$arch"
    tag="kc-appliance-build:$arch"
    echo "==> building kc-appliance for $arch ($platform)"
    mkdir -p "$dest"

    # Build the assembly stage (which holds /out-vmlinuz + /out-initramfs.img),
    # then copy the artifacts out with `create`+`cp`. Unlike `--output
    # type=local`, this works with a remote daemon (podman machine on macOS,
    # docker) as well as a local one.
    "$runtime" build \
        --platform "$platform" \
        --target build \
        -f "$containerfile" \
        -t "$tag" \
        .

    cid="$("$runtime" create "$tag")"
    trap '"$runtime" rm -f "$cid" >/dev/null 2>&1 || true' EXIT
    "$runtime" cp "$cid:/out-vmlinuz" "$dest/vmlinuz"
    "$runtime" cp "$cid:/out-initramfs.img" "$dest/initramfs.img"
    "$runtime" rm -f "$cid" >/dev/null 2>&1 || true
    trap - EXIT

    if [ ! -s "$dest/vmlinuz" ] || [ ! -s "$dest/initramfs.img" ]; then
        echo "error: missing artifacts in $dest" >&2
        exit 1
    fi
    echo "==> $arch: $(ls -la "$dest/vmlinuz" "$dest/initramfs.img" | awk '{print $5, $9}' | tr '\n' ' ')"
done

echo "==> appliance artifacts written under $out_dir/"
