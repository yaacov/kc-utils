#!/usr/bin/env bash
# Shared cleanup helpers for MTV / kc-v2v cluster scenario tests.
# Source after lib/common.sh.

BENCHMARK_PLAN="${BENCHMARK_PLAN:-plan-bench}"
# Legacy sequential plan names (still deleted so leftover CRs clean up).
BENCHMARK_RHEL_PLAN="${BENCHMARK_RHEL_PLAN:-plan-bench-rhel}"
BENCHMARK_WIN_PLAN="${BENCHMARK_WIN_PLAN:-plan-bench-win}"

preflight_oc_mtv() {
  require_bin oc
  if ! oc mtv settings get --setting vddk_image &>/dev/null; then
    echo "ERROR: Cannot read MTV settings. Is MTV installed on this cluster?" >&2
    exit 1
  fi
}

wait_for_namespace_gone() {
  local ns="$1"
  local attempts="${2:-60}"
  local sleep_s="${3:-2}"
  local i

  for i in $(seq 1 "${attempts}"); do
    oc get ns "${ns}" >/dev/null 2>&1 || return 0
    sleep "${sleep_s}"
  done
  echo "WARNING: namespace '${ns}' still present after ${attempts} attempts." >&2
  return 1
}

delete_benchmark_namespace() {
  local ns="$1"
  local wait="${2:-true}"

  echo "Deleting benchmark namespace '${ns}'..."
  oc delete namespace "${ns}" --ignore-not-found 2>/dev/null || true
  if [[ "${wait}" == "true" ]]; then
    wait_for_namespace_gone "${ns}"
  fi
}

create_benchmark_namespace() {
  local ns="$1"
  echo "Creating benchmark namespace '${ns}'..."
  oc create namespace "${ns}"
}

# Delete NS if present, wait until gone, recreate.
fresh_namespace() {
  local ns="$1"
  delete_benchmark_namespace "${ns}" true
  create_benchmark_namespace "${ns}"
}

release_plan_pods() {
  local plan="$1"
  local ns="${2:-${NS}}"

  echo "Deleting plan '${plan}' in namespace '${ns}'..."
  oc mtv delete plan --name "${plan}" -n "${ns}" 2>/dev/null || true
  local i left
  for i in $(seq 1 60); do
    left="$(oc get pods -n "${ns}" --no-headers 2>/dev/null | wc -l | tr -d ' ')"
    [[ "${left}" == "0" ]] && break
    sleep 5
  done
}

stop_migrated_vm() {
  local vm="$1"
  local ns="${2:-${NS}}"

  [[ -n "${vm}" ]] || return 0
  echo "Stopping migrated VM '${vm}' in namespace '${ns}'..."
  oc virt stop "${vm}" -n "${ns}" 2>/dev/null \
    || oc delete vmi "${vm}" -n "${ns}" --ignore-not-found 2>/dev/null \
    || true
}

release_rhel_resources() {
  local ns="${1:-${NS}}"
  local rhel_vm="${2:-${RHEL_VM:-}}"

  release_plan_pods "${BENCHMARK_RHEL_PLAN}" "${ns}"
  stop_migrated_vm "${rhel_vm}" "${ns}"
  echo "Namespace pod count after RHEL release: $(oc get pods -n "${ns}" --no-headers 2>/dev/null | wc -l | tr -d ' ')"
}

release_benchmark_plans() {
  local ns="${1:-${NS}}"
  release_plan_pods "${BENCHMARK_PLAN}" "${ns}"
  release_plan_pods "${BENCHMARK_RHEL_PLAN}" "${ns}"
  release_plan_pods "${BENCHMARK_WIN_PLAN}" "${ns}"
}

# Standalone reset: clear benchmark MTV overrides without saved values.
reset_mtv_benchmark_settings() {
  echo "Resetting MTV benchmark settings..."
  oc mtv settings set --setting virt_v2v_image_fqin --value "" 2>/dev/null \
    || oc mtv settings unset --setting virt_v2v_image_fqin 2>/dev/null \
    || true
  oc mtv settings set --setting feature_windows_wait_for_reboot --value "" 2>/dev/null \
    || oc mtv settings unset --setting feature_windows_wait_for_reboot 2>/dev/null \
    || true
}

benchmark_exit_cleanup() {
  restore_mtv_settings

  if [[ "${SKIP_CLEANUP:-true}" == "true" ]]; then
    echo "SKIP_CLEANUP=true -- leaving namespace '${NS}'."
    return 0
  fi

  delete_benchmark_namespace "${NS}" true
}

run_benchmark_cleanup() {
  local scope="${1:-all}"
  local ns="${NS:-mtv-kc-v2v-bench}"

  preflight_oc_mtv

  case "${scope}" in
    all)
      delete_benchmark_namespace "${ns}" true
      reset_mtv_benchmark_settings
      ;;
    namespace)
      delete_benchmark_namespace "${ns}" true
      ;;
    settings)
      reset_mtv_benchmark_settings
      ;;
    rhel)
      release_rhel_resources "${ns}" "${RHEL_VM:-}"
      ;;
    plans)
      release_benchmark_plans "${ns}"
      ;;
    *)
      echo "ERROR: unknown cleanup scope '${scope}'." >&2
      return 1
      ;;
  esac
}
