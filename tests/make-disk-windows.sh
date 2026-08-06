#!/bin/bash
# Creates a phony Windows Server 2022 disk image for testing kc-prepare/kc-finalize.
# Builds binary hives from upstream .reg text files + minimal-hive using hivexregedit,
# then uploads them into the guestfish image.
# Usage: make-disk-windows.sh <output.img>
set -e

export LIBGUESTFS_BACKEND=direct

IMG="${1:?Usage: make-disk-windows.sh <output.img>}"
TESTS_DIR="$(cd "$(dirname "$0")" && pwd)"
FIXTURES="$TESTS_DIR/fixtures"

rm -f "$IMG"

# Prefer virt-guestfish so RHEL's winsupport NTFS allowlist applies (argv[0]).
GUESTFISH=guestfish
command -v virt-guestfish >/dev/null 2>&1 && GUESTFISH=virt-guestfish

if ! "$GUESTFISH" -a /dev/null run : available "ntfs3g ntfsprogs"; then
    echo "Warning: no NTFS support in libguestfs, skipping Windows image"
    exit 77
fi

# Build binary hive files from .reg text + minimal-hive.
tmpdir=$(mktemp -d)
trap "rm -rf $tmpdir" EXIT

cp "$FIXTURES/minimal-hive" "$tmpdir/SYSTEM"
hivexregedit --merge "$tmpdir/SYSTEM" \
    --prefix 'HKEY_LOCAL_MACHINE\SYSTEM' \
    "$FIXTURES/windows-system.reg"
hivexregedit --merge "$tmpdir/SYSTEM" \
    --prefix 'HKEY_LOCAL_MACHINE\SYSTEM' \
    "$FIXTURES/windows-system-parents.reg"

cp "$FIXTURES/minimal-hive" "$tmpdir/SOFTWARE"
cat "$FIXTURES/win2k22-software.reg" \
    "$FIXTURES/windows-software-all.reg" > "$tmpdir/software-combined.reg"
hivexregedit --merge "$tmpdir/SOFTWARE" \
    --prefix 'HKEY_LOCAL_MACHINE\SOFTWARE' \
    "$tmpdir/software-combined.reg"

"$GUESTFISH" <<EOF
disk-create $IMG raw 512M
add $IMG
run

part-init /dev/sda gpt
part-add /dev/sda p 64 524287
part-add /dev/sda p 524288 -64

mkfs vfat /dev/sda1
mkfs ntfs /dev/sda2

mount /dev/sda2 /
mkdir-p /Windows/System32/Config
mkdir-p /Windows/System32/Drivers
mkdir-p /Windows/TEMP

upload $tmpdir/SOFTWARE /Windows/System32/Config/SOFTWARE
upload $tmpdir/SYSTEM /Windows/System32/Config/SYSTEM

mkdir "/Program Files"
touch /autoexec.bat

umount-all
EOF

echo "Created $IMG"
