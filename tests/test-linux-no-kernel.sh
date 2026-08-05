#!/bin/bash -
# Test: Pipeline gracefully handles no kernel scanner results.
# When no RPM db or dpkg status is found, the pipeline defaults to
# safe virtio caps (block_bus=virtio, net_bus=virtio).

source "$(cd "$(dirname "$0")" && pwd)/functions.sh"
set -e
set -x

skip_if_skipped
requires_linux  # binaries are Linux ELF; cannot execute on other platforms

requires_jq
ensure_built

# Create minimal phony guest with no kernel scanner data
# No RPM database, no dpkg status -- kernel scanners will find nothing
d=$(mktemp -d)
cleanup_fn rm -rf "$d"

mkdir -p "$d/etc/modprobe.d"
mkdir -p "$d/etc"
cat > "$d/etc/fstab" <<'EOF'
/dev/sda1 / ext4 defaults 0 1
EOF

# Deliberately do NOT create:
# - /var/lib/rpm/ (RPM database)
# - /var/lib/dpkg/status (dpkg database)
# - any kernel modules
# This ensures selectedKernel remains nil

# Generate prepare data
prepare_json=$(mktemp)
cleanup_fn rm -f "$prepare_json"
make_linux_prepare_json "$d" "rhel" 9 0 "x86_64" "Test VM" "bios" > "$prepare_json"

# Run converter
output_json=$(mktemp)
cleanup_fn rm -f "$output_json"

"$BIN_DIR/kc-convert-linux" \
    --prepare-data "$prepare_json" \
    --output "$output_json" \
    --mount-root "$d" \
    --offline \
    --log-level debug

# When selectedKernel is nil, the pipeline defaults to virtio
check_json_field "$output_json" '.guestcaps.block_bus' 'virtio'
check_json_field "$output_json" '.guestcaps.net_bus' 'virtio'
check_json_field "$output_json" '.guestcaps.virtio_rng' 'true'
check_json_field "$output_json" '.guestcaps.virtio_balloon' 'true'
check_json_field "$output_json" '.guestcaps.virtio_socket' 'true'
check_json_field "$output_json" '.guestcaps.machine_type' 'q35'

echo "PASS: test-linux-no-kernel"
