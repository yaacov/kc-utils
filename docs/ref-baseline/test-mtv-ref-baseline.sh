#!/bin/bash
#
# Sequential cold-migration runs (RHEL then Windows) with step timing +
# conversion-pod memory sampling.
#
# Prerequisites: oc, oc mtv, jq, GOVC_URL/USERNAME/PASSWORD, VDDK configured.
#
# Env overrides:
#   RHEL_VM / WIN_VM        source VM names (auto-picked from mtv-func* if unset)
#   NS                      namespace (default mtv-ref-baseline)
#   SKIP_CLEANUP            keep NS on exit (default true — for debugging)
#   KEEP_BETWEEN_TESTS      leave RHEL plan/conversion pods after RHEL (default false)
#   DISABLE_WAIT_FOR_REBOOT set feature_windows_wait_for_reboot=false (default true)
#   MEM_INTERVAL            memory sample seconds (default 10)
#   INTERVAL                plan poll seconds (default 10)
#   MAX_ATTEMPTS            plan poll attempts (default 180 => 30m)

set -euo pipefail

NS="${NS:-mtv-ref-baseline}"
PROVIDER="${PROVIDER:-vsphere-test}"
RHEL_VM="${RHEL_VM:-}"
WIN_VM="${WIN_VM:-}"
SKIP_CLEANUP="${SKIP_CLEANUP:-true}"
KEEP_BETWEEN_TESTS="${KEEP_BETWEEN_TESTS:-false}"
DISABLE_WAIT_FOR_REBOOT="${DISABLE_WAIT_FOR_REBOOT:-true}"
INTERVAL="${INTERVAL:-10}"
MAX_ATTEMPTS="${MAX_ATTEMPTS:-180}"
MEM_INTERVAL="${MEM_INTERVAL:-10}"
V2V_LABEL="forklift.app=virt-v2v"
SCENARIO_DIR="$(cd "$(dirname "$0")" && pwd)"
RUN_TS="$(date -u +%Y%m%dT%H%M%SZ)"
SUMMARY_LOG="${SCENARIO_DIR}/test-mtv-ref-baseline-${RUN_TS}.log"
MEM_DIR="${SCENARIO_DIR}/test-mtv-ref-baseline-${RUN_TS}-mem"

log() { echo "$*" | tee -a "${SUMMARY_LOG}"; }

fmt_dur() {
  local s="$1"
  printf "%dm%02ds" $((s / 60)) $((s % 60))
}

get_plan_status() {
  local plan="$1"
  oc mtv get plan --name "${plan}" -n "${NS}" --output json 2>/dev/null \
    | jq -r 'if type == "array" then .[0].status // "Unknown" else .status // "Unknown" end' 2>/dev/null \
    || echo "Unknown"
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

# Prefer a Running virt-v2v pod for the given plan; fall back to newest matching name.
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

# Read cumulative rx/tx bytes from /proc/net/dev (sum of all non-lo interfaces).
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

# Sample conversion pod memory into CSV until plan finishes.
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
  # sets global LAST_PLAN_STATUS
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
  log "RUN ${label}: VM=${vm} plan=${plan}"
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

  # Archive this plan's conversion-pod logs while the pods still exist.
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

  log "RESULT ${label}: status=${status} total=$(fmt_dur "${duration}") vm=${vm}"
  log ""

  if [[ "${status}" != "Completed" && "${status}" != "Succeeded" ]]; then
    return 1
  fi
  return 0
}

# ===================================================================
#  Preflight
# ===================================================================
: > "${SUMMARY_LOG}"
mkdir -p "${MEM_DIR}"

log "=========================================="
log "MTV cold migration reference run"
log "=========================================="
log "Log: ${SUMMARY_LOG}"
log "Mem: ${MEM_DIR}"
log "SKIP_CLEANUP=${SKIP_CLEANUP}"
log "KEEP_BETWEEN_TESTS=${KEEP_BETWEEN_TESTS}"
log "DISABLE_WAIT_FOR_REBOOT=${DISABLE_WAIT_FOR_REBOOT}"
log ""

[[ -n "${GOVC_URL:-}" && -n "${GOVC_USERNAME:-}" && -n "${GOVC_PASSWORD:-}" ]] \
  || { log "ERROR: GOVC_* required"; exit 1; }

if [[ "${DISABLE_WAIT_FOR_REBOOT}" == "true" ]]; then
  log "Disabling feature_windows_wait_for_reboot..."
  oc mtv settings set --setting feature_windows_wait_for_reboot --value false
fi

VDDK_IMAGE=$(oc mtv settings get --setting vddk_image 2>/dev/null | tail -1 | awk '{print $NF}')
log "VDDK: ${VDDK_IMAGE}"
log "virt_v2v_image_fqin: $(oc mtv settings get --setting virt_v2v_image_fqin 2>/dev/null | tail -1)"
log "feature_windows_wait_for_reboot: $(oc mtv settings get --setting feature_windows_wait_for_reboot 2>/dev/null | tail -1)"
log "Default SC: $(oc get sc -o jsonpath='{range .items[?(@.metadata.annotations.storageclass\.kubernetes\.io/is-default-class=="true")]}{.metadata.name}{end}')"
log "Ceph: $(oc get cephcluster -n openshift-storage -o jsonpath='{.items[0].status.ceph.health}' 2>/dev/null || echo n/a)"
log ""

cleanup() {
  if [[ "${SKIP_CLEANUP}" == "true" ]]; then
    log "SKIP_CLEANUP=true -- leaving ${NS}"
    return 0
  fi
  log "Cleaning up namespace ${NS}..."
  oc delete namespace "${NS}" --ignore-not-found 2>/dev/null || true
}
trap cleanup EXIT

# Fresh NS
oc delete namespace "${NS}" --ignore-not-found 2>/dev/null || true
for i in $(seq 1 60); do
  oc get ns "${NS}" >/dev/null 2>&1 || break
  sleep 2
done
oc create namespace "${NS}"

log "Creating providers..."
oc mtv create provider --name "${PROVIDER}" --type vsphere \
  --url "https://${GOVC_URL}/sdk" \
  --username "${GOVC_USERNAME}" \
  --password "${GOVC_PASSWORD}" \
  --provider-insecure-skip-tls \
  -n "${NS}"
oc mtv create provider --name host --type openshift -n "${NS}"
oc wait "provider.forklift.konveyor.io/${PROVIDER}" -n "${NS}" --for=condition=Ready --timeout=300s
oc wait "provider.forklift.konveyor.io/host" -n "${NS}" --for=condition=Ready --timeout=300s
log "Providers ready."
log ""

if [[ -z "${RHEL_VM}" ]]; then
  RHEL_VM=$(oc mtv get inventory vm --provider "${PROVIDER}" -n "${NS}" \
    --query "select name where name ~= 'mtv-func.*' and (name ~= 'rhel' or guestId ~= 'rhel')" \
    --output json 2>/dev/null | jq -r '.[].name' | sort | head -1)
fi
if [[ -z "${WIN_VM}" ]]; then
  WIN_VM=$(oc mtv get inventory vm --provider "${PROVIDER}" -n "${NS}" \
    --query "select name where name ~= 'mtv-func.*' and (name ~= 'win' or guestId ~= 'win')" \
    --output json 2>/dev/null | jq -r '.[].name' | sort | head -1)
fi

[[ -n "${RHEL_VM}" && "${RHEL_VM}" != "null" ]] || { log "ERROR: no RHEL VM"; exit 1; }
[[ -n "${WIN_VM}" && "${WIN_VM}" != "null" ]] || { log "ERROR: no Windows VM"; exit 1; }

log "RHEL_VM=${RHEL_VM}"
log "WIN_VM=${WIN_VM}"
log ""

# Sequential only — never overlap migrations (workers ~15Gi).
# On RHEL failure: stop and keep plan/pods for debugging (no Windows run).
rc_rhel=0
rc_win=0
log "Running RHEL then Windows sequentially (one plan at a time)."
run_one "rhel" "${RHEL_VM}" "plan-ref-rhel" || rc_rhel=$?

if [[ "${rc_rhel}" -ne 0 ]]; then
  log "RHEL failed (exit=${rc_rhel}) — skipping Windows; leaving ${NS} for debugging."
else
  if [[ "${KEEP_BETWEEN_TESTS}" == "true" ]]; then
    log "KEEP_BETWEEN_TESTS=true — leaving plan-ref-rhel + conversion pods for postmortem."
    # Stop migrated RHEL guest to free worker RAM for Windows conversion.
    if [[ -n "${RHEL_VM}" ]]; then
      log "Stopping migrated RHEL VM ${RHEL_VM} (plan/pods kept)..."
      oc virt stop "${RHEL_VM}" -n "${NS}" 2>/dev/null \
        || oc delete vmi "${RHEL_VM}" -n "${NS}" --ignore-not-found 2>/dev/null \
        || true
    fi
  else
    log "RHEL finished OK. Releasing RHEL plan only (free memory) before Windows..."
    oc mtv delete plan --name plan-ref-rhel -n "${NS}" 2>/dev/null || true
    for i in $(seq 1 60); do
      left="$(oc get pods -n "${NS}" --no-headers 2>/dev/null | wc -l | tr -d ' ')"
      [[ "${left}" == "0" ]] && break
      sleep 5
    done
    log "Namespace pod count after RHEL plan delete: $(oc get pods -n "${NS}" --no-headers 2>/dev/null | wc -l | tr -d ' ')"
  fi
  log ""
  run_one "win" "${WIN_VM}" "plan-ref-win" || rc_win=$?
fi

log "=========================================="
log "SUMMARY"
log "=========================================="
log "RHEL (${RHEL_VM}): exit=${rc_rhel}"
log "WIN  (${WIN_VM}): exit=${rc_win}"
log "Full log: ${SUMMARY_LOG}"
log "Memory CSVs: ${MEM_DIR}"
log "Namespace left in place: ${NS} (SKIP_CLEANUP=${SKIP_CLEANUP})"

if [[ "${rc_rhel}" -ne 0 || "${rc_win}" -ne 0 ]]; then
  exit 1
fi
exit 0
