#!/usr/bin/env bash
# Shared helpers for MTV / kc-v2v cluster scenario tests.
# Source from sibling scripts: source "$(dirname "$0")/lib/common.sh"

LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "${LIB_DIR}/.." && pwd)"

# Load tests/scenarios/.env (required). Copy from .env.example on first setup.
load_scenario_env() {
  local env_file="${SCENARIO_DIR}/.env"
  if [[ ! -f "${env_file}" ]]; then
    echo "ERROR: ${env_file} not found." >&2
    echo "Tip: cp tests/scenarios/.env.example tests/scenarios/.env" >&2
    exit 1
  fi
  set -a
  # shellcheck source=/dev/null
  source "${env_file}"
  set +a
}

require_env() {
  local var="$1"
  if [[ -z "${!var:-}" ]]; then
    echo "ERROR: ${var} must be set in tests/scenarios/.env" >&2
    exit 1
  fi
}

load_scenario_env

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

# True if setting looks unset / empty (MTV prints "(empty)" for cleared values).
mtv_setting_is_empty() {
  local v="$1"
  [[ -z "${v}" || "${v}" == "<none>" || "${v}" == "(empty)" || "${v}" == "VALUE" || "${v}" == "null" || "${v}" == "-" ]]
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
    echo "ERROR: GOVC_URL, GOVC_USERNAME, and GOVC_PASSWORD must be set in tests/scenarios/.env" >&2
    exit 1
  fi

  require_env NS
  require_env PROVIDER

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
REF_VIRT_V2V_IMAGE=""
_SETTINGS_SAVED=false

image_sync_log() {
  if [[ -n "${SUMMARY_LOG:-}" ]]; then
    echo "$*" | tee -a "${SUMMARY_LOG}"
  else
    echo "$*"
  fi
}

save_mtv_settings() {
  SAVED_VIRT_V2V_IMAGE="$(mtv_setting_get virt_v2v_image_fqin || true)"
  SAVED_WAIT_FOR_REBOOT="$(mtv_setting_get feature_windows_wait_for_reboot || true)"
  _SETTINGS_SAVED=true
  echo "Saved virt_v2v_image_fqin=${SAVED_VIRT_V2V_IMAGE:-<empty>}"
  echo "Saved feature_windows_wait_for_reboot=${SAVED_WAIT_FOR_REBOOT:-<empty>}"
}

# Operator-default virt-v2v image for the ref leg. Optional REF_V2V_IMAGE in .env;
# otherwise inferred from the controller deploy when it differs from KC_V2V_IMAGE.
capture_ref_virt_v2v_image() {
  local deploy_ref ns deploy deploy_img setting
  REF_VIRT_V2V_IMAGE="${REF_V2V_IMAGE:-}"
  if [[ -n "${REF_VIRT_V2V_IMAGE}" ]]; then
    echo "REF virt-v2v image (from .env): ${REF_VIRT_V2V_IMAGE}"
    return 0
  fi
  deploy_ref="$(forklift_controller_deploy_ref)"
  [[ -n "${deploy_ref}" ]] || return 0
  ns="${deploy_ref%%/*}"
  deploy="${deploy_ref#*/}"
  deploy_img="$(controller_deploy_virt_v2v_image "${ns}" "${deploy}")"
  setting="$(mtv_setting_get virt_v2v_image_fqin || true)"
  if mtv_setting_is_empty "${setting}" && [[ -n "${deploy_img}" ]]; then
    REF_VIRT_V2V_IMAGE="${deploy_img}"
  elif [[ -n "${deploy_img}" && "${deploy_img}" != "${KC_V2V_IMAGE:-}" ]]; then
    REF_VIRT_V2V_IMAGE="${deploy_img}"
  fi
  if [[ -n "${REF_VIRT_V2V_IMAGE}" ]]; then
    echo "REF virt-v2v image (inferred): ${REF_VIRT_V2V_IMAGE}"
  fi
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
    echo "ERROR: conversion image FQIN is empty. Set KC_V2V_IMAGE in tests/scenarios/.env." >&2
    exit 1
  fi
  image_sync_log "Setting virt_v2v_image_fqin=${image}"
  oc mtv settings set --setting virt_v2v_image_fqin --value "${image}"
  wait_for_virt_v2v_image_sync "${image}"
}

# Clear the override so Forklift uses the operator default virt-v2v image.
clear_virt_v2v_image() {
  image_sync_log "Clearing virt_v2v_image_fqin (operator default)"
  oc mtv settings set --setting virt_v2v_image_fqin --value "" 2>/dev/null \
    || oc mtv settings unset --setting virt_v2v_image_fqin 2>/dev/null \
    || true
  if [[ -n "${REF_VIRT_V2V_IMAGE:-}" ]]; then
    wait_for_virt_v2v_image_sync "${REF_VIRT_V2V_IMAGE}"
  else
    wait_for_virt_v2v_image_sync "" "${KC_V2V_IMAGE:-}"
  fi
}

# Conversion pods use VIRT_V2V_IMAGE from the forklift-controller deployment
# template. After set/clear, wait until rollout is done and deploy + MTV setting
# match the expected converter image.
wait_for_virt_v2v_image_sync() {
  local expected="$1"
  local not_image="${2:-}"
  local attempts="${3:-90}"
  local sleep_s="${4:-5}"
  local i current deploy_ref ns deploy deploy_image rollout_ok setting
  deploy_ref="$(forklift_controller_deploy_ref)"
  if [[ -z "${deploy_ref}" ]]; then
    image_sync_log "ERROR: could not find forklift-controller deployment; cannot verify VIRT_V2V_IMAGE." >&2
    return 1
  fi
  ns="${deploy_ref%%/*}"
  deploy="${deploy_ref#*/}"
  image_sync_log "Waiting for forklift-controller rollout + VIRT_V2V_IMAGE sync (ns=${ns} deploy=${deploy})..."
  for i in $(seq 1 "${attempts}"); do
    rollout_ok=false
    if forklift_controller_rollout_complete "${ns}" "${deploy}"; then
      rollout_ok=true
    fi
    deploy_image="$(controller_deploy_virt_v2v_image "${ns}" "${deploy}")"
    current="$(controller_pod_virt_v2v_image "${ns}" "${deploy}")"
    setting="$(mtv_setting_get virt_v2v_image_fqin || true)"
    if [[ -n "${expected}" ]]; then
      if [[ "${rollout_ok}" == "true" && "${deploy_image}" == "${expected}" ]]; then
        if [[ "${expected}" == "${KC_V2V_IMAGE:-}" ]]; then
          if [[ "${setting}" == "${expected}" ]]; then
            image_sync_log "Controller ready: deploy=${deploy_image} setting=${setting} pod=${current:-<n/a>} (attempt ${i}/${attempts})"
            return 0
          fi
        elif mtv_setting_is_empty "${setting}"; then
          image_sync_log "Controller ready: deploy=${deploy_image} setting=<empty> pod=${current:-<n/a>} (attempt ${i}/${attempts})"
          return 0
        fi
      fi
      image_sync_log "  [${i}/${attempts}] rollout=${rollout_ok} deploy=${deploy_image:-<empty>} setting=${setting:-<empty>} pod=${current:-<empty>} want=${expected}"
    elif [[ "${rollout_ok}" == "true" && -n "${deploy_image}" && "${deploy_image}" != "${not_image}" ]]; then
      if mtv_setting_is_empty "${setting}"; then
        image_sync_log "Controller ready: deploy=${deploy_image} setting=<empty> pod=${current:-<n/a>} (attempt ${i}/${attempts})"
        return 0
      fi
      image_sync_log "  [${i}/${attempts}] rollout=${rollout_ok} deploy=${deploy_image:-<empty>} setting=${setting:-<empty>} pod=${current:-<empty>} want=not ${not_image:-<kc>}"
    else
      image_sync_log "  [${i}/${attempts}] rollout=${rollout_ok} deploy=${deploy_image:-<empty>} setting=${setting:-<empty>} pod=${current:-<empty>} want=not ${not_image:-<kc>}"
    fi
    sleep "${sleep_s}"
  done
  image_sync_log "ERROR: timed out waiting for forklift-controller (want='${expected:-<not ${not_image}>}', deploy='${deploy_image:-<empty>}', setting='${setting:-<empty>}', pod='${current:-<empty>}')." >&2
  return 1
}

# Hard gate before creating a plan: conversion pods follow deploy VIRT_V2V_IMAGE.
verify_converter_image_ready() {
  local converter="$1"
  local deploy_ref ns deploy deploy_image setting
  deploy_ref="$(forklift_controller_deploy_ref)"
  if [[ -z "${deploy_ref}" ]]; then
    image_sync_log "ERROR: forklift-controller deployment not found" >&2
    return 1
  fi
  ns="${deploy_ref%%/*}"
  deploy="${deploy_ref#*/}"
  deploy_image="$(controller_deploy_virt_v2v_image "${ns}" "${deploy}")"
  setting="$(mtv_setting_get virt_v2v_image_fqin || true)"
  case "${converter}" in
    kc)
      if [[ "${deploy_image}" != "${KC_V2V_IMAGE}" ]]; then
        image_sync_log "ERROR: kc leg blocked — forklift-controller VIRT_V2V_IMAGE(deploy)=${deploy_image:-<empty>} but need ${KC_V2V_IMAGE} (virt_v2v_image_fqin=${setting:-<empty>}). Conversion pods use the controller deploy env, not the MTV setting alone." >&2
        return 1
      fi
      if [[ "${setting}" != "${KC_V2V_IMAGE}" ]]; then
        image_sync_log "ERROR: kc leg blocked — virt_v2v_image_fqin=${setting:-<empty>} != ${KC_V2V_IMAGE}" >&2
        return 1
      fi
      ;;
    ref)
      if ! mtv_setting_is_empty "${setting}"; then
        image_sync_log "ERROR: ref leg blocked — virt_v2v_image_fqin still ${setting}" >&2
        return 1
      fi
      if [[ -n "${REF_VIRT_V2V_IMAGE:-}" ]]; then
        if [[ "${deploy_image}" != "${REF_VIRT_V2V_IMAGE}" ]]; then
          image_sync_log "ERROR: ref leg blocked — deploy=${deploy_image:-<empty>} want ${REF_VIRT_V2V_IMAGE}" >&2
          return 1
        fi
      elif [[ -z "${deploy_image}" || "${deploy_image}" == "${KC_V2V_IMAGE:-}" ]]; then
        image_sync_log "ERROR: ref leg blocked — deploy=${deploy_image:-<empty>} still matches kc or is unset; set REF_V2V_IMAGE in .env" >&2
        return 1
      fi
      ;;
    *)
      image_sync_log "ERROR: unknown converter '${converter}'" >&2
      return 1
      ;;
  esac
  image_sync_log "Converter image OK (${converter}): deploy=${deploy_image} setting=${setting:-<empty>}"
  return 0
}

forklift_controller_deploy_ref() {
  oc get deploy -A -o json 2>/dev/null \
    | jq -r '
        [
          .items[]
          | select(.metadata.name | test("^(forklift-controller|konveyor-forklift)$"))
          | "\(.metadata.namespace)/\(.metadata.name)"
        ] | .[0] // empty
      ' 2>/dev/null
}

forklift_controller_namespace() {
  local ref
  ref="$(forklift_controller_deploy_ref)"
  [[ -n "${ref}" ]] || return 0
  printf '%s' "${ref%%/*}"
}

forklift_controller_rollout_complete() {
  local ns="$1" deploy="$2"
  oc get "deployment/${deploy}" -n "${ns}" -o json 2>/dev/null \
    | jq -e '
        (.status.observedGeneration // 0) >= (.metadata.generation // 0)
        and (.status.updatedReplicas // 0) == (.spec.replicas // 1)
        and (.status.readyReplicas // 0) == (.spec.replicas // 1)
        and (.status.availableReplicas // 0) == (.spec.replicas // 1)
      ' >/dev/null 2>&1
}

controller_deploy_virt_v2v_image() {
  local ns="$1" deploy="$2"
  oc get "deployment/${deploy}" -n "${ns}" -o json 2>/dev/null \
    | jq -r '
        [
          .spec.template.spec.containers[]?.env[]?
          | select(.name == "VIRT_V2V_IMAGE")
          | .value // empty
        ] | .[0] // empty
      ' 2>/dev/null
}

controller_pod_virt_v2v_image() {
  local ns="$1" deploy="$2"
  local selector
  selector="$(oc get "deployment/${deploy}" -n "${ns}" -o json 2>/dev/null \
    | jq -r '
        .spec.selector.matchLabels
        | to_entries
        | map("\(.key)=\(.value)")
        | join(",")
      ' 2>/dev/null)"
  [[ -n "${selector}" ]] || return 0
  oc get pods -n "${ns}" -l "${selector}" -o json 2>/dev/null \
    | jq -r '
        [
          .items[]
          | select(any(.status.conditions[]?; .type == "Ready" and .status == "True"))
          | {ts: .metadata.creationTimestamp, env: [
              .spec.containers[]?.env[]?
              | select(.name == "VIRT_V2V_IMAGE")
              | .value // empty
            ] | .[0] // empty}
        ]
        | sort_by(.ts)
        | last
        | .env // empty
      ' 2>/dev/null
}

disable_windows_wait_for_reboot() {
  echo "Setting feature_windows_wait_for_reboot=false"
  oc mtv settings set --setting feature_windows_wait_for_reboot --value false
}

# Fetch vCenter TLS certificate for secure provider creation.
fetch_vsphere_cacert() {
  local host="$1"
  local cert_file
  cert_file="$(mktemp "${TMPDIR:-/tmp}/vsphere-cacert.XXXXXX")" || {
    echo "ERROR: failed to create temp file for vSphere CA cert" >&2
    exit 1
  }
  if ! echo | openssl s_client -showcerts -connect "${host}:443" 2>/dev/null \
      | openssl x509 >"${cert_file}"; then
    rm -f "${cert_file}"
    echo "ERROR: failed to fetch vSphere CA cert from ${host}:443" >&2
    exit 1
  fi
  if [[ ! -s "${cert_file}" ]]; then
    rm -f "${cert_file}"
    echo "ERROR: empty vSphere CA cert from ${host}:443" >&2
    exit 1
  fi
  echo "${cert_file}"
}

create_vsphere_and_host_providers() {
  local ns="$1"
  local provider="${2:-vsphere-test}"
  local -a tls_args=()

  if [[ "${PROVIDER_INSECURE_SKIP_TLS:-false}" == "true" ]]; then
    echo "Creating vSphere provider (url=https://${GOVC_URL}/sdk, insecure TLS)..."
    tls_args=(--provider-insecure-skip-tls)
  else
    echo "Creating vSphere provider (url=https://${GOVC_URL}/sdk, CA from ${GOVC_URL}:443)..."
    require_bin openssl
    local cacert_file
    cacert_file="$(fetch_vsphere_cacert "${GOVC_URL}")"
    tls_args=(--cacert "@${cacert_file}")
  fi

  oc mtv create provider --name "${provider}" --type vsphere \
    --url "https://${GOVC_URL}/sdk" \
    --username "${GOVC_USERNAME}" \
    --password "${GOVC_PASSWORD}" \
    "${tls_args[@]}" \
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

list_mtv_func_vms() {
  local ns="$1"
  local provider="$2"
  oc mtv get inventory vm --provider "${provider}" -n "${ns}" \
    --query "select name where name ~= 'mtv-func.*'" \
    --output json 2>/dev/null | jq -r '.[] | .name // empty' | grep -v '^$' | sort
}

# Providers can be Ready before inventory is fully populated. Retry until
# at least `need` mtv-func* VMs appear.
wait_for_mtv_func_vms() {
  local ns="$1"
  local provider="$2"
  local need="${3:-3}"
  local attempts="${4:-60}"
  local sleep_s="${5:-5}"
  local i names count
  echo "Waiting for ${need} mtv-func* inventory VMs..."
  for i in $(seq 1 "${attempts}"); do
    names="$(list_mtv_func_vms "${ns}" "${provider}")"
    count="$(printf '%s\n' "${names}" | grep -c '.' || true)"
    if [[ "${count}" -ge "${need}" ]]; then
      echo "Inventory ready (attempt ${i}/${attempts}):"
      printf '%s\n' "${names}" | sed 's/^/  /'
      return 0
    fi
    sleep "${sleep_s}"
  done
  echo "ERROR: timed out waiting for ${need} mtv-func* inventory VMs (found ${count:-0})." >&2
  return 1
}
