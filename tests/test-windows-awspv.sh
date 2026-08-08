#!/bin/bash -
# Test: AWS PV driver uninstall.
# Creates fake xen driver files and the AWS PV Drivers uninstall
# registry key in SOFTWARE, runs the converter, and verifies that
# xen*.sys files are removed from the drivers directory.

source "$(cd "$(dirname "$0")" && pwd)/functions.sh"
set -e
set -x

skip_if_skipped
requires_linux  # binaries are Linux ELF; cannot execute on other platforms
requires command -v hivexregedit
requires_jq
requires_jq
ensure_built
requires test -f "$UPSTREAM_TESTDATA/minimal-hive"

setup_fake_virtio_drivers_tree

# Create temp working directory.
d=$(mktemp -d)
cleanup_fn rm -rf "$d"

# Build phony Windows guest tree.
root="$d/guest"
mkdir -p "$root/Windows/System32/drivers"
mkdir -p "$root/Program Files"
make_windows_hives "$root"

# Merge AWS PV Drivers uninstall key into the SOFTWARE hive (where the
# awspv plugin checks for it).
cat > "$d/awspv.reg" <<'REGEOF'
Windows Registry Editor Version 5.00

[HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft]

[HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows]

[HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion]

[HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall]

[HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\AWS PV Drivers]
"DisplayName"="AWS PV Drivers"
"DisplayVersion"="8.4.0"
REGEOF
hivexregedit --merge "$root/Windows/System32/config/SOFTWARE" \
    --prefix 'HKEY_LOCAL_MACHINE\SOFTWARE' "$d/awspv.reg"

# Create fake xen driver files that the awspv remove plugin should delete.
for drv in xenvbd xennet xenvif xenbus; do
    echo "fake-xen-driver" > "$root/Windows/System32/drivers/${drv}.sys"
done

# Verify fake driver files exist before conversion.
for drv in xenvbd xennet xenvif xenbus; do
    test -f "$root/Windows/System32/drivers/${drv}.sys"
done

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

# Verify xen driver files have been REMOVED by the awspv remove plugin.
for drv in xenvbd xennet xenvif xenbus; do
    if test -f "$root/Windows/System32/drivers/${drv}.sys"; then
        echo "FAIL: ${drv}.sys should have been removed but still exists"
        exit 1
    fi
done

# Verify guestcaps are still correct after uninstall.
check_json_field "$d/output.json" '.convert.guestcaps.block_bus' 'virtio'
check_json_field "$d/output.json" '.convert.guestcaps.net_bus' 'virtio'

echo "PASS: test-windows-awspv"
