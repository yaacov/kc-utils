#!/bin/bash -
# Test: VirtIO driver copy and registration.
# Creates a fake virtio-win driver tree, runs the converter, and verifies that
# driver files are copied into the guest tree at Windows/Drivers/VirtIO/.
#
# Requires write access to /usr/share/virtio-win (typically root).

source "$(cd "$(dirname "$0")" && pwd)/functions.sh"
set -e
set -x

skip_if_skipped
requires_linux  # binaries are Linux ELF; cannot execute on other platforms
requires command -v hivexregedit
requires_jq
ensure_built
requires test -f "$UPSTREAM_TESTDATA/minimal-hive"

setup_fake_virtio_drivers_tree

# Create temp working directory.
d=$(mktemp -d)
cleanup_fn rm -rf "$d"

# Build phony Windows Server 2022 guest tree.
root="$d/guest"
mkdir -p "$root/Windows/System32/drivers"
mkdir -p "$root/Program Files"
make_windows_hives "$root"

# Generate prepare JSON.
make_windows_prepare_json "$root" 10 0 x86_64 "Windows Server 2022" > "$d/prepare.json"
jq -n --slurpfile p "$d/prepare.json" '{prepare: $p[0]}' > "$d/pipeline.json"

# Run the Windows converter.
"$BIN_DIR/kc-convert-windows" \
    --input "$d/pipeline.json" \
    --output "$d/output.json" \
    --mount-root "$root" \
    --offline \
    --log-level debug

# Verify pipeline succeeded.
test -f "$d/output.json"

# Verify VirtIO drivers were copied into the guest tree.
virtio_dest="$root/Windows/Drivers/VirtIO"
test -d "$virtio_dest"

for drv in viostor vioscsi netkvm; do
    test -f "$virtio_dest/$drv.sys" || {
        echo "FAIL: $drv.sys not found in $virtio_dest"
        exit 1
    }
    test -f "$virtio_dest/$drv.inf" || {
        echo "FAIL: $drv.inf not found in $virtio_dest"
        exit 1
    }
    test -f "$virtio_dest/$drv.cat" || {
        echo "FAIL: $drv.cat not found in $virtio_dest"
        exit 1
    }
done

# Verify PnPutil firstboot script was created (only when drivers found).
pnp_script="$root/Program Files/Guestfs/Firstboot/scripts/2000-install-virtio-drivers.ps1"
test -f "$pnp_script" || {
    echo "FAIL: PnPutil firstboot script not found"
    exit 1
}

# Verify PnPutil script references the driver .inf files.
grep -q 'viostor.inf' "$pnp_script"
grep -q 'vioscsi.inf' "$pnp_script"
grep -q 'netkvm.inf' "$pnp_script"

echo "PASS: test-windows-drivers"
