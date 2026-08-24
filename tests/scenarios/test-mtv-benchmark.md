# Test Plan: MTV conversion benchmark (kc / compare)

## Objective

Cold-migrate three `mtv-func*` VMs from vSphere to OpenShift Virtualization
with MTV, capturing full artifacts (summary log, mem/CPU/net samples,
conversion-pod logs, pipeline timings).

Two modes:

| Mode | What it runs | Use when |
|---|---|---|
| `kc` (default) | Once with `KC_V2V_IMAGE` | Independent kc-v2v benchmark |
| `compare` | Twice: operator-default virt-v2v, then kc-v2v | Full side-by-side compare |

## Prerequisites

- `oc` with `mtv` plugin, `jq`, and `python3` (for the per-run dashboard)
- MTV installed with VDDK image configured
- `tests/scenarios/.env` configured (`GOVC_*`, `KC_V2V_IMAGE`, `NS`, `PROVIDER`)
- At least three inventory VMs matching `mtv-func*` (or pin with `VM1` / `VM2` / `VM3`)

Optional overrides: `VM1` / `VM2` / `VM3`, `NS`, `BENCHMARK_PLAN`,
`SKIP_CLEANUP`, `KEEP_IMAGE_SETTING`, `DISABLE_WAIT_FOR_REBOOT`,
`MEM_INTERVAL`, `INTERVAL`, `MAX_ATTEMPTS`.

Warm / preflight caveats: [docs/apps/forklift-usage.md](../../docs/apps/forklift-usage.md).

## Test Steps

1. Preflight: verify `oc`, MTV settings, and VDDK image; save current
   `virt_v2v_image_fqin` and `feature_windows_wait_for_reboot`
2. **compare only — ref leg:** clear `virt_v2v_image_fqin`, wait for forklift-controller
   `VIRT_V2V_IMAGE` sync, create namespace + providers, wait for inventory
   (unless VMs pinned), create one cold plan with three VMs (`plan-bench`);
   sample all conversion pods in parallel; archive pod logs
3. **kc leg:** set image to `KC_V2V_IMAGE`, wait for controller sync,
   fresh namespace + providers, repeat the three-VM plan as the **kc** leg
   (in `MODE=kc`, only this leg runs)
4. Generate `runs/test-mtv-benchmark-<ts>.html` dashboard from CSVs
5. By default, leave MTV settings, plan/pods, and namespace in place
   (`KEEP_IMAGE_SETTING=true`, `SKIP_CLEANUP=true`).
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

- Every planned leg reaches `Completed` or `Succeeded` for the plan **and**
  all three VMs
- In `compare` mode, a failed ref leg skips the kc leg and fails overall

## Fail Criteria

- Providers do not become Ready
- The plan ends in `Failed`, `Error`, `Cancelled`, or times out (~30 minutes)
- Any VM in the plan fails (conversion pods still run in parallel)

## How to Run

```bash
# Build and push the image under test
make build-kc-v2v-image push-kc-v2v-image

cp tests/scenarios/.env.example tests/scenarios/.env
# edit .env

# Independent kc-v2v benchmark (default)
MODE=kc ./tests/scenarios/test-mtv-benchmark.sh

# Full compare: operator-default virt-v2v then kc-v2v
MODE=compare ./tests/scenarios/test-mtv-benchmark.sh

# Clean up afterward
./tests/scenarios/clean-env.sh
```

Narrative comparison of archived results:
[virt-v2v-vs-kc-v2v.md](virt-v2v-vs-kc-v2v.md). Published tables and dashboard:
[docs/architecture/ref-baseline/README.md](../../docs/architecture/ref-baseline/README.md).
