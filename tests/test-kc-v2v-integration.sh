#!/bin/bash -
# Test: kc-v2v supporting packages (v2v config parsing, inspection XML, HTTP handlers).

source "$(cd "$(dirname "$0")" && pwd)/functions.sh"
set -e

skip_if_skipped
requires_linux  # binaries are Linux ELF; cannot execute on other platforms
ensure_built

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$repo_root"
go test ./pkg/v2v/... ./internal/v2v/... -count=1

echo "PASS: test-kc-v2v-integration"
