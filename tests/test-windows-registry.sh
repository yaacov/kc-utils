#!/bin/bash -
# Test: Windows registry modifications — DevicePath and CrashControl.
# Verifies that kc-convert-windows:
# - Updates SOFTWARE hive DevicePath to include VirtIO driver path
# - Sets SYSTEM hive CrashControl\AutoReboot to 0

source "$(cd "$(dirname "$0")" && pwd)/functions.sh"
set -e
set -x

skip_if_skipped
requires_linux  # binaries are Linux ELF; cannot execute on other platforms
requires command -v hivexregedit
requires command -v hivexget
requires_jq
ensure_built
requires test -f "$UPSTREAM_TESTDATA/minimal-hive"
requires command -v bsdtar

setup_fake_virtio_drivers

d=$(mktemp -d)
cleanup_fn rm -rf "$d"

root="$d/guest"
mkdir -p "$root/Windows/System32/drivers"
mkdir -p "$root/Program Files"
make_windows_hives "$root"

make_windows_prepare_json "$root" 10 0 x86_64 "Windows Server 2022" bios > "$d/prepare.json"

"$BIN_DIR/kc-convert-windows" \
    --prepare-data "$d/prepare.json" \
    --output "$d/output.json" \
    --mount-root "$root" \
    --offline \
    --log-level debug

test -f "$d/output.json"

# Verify DevicePath in SOFTWARE hive includes VirtIO path
software_hive="$root/Windows/System32/config/SOFTWARE"
device_path=$(hivexget "$software_hive" \
    'Microsoft\Windows\CurrentVersion' DevicePath 2>/dev/null) || true
if [ -z "$device_path" ]; then
    echo "FAIL: could not read DevicePath from SOFTWARE hive"
    exit 1
fi
echo "DevicePath = $device_path"
echo "$device_path" | grep -qF 'Drivers\VirtIO' || {
    echo "FAIL: DevicePath does not contain VirtIO driver path"
    exit 1
}

# Verify CrashControl AutoReboot in SYSTEM hive is set to 0
system_hive="$root/Windows/System32/config/SYSTEM"
auto_reboot=$(hivexget "$system_hive" \
    'ControlSet001\Control\CrashControl' AutoReboot 2>/dev/null) || true
if [ -z "$auto_reboot" ]; then
    echo "FAIL: could not read CrashControl\\AutoReboot from SYSTEM hive"
    exit 1
fi
echo "AutoReboot = $auto_reboot"
# hivexget outputs DWORD as "dword:00000000"
echo "$auto_reboot" | grep -qE '(^0$|dword:0+$)' || {
    echo "FAIL: AutoReboot should be 0, got: $auto_reboot"
    exit 1
}

echo "PASS: test-windows-registry"
