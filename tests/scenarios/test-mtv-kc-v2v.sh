#!/usr/bin/env bash
#
# E2E smoke: MTV cold migration of one mtv-func RHEL VM then one Windows VM
# using the kc-v2v conversion image.
#
# Flow: set virt_v2v_image_fqin -> namespace + providers -> RHEL plan -> Windows
#       plan -> restore settings -> cleanup (unless SKIP_CLEANUP).
#
# Prerequisites: oc, oc mtv plugin, jq, GOVC_URL/USERNAME/PASSWORD,
#   MTV installed with VDDK configured, and KC_V2V_IMAGE pointing at a
#   pushable/pullable kc-v2v image (default from Makefile).
#
# Env overrides:
#   KC_V2V_IMAGE            conversion image FQIN (required unless set by make)
#   RHEL_VM / WIN_VM        source VM names (auto-picked from mtv-func* if unset)
#   NS                      namespace (default mtv-kc-v2v-test)
#   PROVIDER                vSphere provider name (default vsphere-test)
#   SKIP_CLEANUP            keep NS on exit (default false)
#   KEEP_IMAGE_SETTING      leave virt_v2v_image_fqin / reboot flag (default false)
#   DISABLE_WAIT_FOR_REBOOT set feature_windows_wait_for_reboot=false (default true)
#   INTERVAL / MAX_ATTEMPTS plan poll (defaults 15s / 120 => 30m per plan)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=common.sh
source "${SCRIPT_DIR}/common.sh"

NS="${NS:-mtv-kc-v2v-test}"
PROVIDER="${PROVIDER:-vsphere-test}"
RHEL_VM="${RHEL_VM:-}"
WIN_VM="${WIN_VM:-}"
SKIP_CLEANUP="${SKIP_CLEANUP:-false}"
KEEP_IMAGE_SETTING="${KEEP_IMAGE_SETTING:-false}"
DISABLE_WAIT_FOR_REBOOT="${DISABLE_WAIT_FOR_REBOOT:-true}"
INTERVAL="${INTERVAL:-15}"
MAX_ATTEMPTS="${MAX_ATTEMPTS:-120}"
KC_V2V_IMAGE="${KC_V2V_IMAGE:-}"

cleanup() {
  restore_mtv_settings
  if [[ "${SKIP_CLEANUP}" == "true" ]]; then
    echo "SKIP_CLEANUP=true -- preserving resources in namespace '${NS}'."
    return 0
  fi
  echo "Cleaning up namespace ${NS}..."
  oc delete namespace "${NS}" --ignore-not-found 2>/dev/null || true
  echo "Cleanup done."
}
trap cleanup EXIT

echo "=========================================="
echo "MTV-kc-v2v Smoke (RHEL then Windows)"
echo "=========================================="
echo ""

echo "Preflight..."
preflight_mtv_cluster

if [[ -z "${KC_V2V_IMAGE}" ]]; then
  echo "ERROR: KC_V2V_IMAGE must be set (e.g. quay.io/you/kc-v2v:devel-amd64)." >&2
  echo "Or run: make test-cluster-smoke" >&2
  exit 1
fi
echo "KC_V2V_IMAGE=${KC_V2V_IMAGE}"
echo ""

save_mtv_settings
set_virt_v2v_image "${KC_V2V_IMAGE}"
if [[ "${DISABLE_WAIT_FOR_REBOOT}" == "true" ]]; then
  disable_windows_wait_for_reboot
fi
echo ""

# Fresh namespace (also clears leftover plans from prior runs)
fresh_namespace "${NS}"
echo ""

create_vsphere_and_host_providers "${NS}" "${PROVIDER}"
echo ""

echo "Selecting source VMs..."
if [[ -z "${RHEL_VM}" ]]; then
  RHEL_VM="$(pick_rhel_vm "${NS}" "${PROVIDER}")"
fi
if [[ -z "${WIN_VM}" ]]; then
  WIN_VM="$(pick_win_vm "${NS}" "${PROVIDER}")"
fi

if [[ -z "${RHEL_VM}" || "${RHEL_VM}" == "null" ]]; then
  echo "ERROR: No mtv-func RHEL VMs found in inventory." >&2
  exit 1
fi
if [[ -z "${WIN_VM}" || "${WIN_VM}" == "null" ]]; then
  echo "ERROR: No mtv-func Windows VMs found in inventory." >&2
  exit 1
fi

echo "RHEL_VM=${RHEL_VM}"
echo "WIN_VM=${WIN_VM}"
echo ""

# Sequential only — never overlap migrations.
rc_rhel=0
rc_win=0
echo "Running RHEL then Windows sequentially (one plan at a time)."
run_cold_smoke_plan "rhel" "${RHEL_VM}" "plan-smoke-rhel" "${NS}" "${PROVIDER}" || rc_rhel=$?

if [[ "${rc_rhel}" -ne 0 ]]; then
  echo "RHEL failed (exit=${rc_rhel}) — skipping Windows; leaving ${NS} for debugging."
  SKIP_CLEANUP=true
else
  echo "RHEL finished OK. Releasing RHEL plan (free memory) before Windows..."
  release_plan_pods "plan-smoke-rhel" "${NS}"
  # Stop migrated RHEL guest if still running to free worker RAM.
  oc virt stop "${RHEL_VM}" -n "${NS}" 2>/dev/null \
    || oc delete vmi "${RHEL_VM}" -n "${NS}" --ignore-not-found 2>/dev/null \
    || true
  echo ""
  run_cold_smoke_plan "win" "${WIN_VM}" "plan-smoke-win" "${NS}" "${PROVIDER}" || rc_win=$?
fi

echo "=========================================="
echo "SUMMARY"
echo "=========================================="
echo "Image:  ${KC_V2V_IMAGE}"
echo "RHEL (${RHEL_VM}): exit=${rc_rhel}"
echo "WIN  (${WIN_VM}): exit=${rc_win}"
echo "Namespace: ${NS} (SKIP_CLEANUP=${SKIP_CLEANUP})"
echo ""

if [[ "${rc_rhel}" -eq 0 && "${rc_win}" -eq 0 ]]; then
  echo "TEST PASSED: MTV-kc-v2v smoke"
  exit 0
fi

echo "TEST FAILED: MTV-kc-v2v smoke (rhel=${rc_rhel} win=${rc_win})"
exit 1
