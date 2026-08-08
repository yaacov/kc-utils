#!/usr/bin/env bash
# Shared helpers for MTV / kc-v2v cluster scenario tests.
# Source from sibling scripts: source "$(dirname "$0")/lib/common.sh"

LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "${LIB_DIR}/.." && pwd)"

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

# Save / restore helpers for cluster settings touched by benchmark runs.
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
  if [[ "${KEEP_IMAGE_SETTING:-true}" == "true" ]]; then
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

# Clear the override so Forklift uses the operator default virt-v2v image.
clear_virt_v2v_image() {
  echo "Clearing virt_v2v_image_fqin (operator default)"
  oc mtv settings set --setting virt_v2v_image_fqin --value "" 2>/dev/null \
    || oc mtv settings unset --setting virt_v2v_image_fqin 2>/dev/null \
    || true
}

disable_windows_wait_for_reboot() {
  echo "Setting feature_windows_wait_for_reboot=false"
  oc mtv settings set --setting feature_windows_wait_for_reboot --value false
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
    --query "select name where name ~= 'mtv-func.*' and name ~= 'rhel'" \
    --output json 2>/dev/null | jq -r '.[].name' | sort | head -1
}

pick_win_vm() {
  local ns="$1"
  local provider="$2"
  oc mtv get inventory vm --provider "${provider}" -n "${ns}" \
    --query "select name where name ~= 'mtv-func.*' and name ~= 'win'" \
    --output json 2>/dev/null | jq -r '.[].name' | sort | head -1
}

# Providers can be Ready before inventory is fully populated. Retry picks.
wait_for_mtv_func_vms() {
  local ns="$1"
  local provider="$2"
  local attempts="${3:-60}"
  local sleep_s="${4:-5}"
  local i rhel win
  echo "Waiting for mtv-func RHEL + Windows inventory VMs..."
  for i in $(seq 1 "${attempts}"); do
    rhel="$(pick_rhel_vm "${ns}" "${provider}")"
    win="$(pick_win_vm "${ns}" "${provider}")"
    if [[ -n "${rhel}" && "${rhel}" != "null" && -n "${win}" && "${win}" != "null" ]]; then
      echo "Inventory ready (attempt ${i}/${attempts}): rhel=${rhel} win=${win}"
      return 0
    fi
    sleep "${sleep_s}"
  done
  echo "ERROR: timed out waiting for mtv-func RHEL/Windows inventory VMs." >&2
  return 1
}
