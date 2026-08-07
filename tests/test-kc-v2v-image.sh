#!/bin/bash -
# Smoke-test the built kc-v2v container image.
#
# Run inside the image (see `make test-kc-v2v-image`):
#   - required RPMs present (guestfs, winsupport, FS tools)
#   - libguestfs-winsupport appliance bits installed
#   - guestfish appliance can report + mount key filesystems (needs /dev/kvm)
#
# Exit 0 pass (or packages OK + appliance skipped without kvm), 1 fail.
# Set REQUIRE_GUESTFS=1 (or REQUIRE_NTFS=1) to require /dev/kvm + FS checks.
# Clevis/NBDE (clevisluks=yes) is required; make test-kc-v2v-image sets REQUIRE_CLEVIS=1.

set -euo pipefail

# This test is designed for the kc-v2v image (make test-kc-v2v-image).
# Skip when run in the generic test container.
if [ ! -x /usr/bin/kc-v2v ] && [ ! -x /usr/lib/kc-utils/kc-prepare ]; then
    exit 77
fi

fail=0
pass() { echo "PASS: $*"; }
fail_msg() { echo "FAIL: $*"; fail=1; }
skip_msg() { echo "SKIP: $*"; }

require_guestfs=0
if [ "${REQUIRE_GUESTFS:-}" = "1" ] || [ "${REQUIRE_NTFS:-}" = "1" ]; then
	require_guestfs=1
fi

echo "=== kc-v2v image smoke test ==="
echo "image check host=$(hostname 2>/dev/null || true) uid=$(id -u) HOME=${HOME:-}"

# ── Package presence (matches build/kc-v2v/Containerfile) ────────────
for pkg in \
	libguestfs \
	libguestfs-winsupport \
	libguestfs-xfs \
	guestfs-tools \
	ntfs-3g \
	e2fsprogs \
	xfsprogs \
	btrfs-progs \
	hivex \
	perl-hivex \
	qemu-img \
	cryptsetup \
	lvm2 \
	clevis \
	clevis-luks
do
	if rpm -q "$pkg" >/dev/null 2>&1; then
		pass "rpm $pkg ($(rpm -q "$pkg"))"
	else
		fail_msg "rpm $pkg not installed"
	fi
done

# ── Binaries used by the pipeline ────────────────────────────────────
for bin in guestfish virt-guestfish qemu-img hivexregedit cryptsetup kc-v2v kc-prepare kc-copy; do
	if command -v "$bin" >/dev/null 2>&1 || [ -x "/usr/lib/kc-utils/$bin" ]; then
		pass "binary $bin"
	else
		fail_msg "binary $bin missing"
	fi
done

# RHEL winsupport only allows NTFS mounts when program name starts with virt-.
if [ -L /usr/bin/virt-guestfish ] || [ -x /usr/bin/virt-guestfish ]; then
	pass "virt-guestfish present (RHEL NTFS allowlist argv[0])"
else
	fail_msg "virt-guestfish missing — NTFS mounts via guestfish will fail on RHEL/CentOS"
fi
GUESTFISH=virt-guestfish
command -v virt-guestfish >/dev/null 2>&1 || GUESTFISH=guestfish

# ── Winsupport appliance payloads (what actually enables NTFS mount) ─
libdir=
for d in /usr/lib64/guestfs/supermin.d /usr/lib/guestfs/supermin.d; do
	if [ -d "$d" ]; then
		libdir=$d
		break
	fi
done
if [ -z "$libdir" ]; then
	fail_msg "guestfs supermin.d directory not found"
else
	pass "supermin.d at $libdir"
	for f in zz-winsupport.tar.gz zz-winsupport-deps; do
		if [ -f "$libdir/$f" ]; then
			pass "winsupport file $f ($(wc -c <"$libdir/$f") bytes)"
		else
			if [ "$f" = "zz-winsupport.tar.gz" ]; then
				fail_msg "missing $libdir/$f (NTFS mount will fail with 'unsupported filesystem type')"
			else
				echo "WARN: missing optional $libdir/$f"
			fi
		fi
	done
fi

# Fail early on package/file problems before spending time on appliance.
if [ "$fail" -ne 0 ]; then
	echo ""
	echo "Image package/file checks failed — not running appliance FS probes."
	exit 1
fi

# ── Appliance capabilities + create/mount each FS ────────────────────
export LIBGUESTFS_BACKEND="${LIBGUESTFS_BACKEND:-direct}"
export HOME="${HOME:-/var/tmp}"

if [ ! -e /dev/kvm ]; then
	if [ "$require_guestfs" -eq 1 ]; then
		fail_msg "/dev/kvm missing and REQUIRE_GUESTFS/REQUIRE_NTFS=1"
		exit 1
	fi
	skip_msg "no /dev/kvm — package checks passed; skipping guestfish FS checks (set REQUIRE_GUESTFS=1 to require them)"
	echo ""
	echo "Results: PASS (appliance FS checks skipped)"
	exit 0
fi

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

out=$tmp/supported.txt
if ! "$GUESTFISH" -a /dev/null run : supported >"$out" 2>"$tmp/guestfish.err"; then
	echo "guestfish stderr:"
	cat "$tmp/guestfish.err" || true
	fail_msg "guestfish appliance launch / supported failed"
	exit 1
fi
pass "guestfish appliance launched (supported, bin=$GUESTFISH)"

# Feature name → human label. These must be "yes" in `guestfish … supported`.
# ntfs3g: Windows guests (via libguestfs-winsupport)
# xfs / btrfs: Linux guests (libguestfs-xfs / btrfs-progs in appliance)
# hivex: Windows registry offline edits
# lvm2 / luks: common Linux disk layouts
check_supported() {
	local feat=$1
	local why=$2
	local val
	val=$(awk -v f="$feat" '$1==f {print $2; exit}' "$out")
	if [ "$val" = "yes" ]; then
		pass "guestfish supported $feat=yes ($why)"
	else
		grep -E "^${feat} " "$out" || true
		fail_msg "guestfish supported $feat=${val:-missing} (want yes) — $why"
	fi
}

check_supported ntfs3g "Windows NTFS mount"
check_supported xfs "Linux XFS"
check_supported btrfs "Linux btrfs"
check_supported hivex "Windows registry (hivex)"
check_supported lvm2 "Linux LVM"
check_supported luks "LUKS"

# Clevis/NBDE — required; make test-kc-v2v-image runs with REQUIRE_CLEVIS=1
clevis_val=$(awk '$1=="clevisluks" {print $2; exit}' "$out")
if [ "$clevis_val" = "yes" ]; then
	pass "guestfish supported clevisluks=yes (NBDE Clevis unlock)"
else
	fail_msg "guestfish supported clevisluks=${clevis_val:-missing} (want yes) — Clevis/NBDE"
fi

# ext* is usually covered by the core appliance; report but do not hard-fail
# on a missing feature name (varies by libguestfs version).
for feat in ext2 linuxfs; do
	val=$(awk -v f="$feat" '$1==f {print $2; exit}' "$out")
	if [ -n "$val" ]; then
		if [ "$val" = "yes" ]; then
			pass "guestfish supported $feat=yes"
		else
			echo "WARN: guestfish supported $feat=$val"
		fi
	fi
done

echo "--- guestfish supported (filesystem-related) ---"
grep -Ei '^(ntfs|xfs|btrfs|ext|hivex|lvm|luks|clevis|linuxfs)' "$out" || true

# Create + mount a tiny filesystem for each type we ship support for.
# guestfish -N fs:TYPE builds a disk, formats, mounts at /, then runs the cmd.
mount_fs() {
	local fstype=$1
	local img=$tmp/fs-$fstype.img
	local err=$tmp/fs-$fstype.err
	# -N auto-mounts; argv[0] must be virt-* for NTFS on RHEL/CentOS.
	if "$GUESTFISH" -N "$img"=fs:"$fstype":64M -- ls / >/dev/null 2>"$err"; then
		pass "$GUESTFISH -N fs:$fstype mount + ls /"
	else
		echo "--- fs:$fstype stderr ---"
		cat "$err" || true
		fail_msg "guestfish could not create/mount $fstype (fs:$fstype)"
	fi
}

mount_fs ext4
mount_fs xfs
mount_fs btrfs
mount_fs ntfs

echo ""
if [ "$fail" -ne 0 ]; then
	echo "Results: FAIL"
	exit 1
fi
echo "Results: PASS"
exit 0
