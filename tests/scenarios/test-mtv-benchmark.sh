#!/usr/bin/env bash
#
# MTV cold-migration benchmark: three VMs in one plan, with full artifact
# capture (summary log, mem/CPU/net CSV, conversion-pod logs, pipeline
# timings). Conversion pods are sampled in parallel.
#
# Modes (MODE):
#   kc       Run once with KC_V2V_IMAGE (independent kc-v2v benchmark). Default.
#   compare  Run twice: operator-default virt-v2v then kc-v2v (full compare).
#
# Prerequisites: oc, oc mtv, jq, tests/scenarios/.env configured, VDDK on cluster.
#
# Env overrides (in .env or shell after load):
#   MODE                    kc | compare (default kc)
#   KC_V2V_IMAGE            conversion image FQIN (required in .env)
#   VM1 / VM2 / VM3         source VM names (auto-picked from mtv-func* if unset)
#   NS                      namespace (required in .env)
#   PROVIDER                vSphere provider name (required in .env)
#   PROVIDER_INSECURE_SKIP_TLS  true = skip TLS verify; false = fetch CA from GOVC_URL:443
#   SKIP_CLEANUP            keep NS on exit (default true); use cleanup script to remove later
#   KEEP_IMAGE_SETTING      leave virt_v2v_image_fqin / reboot flag (default true)
#   DISABLE_WAIT_FOR_REBOOT set feature_windows_wait_for_reboot=false (default true)
#   MEM_INTERVAL            memory sample seconds (default 10)
#   INTERVAL                plan poll seconds (default 10)
#   MAX_ATTEMPTS            plan poll attempts (default 180 => 30m)
#   BENCHMARK_PLAN          plan name (default plan-bench)
#
# Artifacts under runs/:
#   runs/test-mtv-benchmark-<ts>-<converter>.log
#   runs/test-mtv-benchmark-<ts>-<converter>-mem/
#   runs/test-mtv-benchmark-<ts>.html  (dashboard)
# Archive under docs/architecture/ref-baseline/runs/ when publishing a comparison.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"
# shellcheck source=lib/cleanup.sh
source "${SCRIPT_DIR}/lib/cleanup.sh"

MODE="${MODE:-kc}"
# Prefer VM1/VM2/VM3; accept leftover RHEL_VM/WIN_VM from older .env files.
VM1="${VM1:-${RHEL_VM:-}}"
VM2="${VM2:-${WIN_VM:-}}"
VM3="${VM3:-}"
BENCH_LABELS=(vm1 vm2 vm3)
BENCH_VMS=("${VM1}" "${VM2}" "${VM3}")
SKIP_CLEANUP="${SKIP_CLEANUP:-true}"
KEEP_IMAGE_SETTING="${KEEP_IMAGE_SETTING:-true}"
BENCHMARK_PLAN="${BENCHMARK_PLAN:-plan-bench}"
DISABLE_WAIT_FOR_REBOOT="${DISABLE_WAIT_FOR_REBOOT:-true}"
INTERVAL="${INTERVAL:-10}"
MAX_ATTEMPTS="${MAX_ATTEMPTS:-180}"
MEM_INTERVAL="${MEM_INTERVAL:-10}"
V2V_LABEL="forklift.app=virt-v2v"
RUN_TS="$(date -u +%Y%m%dT%H%M%SZ)"
RUNS_DIR="${SCENARIO_DIR}/runs"
ARTIFACT_PREFIX="${RUNS_DIR}/test-mtv-benchmark-${RUN_TS}"
mkdir -p "${RUNS_DIR}"

# Per-leg state (set by begin_leg / monitor_plan)
CONVERTER=""
SUMMARY_LOG=""
MEM_DIR=""
LAST_PLAN_STATUS=""
LAST_VM_STATUS=()
LAST_VM_DURATION=()

# Background monitor PIDs (memory sampler). Ctrl+C must kill these or they
# keep polling after the main script is interrupted.
_BG_PIDS=()
_CLEANED_UP=false

track_bg_pid() {
  _BG_PIDS+=("$1")
}

untrack_bg_pid() {
  local target="$1" pid
  local kept=()
  for pid in "${_BG_PIDS[@]:-}"; do
    [[ "${pid}" == "${target}" ]] || kept+=("${pid}")
  done
  _BG_PIDS=("${kept[@]:-}")
}

kill_bg_jobs() {
  local pid job
  for pid in "${_BG_PIDS[@]:-}"; do
    kill "${pid}" 2>/dev/null || true
  done
  while read -r job; do
    [[ -n "${job}" ]] || continue
    kill "${job}" 2>/dev/null || true
  done < <(jobs -p 2>/dev/null || true)
  wait 2>/dev/null || true
  _BG_PIDS=()
}

usage() {
  cat <<EOF
Usage: MODE=kc|ref|compare $0

  Configure tests/scenarios/.env first (see .env.example).

  MODE=kc       Independent kc-v2v benchmark (default)
  MODE=ref      Operator-default virt-v2v only
  MODE=compare  operator-default virt-v2v then kc-v2v

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

is_terminal_status() {
  case "${1:-}" in
    Completed|Succeeded|Failed|Error|Cancelled) return 0 ;;
    *) return 1 ;;
  esac
}

is_success_status() {
  case "${1:-}" in
    Completed|Succeeded) return 0 ;;
    *) return 1 ;;
  esac
}

# Inventory id for a named VM on the Plan CR (status, then spec).
vm_id() {
  local plan="$1" vm="$2"
  oc get "plan.forklift.konveyor.io/${plan}" -n "${NS}" -o json 2>/dev/null \
    | jq -r --arg vm "${vm}" '
        ((.status.migration.vms // []) + (.spec.vms // []))
        | map(select(.name == $vm) | .id)
        | map(select(. != null and . != ""))
        | .[0] // empty
      ' 2>/dev/null || true
}

pipeline_json() {
  local plan="$1" vm="$2"
  oc get "plan.forklift.konveyor.io/${plan}" -n "${NS}" -o json 2>/dev/null \
    | jq -c --arg vm "${vm}" '
        [.status.migration.vms[]? | select(.name == $vm) | .pipeline // []]
        | .[0] // []
      '
}

vm_phase() {
  local plan="$1" vm="$2"
  oc get "plan.forklift.konveyor.io/${plan}" -n "${NS}" -o json 2>/dev/null \
    | jq -r --arg vm "${vm}" '
        [.status.migration.vms[]? | select(.name == $vm) | .phase // ""]
        | .[0] // empty
      ' 2>/dev/null || true
}

fetch_plan_json() {
  local plan="$1"
  oc get "plan.forklift.konveyor.io/${plan}" -n "${NS}" -o json 2>/dev/null || echo '{}'
}

# Status string from a Plan CR (or oc mtv get JSON). Prefers condition types
# when .status is an object.
plan_status_from_json() {
  jq -r '
    if type == "array" then
      .[0].status // "Unknown"
    elif (.status | type) == "string" then
      .status // "Unknown"
    else
      def cond($t):
        [.status.conditions[]? | select(.type == $t) | .status][0] // "False";
      if cond("Succeeded") == "True" then "Succeeded"
      elif cond("Failed") == "True" then "Failed"
      elif cond("Canceled") == "True" then "Cancelled"
      elif cond("Cancelled") == "True" then "Cancelled"
      elif cond("Executing") == "True" then "Executing"
      elif cond("Ready") == "True" then "Ready"
      else "Unknown"
      end
    end
  ' 2>/dev/null || echo "Unknown"
}

vm_phase_from_json() {
  local vm="$1"
  jq -r --arg vm "${vm}" '
    [.status.migration.vms[]? | select(.name == $vm) | .phase // ""]
    | .[0] // empty
  ' 2>/dev/null || true
}

pipeline_summary_from_json() {
  local vm="$1"
  jq -r --arg vm "${vm}" '
    [.status.migration.vms[]? | select(.name == $vm) | .pipeline // []]
    | .[0] // []
    | [.[] | "\(.name)=\(.phase // "-")"] | join(" ")
  ' 2>/dev/null || true
}

summarize_pipeline() {
  local plan="$1" vm="$2"
  log "  Pipeline step timings (${plan} / ${vm}):"
  oc get "plan.forklift.konveyor.io/${plan}" -n "${NS}" -o json 2>/dev/null \
    | jq -r --arg vm "${vm}" '
      ([.status.migration.vms[]? | select(.name == $vm)][0].pipeline // [])[]
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

# Match the virt-v2v pod for one VM in a (possibly multi-VM) plan.
# Pods are named like {plan}-vm-{id}-{suffix}; also try label vmID.
# When vm is set, wait until the Plan CR has that VM's inventory id so the
# two parallel monitors do not attach to the same pod.
find_v2v_pod() {
  local plan="${1:-}" vm="${2:-}"
  local vmid=""
  if [[ -n "${vm}" ]]; then
    vmid="$(vm_id "${plan}" "${vm}")"
    [[ -n "${vmid}" ]] || return 0
  fi
  oc get pods -n "${NS}" -l "${V2V_LABEL}" -o json 2>/dev/null \
    | jq -r --arg plan "${plan}" --arg vmid "${vmid}" '
        [.items[]
          | select($plan == "" or (.metadata.name | startswith($plan)))
          | select(
              $vmid == ""
              or (.metadata.name | contains($vmid))
              or ((.metadata.labels.vmID // .metadata.labels.vmid // "") == $vmid)
            )
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
  # Parent owns Ctrl+C / EXIT cleanup; reset inherited traps so this sampler
  # dies on signal or explicit kill instead of racing the main cleanup path.
  trap - INT TERM EXIT

  local plan="$1" label="$2" vm="$3"
  local csv="${MEM_DIR}/${label}-virt-v2v-memory.csv"
  mkdir -p "${MEM_DIR}"
  echo "timestamp_utc,elapsed_s,pod,node,mem_working_set_mi,mem_rss_mi_cgroup,cpu_m,net_rx_bytes,net_tx_bytes,phase" > "${csv}"

  local start_ts pod node mem_top cpu_top rss_mi phase elapsed top_line net_raw net_rx net_tx
  start_ts=$(date +%s)
  log "  Memory/net monitor (${label}) -> ${csv} (every ${MEM_INTERVAL}s)"

  while true; do
    local status vm_p
    status="$(get_plan_status "${plan}")"
    if is_terminal_status "${status}"; then
      break
    fi
    vm_p="$(vm_phase "${plan}" "${vm}")"
    if is_terminal_status "${vm_p}"; then
      break
    fi

    pod="$(find_v2v_pod "${plan}" "${vm}")"
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
      printf "    [mon %s] +%s pod=%s node=%s top=%sMi cgroup=%sMi cpu=%sm rx=%s tx=%s phase=%s\n" \
        "${label}" "$(fmt_dur "${elapsed}")" "${pod}" "${node:-?}" "${mem_top:-?}" "${rss_mi:-?}" "${cpu_top:-?}" \
        "${net_rx:-?}" "${net_tx:-?}" "${phase:-?}" \
        | tee -a "${SUMMARY_LOG}"
    fi
    sleep "${MEM_INTERVAL}" || exit 130
  done

  if [[ "$(wc -l < "${csv}")" -gt 1 ]]; then
    local peak_top peak_cgroup max_rx max_tx
    peak_top="$(awk -F, 'NR>1 && $5!="" {if ($5+0>max) max=$5+0} END{print max+0}' "${csv}")"
    peak_cgroup="$(awk -F, 'NR>1 && $6!="" {if ($6+0>max) max=$6+0} END{print max+0}' "${csv}")"
    max_rx="$(awk -F, 'NR>1 && $8!="" {if ($8+0>max) max=$8+0} END{print max+0}' "${csv}")"
    max_tx="$(awk -F, 'NR>1 && $9!="" {if ($9+0>max) max=$9+0} END{print max+0}' "${csv}")"
    log "  Peak conversion memory (${label}): metrics-server=${peak_top}Mi  cgroup=${peak_cgroup}Mi"
    log "  Network totals (${label}, cumulative at last sample): rx=$((max_rx / 1024 / 1024))Mi  tx=$((max_tx / 1024 / 1024))Mi"
  else
    log "  No conversion memory samples collected (${label})"
  fi
}

wait_mem_monitor() {
  local pid="$1"
  if ! wait "${pid}" 2>/dev/null; then
    kill "${pid}" 2>/dev/null || true
    wait "${pid}" 2>/dev/null || true
  fi
  untrack_bg_pid "${pid}"
}

# Fill empty BENCH_VMS slots from mtv-func* inventory, skipping names already used.
fill_unset_bench_vms() {
  local ns="$1"
  local provider="$2"
  local used=" " name i
  for i in "${!BENCH_VMS[@]}"; do
    if [[ -n "${BENCH_VMS[$i]}" && "${BENCH_VMS[$i]}" != "null" ]]; then
      used+="${BENCH_VMS[$i]} "
    fi
  done
  while read -r name; do
    [[ -z "${name}" || "${name}" == "null" ]] && continue
    [[ "${used}" == *" ${name} "* ]] && continue
    for i in "${!BENCH_VMS[@]}"; do
      if [[ -z "${BENCH_VMS[$i]}" || "${BENCH_VMS[$i]}" == "null" ]]; then
        BENCH_VMS[$i]="${name}"
        used+="${name} "
        break
      fi
    done
  done <<< "$(list_mtv_func_vms "${ns}" "${provider}")"
}

sync_bench_vm_vars() {
  VM1="${BENCH_VMS[0]:-}"
  VM2="${BENCH_VMS[1]:-}"
  VM3="${BENCH_VMS[2]:-}"
}

bench_vm_list() {
  local IFS=,
  echo "${BENCH_VMS[*]}"
}

label_upper() {
  printf '%s' "$1" | tr '[:lower:]' '[:upper:]'
}

monitor_plan() {
  local plan="$1"
  local start_time="${2:-}"
  local start_iso="${3:-}"
  local attempt=0 status i label vm end_iso
  local phases=() pipes=() done_ts=() mem_pids=()
  if [[ -z "${start_time}" ]]; then
    start_time=$(date +%s)
  fi
  if [[ -z "${start_iso}" ]]; then
    start_iso=$(date -u +%Y-%m-%dT%H:%M:%SZ)
  fi
  LAST_PLAN_STATUS="Unknown"
  LAST_VM_STATUS=()
  LAST_VM_DURATION=()

  for i in "${!BENCH_LABELS[@]}"; do
    done_ts[$i]=0
    LAST_VM_STATUS[$i]=""
    LAST_VM_DURATION[$i]=0
    monitor_conversion_memory "${plan}" "${BENCH_LABELS[$i]}" "${BENCH_VMS[$i]}" &
    mem_pids[$i]=$!
    track_bg_pid "${mem_pids[$i]}"
  done

  local desc=""
  for i in "${!BENCH_LABELS[@]}"; do
    desc+="${BENCH_LABELS[$i]}=${BENCH_VMS[$i]} "
  done
  log "--- Monitoring plan ${plan} (${desc% }) ---"
  local prev_pipe="" plan_json
  while [ "${attempt}" -lt "${MAX_ATTEMPTS}" ]; do
    attempt=$((attempt + 1))
    plan_json="$(fetch_plan_json "${plan}")"
    status="$(plan_status_from_json <<<"${plan_json}")"
    local elapsed=$(( $(date +%s) - start_time ))
    local pipe_summary="" phase_line=""
    for i in "${!BENCH_LABELS[@]}"; do
      label="${BENCH_LABELS[$i]}"
      vm="${BENCH_VMS[$i]}"
      phases[$i]="$(vm_phase_from_json "${vm}" <<<"${plan_json}")"
      pipes[$i]="$(pipeline_summary_from_json "${vm}" <<<"${plan_json}")"
      pipe_summary+="${label}:[${pipes[$i]}] "
      phase_line+="${label}=${phases[$i]:-?} "
      if is_terminal_status "${phases[$i]}" && [[ "${done_ts[$i]}" -eq 0 ]]; then
        done_ts[$i]=$(date +%s)
      fi
    done
    pipe_summary="${pipe_summary% }"
    phase_line="${phase_line% }"

    if [[ "${pipe_summary}" != "${prev_pipe}" ]]; then
      log "  [step-change] $(fmt_dur "${elapsed}") plan=${status} ${phase_line} ${pipe_summary}"
      prev_pipe="${pipe_summary}"
    else
      printf "  [%02d/%02d] %s  Status=%s %s\n" \
        "${attempt}" "${MAX_ATTEMPTS}" "$(fmt_dur "${elapsed}")" "${status}" \
        "${phase_line}" \
        | tee -a "${SUMMARY_LOG}"
    fi

    if is_terminal_status "${status}"; then
      break
    fi
    sleep "${INTERVAL}"
  done

  if ! is_terminal_status "${status}"; then
    log "  Timed out waiting for plan ${plan} (status=${status:-Unknown}, attempts=${MAX_ATTEMPTS}); stopping samplers."
    for i in "${!mem_pids[@]}"; do
      kill "${mem_pids[$i]}" 2>/dev/null || true
    done
  fi

  for i in "${!mem_pids[@]}"; do
    wait_mem_monitor "${mem_pids[$i]}"
  done

  local now duration
  now=$(date +%s)
  duration=$(( now - start_time ))
  LAST_PLAN_STATUS="${status}"
  for i in "${!BENCH_LABELS[@]}"; do
    LAST_VM_STATUS[$i]="${phases[$i]}"
    if [[ "${done_ts[$i]}" -gt 0 ]]; then
      LAST_VM_DURATION[$i]=$(( done_ts[$i] - start_time ))
    else
      LAST_VM_DURATION[$i]=$(( now - start_time ))
    fi
  done
  end_iso=$(date -u +%Y-%m-%dT%H:%M:%SZ)
  log ""
  log "Migration ended at ${end_iso}"
  log "Migration lifetime: $(fmt_dur "${duration}") (start=${start_iso} end=${end_iso})"
  log "  Plan ${plan} finished: status=${status} duration=$(fmt_dur "${duration}")"
  for i in "${!BENCH_LABELS[@]}"; do
    summarize_pipeline "${plan}" "${BENCH_VMS[$i]}"
  done
  log "  VM details:"
  oc mtv get plan --name "${plan}" -n "${NS}" --vms 2>&1 | tee -a "${SUMMARY_LOG}" || true
  log ""
}

save_conversion_logs() {
  local plan="$1" label="$2" vm="$3"
  local vmid pod logfile
  vmid="$(vm_id "${plan}" "${vm}")"
  if [[ -z "${vmid}" ]]; then
    log "  WARNING: no inventory id for VM ${vm}; conversion-pod log may be missing"
    return 0
  fi
  while read -r pod; do
    [[ -z "${pod}" ]] && continue
    logfile="${MEM_DIR}/${label}-${pod}.log"
    log "  Saving conversion pod log (${label}): ${logfile}"
    oc logs -n "${NS}" "${pod}" --all-containers=true >"${logfile}" 2>&1 || true
  done < <(oc get pods -n "${NS}" -l "${V2V_LABEL}" -o json 2>/dev/null \
    | jq -r --arg plan "${plan}" --arg vmid "${vmid}" '
        .items[]
        | select(.metadata.name | startswith($plan))
        | select(
            (.metadata.name | contains($vmid))
            or ((.metadata.labels.vmID // .metadata.labels.vmid // "") == $vmid)
          )
        | .metadata.name
      ' 2>/dev/null || true)
}

run_plan() {
  local plan="${BENCHMARK_PLAN}"
  local i label vm_list start_s start_iso
  vm_list="$(bench_vm_list)"
  log "=========================================="
  log "RUN: VMs=${vm_list} plan=${plan} converter=${CONVERTER}"
  log "=========================================="

  oc mtv delete plan --name "${plan}" -n "${NS}" 2>/dev/null || true
  sleep 2

  local create_out
  if ! create_out="$(oc mtv create plan --name "${plan}" --source "${PROVIDER}" --target host \
    --vms "${vm_list}" \
    --run-preflight-inspection false \
    -n "${NS}" 2>&1)"; then
    log "ERROR: failed to create plan ${plan}"
    log "${create_out}"
    log "DEBUG: ns phase=$(oc get ns "${NS}" -o jsonpath='{.status.phase}' 2>/dev/null || echo missing)"
    oc mtv get plan -n "${NS}" 2>&1 | tee -a "${SUMMARY_LOG}" || true
    oc get vm,vmi,pvc -n "${NS}" --no-headers 2>&1 | tee -a "${SUMMARY_LOG}" || true
    return 1
  fi
  log "${create_out}"

  log "Waiting for plan Ready..."
  local wait_out
  if ! wait_out="$(oc wait "plan.forklift.konveyor.io/${plan}" -n "${NS}" \
    --for=condition=Ready --timeout=300s 2>&1)"; then
    log "ERROR: plan ${plan} did not become Ready"
    log "${wait_out}"
    oc get "plan.forklift.konveyor.io/${plan}" -n "${NS}" -o yaml 2>&1 \
      | tee -a "${SUMMARY_LOG}" || true
    return 1
  fi
  log "${wait_out}"

  oc mtv start plan --name "${plan}" -n "${NS}"
  start_s=$(date +%s)
  start_iso=$(date -u +%Y-%m-%dT%H:%M:%SZ)
  log "Migration started at ${start_iso}"

  monitor_plan "${plan}" "${start_s}" "${start_iso}"

  for i in "${!BENCH_LABELS[@]}"; do
    save_conversion_logs "${plan}" "${BENCH_LABELS[$i]}" "${BENCH_VMS[$i]}"
  done

  for i in "${!BENCH_LABELS[@]}"; do
    label="${BENCH_LABELS[$i]}"
    log "RESULT ${label}: status=${LAST_VM_STATUS[$i]:-Unknown} total=$(fmt_dur "${LAST_VM_DURATION[$i]:-0}") vm=${BENCH_VMS[$i]} converter=${CONVERTER}"
  done
  log ""

  if ! is_success_status "${LAST_PLAN_STATUS}"; then
    return 1
  fi
  for i in "${!BENCH_LABELS[@]}"; do
    if ! is_success_status "${LAST_VM_STATUS[$i]}"; then
      return 1
    fi
  done
  return 0
}

# Run one converter leg: three VMs in a single plan. Sets image, fresh NS,
# providers. Args: converter (kc|ref). Returns non-zero if the plan or any
# VM fails.
run_converter_leg() {
  local converter="$1"
  local rc=0 i label unset_any=false need vm_rc seen=" " vm

  begin_leg "${converter}"

  log "=========================================="
  log "MTV cold migration benchmark — ${converter}"
  log "=========================================="
  log "Mode: ${MODE}"
  log "Log: ${SUMMARY_LOG}"
  log "Mem: ${MEM_DIR}"
  log "NS=${NS} SKIP_CLEANUP=${SKIP_CLEANUP}"
  log ""

  case "${converter}" in
    kc)
      set_virt_v2v_image "${KC_V2V_IMAGE}" || return 1
      ;;
    ref)
      clear_virt_v2v_image || return 1
      ;;
    *)
      log "ERROR: unknown converter '${converter}'"
      return 1
      ;;
  esac

  verify_converter_image_ready "${converter}" || return 1

  if [[ "${DISABLE_WAIT_FOR_REBOOT}" == "true" ]]; then
    disable_windows_wait_for_reboot
  fi

  log "VDDK: $(mtv_setting_get vddk_image)"
  log "virt_v2v_image_fqin: $(mtv_setting_get virt_v2v_image_fqin)"
  deploy_ref="$(forklift_controller_deploy_ref)"
  if [[ -n "${deploy_ref}" ]]; then
    log "forklift-controller deploy=${deploy_ref#*/} VIRT_V2V_IMAGE(deploy)=$(controller_deploy_virt_v2v_image "${deploy_ref%%/*}" "${deploy_ref#*/}") pod=$(controller_pod_virt_v2v_image "${deploy_ref%%/*}" "${deploy_ref#*/}")"
  fi
  log "feature_windows_wait_for_reboot: $(mtv_setting_get feature_windows_wait_for_reboot)"
  log "Default SC: $(oc get sc -o jsonpath='{range .items[?(@.metadata.annotations.storageclass\.kubernetes\.io/is-default-class=="true")]}{.metadata.name}{end}')"
  log "Ceph: $(oc get cephcluster -n openshift-storage -o jsonpath='{.items[0].status.ceph.health}' 2>/dev/null || echo n/a)"
  log ""

  if ! fresh_namespace "${NS}"; then
    log "ERROR: failed to recreate namespace ${NS} for ${converter} leg"
    return 1
  fi
  log "Creating providers..."
  if ! create_vsphere_and_host_providers "${NS}" "${PROVIDER}"; then
    log "ERROR: failed to create providers in ${NS}"
    return 1
  fi
  log ""

  need="${#BENCH_LABELS[@]}"
  for i in "${!BENCH_VMS[@]}"; do
    if [[ -z "${BENCH_VMS[$i]}" || "${BENCH_VMS[$i]}" == "null" ]]; then
      unset_any=true
      break
    fi
  done
  if [[ "${unset_any}" == "true" ]]; then
    wait_for_mtv_func_vms "${NS}" "${PROVIDER}" "${need}" || return 1
    fill_unset_bench_vms "${NS}" "${PROVIDER}"
    sync_bench_vm_vars
  fi

  for i in "${!BENCH_LABELS[@]}"; do
    label="${BENCH_LABELS[$i]}"
    vm="${BENCH_VMS[$i]}"
    if [[ -z "${vm}" || "${vm}" == "null" ]]; then
      log "ERROR: no source VM for ${label} (pin VM$((i + 1)) or add mtv-func* inventory VMs)"
      return 1
    fi
    if [[ "${seen}" == *" ${vm} "* ]]; then
      log "ERROR: duplicate source VM '${vm}' (${label})"
      return 1
    fi
    seen+="${vm} "
  done

  for i in "${!BENCH_LABELS[@]}"; do
    log "$(label_upper "${BENCH_LABELS[$i]}")=${BENCH_VMS[$i]}"
  done
  log ""
  log "Running ${need} VMs in one plan (${BENCHMARK_PLAN}); conversion pods sampled in parallel."

  run_plan || rc=$?

  if [[ "${rc}" -ne 0 ]]; then
    log "Plan failed (exit=${rc}) — leaving ${NS} for debugging."
    SKIP_CLEANUP=true
  fi

  log "=========================================="
  log "LEG SUMMARY (${converter})"
  log "=========================================="
  log "Image: $(mtv_setting_get virt_v2v_image_fqin)"
  log "Plan ${BENCHMARK_PLAN}: status=${LAST_PLAN_STATUS:-Unknown}"
  for i in "${!BENCH_LABELS[@]}"; do
    vm_rc=0
    is_success_status "${LAST_VM_STATUS[$i]:-}" || vm_rc=1
    log "$(label_upper "${BENCH_LABELS[$i]}") (${BENCH_VMS[$i]}): status=${LAST_VM_STATUS[$i]:-Unknown} exit=${vm_rc}"
    if [[ "${vm_rc}" -ne 0 ]]; then
      rc=1
    fi
  done
  log "Full log: ${SUMMARY_LOG}"
  log "Memory CSVs + pod logs: ${MEM_DIR}"
  log ""

  if [[ "${rc}" -ne 0 ]]; then
    return 1
  fi
  return 0
}

# ===================================================================
#  Main
# ===================================================================
case "${MODE}" in
  kc|ref|compare) ;;
  -h|--help|help)
    usage
    exit 0
    ;;
  *)
    echo "ERROR: MODE must be 'kc', 'ref', or 'compare' (got '${MODE}')." >&2
    usage >&2
    exit 1
    ;;
esac

require_env KC_V2V_IMAGE

echo "=========================================="
echo "MTV conversion benchmark"
echo "=========================================="
echo "MODE=${MODE}"
echo "KC_V2V_IMAGE=${KC_V2V_IMAGE}"
echo "Artifact prefix: ${ARTIFACT_PREFIX}-<converter>"
echo ""

preflight_mtv_cluster
save_mtv_settings
capture_ref_virt_v2v_image

cleanup() {
  if [[ "${_CLEANED_UP}" == "true" ]]; then
    return 0
  fi
  _CLEANED_UP=true
  kill_bg_jobs
  benchmark_exit_cleanup
}

on_interrupt() {
  echo "" >&2
  echo "Interrupted (Ctrl+C) — stopping background monitors and exiting." >&2
  echo "Note: in-cluster MTV plans keep running; use clean-env.sh to cancel/cleanup." >&2
  cleanup
  trap - EXIT INT TERM
  exit 130
}

trap cleanup EXIT
trap on_interrupt INT TERM

overall_rc=0

case "${MODE}" in
  kc)
    run_converter_leg "kc" || overall_rc=$?
    ;;
  ref)
    run_converter_leg "ref" || overall_rc=$?
    ;;
  compare)
    run_converter_leg "ref" || overall_rc=$?
    if [[ "${overall_rc}" -ne 0 ]]; then
      echo "ref leg failed — skipping kc leg."
    else
      echo ""
      echo "ref leg OK. Starting kc leg..."
      echo ""
      run_converter_leg "kc" || overall_rc=$?
    fi
    ;;
esac

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
