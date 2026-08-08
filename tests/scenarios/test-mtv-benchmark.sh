#!/usr/bin/env bash
#
# MTV cold-migration benchmark: RHEL then Windows, with full artifact capture
# (summary log, mem/CPU/net CSV, conversion-pod logs, pipeline timings).
#
# Modes (MODE):
#   kc       Run once with KC_V2V_IMAGE (independent kc-v2v benchmark). Default.
#   compare  Run twice: kc-v2v then operator-default virt-v2v (full compare).
#
# Prerequisites: oc, oc mtv, jq, GOVC_URL/USERNAME/PASSWORD, VDDK configured,
#   and KC_V2V_IMAGE set to a cluster-pullable kc-v2v FQIN.
#
# Env overrides:
#   MODE                    kc | compare (default kc)
#   KC_V2V_IMAGE            conversion image FQIN (required)
#   RHEL_VM / WIN_VM        source VM names (auto-picked from mtv-func* if unset)
#   NS                      namespace (default mtv-kc-v2v-bench)
#   PROVIDER                vSphere provider name (default vsphere-test)
#   SKIP_CLEANUP            keep NS on exit (default true); use cleanup script to remove later
#   KEEP_BETWEEN_TESTS      leave RHEL plan/pods after RHEL (default true)
#   KEEP_IMAGE_SETTING      leave virt_v2v_image_fqin / reboot flag (default true)
#   DISABLE_WAIT_FOR_REBOOT set feature_windows_wait_for_reboot=false (default true)
#   MEM_INTERVAL            memory sample seconds (default 10)
#   INTERVAL                plan poll seconds (default 10)
#   MAX_ATTEMPTS            plan poll attempts (default 180 => 30m)
#
# Artifacts under runs/:
#   runs/test-mtv-benchmark-<ts>-<converter>.log
#   runs/test-mtv-benchmark-<ts>-<converter>-mem/
#   runs/test-mtv-benchmark-<ts>.html  (dashboard)
# Archive under docs/ref-baseline/runs/ when publishing a comparison.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"
# shellcheck source=lib/cleanup.sh
source "${SCRIPT_DIR}/lib/cleanup.sh"

MODE="${MODE:-kc}"
NS="${NS:-mtv-kc-v2v-bench}"
PROVIDER="${PROVIDER:-vsphere-test}"
RHEL_VM="${RHEL_VM:-}"
WIN_VM="${WIN_VM:-}"
SKIP_CLEANUP="${SKIP_CLEANUP:-true}"
KEEP_BETWEEN_TESTS="${KEEP_BETWEEN_TESTS:-true}"
KEEP_IMAGE_SETTING="${KEEP_IMAGE_SETTING:-true}"
DISABLE_WAIT_FOR_REBOOT="${DISABLE_WAIT_FOR_REBOOT:-true}"
INTERVAL="${INTERVAL:-10}"
MAX_ATTEMPTS="${MAX_ATTEMPTS:-180}"
MEM_INTERVAL="${MEM_INTERVAL:-10}"
KC_V2V_IMAGE="${KC_V2V_IMAGE:-}"
V2V_LABEL="forklift.app=virt-v2v"
RUN_TS="$(date -u +%Y%m%dT%H%M%SZ)"
RUNS_DIR="${SCENARIO_DIR}/runs"
ARTIFACT_PREFIX="${RUNS_DIR}/test-mtv-benchmark-${RUN_TS}"
mkdir -p "${RUNS_DIR}"

# Per-leg state (set by begin_leg)
CONVERTER=""
SUMMARY_LOG=""
MEM_DIR=""

usage() {
  cat <<EOF
Usage: MODE=kc|compare KC_V2V_IMAGE=<fqin> $0

  MODE=kc       Independent kc-v2v benchmark (default)
  MODE=compare  kc-v2v then operator-default virt-v2v

Artifacts: ${ARTIFACT_PREFIX}-<converter>.log and -mem/
EOF
}

log() { echo "$*" | tee -a "${SUMMARY_LOG}"; }

begin_leg() {
  CONVERTER="$1"
  SUMMARY_LOG="${ARTIFACT_PREFIX}-${CONVERTER}.log"
  MEM_DIR="${ARTIFACT_PREFIX}-${CONVERTER}-mem"
  : > "${SUMMARY_LOG}"
  mkdir -p "${MEM_DIR}"
}

pipeline_json() {
  local plan="$1"
  oc get "plan.forklift.konveyor.io/${plan}" -n "${NS}" -o json 2>/dev/null \
    | jq -c '.status.migration.vms[0].pipeline // []'
}

vm_phase() {
  local plan="$1"
  oc get "plan.forklift.konveyor.io/${plan}" -n "${NS}" \
    -o jsonpath='{.status.migration.vms[0].phase}' 2>/dev/null || true
}

summarize_pipeline() {
  local plan="$1"
  log "  Pipeline step timings (${plan}):"
  oc get "plan.forklift.konveyor.io/${plan}" -n "${NS}" -o json 2>/dev/null \
    | jq -r '
      (.status.migration.vms[0].pipeline // [])[]
      | [
          .name,
          (.phase // "-"),
          (if .started then (.started|fromdateiso8601) else empty end),
          (if .completed then (.completed|fromdateiso8601) else empty end)
        ]
      | if length == 4 then
          "    \(.[0]): \(.[1])  dur=\((.[3]-.[2])/60|floor)m\((.[3]-.[2])%60|floor)s  started=\(.[2]|todateiso8601)  completed=\(.[3]|todateiso8601)"
        else
          "    \(.[0]): \(.[1]//"-")  (no completed timestamp)"
        end
      ' 2>/dev/null | tee -a "${SUMMARY_LOG}" || log "    (unable to parse pipeline)"
}

find_v2v_pod() {
  local plan="${1:-}"
  oc get pods -n "${NS}" -l "${V2V_LABEL}" -o json 2>/dev/null \
    | jq -r --arg plan "${plan}" '
        [.items[]
          | select($plan == "" or (.metadata.name | startswith($plan)))
        ]
        | sort_by(.metadata.creationTimestamp)
        | (map(select(.status.phase == "Running")) | last)
          // last
        | .metadata.name // empty
      ' 2>/dev/null || true
}

cgroup_mem_mi() {
  local pod="$1" c name
  for c in virt-v2v $(oc get pod -n "${NS}" "${pod}" -o jsonpath='{.spec.containers[*].name}' 2>/dev/null); do
    name="${c}"
    [[ -n "${name}" ]] || continue
    local bytes
    bytes="$(oc exec -n "${NS}" "${pod}" -c "${name}" -- sh -c '
      if [ -f /sys/fs/cgroup/memory.current ]; then cat /sys/fs/cgroup/memory.current
      elif [ -f /sys/fs/cgroup/memory/memory.usage_in_bytes ]; then cat /sys/fs/cgroup/memory/memory.usage_in_bytes
      else echo 0; fi' 2>/dev/null || echo 0)"
    if [[ "${bytes}" =~ ^[0-9]+$ ]] && [[ "${bytes}" -gt 0 ]]; then
      echo $((bytes / 1024 / 1024))
      return 0
    fi
  done
  echo ""
}

pod_net_bytes() {
  local pod="$1" c name
  for c in virt-v2v $(oc get pod -n "${NS}" "${pod}" -o jsonpath='{.spec.containers[*].name}' 2>/dev/null); do
    name="${c}"
    [[ -n "${name}" ]] || continue
    local result
    result="$(oc exec -n "${NS}" "${pod}" -c "${name}" -- sh -c '
      awk "NR>2 && \$1 !~ /lo:/ {
        split(\$1,a,\":\"); rx+=\$2; tx+=\$10
      } END { print rx+0, tx+0 }
      " /proc/net/dev' 2>/dev/null || true)"
    if [[ -n "${result}" && "${result}" != "0 0" ]]; then
      echo "${result}"
      return 0
    fi
  done
  echo ""
}

monitor_conversion_memory() {
  local plan="$1" label="$2"
  local csv="${MEM_DIR}/${label}-virt-v2v-memory.csv"
  mkdir -p "${MEM_DIR}"
  echo "timestamp_utc,elapsed_s,pod,node,mem_working_set_mi,mem_rss_mi_cgroup,cpu_m,net_rx_bytes,net_tx_bytes,phase" > "${csv}"

  local start_ts pod node mem_top cpu_top rss_mi phase elapsed top_line net_raw net_rx net_tx
  start_ts=$(date +%s)
  log "  Memory/net monitor -> ${csv} (every ${MEM_INTERVAL}s)"

  while true; do
    local status
    status="$(get_plan_status "${plan}")"
    case "${status}" in
      Completed|Succeeded|Failed|Error|Cancelled) break ;;
    esac

    pod="$(find_v2v_pod "${plan}")"
    if [[ -n "${pod}" ]]; then
      node="$(oc get pod -n "${NS}" "${pod}" -o jsonpath='{.spec.nodeName}' 2>/dev/null || true)"
      phase="$(oc get pod -n "${NS}" "${pod}" -o jsonpath='{.status.phase}' 2>/dev/null || true)"
      top_line="$(oc adm top pod -n "${NS}" "${pod}" --no-headers 2>/dev/null || true)"
      cpu_top="$(awk '{print $2}' <<<"${top_line}" | tr -d 'm')"
      mem_top="$(awk '{print $3}' <<<"${top_line}" | tr -d 'Mi')"
      rss_mi="$(cgroup_mem_mi "${pod}")"
      net_raw="$(pod_net_bytes "${pod}")"
      net_rx="$(awk '{print $1}' <<<"${net_raw}")"
      net_tx="$(awk '{print $2}' <<<"${net_raw}")"
      elapsed=$(( $(date +%s) - start_ts ))
      echo "$(date -u +%Y-%m-%dT%H:%M:%SZ),${elapsed},${pod},${node},${mem_top},${rss_mi},${cpu_top},${net_rx},${net_tx},${phase}" >> "${csv}"
      printf "    [mon] +%s pod=%s node=%s top=%sMi cgroup=%sMi cpu=%sm rx=%s tx=%s phase=%s\n" \
        "$(fmt_dur "${elapsed}")" "${pod}" "${node:-?}" "${mem_top:-?}" "${rss_mi:-?}" "${cpu_top:-?}" \
        "${net_rx:-?}" "${net_tx:-?}" "${phase:-?}" \
        | tee -a "${SUMMARY_LOG}"
    fi
    sleep "${MEM_INTERVAL}"
  done

  if [[ "$(wc -l < "${csv}")" -gt 1 ]]; then
    local peak_top peak_cgroup max_rx max_tx
    peak_top="$(awk -F, 'NR>1 && $5!="" {if ($5+0>max) max=$5+0} END{print max+0}' "${csv}")"
    peak_cgroup="$(awk -F, 'NR>1 && $6!="" {if ($6+0>max) max=$6+0} END{print max+0}' "${csv}")"
    max_rx="$(awk -F, 'NR>1 && $8!="" {if ($8+0>max) max=$8+0} END{print max+0}' "${csv}")"
    max_tx="$(awk -F, 'NR>1 && $9!="" {if ($9+0>max) max=$9+0} END{print max+0}' "${csv}")"
    log "  Peak conversion memory: metrics-server=${peak_top}Mi  cgroup=${peak_cgroup}Mi"
    log "  Network totals (cumulative at last sample): rx=$((max_rx / 1024 / 1024))Mi  tx=$((max_tx / 1024 / 1024))Mi"
  else
    log "  No conversion memory samples collected"
  fi
}

monitor_plan() {
  local plan="$1" label="$2"
  local start_time attempt=0 status
  start_time=$(date +%s)
  LAST_PLAN_STATUS="Unknown"

  monitor_conversion_memory "${plan}" "${label}" &
  local mem_pid=$!

  log "--- Monitoring plan ${plan} (${label}) ---"
  local prev_pipe=""
  while [ "${attempt}" -lt "${MAX_ATTEMPTS}" ]; do
    attempt=$((attempt + 1))
    status="$(get_plan_status "${plan}")"
    local elapsed=$(( $(date +%s) - start_time ))
    local phase pipe_summary
    phase="$(vm_phase "${plan}")"
    pipe_summary="$(pipeline_json "${plan}" | jq -r '[.[] | "\(.name)=\(.phase // "-")"] | join(" ")' 2>/dev/null || true)"

    if [[ "${pipe_summary}" != "${prev_pipe}" ]]; then
      log "  [step-change] $(fmt_dur "${elapsed}") phase=${phase:-?} ${pipe_summary}"
      prev_pipe="${pipe_summary}"
    else
      printf "  [%02d/%02d] %s  Status=%s phase=%s\n" \
        "${attempt}" "${MAX_ATTEMPTS}" "$(fmt_dur "${elapsed}")" "${status}" "${phase:-?}" \
        | tee -a "${SUMMARY_LOG}"
    fi

    case "${status}" in
      Completed|Succeeded|Failed|Error|Cancelled)
        break
        ;;
    esac
    sleep "${INTERVAL}"
  done

  wait "${mem_pid}" 2>/dev/null || true

  local duration=$(( $(date +%s) - start_time ))
  LAST_PLAN_STATUS="${status}"
  log ""
  log "  Plan ${plan} finished: status=${status} duration=$(fmt_dur "${duration}")"
  summarize_pipeline "${plan}"
  log "  VM details:"
  oc mtv get plan --name "${plan}" -n "${NS}" --vms 2>&1 | tee -a "${SUMMARY_LOG}" || true
  log ""
}

run_one() {
  local label="$1" vm="$2" plan="$3"
  log "=========================================="
  log "RUN ${label}: VM=${vm} plan=${plan} converter=${CONVERTER}"
  log "=========================================="

  oc mtv delete plan --name "${plan}" -n "${NS}" 2>/dev/null || true
  sleep 2

  oc mtv create plan --name "${plan}" --source "${PROVIDER}" --target host \
    --vms "${vm}" \
    --run-preflight-inspection false \
    -n "${NS}"

  log "Waiting for plan Ready..."
  oc wait "plan.forklift.konveyor.io/${plan}" -n "${NS}" \
    --for=condition=Ready --timeout=300s

  local start_time
  start_time=$(date +%s)
  oc mtv start plan --name "${plan}" -n "${NS}"
  log "Migration started at $(date -u +%Y-%m-%dT%H:%M:%SZ)"

  monitor_plan "${plan}" "${label}"
  local duration=$(( $(date +%s) - start_time ))
  local status="${LAST_PLAN_STATUS}"

  local pod logfile
  while read -r pod; do
    [[ -z "${pod}" ]] && continue
    logfile="${MEM_DIR}/${label}-${pod}.log"
    log "  Saving conversion pod log: ${logfile}"
    oc logs -n "${NS}" "${pod}" --all-containers=true >"${logfile}" 2>&1 || true
  done < <(oc get pods -n "${NS}" -l "${V2V_LABEL}" -o json 2>/dev/null \
    | jq -r --arg plan "${plan}" '
        .items[]
        | select(.metadata.name | startswith($plan))
        | .metadata.name
      ' 2>/dev/null || true)

  log "RESULT ${label}: status=${status} total=$(fmt_dur "${duration}") vm=${vm} converter=${CONVERTER}"
  log ""

  if [[ "${status}" != "Completed" && "${status}" != "Succeeded" ]]; then
    return 1
  fi
  return 0
}

release_after_rhel() {
  if [[ "${KEEP_BETWEEN_TESTS}" == "true" ]]; then
    log "KEEP_BETWEEN_TESTS=true — leaving plan-bench-rhel + conversion pods for postmortem."
    if [[ -n "${RHEL_VM}" ]]; then
      log "Stopping migrated RHEL VM ${RHEL_VM} (plan/pods kept)..."
      stop_migrated_vm "${RHEL_VM}" "${NS}"
    fi
    return 0
  fi

  log "RHEL finished OK. Releasing RHEL plan (free memory) before Windows..."
  release_rhel_resources "${NS}" "${RHEL_VM}"
}

# Run one converter leg: RHEL then Windows. Sets image, fresh NS, providers.
# Args: converter (kc|ref)
# Returns non-zero if either plan fails.
run_converter_leg() {
  local converter="$1"
  local rc_rhel=0 rc_win=0

  begin_leg "${converter}"

  log "=========================================="
  log "MTV cold migration benchmark — ${converter}"
  log "=========================================="
  log "Mode: ${MODE}"
  log "Log: ${SUMMARY_LOG}"
  log "Mem: ${MEM_DIR}"
  log "NS=${NS} SKIP_CLEANUP=${SKIP_CLEANUP} KEEP_BETWEEN_TESTS=${KEEP_BETWEEN_TESTS}"
  log ""

  case "${converter}" in
    kc)
      set_virt_v2v_image "${KC_V2V_IMAGE}"
      ;;
    ref)
      clear_virt_v2v_image
      ;;
    *)
      log "ERROR: unknown converter '${converter}'"
      return 1
      ;;
  esac

  if [[ "${DISABLE_WAIT_FOR_REBOOT}" == "true" ]]; then
    disable_windows_wait_for_reboot
  fi

  log "VDDK: $(mtv_setting_get vddk_image)"
  log "virt_v2v_image_fqin: $(mtv_setting_get virt_v2v_image_fqin)"
  log "feature_windows_wait_for_reboot: $(mtv_setting_get feature_windows_wait_for_reboot)"
  log "Default SC: $(oc get sc -o jsonpath='{range .items[?(@.metadata.annotations.storageclass\.kubernetes\.io/is-default-class=="true")]}{.metadata.name}{end}')"
  log "Ceph: $(oc get cephcluster -n openshift-storage -o jsonpath='{.items[0].status.ceph.health}' 2>/dev/null || echo n/a)"
  log ""

  fresh_namespace "${NS}"
  log "Creating providers..."
  create_vsphere_and_host_providers "${NS}" "${PROVIDER}"
  log ""

  if [[ -z "${RHEL_VM}" || -z "${WIN_VM}" ]]; then
    wait_for_mtv_func_vms "${NS}" "${PROVIDER}" || return 1
  fi

  if [[ -z "${RHEL_VM}" ]]; then
    RHEL_VM="$(pick_rhel_vm "${NS}" "${PROVIDER}")"
  fi
  if [[ -z "${WIN_VM}" ]]; then
    WIN_VM="$(pick_win_vm "${NS}" "${PROVIDER}")"
  fi

  if [[ -z "${RHEL_VM}" || "${RHEL_VM}" == "null" ]]; then
    log "ERROR: no mtv-func RHEL VM found"
    return 1
  fi
  if [[ -z "${WIN_VM}" || "${WIN_VM}" == "null" ]]; then
    log "ERROR: no mtv-func Windows VM found"
    return 1
  fi

  log "RHEL_VM=${RHEL_VM}"
  log "WIN_VM=${WIN_VM}"
  log ""
  log "Running RHEL then Windows sequentially (one plan at a time)."

  run_one "rhel" "${RHEL_VM}" "plan-bench-rhel" || rc_rhel=$?

  if [[ "${rc_rhel}" -ne 0 ]]; then
    log "RHEL failed (exit=${rc_rhel}) — skipping Windows; leaving ${NS} for debugging."
    SKIP_CLEANUP=true
  else
    release_after_rhel
    log ""
    run_one "win" "${WIN_VM}" "plan-bench-win" || rc_win=$?
  fi

  log "=========================================="
  log "LEG SUMMARY (${converter})"
  log "=========================================="
  log "Image: $(mtv_setting_get virt_v2v_image_fqin)"
  log "RHEL (${RHEL_VM}): exit=${rc_rhel}"
  log "WIN  (${WIN_VM}): exit=${rc_win}"
  log "Full log: ${SUMMARY_LOG}"
  log "Memory CSVs + pod logs: ${MEM_DIR}"
  log ""

  if [[ "${rc_rhel}" -ne 0 || "${rc_win}" -ne 0 ]]; then
    return 1
  fi
  return 0
}

# ===================================================================
#  Main
# ===================================================================
case "${MODE}" in
  kc|compare) ;;
  -h|--help|help)
    usage
    exit 0
    ;;
  *)
    echo "ERROR: MODE must be 'kc' or 'compare' (got '${MODE}')." >&2
    usage >&2
    exit 1
    ;;
esac

if [[ -z "${KC_V2V_IMAGE}" ]]; then
  echo "ERROR: KC_V2V_IMAGE must be set (e.g. quay.io/you/kc-v2v:devel-amd64)." >&2
  exit 1
fi

echo "=========================================="
echo "MTV conversion benchmark"
echo "=========================================="
echo "MODE=${MODE}"
echo "KC_V2V_IMAGE=${KC_V2V_IMAGE}"
echo "Artifact prefix: ${ARTIFACT_PREFIX}-<converter>"
echo ""

preflight_mtv_cluster
save_mtv_settings

cleanup() {
  benchmark_exit_cleanup
}
trap cleanup EXIT

overall_rc=0

run_converter_leg "kc" || overall_rc=$?

if [[ "${MODE}" == "compare" ]]; then
  if [[ "${overall_rc}" -ne 0 ]]; then
    echo "kc leg failed — skipping ref (default) leg."
  else
    echo ""
    echo "kc leg OK. Starting ref (operator default) leg..."
    echo ""
    run_converter_leg "ref" || overall_rc=$?
  fi
fi

echo "=========================================="
echo "OVERALL SUMMARY"
echo "=========================================="
echo "MODE=${MODE}"
echo "KC_V2V_IMAGE=${KC_V2V_IMAGE}"
echo "Artifacts: ${ARTIFACT_PREFIX}-*.log / ${ARTIFACT_PREFIX}-*-mem/"

DASHBOARD_HTML="${ARTIFACT_PREFIX}.html"
if python3 "${SCRIPT_DIR}/lib/generate-run-dashboard.py" "${ARTIFACT_PREFIX}"; then
  echo "Dashboard: ${DASHBOARD_HTML}"
else
  echo "WARNING: failed to generate dashboard HTML" >&2
fi

echo "Namespace: ${NS} (SKIP_CLEANUP=${SKIP_CLEANUP})"
if [[ "${overall_rc}" -eq 0 ]]; then
  echo "TEST PASSED: MTV benchmark (${MODE})"
  exit 0
fi
echo "TEST FAILED: MTV benchmark (${MODE}) exit=${overall_rc}"
exit 1
