#!/bin/bash
# Runnable example: prepare → convert → finalize on a phony Linux disk image.
#
# Usage (from repository root, as root):
#   sudo docs/examples/run-linux-disk.sh
#
# Options:
#   KC_BIN_DIR=/path/to/bin   override binary location (default: ./bin)
#   KC_WORK_DIR=/path         working directory (default: mktemp)
#   KC_MOUNT_ROOT=/path       guest mount point (default: /tmp/kc-guest)
#   KC_DISK=/path/to.img      use existing disk image instead of creating one
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
BIN_DIR="${KC_BIN_DIR:-$ROOT_DIR/bin}"
TESTS_DIR="$ROOT_DIR/tests"
MOUNT_ROOT="${KC_MOUNT_ROOT:-/tmp/kc-guest}"
WORK_DIR="${KC_WORK_DIR:-}"
OFFLINE="${KC_OFFLINE:-true}"

if [ "$(id -u)" -ne 0 ]; then
    echo "error: run as root (loop devices and mounts require privileges)" >&2
    exit 1
fi

if [ "$(uname -s)" != "Linux" ]; then
    echo "error: kc-prepare and kc-finalize require Linux" >&2
    exit 1
fi

for bin in kc-prepare kc-convert-linux kc-convert-windows kc-finalize; do
    if [ ! -x "$BIN_DIR/$bin" ]; then
        echo "error: $BIN_DIR/$bin not found; run 'make build' first" >&2
        exit 1
    fi
done

command -v jq >/dev/null 2>&1 || {
    echo "error: jq is required" >&2
    exit 1
}

if [ -z "$WORK_DIR" ]; then
    WORK_DIR=$(mktemp -d /tmp/kc-example.XXXXXX)
    trap 'rm -rf "$WORK_DIR"' EXIT
fi
mkdir -p "$WORK_DIR" "$MOUNT_ROOT"

if [ -n "${KC_DISK:-}" ]; then
    DISK="$KC_DISK"
else
    DISK="$WORK_DIR/linux.img"
    if ! command -v guestfish >/dev/null 2>&1; then
        echo "error: guestfish not found; set KC_DISK to an existing disk image" >&2
        exit 1
    fi
    echo "Creating test disk image: $DISK"
    "$TESTS_DIR/make-disk-linux.sh" "$DISK"
fi

PREPARE_INPUT="$WORK_DIR/prepare-input.json"
cat > "$PREPARE_INPUT" <<EOF
{
  "disks": [{"path": "$DISK", "format": "raw"}],
  "source": {"name": "example-rhel", "type": "disk"},
  "options": {"tmp_dir": "$WORK_DIR"}
}
EOF

echo "=== kc-prepare ==="
"$BIN_DIR/kc-prepare" \
    --input "$PREPARE_INPUT" \
    --output "$WORK_DIR/pipeline.json" \
    --mount-root "$MOUNT_ROOT" \
    --log-level info

CONVERTER=$(jq -r '.converter' "$WORK_DIR/pipeline.json")
echo "Converter selected: $CONVERTER"

OFFLINE_FLAG=""
[ "$OFFLINE" = true ] && OFFLINE_FLAG="--offline"

echo "=== $CONVERTER ==="
"$BIN_DIR/$CONVERTER" \
    --input "$WORK_DIR/pipeline.json" \
    --output "$WORK_DIR/pipeline.json" \
    --mount-root "$MOUNT_ROOT" \
    $OFFLINE_FLAG \
    --log-level info

echo "=== kc-finalize ==="
"$BIN_DIR/kc-finalize" \
    --input "$WORK_DIR/pipeline.json" \
    --output "$WORK_DIR/pipeline.json" \
    --mount-root "$MOUNT_ROOT" \
    --log-level info

echo ""
echo "Conversion complete."
echo "  pipeline data: $WORK_DIR/pipeline.json"
echo ""
jq '{converter: .prepare.converter, root_device: .prepare.root_device, inspect: {type: .prepare.inspect.type, distro: .prepare.inspect.distro, product_name: .prepare.inspect.product_name}}' \
    "$WORK_DIR/pipeline.json"
echo ""
jq '{block_bus: .target.guestcaps.block_bus, net_bus: .target.guestcaps.net_bus, arch: .target.guestcaps.arch}' \
    "$WORK_DIR/pipeline.json"
