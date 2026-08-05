#!/bin/bash -
# Test: EC2 service disabling in SYSTEM hive.
# Creates the "Program Files/Amazon" directory (EC2 detection trigger)
# and EC2-related service keys with Start=2 (auto-start), runs the
# converter, then verifies that the services have Start=4 (disabled).

source "$(cd "$(dirname "$0")" && pwd)/functions.sh"
set -e
set -x

skip_if_skipped
requires_linux  # binaries are Linux ELF; cannot execute on other platforms
requires command -v hivexregedit
requires_jq
requires command -v bsdtar
ensure_built
requires test -f "$UPSTREAM_TESTDATA/minimal-hive"

# Set up fake virtio-win ISO so guestcaps reports virtio.
setup_fake_virtio_drivers

# Create temp working directory.
d=$(mktemp -d)
cleanup_fn rm -rf "$d"

# Build phony Windows guest tree.
root="$d/guest"
mkdir -p "$root/Windows/System32/drivers"
mkdir -p "$root/Program Files"
make_windows_hives "$root"

# Create the Amazon directory -- this triggers EC2 cleanup (Block 15).
mkdir -p "$root/Program Files/Amazon"

# Merge EC2-related service keys with Start=2 (auto-start) into SYSTEM hive.
cat > "$d/ec2-services.reg" <<'REGEOF'
Windows Registry Editor Version 5.00

[ControlSet001\Services\AWSPVDrivers]
"Start"=dword:00000002
"Type"=dword:00000001
"ImagePath"="system32\\drivers\\awspv.sys"

[ControlSet001\Services\AmazonSSMAgent]
"Start"=dword:00000002
"Type"=dword:00000010
"ImagePath"="C:\\Program Files\\Amazon\\SSM\\amazon-ssm-agent.exe"
REGEOF
hivexregedit --merge "$root/Windows/System32/config/SYSTEM" "$d/ec2-services.reg"

# Generate prepare JSON.
make_windows_prepare_json "$root" 10 0 x86_64 "Windows Server 2022" > "$d/prepare.json"

# Run the Windows converter.
"$BIN_DIR/kc-convert-windows" \
    --prepare-data "$d/prepare.json" \
    --output "$d/output.json" \
    --mount-root "$root" \
    --offline \
    --log-level debug

# Verify pipeline succeeded.
test -f "$d/output.json"

# Verify EC2 services were disabled (Start changed from 2 to 4).
# Export each service key from the SYSTEM hive and check Start value.
for svc in AWSPVDrivers AmazonSSMAgent; do
    hivexregedit --export "$root/Windows/System32/config/SYSTEM" \
        "\\ControlSet001\\Services\\$svc" > "$d/after-${svc}.reg"

    if ! grep -q 'dword:00000004' "$d/after-${svc}.reg"; then
        echo "FAIL: $svc Start value was not changed to 4 (disabled)"
        echo "Registry export:"
        cat "$d/after-${svc}.reg"
        exit 1
    fi
done

# Verify guestcaps are correct.
check_json_field "$d/output.json" '.guestcaps.block_bus' 'virtio'
check_json_field "$d/output.json" '.guestcaps.net_bus' 'virtio'

echo "PASS: test-windows-ec2"
