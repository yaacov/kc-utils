#!/usr/bin/env bash
# Parse kc-v2v virt-v2v pod logs into copy / prepare / convert / finalize durations.
# Usage: lib/parse-kc-v2v-pod-phases.sh <pod-log-file>
set -euo pipefail

LOG="${1:-}"
if [[ -z "${LOG}" || ! -f "${LOG}" ]]; then
  echo "Usage: $0 <pod-log-file>" >&2
  exit 1
fi

# Extract ISO timestamp from structured log lines
ts() {
  # time=2026-08-03T09:37:00.427Z
  sed -n 's/.*time=\([0-9T:.-]*Z\).*/\1/p' | head -1
}

epoch() {
  # macOS date
  date -u -j -f "%Y-%m-%dT%H:%M:%S" "$(echo "$1" | sed 's/\.[0-9]*Z$//; s/T/ /')" "+%s" 2>/dev/null \
    || date -u -d "${1}" "+%s" 2>/dev/null
}

fmt_dur() {
  local s="$1"
  if [[ -z "${s}" || "${s}" -lt 0 ]]; then
    echo "n/a"
    return
  fi
  if [[ "${s}" -lt 60 ]]; then
    echo "${s}s"
  else
    echo "$((s / 60))m $((s % 60))s"
  fi
}

COPY_START=$(rg -m1 'msg=exec bin=.*/kc-copy|msg="VDDK copy starting"' "${LOG}" | ts || true)
COPY_END=$(rg -m1 'msg="VDDK copy complete"' "${LOG}" | ts || true)
# If no VDDK copy (disks already present), copy may be skipped
PREPARE_START=$(rg -m1 'msg=exec bin=.*/kc-prepare|msg="kc-prepare starting"' "${LOG}" | ts || true)
# convert starts when kc-convert-* is exec'd
CONVERT_START=$(rg -m1 'msg=exec bin=.*/kc-convert-' "${LOG}" | ts || true)
FINALIZE_START=$(rg -m1 'msg=exec bin=.*/kc-finalize' "${LOG}" | ts || true)
# End of finalize / conversion success
FINALIZE_END=$(rg -m1 'msg="conversion complete"|msg="kc-finalize complete"|msg="wrote inspection"|conversion: succeeded' "${LOG}" | ts || true)
# Fallback: last timestamp in log
if [[ -z "${FINALIZE_END}" ]]; then
  FINALIZE_END=$(rg 'time=' "${LOG}" | tail -1 | ts || true)
fi

# Prepare ends when convert starts (or finalize if convert skipped)
PREPARE_END="${CONVERT_START:-${FINALIZE_START}}"
CONVERT_END="${FINALIZE_START}"

echo "=== kc-v2v phase timestamps ==="
echo "copy_start:     ${COPY_START:-none}"
echo "copy_end:       ${COPY_END:-none}"
echo "prepare_start:  ${PREPARE_START:-none}"
echo "convert_start:  ${CONVERT_START:-none}"
echo "finalize_start: ${FINALIZE_START:-none}"
echo "finalize_end:   ${FINALIZE_END:-none}"
echo ""

if [[ -n "${COPY_START}" && -n "${COPY_END}" ]]; then
  COPY_S=$(( $(epoch "${COPY_END}") - $(epoch "${COPY_START}") ))
else
  COPY_S=""
fi
if [[ -n "${PREPARE_START}" && -n "${PREPARE_END}" ]]; then
  PREPARE_S=$(( $(epoch "${PREPARE_END}") - $(epoch "${PREPARE_START}") ))
else
  PREPARE_S=""
fi
if [[ -n "${CONVERT_START}" && -n "${CONVERT_END}" ]]; then
  CONVERT_S=$(( $(epoch "${CONVERT_END}") - $(epoch "${CONVERT_START}") ))
else
  CONVERT_S=""
fi
if [[ -n "${FINALIZE_START}" && -n "${FINALIZE_END}" ]]; then
  FINALIZE_S=$(( $(epoch "${FINALIZE_END}") - $(epoch "${FINALIZE_START}") ))
else
  FINALIZE_S=""
fi

echo "=== kc-v2v phase durations ==="
printf "copy:     %s\n" "$(fmt_dur "${COPY_S:-}")"
printf "prepare:  %s\n" "$(fmt_dur "${PREPARE_S:-}")"
printf "convert:  %s\n" "$(fmt_dur "${CONVERT_S:-}")"
printf "finalize: %s\n" "$(fmt_dur "${FINALIZE_S:-}")"

# Machine-readable for report updates
echo ""
echo "COPY_S=${COPY_S:-}"
echo "PREPARE_S=${PREPARE_S:-}"
echo "CONVERT_S=${CONVERT_S:-}"
echo "FINALIZE_S=${FINALIZE_S:-}"
