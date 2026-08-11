# Test Plan: MTV conversion benchmark (kc / compare)

## Objective

Cold-migrate one `mtv-func*` RHEL VM and one Windows VM from vSphere to
OpenShift Virtualization with MTV, capturing full artifacts (summary log,
mem/CPU/net samples, conversion-pod logs, pipeline timings).

Two modes:

| Mode | What it runs | Use when |
|---|---|---|
| `kc` (default) | Once with `KC_V2V_IMAGE` | Independent kc-v2v benchmark |
| `compare` | Twice: kc-v2v, then operator-default virt-v2v | Full side-by-side compare |

## Prerequisites

- `oc` with `mtv` plugin, `jq`, and `python3` (for the per-run dashboard)
- MTV installed with VDDK image configured
- `tests/scenarios/.env` configured (`GOVC_*`, `KC_V2V_IMAGE`, `NS`, `PROVIDER`)
- At least one inventory VM matching `mtv-func*` with a RHEL guest and one with Windows
  (or pin with `RHEL_VM` / `WIN_VM`)

Optional overrides: `RHEL_VM` / `WIN_VM`, `NS`, `SKIP_CLEANUP`,
`KEEP_BETWEEN_TESTS`, `KEEP_IMAGE_SETTING`, `DISABLE_WAIT_FOR_REBOOT`,
`MEM_INTERVAL`, `INTERVAL`, `MAX_ATTEMPTS`.

Warm / preflight caveats: [docs/apps/forklift-usage.md](../../docs/apps/forklift-usage.md).

## Test Steps

1. Preflight: verify `oc`, MTV settings, and VDDK image; save current
   `virt_v2v_image_fqin` and `feature_windows_wait_for_reboot`
2. **kc leg:** set image to `KC_V2V_IMAGE`, wait for forklift-controller
   `VIRT_V2V_IMAGE` sync, create namespace + providers, wait for inventory
   (unless VMs pinned), cold plan RHEL then Windows; sample conversion-pod
   metrics; archive pod logs
3. **compare only:** clear `virt_v2v_image_fqin`, wait for controller sync,
   fresh namespace + providers, repeat RHEL then Windows as the **ref** leg
4. Generate `runs/test-mtv-benchmark-<ts>.html` dashboard from CSVs
5. By default, leave MTV settings, RHEL plan/pods, and namespace in place
   (`KEEP_IMAGE_SETTING=true`, `KEEP_BETWEEN_TESTS=true`, `SKIP_CLEANUP=true`).
   Run [`clean-env.sh`](clean-env.sh) afterward,
   or set `SKIP_CLEANUP=false KEEP_IMAGE_SETTING=false` to clean up on exit.

## Artifacts

Under `tests/scenarios/runs/` (gitignored):

- `test-mtv-benchmark-<ts>-kc.log` / `-kc-mem/`
- `test-mtv-benchmark-<ts>-ref.log` / `-ref-mem/` (compare mode only)
- `test-mtv-benchmark-<ts>.html` — Chart.js dashboard for this run

Each `-mem/` directory holds per-VM CSVs and conversion-pod logs. Archive under
[`docs/architecture/ref-baseline/runs/`](../../docs/architecture/ref-baseline/runs/) when publishing.

Regenerate a dashboard later:

```bash
python3 tests/scenarios/lib/generate-run-dashboard.py \
  tests/scenarios/runs/test-mtv-benchmark-<ts>
```

## Pass Criteria

- Every planned leg reaches `Completed` or `Succeeded` for both VMs
- In `compare` mode, a failed kc leg skips the ref leg and fails overall

## Fail Criteria

- Providers do not become Ready
- Any plan ends in `Failed`, `Error`, `Cancelled`, or times out (~30 minutes)
- RHEL failure skips Windows for that leg

## How to Run

```bash
# Build and push the image under test
make build-kc-v2v-image push-kc-v2v-image

cp tests/scenarios/.env.example tests/scenarios/.env
# edit .env

# Independent kc-v2v benchmark (default)
MODE=kc ./tests/scenarios/test-mtv-benchmark.sh

# Full compare: kc-v2v then operator default
MODE=compare ./tests/scenarios/test-mtv-benchmark.sh

# Clean up afterward
./tests/scenarios/clean-env.sh
```

Narrative comparison of archived results:
[virt-v2v-vs-kc-v2v.md](virt-v2v-vs-kc-v2v.md). Published tables and dashboard:
[docs/architecture/ref-baseline/README.md](../../docs/architecture/ref-baseline/README.md).
