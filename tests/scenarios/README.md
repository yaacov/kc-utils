# Cluster scenario tests (MTV / kc-v2v)

Manual end-to-end benchmarks against a live OpenShift cluster with MTV
(Forklift) and a vSphere source. Not part of GitHub CI.

There is one runner: [`test-mtv-benchmark.sh`](test-mtv-benchmark.sh).
Cleanup: [`test-mtv-benchmark-cleanup.sh`](test-mtv-benchmark-cleanup.sh).

## Layout

```
tests/scenarios/
├── test-mtv-benchmark.sh          # runner (MODE=kc | compare)
├── test-mtv-benchmark-cleanup.sh  # standalone cleanup
├── test-mtv-benchmark.md          # test plan
├── virt-v2v-vs-kc-v2v.md           # published comparison narrative
├── lib/
│   ├── common.sh                  # preflight, settings, providers, inventory wait
│   ├── cleanup.sh                 # namespace/plan/settings cleanup helpers
│   ├── generate-run-dashboard.py
│   └── parse-kc-v2v-pod-phases.sh
└── runs/                          # gitignored live outputs
    └── .gitignore
```

Published archives and the static comparison dashboard live under
[`docs/ref-baseline/`](../../docs/ref-baseline/).

## Prerequisites

- `oc` with the `mtv` plugin, `jq`, `python3` (dashboard generation)
- MTV installed; `vddk_image` configured
- `GOVC_URL`, `GOVC_USERNAME`, `GOVC_PASSWORD` for vSphere with `mtv-func*` VMs
- A cluster-pullable kc-v2v image (`make build-kc-v2v-image push-kc-v2v-image`)

Integration reference: [docs/forklift-usage.md](../../docs/forklift-usage.md).

## Quick start

```bash
make build-kc-v2v-image push-kc-v2v-image
export KC_V2V_IMAGE=quay.io/you/kc-v2v:devel-amd64
export GOVC_URL=... GOVC_USERNAME=... GOVC_PASSWORD=...

# Optional but recommended: pin source VMs
export RHEL_VM=mtv-func-cold-rhel9-staticips
export WIN_VM=mtv-func-win2008

# Independent kc-v2v benchmark (logs + mem/CPU/net + pod logs + HTML)
MODE=kc ./tests/scenarios/test-mtv-benchmark.sh

# Full compare: kc-v2v then operator-default virt-v2v
MODE=compare ./tests/scenarios/test-mtv-benchmark.sh

# Clean up after a run (default: delete namespace + reset MTV settings)
./tests/scenarios/test-mtv-benchmark-cleanup.sh

# Or opt in to cleanup at the end of a benchmark run
SKIP_CLEANUP=false KEEP_IMAGE_SETTING=false MODE=kc ./tests/scenarios/test-mtv-benchmark.sh
```

Each leg sets or clears `virt_v2v_image_fqin`, waits for the forklift-controller
rollout to sync `VIRT_V2V_IMAGE`, then creates a fresh namespace and providers.
Inventory VM discovery retries until `mtv-func*` RHEL and Windows VMs appear
(unless `RHEL_VM` / `WIN_VM` are pinned).

## Artifacts (`runs/`)

Live outputs (gitignored):

| Path | Contents |
|---|---|
| `runs/test-mtv-benchmark-<ts>-kc.log` | kc leg summary |
| `runs/test-mtv-benchmark-<ts>-kc-mem/` | CSVs + conversion-pod logs |
| `runs/test-mtv-benchmark-<ts>-ref.log` / `-ref-mem/` | compare mode only |
| `runs/test-mtv-benchmark-<ts>.html` | Chart.js dashboard for this run |

Regenerate a dashboard:

```bash
python3 tests/scenarios/lib/generate-run-dashboard.py \
  tests/scenarios/runs/test-mtv-benchmark-<ts>
```

To publish a comparison, copy `runs/test-mtv-benchmark-<ts>-*` into
[`docs/ref-baseline/runs/`](../../docs/ref-baseline/runs/) and update
[`docs/ref-baseline/README.md`](../../docs/ref-baseline/README.md).

## Scripts

| Path | Purpose |
|---|---|
| [test-mtv-benchmark.sh](test-mtv-benchmark.sh) | Benchmark runner (`MODE=kc` or `MODE=compare`) |
| [test-mtv-benchmark-cleanup.sh](test-mtv-benchmark-cleanup.sh) | Standalone cleanup (`--all`, `--namespace-only`, `--settings-only`, `--release-rhel`) |
| [test-mtv-benchmark.md](test-mtv-benchmark.md) | Test plan |
| [lib/common.sh](lib/common.sh) | Shared helpers (settings, providers, inventory wait, controller sync) |
| [lib/cleanup.sh](lib/cleanup.sh) | Cleanup helpers (namespace, plans, MTV settings) |
| [lib/generate-run-dashboard.py](lib/generate-run-dashboard.py) | Build per-run HTML from CSVs |
| [lib/parse-kc-v2v-pod-phases.sh](lib/parse-kc-v2v-pod-phases.sh) | Parse kc-v2v pod logs into phase durations |
| [virt-v2v-vs-kc-v2v.md](virt-v2v-vs-kc-v2v.md) | Architecture and performance comparison narrative |

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `MODE` | `kc` | `kc` = independent kc run; `compare` = kc then ref |
| `KC_V2V_IMAGE` | (required) | kc-v2v image FQIN |
| `GOVC_*` | — | vSphere credentials |
| `RHEL_VM` / `WIN_VM` | auto-picked | Pin source VMs (recommended) |
| `NS` | `mtv-kc-v2v-bench` | Test namespace |
| `PROVIDER` | `vsphere-test` | vSphere provider name |
| `SKIP_CLEANUP` | `true` | Keep namespace after exit |
| `KEEP_BETWEEN_TESTS` | `true` | Leave RHEL plan/pods while running Windows |
| `KEEP_IMAGE_SETTING` | `true` | Do not restore `virt_v2v_image_fqin` |
| `DISABLE_WAIT_FOR_REBOOT` | `true` | Set `feature_windows_wait_for_reboot=false` |
| `MEM_INTERVAL` | `10` | Seconds between metric samples |
| `INTERVAL` | `10` | Seconds between plan status polls |
| `MAX_ATTEMPTS` | `180` | Max poll attempts (~30 min per plan) |
