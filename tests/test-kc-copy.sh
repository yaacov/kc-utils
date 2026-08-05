#!/bin/bash
# test-kc-copy.sh — unit tests for pkg/copy
set -e

source "$(cd "$(dirname "$0")" && pwd)/functions.sh"
skip_if_skipped
requires_linux  # binaries are Linux ELF; cannot execute on other platforms
ensure_built

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$repo_root"

go test ./pkg/copy/... -count=1

echo "PASS: test-kc-copy"
