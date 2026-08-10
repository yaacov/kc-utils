#!/bin/bash -
# Test: Windows Server 2008 uses archived 2k8 driver dir when present.
source "$(cd "$(dirname "$0")" && pwd)/functions.sh"
set -e
set -x

skip_if_skipped
requires_linux
requires command -v hivexregedit
requires_jq
ensure_built
requires test -f "$UPSTREAM_TESTDATA/minimal-hive"

iso_dir="/usr/share/virtio-win"
if ! mkdir -p "$iso_dir/drivers/by-os/amd64/2k8" "$iso_dir/drivers/by-os/amd64/2k8R2" 2>/dev/null; then
    echo "$0: cannot create $iso_dir (need root?), skipping"
    exit 77
fi
cleanup_fn rm -rf /usr/share/virtio-win

for drv in viostor vioscsi netkvm; do
    for ext in inf sys cat; do
        echo "fake-2k8-$drv" > "$iso_dir/drivers/by-os/amd64/2k8/$drv.$ext"
        echo "fake-2k8r2-$drv" > "$iso_dir/drivers/by-os/amd64/2k8R2/$drv.$ext"
    done
done

d=$(mktemp -d)
cleanup_fn rm -rf "$d"

root="$d/guest"
mkdir -p "$root/Windows/System32/drivers"
mkdir -p "$root/Program Files"
make_windows_hives "$root"

make_windows_prepare_json "$root" 6 0 x86_64 "Windows Server (R) 2008 Enterprise" > "$d/prepare.json"
jq -n --slurpfile p "$d/prepare.json" '{prepare: $p[0]}' > "$d/pipeline.json"

"$BIN_DIR/kc-convert-windows" \
    --input "$d/pipeline.json" \
    --output "$d/output.json" \
    --mount-root "$root" \
    --offline \
    --log-level debug

test -f "$d/output.json"
grep -q 'fake-2k8-viostor' "$root/Windows/Drivers/VirtIO/viostor.inf"

bat="$root/Program Files/Guestfs/Firstboot/firstboot.bat"
test -f "$bat"
grep -q 'ExecutionPolicy' "$bat"
grep -Fq 'C:\Windows\System32\shutdown.exe /r /t 5 /f' "$bat"

echo "PASS: test-windows-win2008"
