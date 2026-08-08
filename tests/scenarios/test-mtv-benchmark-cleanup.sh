#!/usr/bin/env bash
#
# Clean up MTV benchmark resources on a live cluster.
#
# Default (--all): delete the benchmark namespace and reset MTV settings
# touched by the benchmark (virt_v2v_image_fqin, feature_windows_wait_for_reboot).
#
# Examples:
#   ./tests/scenarios/test-mtv-benchmark-cleanup.sh
#   NS=mtv-kc-v2v-bench ./tests/scenarios/test-mtv-benchmark-cleanup.sh --namespace-only
#   RHEL_VM=mtv-func-cold-rhel9-staticips ./tests/scenarios/test-mtv-benchmark-cleanup.sh --release-rhel
#
# Env:
#   NS        Benchmark namespace (default mtv-kc-v2v-bench)
#   RHEL_VM   Migrated RHEL VM name (for --release-rhel)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"
# shellcheck source=lib/cleanup.sh
source "${SCRIPT_DIR}/lib/cleanup.sh"

NS="${NS:-mtv-kc-v2v-bench}"
SCOPE="all"

usage() {
  cat <<EOF
Usage: $0 [options]

Clean up MTV benchmark resources on the current OpenShift cluster.

Options:
  --all              Delete namespace and reset MTV settings (default)
  --namespace-only   Delete the benchmark namespace only
  --settings-only    Reset MTV benchmark settings only
  --release-rhel     Delete RHEL plan/pods and stop the migrated RHEL VM
  --release-plans    Delete benchmark plans (RHEL + Windows) in the namespace
  -h, --help         Show this help

Environment:
  NS        Benchmark namespace (default: mtv-kc-v2v-bench)
  RHEL_VM   Migrated RHEL VM name (for --release-rhel)
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --all)
      SCOPE="all"
      ;;
    --namespace-only)
      SCOPE="namespace"
      ;;
    --settings-only)
      SCOPE="settings"
      ;;
    --release-rhel)
      SCOPE="rhel"
      ;;
    --release-plans)
      SCOPE="plans"
      ;;
    -h|--help|help)
      usage
      exit 0
      ;;
    *)
      echo "ERROR: unknown option '$1'." >&2
      usage >&2
      exit 1
      ;;
  esac
  shift
done

echo "MTV benchmark cleanup (scope=${SCOPE}, NS=${NS})"
run_benchmark_cleanup "${SCOPE}"
echo "Cleanup complete."
