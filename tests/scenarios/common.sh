#!/usr/bin/env bash
# Shared helpers for MTV / kc-v2v cluster scenario tests.
# Source from scripts in this directory: source "$(dirname "$0")/common.sh"

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

fmt_dur() {
  local s="$1"
  printf "%dm%02ds" $((s / 60)) $((s % 60))
}

get_plan_status() {
  local plan="$1"
  local ns="${2:-${NS}}"
  oc mtv get plan --name "${plan}" -n "${ns}" --output json 2>/dev/null \
    | jq -r 'if type == "array" then .[0].status // "Unknown" else .status // "Unknown" end' 2>/dev/null \
    || echo "Unknown"
}

get_plan_vms() {
  local plan="$1"
  local ns="${2:-${NS}}"
  oc mtv get plan --name "${plan}" -n "${ns}" --vms 2>&1
}

# Poll until plan reaches a terminal status or times out.
# Uses INTERVAL / MAX_ATTEMPTS from the caller environment.
# Sets LAST_PLAN_STATUS. Returns 0 on terminal status, 1 on timeout.
wait_for_plan() {
  local plan="$1"
  local start_time="$2"
  local ns="${3:-${NS}}"
  local attempt=0
  local status
  local interval="${INTERVAL:-15}"
  local max_attempts="${MAX_ATTEMPTS:-120}"

  echo "--- Monitoring plan: ${plan} ---"

  while [ "${attempt}" -lt "${max_attempts}" ]; do
    attempt=$((attempt + 1))
    status="$(get_plan_status "${plan}" "${ns}")"
    local elapsed=$(( $(date +%s) - start_time ))

    printf "  [%02d/%02d] %s  Status: %s\n" \
      "${attempt}" "${max_attempts}" \
      "$(fmt_dur "${elapsed}")" \
      "${status}"

    case "${status}" in
      Completed|Succeeded|Failed|Error|Cancelled)
        LAST_PLAN_STATUS="${status}"
        echo ""
        echo "  Plan ${plan} finished:"
        echo "    Status:   ${status}"
        echo "    Duration: $(fmt_dur "$(( $(date +%s) - start_time ))")"
        echo ""
        echo "  VM-level details:"
        get_plan_vms "${plan}" "${ns}"
        echo ""
        return 0
        ;;
    esac

    sleep "${interval}"
  done

  LAST_PLAN_STATUS="$(get_plan_status "${plan}" "${ns}")"
  echo ""
  echo "  Plan ${plan} timed out after $((max_attempts * interval))s"
  echo "    Status:   ${LAST_PLAN_STATUS}"
  echo "    Duration: $(fmt_dur "$(( $(date +%s) - start_time ))")"
  echo ""
  echo "  VM-level details:"
  get_plan_vms "${plan}" "${ns}"
  echo ""
  return 1
}

# Read a single MTV setting value (last field of the get output table).
mtv_setting_get() {
  local setting="$1"
  oc mtv settings get --setting "${setting}" 2>/dev/null \
    | tail -1 | awk '{print $NF}'
}

# True if setting looks unset / empty.
mtv_setting_is_empty() {
  local v="$1"
  [[ -z "${v}" || "${v}" == "<none>" || "${v}" == "VALUE" || "${v}" == "null" ]]
}

require_bin() {
  local bin="$1"
  if ! command -v "${bin}" &>/dev/null; then
    echo "ERROR: '${bin}' not found in PATH." >&2
    exit 1
  fi
}

# Verify oc, jq, GOVC_*, MTV, and VDDK image.
preflight_mtv_cluster() {
  require_bin oc
  require_bin jq

  if [[ -z "${GOVC_URL:-}" || -z "${GOVC_USERNAME:-}" || -z "${GOVC_PASSWORD:-}" ]]; then
    echo "ERROR: GOVC_URL, GOVC_USERNAME, and GOVC_PASSWORD must be set." >&2
    exit 1
  fi

  if ! oc mtv settings get --setting vddk_image &>/dev/null; then
    echo "ERROR: Cannot read MTV settings. Is MTV installed on this cluster?" >&2
    exit 1
  fi

  local vddk
  vddk="$(mtv_setting_get vddk_image)"
  if mtv_setting_is_empty "${vddk}"; then
    echo "ERROR: VDDK image not configured. Required for vSphere migrations." >&2
    echo "Set it with: oc mtv settings set --setting vddk_image --value <image>" >&2
    exit 1
  fi
  echo "MTV controller found. VDDK image: ${vddk}"
}

# Save / restore helpers for cluster settings touched by smoke tests.
# Call save_mtv_settings once, then restore_mtv_settings from EXIT trap.
SAVED_VIRT_V2V_IMAGE=""
SAVED_WAIT_FOR_REBOOT=""
_SETTINGS_SAVED=false

save_mtv_settings() {
  SAVED_VIRT_V2V_IMAGE="$(mtv_setting_get virt_v2v_image_fqin || true)"
  SAVED_WAIT_FOR_REBOOT="$(mtv_setting_get feature_windows_wait_for_reboot || true)"
  _SETTINGS_SAVED=true
  echo "Saved virt_v2v_image_fqin=${SAVED_VIRT_V2V_IMAGE:-<empty>}"
  echo "Saved feature_windows_wait_for_reboot=${SAVED_WAIT_FOR_REBOOT:-<empty>}"
}

restore_mtv_settings() {
  if [[ "${_SETTINGS_SAVED}" != "true" ]]; then
    return 0
  fi
  if [[ "${KEEP_IMAGE_SETTING:-false}" == "true" ]]; then
    echo "KEEP_IMAGE_SETTING=true — leaving cluster settings as configured by this run."
    return 0
  fi

  echo "Restoring MTV settings..."
  if mtv_setting_is_empty "${SAVED_VIRT_V2V_IMAGE}"; then
    oc mtv settings set --setting virt_v2v_image_fqin --value "" 2>/dev/null \
      || oc mtv settings unset --setting virt_v2v_image_fqin 2>/dev/null \
      || true
  else
    oc mtv settings set --setting virt_v2v_image_fqin --value "${SAVED_VIRT_V2V_IMAGE}" 2>/dev/null || true
  fi

  if mtv_setting_is_empty "${SAVED_WAIT_FOR_REBOOT}"; then
    oc mtv settings set --setting feature_windows_wait_for_reboot --value "" 2>/dev/null \
      || oc mtv settings unset --setting feature_windows_wait_for_reboot 2>/dev/null \
      || true
  else
    oc mtv settings set --setting feature_windows_wait_for_reboot --value "${SAVED_WAIT_FOR_REBOOT}" 2>/dev/null || true
  fi
}

set_virt_v2v_image() {
  local image="$1"
  if [[ -z "${image}" ]]; then
    echo "ERROR: conversion image FQIN is empty. Set KC_V2V_IMAGE." >&2
    exit 1
  fi
  echo "Setting virt_v2v_image_fqin=${image}"
  oc mtv settings set --setting virt_v2v_image_fqin --value "${image}"
}

disable_windows_wait_for_reboot() {
  echo "Setting feature_windows_wait_for_reboot=false"
  oc mtv settings set --setting feature_windows_wait_for_reboot --value false
}

# Delete NS if present, wait until gone, recreate.
fresh_namespace() {
  local ns="$1"
  oc delete namespace "${ns}" --ignore-not-found 2>/dev/null || true
  local i
  for i in $(seq 1 60); do
    oc get ns "${ns}" >/dev/null 2>&1 || break
    sleep 2
  done
  oc create namespace "${ns}"
}

create_vsphere_and_host_providers() {
  local ns="$1"
  local provider="${2:-vsphere-test}"

  echo "Creating vSphere provider (url=https://${GOVC_URL}/sdk)..."
  oc mtv create provider --name "${provider}" --type vsphere \
    --url "https://${GOVC_URL}/sdk" \
    --username "${GOVC_USERNAME}" \
    --password "${GOVC_PASSWORD}" \
    --provider-insecure-skip-tls \
    -n "${ns}"

  echo "Creating OpenShift provider..."
  oc mtv create provider --name host --type openshift -n "${ns}"

  echo "Waiting for providers..."
  oc wait "provider.forklift.konveyor.io/${provider}" -n "${ns}" \
    --for=condition=Ready --timeout=300s
  oc wait "provider.forklift.konveyor.io/host" -n "${ns}" \
    --for=condition=Ready --timeout=300s
  echo "Providers ready."
}

pick_rhel_vm() {
  local ns="$1"
  local provider="$2"
  oc mtv get inventory vm --provider "${provider}" -n "${ns}" \
    --query "select name where name ~= 'mtv-func.*' and (name ~= 'rhel' or guestId ~= 'rhel')" \
    --output json 2>/dev/null | jq -r '.[].name' | sort | head -1
}

pick_win_vm() {
  local ns="$1"
  local provider="$2"
  oc mtv get inventory vm --provider "${provider}" -n "${ns}" \
    --query "select name where name ~= 'mtv-func.*' and (name ~= 'win' or guestId ~= 'win')" \
    --output json 2>/dev/null | jq -r '.[].name' | sort | head -1
}

# Create + start a cold plan; wait for terminal status (no metrics sampling).
# Sets LAST_PLAN_STATUS. Returns 0 on Completed/Succeeded, 1 otherwise.
run_cold_smoke_plan() {
  local label="$1"
  local vm="$2"
  local plan="$3"
  local ns="${4:-${NS}}"
  local provider="${5:-${PROVIDER}}"

  echo "=========================================="
  echo "RUN ${label}: VM=${vm} plan=${plan}"
  echo "=========================================="

  oc mtv delete plan --name "${plan}" -n "${ns}" 2>/dev/null || true
  sleep 2

  if ! oc mtv create plan --name "${plan}" --source "${provider}" --target host \
    --vms "${vm}" \
    --run-preflight-inspection false \
    -n "${ns}"; then
    echo "ERROR: failed to create plan ${plan}" >&2
    return 1
  fi

  echo "Waiting for plan Ready..."
  if ! oc wait "plan.forklift.konveyor.io/${plan}" -n "${ns}" \
    --for=condition=Ready --timeout=300s; then
    echo "ERROR: plan ${plan} did not become Ready" >&2
    return 1
  fi
  echo "Plan ready."

  local start_time
  start_time=$(date +%s)
  if ! oc mtv start plan --name "${plan}" -n "${ns}"; then
    echo "ERROR: failed to start plan ${plan}" >&2
    return 1
  fi
  echo "Migration started."
  echo ""

  wait_for_plan "${plan}" "${start_time}" "${ns}" || true
  local status="${LAST_PLAN_STATUS}"
  local duration=$(( $(date +%s) - start_time ))

  echo "RESULT ${label}: status=${status} total=$(fmt_dur "${duration}") vm=${vm}"
  echo ""

  if [[ "${status}" == "Completed" || "${status}" == "Succeeded" ]]; then
    return 0
  fi
  return 1
}

release_plan_pods() {
  local plan="$1"
  local ns="${2:-${NS}}"
  oc mtv delete plan --name "${plan}" -n "${ns}" 2>/dev/null || true
  local i left
  for i in $(seq 1 60); do
    left="$(oc get pods -n "${ns}" --no-headers 2>/dev/null | wc -l | tr -d ' ')"
    [[ "${left}" == "0" ]] && break
    sleep 5
  done
}
