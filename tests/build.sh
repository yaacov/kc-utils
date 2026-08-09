#!/bin/bash
set -euo pipefail
TESTS_DIR="$(cd "$(dirname "$0")" && pwd)"
TOP_DIR="$(cd "$TESTS_DIR/.." && pwd)"
BIN_DIR="$TESTS_DIR/bin"

mkdir -p "$BIN_DIR"
cd "$TOP_DIR"

if [ "$(uname -s)" = "Linux" ]; then
    bins="kc-prepare kc-convert-linux kc-convert-windows kc-finalize"
else
    bins="kc-convert-linux kc-convert-windows"
fi

for bin in $bins; do
    echo "Building $bin..."
    CGO_ENABLED=0 GOOS=linux go build -trimpath -o "$BIN_DIR/$bin" "./cmd/$bin"
done
echo "Binaries built in $BIN_DIR/"
