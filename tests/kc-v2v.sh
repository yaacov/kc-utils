#!/bin/bash
# kc-v2v.sh -- virt-v2v-in-place workalike using kc-utils binaries
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BIN_DIR="${KC_BIN_DIR:-$SCRIPT_DIR/bin}"

LOG_LEVEL="info"
MOUNT_ROOT="/tmp/kc-guest"
OFFLINE=false
USE_GUESTFS=false
WORK_DIR=""
GUESTFISH_OWNED=false

usage() {
    cat <<'EOF'
Usage: kc-v2v.sh [options]

Options:
  --disk PATH          Disk image to convert (required)
  --mount-root PATH    Guest mount root (default: /tmp/kc-guest)
  --offline            Skip network-dependent operations
  --guestfs            Use libguestfs appliance instead of privileged mount syscalls
  --log-level LEVEL    debug, info, warn, error (default: info)
  --work-dir PATH      Working dir for intermediate JSON (default: auto tmpdir)
  -h, --help           Show this help
EOF
    exit 1
}

cleanup() {
    if $GUESTFISH_OWNED && [ -n "${GUESTFISH_PID:-}" ]; then
        guestfish --remote="$GUESTFISH_PID" -- exit >/dev/null 2>&1 || true
    fi
    if [ -n "${WORK_DIR_CLEANUP:-}" ]; then
        rm -rf "$WORK_DIR_CLEANUP"
    fi
}
trap cleanup EXIT

DISK=""
while [[ $# -gt 0 ]]; do
    case "$1" in
        --disk)       DISK="$2"; shift 2 ;;
        --mount-root) MOUNT_ROOT="$2"; shift 2 ;;
        --offline)    OFFLINE=true; shift ;;
        --guestfs)    USE_GUESTFS=true; shift ;;
        --log-level)  LOG_LEVEL="$2"; shift 2 ;;
        --work-dir)   WORK_DIR="$2"; shift 2 ;;
        -h|--help)    usage ;;
        *)            echo "Unknown option: $1"; usage ;;
    esac
done

[ -z "$DISK" ] && { echo "error: --disk is required"; usage; }

if [ -z "$WORK_DIR" ]; then
    WORK_DIR=$(mktemp -d /tmp/kc-v2v.XXXXXX)
    WORK_DIR_CLEANUP="$WORK_DIR"
fi

cat > "$WORK_DIR/input.json" <<EOF
{
  "disks": [{"path": "$DISK", "format": "raw"}],
  "source": {"name": "$(basename "$DISK" .qcow2)", "type": "disk"},
  "options": {"tmp_dir": "$WORK_DIR"}
}
EOF

GUESTFS_FLAG=""
if $USE_GUESTFS; then
    GUESTFS_FLAG="--guestfs"
    export LIBGUESTFS_BACKEND=direct
    # Mirror Go kc-v2v: one shared guestfish --listen for prepare/convert/finalize.
    # Prefer virt-guestfish so RHEL NTFS mounts are allowlisted (argv[0]).
    gf=guestfish
    command -v virt-guestfish >/dev/null 2>&1 && gf=virt-guestfish
    eval "$("$gf" --listen)"
    export KC_GUESTFISH_PID="$GUESTFISH_PID"
    GUESTFISH_OWNED=true
    sock="/tmp/.guestfish-$(id -u)/socket-$GUESTFISH_PID"
    for _ in $(seq 1 100); do
        if [ -S "$sock" ] || [ -e "$sock" ]; then
            break
        fi
        sleep 0.05
    done
    if [ ! -e "$sock" ]; then
        echo "error: guestfish listen socket not ready at $sock (pid=$GUESTFISH_PID)" >&2
        exit 1
    fi
    echo "=== guestfish shared listener pid=$GUESTFISH_PID ==="
fi

echo "=== kc-prepare ==="
"$BIN_DIR/kc-prepare" \
    --input "$WORK_DIR/input.json" \
    --output "$WORK_DIR/prepare-out.json" \
    --mount-root "$MOUNT_ROOT" \
    --log-level "$LOG_LEVEL" \
    $GUESTFS_FLAG

CONVERTER=$(jq -r '.prepare.converter' "$WORK_DIR/prepare-out.json")
echo "Detected converter: $CONVERTER"

OFFLINE_FLAG=""
$OFFLINE && OFFLINE_FLAG="--offline"

echo "=== $CONVERTER ==="
"$BIN_DIR/$CONVERTER" \
    --input "$WORK_DIR/prepare-out.json" \
    --output "$WORK_DIR/convert-out.json" \
    --mount-root "$MOUNT_ROOT" \
    $OFFLINE_FLAG \
    --log-level "$LOG_LEVEL" \
    $GUESTFS_FLAG

echo "=== kc-finalize ==="
"$BIN_DIR/kc-finalize" \
    --input "$WORK_DIR/convert-out.json" \
    --output "$WORK_DIR/target-meta.json" \
    --mount-root "$MOUNT_ROOT" \
    --log-level "$LOG_LEVEL" \
    $GUESTFS_FLAG

echo "=== Conversion complete ==="
echo "Target metadata: $WORK_DIR/target-meta.json"
