#!/usr/bin/env bash
#
# Remove local benchmark artifacts under tests/scenarios/runs/.
# Keeps .gitignore so the directory stays tracked.
#
# Examples:
#   ./tests/scenarios/clean-runs.sh
#   ./tests/scenarios/clean-runs.sh --dry-run

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RUNS_DIR="${SCRIPT_DIR}/runs"
DRY_RUN=false

usage() {
  cat <<EOF
Usage: $0 [options]

Remove local benchmark artifacts from tests/scenarios/runs/
(logs, *-mem/ directories, HTML dashboards). Preserves .gitignore.

Options:
  --dry-run   List what would be removed without deleting
  -h, --help  Show this help
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dry-run)
      DRY_RUN=true
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

if [[ ! -d "${RUNS_DIR}" ]]; then
  echo "No runs directory at ${RUNS_DIR} — nothing to clean."
  exit 0
fi

shopt -s nullglob dotglob
entries=("${RUNS_DIR}"/*)
shopt -u nullglob dotglob

to_remove=()
for entry in "${entries[@]}"; do
  base="$(basename "${entry}")"
  [[ "${base}" == ".gitignore" ]] && continue
  to_remove+=("${entry}")
done

if [[ ${#to_remove[@]} -eq 0 ]]; then
  echo "runs/ is already empty (only .gitignore)."
  exit 0
fi

if [[ "${DRY_RUN}" == "true" ]]; then
  echo "Would remove ${#to_remove[@]} item(s) from ${RUNS_DIR}:"
  for entry in "${to_remove[@]}"; do
    echo "  ${entry#"${RUNS_DIR}"/}"
  done
  exit 0
fi

echo "Cleaning ${RUNS_DIR} (${#to_remove[@]} item(s))..."
for entry in "${to_remove[@]}"; do
  rm -rf "${entry}"
  echo "  removed ${entry#"${RUNS_DIR}"/}"
done
echo "Done. .gitignore preserved."
