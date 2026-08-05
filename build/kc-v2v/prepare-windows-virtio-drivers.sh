#!/usr/bin/bash
# Copy virtio-win 1.9.12-4.el7 (RPM or ISO) into build/kc-v2v/vendor/ for pre–Win 8
# by-os dirs that the modern virtio-win RPM does not ship.
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
vendor_dir="${script_dir}/vendor"

rpm_name="virtio-win-1.9.12-4.el7.noarch.rpm"
iso_name="virtio-win-1.9.12.iso"
dest_rpm="${vendor_dir}/${rpm_name}"
dest_iso="${vendor_dir}/${iso_name}"

# Optional: when FORKLIFT_ROOT points at a Forklift clone, read its RPM lock file
# and check that repo's vendor/ paths.
if [ -n "${FORKLIFT_ROOT:-}" ]; then
    lock_file="${FORKLIFT_ROOT}/.konflux/virt-v2v/winlegacyiso/rpms.lock.yaml"
    if [ -f "$lock_file" ]; then
        locked="$(grep -E 'virtio-win-[0-9].*\.noarch\.rpm' "$lock_file" | head -1 | sed -E 's/.*(virtio-win-[^ ]+\.noarch\.rpm).*/\1/')"
        if [ -n "$locked" ]; then
            rpm_name="$locked"
            dest_rpm="${vendor_dir}/${rpm_name}"
        fi
    fi
fi

mkdir -p "$vendor_dir"

if [ -f "$dest_rpm" ] || [ -f "$dest_iso" ]; then
    echo "Windows virtio-win vendor artifact already present under ${vendor_dir}"
    exit 0
fi

try_copy() {
    local src="$1"
    if [ -f "$src" ]; then
        cp -a "$src" "$2"
        echo "Copied $(basename "$2") from ${src}"
        return 0
    fi
    return 1
}

if [ -n "${VIRTIO_WIN_RPM:-}" ] && try_copy "$VIRTIO_WIN_RPM" "$dest_rpm"; then
    exit 0
fi
if [ -n "${VIRTIO_WIN_ISO:-}" ] && try_copy "$VIRTIO_WIN_ISO" "$dest_iso"; then
    exit 0
fi

if [ -n "${FORKLIFT_ROOT:-}" ]; then
    for src in \
        "${FORKLIFT_ROOT}/build/kc-v2v/vendor/${rpm_name}" \
        "${FORKLIFT_ROOT}/build/virt-v2v/vendor/${rpm_name}"; do
        if try_copy "$src" "$dest_rpm"; then
            exit 0
        fi
    done

    for src in \
        "${FORKLIFT_ROOT}/build/kc-v2v/vendor/${iso_name}" \
        "${FORKLIFT_ROOT}/build/virt-v2v/vendor/${iso_name}"; do
        if try_copy "$src" "$dest_iso"; then
            exit 0
        fi
    done
fi

container_runtime_ready() {
    local runtime="$1"
    command -v "$runtime" >/dev/null 2>&1 || return 1
    "$runtime" info >/dev/null 2>&1
}

extract_iso_from_image() {
    local image="$1"
    local runtime="${CONTAINER_CMD:-podman}"
    container_runtime_ready "$runtime" || return 1
    if ! "$runtime" image exists "$image" >/dev/null 2>&1; then
        return 1
    fi
    local cid
    cid="$("$runtime" create "$image")"
    trap "$runtime rm -f '$cid' >/dev/null 2>&1 || true" RETURN
    for iso_path in /usr/local/virtio-win-legacy.iso /usr/share/virtio-win/virtio-win.iso; do
        if "$runtime" cp "${cid}:${iso_path}" "$dest_iso" 2>/dev/null; then
            echo "Extracted ${iso_name} from container image ${image}:${iso_path}"
            return 0
        fi
    done
    return 1
}

if [ -n "${FORKLIFT_VIRT_V2V_IMAGE:-}" ] && extract_iso_from_image "$FORKLIFT_VIRT_V2V_IMAGE"; then
    exit 0
fi

if [ -n "${FORKLIFT_ROOT:-}" ] && container_runtime_ready "${CONTAINER_CMD:-podman}"; then
    runtime="${CONTAINER_CMD:-podman}"
    downstream="${FORKLIFT_ROOT}/build/virt-v2v/Containerfile-downstream"
    if [ -f "$downstream" ]; then
        echo "Building Forklift winlegacyiso stage (FORKLIFT_ROOT=${FORKLIFT_ROOT})..."
        if "$runtime" build --target winlegacyiso \
            -f "$downstream" \
            -t kc-utils-winlegacyiso:local \
            "$FORKLIFT_ROOT"; then
            cid="$("$runtime" create kc-utils-winlegacyiso:local)"
            trap "$runtime rm -f '$cid' >/dev/null 2>&1 || true" RETURN
            if "$runtime" cp "${cid}:/usr/share/virtio-win/virtio-win.iso" "$dest_iso"; then
                echo "Extracted ${iso_name} from Forklift winlegacyiso build"
                exit 0
            fi
        fi
    fi
fi

cat >&2 <<EOF
WARNING: virtio-win vendor artifact not found for pre–Win 8 by-os dirs.
         Image build will succeed without 2k8/2k3/xp/vista; those guests will fail at conversion time.

To stage legacy drivers, place one of the following under build/kc-v2v/vendor/:

  ${rpm_name}
  ${iso_name}

Or set an explicit source:

  VIRTIO_WIN_RPM=/path/to/${rpm_name} make prepare-windows-virtio-drivers
  VIRTIO_WIN_ISO=/path/to/virtio-win.iso make prepare-windows-virtio-drivers

Optional: set FORKLIFT_ROOT to a Forklift clone to copy from its vendor paths or
build its winlegacyiso stage. Set FORKLIFT_VIRT_V2V_IMAGE to extract the ISO from
an existing virt-v2v container image.

There is no open/free public source for these drivers. The known-good artifact is
virtio-win 1.9.12-4.el7 (RHEL supplementary channel; requires RHEL entitlement).
See build/kc-v2v/vendor/README.md.

Set PREPARE_VIRTIO_WIN_STRICT=1 to fail instead of skipping.

EOF
if [ "${PREPARE_VIRTIO_WIN_STRICT:-0}" = "1" ]; then
    exit 1
fi
exit 0
