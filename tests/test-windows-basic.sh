#!/bin/bash -
# Test: Basic Windows Server 2022 conversion pipeline.
# Verifies pipeline exits 0, output.json is created, and guestcaps
# fields (block_bus, net_bus, arch, machine_type) are correct.

source "$(cd "$(dirname "$0")" && pwd)/functions.sh"
set -e
set -x

skip_if_skipped
requires_linux  # binaries are Linux ELF; cannot execute on other platforms
requires command -v hivexregedit
requires_jq
ensure_built
requires test -f "$UPSTREAM_TESTDATA/minimal-hive"

# Set up fake virtio-win driver tree so guestcaps reports virtio.
setup_fake_virtio_drivers_tree

# Create temp working directory.
d=$(mktemp -d)
cleanup_fn rm -rf "$d"

# Build phony Windows Server 2022 guest tree.
root="$d/guest"
mkdir -p "$root/Windows/System32/drivers"
mkdir -p "$root/Program Files"
make_windows_hives "$root"

# Generate prepare JSON for Windows Server 2022.
make_windows_prepare_json "$root" 10 0 x86_64 "Windows Server 2022" bios > "$d/prepare.json"

# Run the Windows converter.
"$BIN_DIR/kc-convert-windows" \
    --prepare-data "$d/prepare.json" \
    --output "$d/output.json" \
    --mount-root "$root" \
    --offline \
    --log-level debug

# Verify output.json was created.
test -f "$d/output.json"

# Verify guestcaps fields.
check_json_field "$d/output.json" '.guestcaps.block_bus' 'virtio'
check_json_field "$d/output.json" '.guestcaps.net_bus' 'virtio'
check_json_field "$d/output.json" '.guestcaps.arch' 'x86_64'
check_json_field "$d/output.json" '.guestcaps.machine_type' 'q35'
check_json_field "$d/output.json" '.guestcaps.virtio_rng' 'true'
check_json_field "$d/output.json" '.guestcaps.virtio_balloon' 'true'
check_json_field "$d/output.json" '.guestcaps.virtio_socket' 'true'
check_json_field "$d/output.json" '.guestcaps.isa_pvpanic' 'true'
check_json_field "$d/output.json" '.guestcaps.virtio_1_0' 'true'
check_json_field "$d/output.json" '.guestcaps.rtc_utc' 'false'

# Compare guestcaps against golden output.
expected="$TESTS_DIR/expected/windows-basic-caps.json"
if [ -f "$expected" ]; then
    # Extract guestcaps from actual output and compare with expected.
    actual_caps=$(jq '.guestcaps' "$d/output.json")
    expected_caps=$(jq '.guestcaps' "$expected")
    if [ "$actual_caps" != "$expected_caps" ]; then
        echo "FAIL: guestcaps mismatch with golden output"
        echo "Expected: $expected_caps"
        echo "Actual:   $actual_caps"
        exit 1
    fi
fi

echo "PASS: test-windows-basic"
