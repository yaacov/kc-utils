#!/bin/bash -
# Test: dynamicscripts plugin — Linux firstboot script installation.

source "$(cd "$(dirname "$0")" && pwd)/functions.sh"
set -e
set -x

skip_if_skipped
requires_linux
requires_jq
ensure_built

d=$(mktemp -d)
cleanup_fn rm -rf "$d"

root="$d/guest"
scripts="$d/scripts"
mkdir -p "$root/etc/systemd/system" "$root/usr/local/bin" "$scripts"
echo '#!/bin/bash echo hello' > "$scripts/10_linux_firstboot_hello.sh"
chmod 755 "$scripts/10_linux_firstboot_hello.sh"

cat > "$d/prepare.json" <<EOF
{
  "status": "complete",
  "inspect": {"type": "linux", "distro": "rhel", "major_version": 9, "arch": "x86_64"},
  "firmware": {"type": "bios"},
  "boot_device": {"disk_index": 0},
  "source": {"name": "testvm", "type": "disk"},
  "disks": [],
  "options": {"dynamic_scripts_dir": "$scripts", "hostname": "testhost"}
}
EOF

cat > "$d/convert.json" <<'EOF'
{"guestcaps": {"block_bus": "virtio", "net_bus": "virtio", "arch": "x86_64"}}
EOF

"$BIN_DIR/kc-finalize" \
    --prepare-data "$d/prepare.json" \
    --convert-data "$d/convert.json" \
    --output "$d/target-meta.json" \
    --mount-root "$root" \
    --log-level debug

test -f "$root/usr/local/bin/kc-dynamic-10_linux_firstboot_hello.sh"
grep -q 'hello' "$root/usr/local/bin/kc-dynamic-10_linux_firstboot_hello.sh"

echo "PASS: test-dynamicscripts-linux"
