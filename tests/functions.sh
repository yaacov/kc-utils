#!/bin/bash
# Common functions for kc-utils e2e tests.
# Each test script starts with: source ./functions.sh

# Directories
TESTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TOP_DIR="$(cd "$TESTS_DIR/.." && pwd)"
BIN_DIR="$TESTS_DIR/bin"
UPSTREAM_TESTDATA="$TESTS_DIR/fixtures"

# Cleanup hooks (same pattern as upstream virt-v2v)
declare -a _cleanup_hook
cleanup_fn() { _cleanup_hook[${#_cleanup_hook[@]}]="$@"; }
_run_cleanup_hooks() {
    local _status=$? _i
    set +e
    trap '' INT QUIT TERM EXIT ERR
    for (( _i=0; _i<${#_cleanup_hook[@]}; ++_i )); do
        ${_cleanup_hook[_i]}
    done
    exit $_status
}
trap _run_cleanup_hooks INT QUIT TERM EXIT ERR

# skip_if_skipped: check SKIP_<TEST_NAME> env var
skip_if_skipped() {
    local v
    if [ -n "$1" ]; then
        v="SKIP_$(basename "$1" | tr 'a-z.-' 'A-Z__')"
    else
        v="SKIP_$(basename "$0" | tr 'a-z.-' 'A-Z__')"
    fi
    if [ -n "${!v}" ]; then
        echo "$0: test skipped because \$$v is set"
        exit 77
    fi
}

# requires: skip (exit 77) if command fails
requires() {
    ( "$@" ) </dev/null >/dev/null 2>&1 || {
        echo "$0: test prerequisite '$*' is missing"
        exit 77
    }
}

# requires_linux: skip if not on Linux
requires_linux() {
    if [ "$(uname -s)" != "Linux" ]; then
        echo "$0: test skipped (requires Linux)"
        exit 77
    fi
}

# requires_jq: skip if jq not available
requires_jq() {
    requires jq --version
}

# root_test: skip unless running as root
root_test() {
    if [ "$(id -u)" -ne 0 ]; then
        echo "$0: use 'sudo make test-e2e-root' to run this test"
        exit 77
    fi
}

# requires_loop_partitions: skip if loop device partition scanning doesn't
# create device nodes (e.g., inside Docker Desktop containers).
requires_loop_partitions() {
    local img dev rc=0
    img=$(mktemp /tmp/kc-loop-check.XXXXXX)
    truncate -s 64M "$img"
    printf '\x55\xaa' | dd of="$img" bs=1 seek=510 conv=notrunc >/dev/null 2>&1
    printf '\x00\x00\x00\x00\x80\x00\x01\x00\x83\xfe\x3f\x01\x3f\x00\x00\x00\xc1\xff\x01\x00' \
        | dd of="$img" bs=1 seek=446 conv=notrunc >/dev/null 2>&1
    dev=$(losetup --partscan --find --show "$img" 2>/dev/null) || rc=$?
    if [ $rc -ne 0 ] || [ -z "$dev" ]; then
        rm -f "$img"
        echo "$0: losetup failed, skipping"
        exit 77
    fi
    if ! test -b "${dev}p1" 2>/dev/null; then
        losetup -d "$dev" 2>/dev/null
        rm -f "$img"
        echo "$0: loop partition device nodes not available (Docker Desktop?), skipping"
        exit 77
    fi
    losetup -d "$dev" 2>/dev/null
    rm -f "$img"
}

# ensure_built: build test binaries if not present
ensure_built() {
    if [ ! -x "$BIN_DIR/kc-convert-linux" ]; then
        "$TESTS_DIR/build.sh" || { echo "$0: build failed"; exit 1; }
    fi
}

# check_json_field: verify a JSON field value using jq
# Usage: check_json_field file.json '.convert.guestcaps.block_bus' 'virtio'
check_json_field() {
    local file="$1" path="$2" expected="$3"
    local got
    got=$(jq -r "$path" "$file")
    if [ "$got" != "$expected" ]; then
        echo "FAIL: $path in $file: expected '$expected', got '$got'"
        return 1
    fi
}

# assert_systemd_unit_disabled: verify a systemd unit is masked and vendor wants removed
# Usage: assert_systemd_unit_disabled <guest_root> <unit.service>
assert_systemd_unit_disabled() {
    local root="$1" unit="$2"
    local mask="$root/etc/systemd/system/$unit"
    local vendor_wants="$root/usr/lib/systemd/system/multi-user.target.wants/$unit"
    if [ -e "$vendor_wants" ] || [ -L "$vendor_wants" ]; then
        echo "FAIL: vendor wants symlink still exists: $vendor_wants"
        return 1
    fi
    if [ ! -L "$mask" ] || [ "$(readlink "$mask")" != "/dev/null" ]; then
        echo "FAIL: unit not masked to /dev/null: $mask"
        return 1
    fi
}

# make_linux_prepare_json: generate a prepare section JSON for Linux tests
# Usage: make_linux_prepare_json <root> <distro> <major> <minor> <arch> <product> <firmware> > file.json
make_linux_prepare_json() {
    local root="$1" distro="$2" major="$3" minor="$4" arch="$5" product="$6" firmware="${7:-bios}"
    cat <<EOF
{
  "status": "ok",
  "converter": "linux",
  "inspect": {
    "type": "linux",
    "distro": "$distro",
    "major_version": $major,
    "minor_version": $minor,
    "arch": "$arch",
    "product_name": "$product"
  },
  "firmware": {"type": "$firmware"},
  "boot_device": {"disk_index": 0},
  "free_space": [],
  "source": {"name": "test-vm", "type": "none"},
  "disks": [],
  "mount_root": "$root"
}
EOF
}

# setup_fake_virtio_drivers_tree: populate a minimal virtio-win by-os tree for tests.
# Creates driver files under /usr/share/virtio-win/drivers/by-os/amd64/2k22/.
# Usage: setup_fake_virtio_drivers_tree
setup_fake_virtio_drivers_tree() {
    local iso_dir="/usr/share/virtio-win"
    if ! mkdir -p "$iso_dir/drivers/by-os/amd64/2k22" 2>/dev/null; then
        echo "$0: cannot create $iso_dir (need root?), skipping"
        exit 77
    fi
    cleanup_fn rm -rf /usr/share/virtio-win
    local drv osver ext
    for drv in viostor vioscsi netkvm viorng balloon vioserial; do
        for ext in inf sys cat; do
            echo "fake-$drv" > "$iso_dir/drivers/by-os/amd64/2k22/$drv.$ext"
        done
    done
    mkdir -p "$iso_dir/guest-agent"
    echo "fake-ga" > "$iso_dir/guest-agent/qemu-ga-x86_64.msi"
}

# install_stub_dracut: place a no-op dracut inside a fake guest root so that
# the initramfs rebuild step succeeds in test chroots where no real dracut
# binary exists. The chroot needs a working /bin/sh for scripts, so we copy
# the host shell (resolving symlinks) and its shared libraries if missing.
# Usage: install_stub_dracut <root>
install_stub_dracut() {
    local root="$1"
    if [ ! -x "$root/bin/sh" ]; then
        local sh_real
        sh_real=$(readlink -f "$(command -v sh)")
        mkdir -p "$root/bin"
        cp "$sh_real" "$root/bin/sh"
        local lib
        for lib in $(ldd "$sh_real" 2>/dev/null | grep -o '/[^ ]*'); do
            if [ -f "$lib" ]; then
                mkdir -p "$root$(dirname "$lib")"
                cp "$lib" "$root$lib" 2>/dev/null || true
            fi
        done
    fi
    mkdir -p "$root/usr/sbin" "$root/usr/bin" "$root/sbin"
    local loc
    for loc in "$root/usr/sbin/dracut" "$root/usr/bin/dracut" "$root/sbin/dracut"; do
        cat > "$loc" <<'STUB'
#!/bin/sh
# Stub dracut for tests: find the output image positional arg and write
# a fake initramfs so the post-rebuild verification check passes.
skip_next=false
outfile=""
for arg; do
    if $skip_next; then skip_next=false; continue; fi
    case "$arg" in
        --add-drivers|--modules|--kver) skip_next=true ;;
        --*) ;;
        *)
            if [ -z "$outfile" ]; then outfile="$arg"; fi
            ;;
    esac
done
if [ -n "$outfile" ]; then
    echo "FAKE_INITRAMFS_REBUILT" > "$outfile"
fi
exit 0
STUB
        chmod +x "$loc"
    done
}

FIXTURES_DIR="$TESTS_DIR/fixtures"

# make_windows_hives: create SYSTEM and SOFTWARE hives from upstream test data
# Usage: make_windows_hives <root>
# Creates hives at <root>/Windows/System32/config/{SYSTEM,SOFTWARE}
make_windows_hives() {
    local root="$1"
    mkdir -p "$root/Windows/System32/config"

    cp "$UPSTREAM_TESTDATA/minimal-hive" "$root/Windows/System32/config/SYSTEM"
    hivexregedit --merge "$root/Windows/System32/config/SYSTEM" \
        --prefix 'HKEY_LOCAL_MACHINE\SYSTEM' \
        "$UPSTREAM_TESTDATA/windows-system.reg"
    hivexregedit --merge "$root/Windows/System32/config/SYSTEM" \
        --prefix 'HKEY_LOCAL_MACHINE\SYSTEM' \
        "$FIXTURES_DIR/windows-system-parents.reg"

    cp "$UPSTREAM_TESTDATA/minimal-hive" "$root/Windows/System32/config/SOFTWARE"
    cat "$UPSTREAM_TESTDATA/win2k22-software.reg" \
        "$UPSTREAM_TESTDATA/windows-software-all.reg" > "$root/software-combined.reg"
    hivexregedit --merge "$root/Windows/System32/config/SOFTWARE" \
        --prefix 'HKEY_LOCAL_MACHINE\SOFTWARE' \
        "$root/software-combined.reg"
    rm -f "$root/software-combined.reg"
}

# make_windows_prepare_json: generate a prepare section JSON for Windows tests
# Usage: make_windows_prepare_json <root> <major> <minor> <arch> <product> <firmware> > file.json
make_windows_prepare_json() {
    local root="$1" major="$2" minor="$3" arch="$4" product="$5" firmware="${6:-bios}"
    cat <<EOF
{
  "status": "ok",
  "converter": "windows",
  "inspect": {
    "type": "windows",
    "distro": "windows",
    "major_version": $major,
    "minor_version": $minor,
    "arch": "$arch",
    "product_name": "$product"
  },
  "inspect_windows": {
    "system_root": "Windows",
    "current_control_set": 1,
    "system_hive": "Windows/System32/config/SYSTEM",
    "software_hive": "Windows/System32/config/SOFTWARE",
    "drive_mappings": {"C": "/"}
  },
  "firmware": {"type": "$firmware"},
  "boot_device": {"disk_index": 0},
  "free_space": [],
  "source": {"name": "test-vm", "type": "none"},
  "disks": [],
  "mount_root": "$root"
}
EOF
}

# make_multiboot_linux_img: two ext4 roots (RHEL on p1, Debian on p2)
# Usage: make_multiboot_linux_img <output.img>
make_multiboot_linux_img() {
    local img="${1:?Usage: make_multiboot_linux_img <output.img>}"
    truncate -s 256M "$img"
    guestfish --rw -a "$img" -m /dev/sda1:/ -m /dev/sda2:/ <<'EOF'
run
part-init /dev/sda mbr
part-add /dev/sda p 2048 133119
part-add /dev/sda p 133120 -2048
mkfs ext4 /dev/sda1
mkfs ext4 /dev/sda2
mount /dev/sda1 /
write /etc/os-release "NAME=\"Red Hat Enterprise Linux\"\nID=\"rhel\"\nVERSION_ID=\"9\"\nPRETTY_NAME=\"RHEL 9\"\n"
umount /
mount /dev/sda2 /
write /etc/os-release "NAME=\"Debian GNU/Linux\"\nID=\"debian\"\nVERSION_ID=\"12\"\nPRETTY_NAME=\"Debian 12\"\n"
umount-all
EOF
}

# make_lvm_linux_img: boot ext4 + LVM root with UUID-based fstab
# Usage: make_lvm_linux_img <output.img>
make_lvm_linux_img() {
    local img="${1:?Usage: make_lvm_linux_img <output.img>}"
    truncate -s 512M "$img"
    guestfish --rw -a "$img" <<'EOF'
run
part-init /dev/sda mbr
part-add /dev/sda p 2048 524287
part-add /dev/sda p 524288 -2048
mkfs ext4 /dev/sda1
pvcreate /dev/sda2
vgcreate kclvm /dev/sda2
lvcreate -l 100%FREE -n root kclvm
mkfs ext4 /dev/kclvm/root
mount /dev/kclvm/root /
write /etc/os-release "NAME=\"Red Hat Enterprise Linux\"\nID=\"rhel\"\nVERSION_ID=\"9\"\nPRETTY_NAME=\"RHEL 9 LVM\"\n"
mkdir-p /boot/grub2
sh "ROOTUUID=$(blkid -o value -s UUID /dev/kclvm/root); BOOTUUID=$(blkid -o value -s UUID /dev/sda1); printf 'UUID=%s /boot ext4 defaults 0 2\nUUID=%s / ext4 defaults 0 1\n' \"$BOOTUUID\" \"$ROOTUUID\" > /etc/fstab"
umount /
mount /dev/sda1 /boot
write /boot/grub2/grub.cfg "menuentry 'RHEL LVM' { }\n"
umount-all
EOF
}
