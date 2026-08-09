#!/bin/bash -
# Test: dynamicscripts plugin — Windows firstboot script copy.

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
mkdir -p "$root/Program Files/Guestfs/Firstboot/scripts" "$scripts"
echo 'Write-Host "joined"' > "$scripts/20_win_firstboot_join-domain.ps1"

cat > "$d/prepare.json" <<EOF
{
  "status": "complete",
  "inspect": {"type": "windows", "major_version": 10, "arch": "x86_64"},
  "firmware": {"type": "bios"},
  "boot_device": {"disk_index": 0},
  "source": {"name": "testvm", "type": "disk"},
  "disks": [],
  "options": {"dynamic_scripts_dir": "$scripts"}
}
EOF

cat > "$d/convert.json" <<'EOF'
{"guestcaps": {"block_bus": "virtio", "net_bus": "virtio", "arch": "x86_64"}}
EOF

jq -n --slurpfile p "$d/prepare.json" --slurpfile c "$d/convert.json" \
    '{prepare: $p[0], convert: $c[0]}' > "$d/pipeline.json"

"$BIN_DIR/kc-finalize" \
    --input "$d/pipeline.json" \
    --output "$d/target-meta.json" \
    --mount-root "$root" \
    --log-level debug

dest="$root/Program Files/Guestfs/Firstboot/scripts/20_win_firstboot_join-domain.ps1"
test -f "$dest"
grep -q 'joined' "$dest"

echo "PASS: test-dynamicscripts-windows"
