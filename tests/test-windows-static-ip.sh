#!/bin/bash -
# Test: Static IP firstboot script generation from prepare JSON options.

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

d=$(mktemp -d)
cleanup_fn rm -rf "$d"

root="$d/guest"
mkdir -p "$root/Windows/System32/drivers"
mkdir -p "$root/Program Files"
make_windows_hives "$root"

# Prepare JSON with static IPs in options (echoed from kc-prepare).
cat > "$d/prepare.json" <<'EOF'
{
  "status": "complete",
  "converter": "kc-convert-windows",
  "inspect": {"type": "windows", "major_version": 10, "minor_version": 0, "arch": "x86_64", "product_name": "Windows Server 2022"},
  "firmware": {"type": "bios"},
  "boot_device": {"disk_index": 0},
  "source": {"name": "testvm", "type": "disk"},
  "disks": [],
  "options": {
    "static_ips": [
      {"mac": "52:54:00:aa:bb:cc", "ip": "192.168.1.10", "gateway": "192.168.1.1", "netmask": "255.255.255.0", "dns": ["8.8.8.8"]}
    ]
  }
}
EOF

"$BIN_DIR/kc-convert-windows" \
    --prepare-data "$d/prepare.json" \
    --output "$d/output.json" \
    --mount-root "$root" \
    --offline \
    --log-level debug

firstboot_scripts="$root/Program Files/Guestfs/Firstboot/scripts"
test -f "$firstboot_scripts/2500-static-ip.ps1"
grep -q '192.168.1.10' "$firstboot_scripts/2500-static-ip.ps1"
grep -q 'Get-NetAdapter' "$firstboot_scripts/2500-static-ip.ps1"

echo "PASS: test-windows-static-ip"
