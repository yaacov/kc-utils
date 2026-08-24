# Cluster scenario tests (MTV / kc-v2v)

Manual end-to-end benchmarks against a live OpenShift cluster with MTV
(Forklift) and a vSphere source. Not part of GitHub CI.

There is one runner: [`test-mtv-benchmark.sh`](test-mtv-benchmark.sh).
Cluster cleanup: [`clean-env.sh`](clean-env.sh).
Local `runs/` cleanup: [`clean-runs.sh`](clean-runs.sh).

## Layout

```
tests/scenarios/
├── test-mtv-benchmark.sh          # runner (MODE=kc | compare)
├── clean-env.sh                   # standalone cluster cleanup
├── clean-runs.sh                  # wipe local runs/ artifacts
├── test-mtv-benchmark.md          # test plan
├── .env.example                   # template — copy to .env (gitignored)
├── virt-v2v-vs-kc-v2v.md           # published comparison narrative
├── lib/
│   ├── common.sh                  # .env load, preflight, settings, providers
│   ├── cleanup.sh                 # namespace/plan/settings cleanup helpers
│   ├── generate-run-dashboard.py
│   └── parse-kc-v2v-pod-phases.sh
└── runs/                          # gitignored live outputs
    └── .gitignore
```

Published archives and the static comparison dashboard live under
[`docs/architecture/ref-baseline/`](../../docs/architecture/ref-baseline/).

## Prerequisites

- `oc` with the `mtv` plugin, `jq`, `python3` (dashboard generation)
- MTV installed; `vddk_image` configured
- `tests/scenarios/.env` with vSphere creds, `KC_V2V_IMAGE`, `NS`, and `PROVIDER`
  (copy from `.env.example`)
- MTV installed; `vddk_image` configured
- vSphere inventory with at least three `mtv-func*` VMs (or pin `VM1` / `VM2` / `VM3` in `.env`)

Integration reference: [docs/apps/forklift-usage.md](../../docs/apps/forklift-usage.md).

## Quick start

```bash
make build-kc-v2v-image push-kc-v2v-image
cp tests/scenarios/.env.example tests/scenarios/.env
# edit .env: GOVC_*, KC_V2V_IMAGE, NS, PROVIDER; optional VM1 / VM2 / VM3

# Independent kc-v2v benchmark (logs + mem/CPU/net + pod logs + HTML)
MODE=kc ./tests/scenarios/test-mtv-benchmark.sh

# Full compare: operator-default virt-v2v then kc-v2v
MODE=compare ./tests/scenarios/test-mtv-benchmark.sh

# Clean up after a run (default: delete namespace + reset MTV settings)
./tests/scenarios/clean-env.sh

# Wipe local runs/ artifacts (logs, *-mem/, HTML); keeps .gitignore
./tests/scenarios/clean-runs.sh

# Or opt in to cleanup at the end of a benchmark run
SKIP_CLEANUP=false KEEP_IMAGE_SETTING=false MODE=kc ./tests/scenarios/test-mtv-benchmark.sh
```

Each leg sets or clears `virt_v2v_image_fqin`, waits for the forklift-controller
rollout to sync `VIRT_V2V_IMAGE`, then creates a fresh namespace and providers.
Inventory VM discovery retries until three `mtv-func*` VMs appear
(unless `VM1` / `VM2` / `VM3` are pinned). All three VMs are migrated in one plan
(`plan-bench`); conversion-pod metrics are sampled in parallel.

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
[`docs/architecture/ref-baseline/runs/`](../../docs/architecture/ref-baseline/runs/) and update
[`docs/architecture/ref-baseline/README.md`](../../docs/architecture/ref-baseline/README.md).

## Scripts

| Path | Purpose |
|---|---|
| [test-mtv-benchmark.sh](test-mtv-benchmark.sh) | Benchmark runner (`MODE=kc` or `MODE=compare`) |
| [clean-env.sh](clean-env.sh) | Standalone cluster cleanup (`--all`, `--namespace-only`, `--settings-only`, `--release-rhel`, `--release-plans`; `RHEL_VM` names the migrated RHEL VM for `--release-rhel`) |
| [clean-runs.sh](clean-runs.sh) | Remove local `runs/` artifacts (`--dry-run` to preview) |
| [test-mtv-benchmark.md](test-mtv-benchmark.md) | Test plan |
| [lib/common.sh](lib/common.sh) | Shared helpers (settings, providers, inventory wait, controller sync) |
| [lib/cleanup.sh](lib/cleanup.sh) | Cleanup helpers (namespace, plans, MTV settings) |
| [lib/generate-run-dashboard.py](lib/generate-run-dashboard.py) | Build per-run HTML from CSVs |
| [lib/parse-kc-v2v-pod-phases.sh](lib/parse-kc-v2v-pod-phases.sh) | Parse kc-v2v pod logs into phase durations |
| [virt-v2v-vs-kc-v2v.md](virt-v2v-vs-kc-v2v.md) | Architecture and performance comparison narrative |

## Environment variables

| Variable | Required in `.env` | Description |
|---|---|---|
| `GOVC_URL` / `GOVC_USERNAME` / `GOVC_PASSWORD` | yes | vSphere credentials |
| `PROVIDER_INSECURE_SKIP_TLS` | no | `true` skips TLS verify; `false` (default) fetches CA from `GOVC_URL:443` |
| `KC_V2V_IMAGE` | yes | kc-v2v image FQIN |
| `NS` | yes | Test namespace |
| `PROVIDER` | yes | vSphere provider name |
| `VM1` / `VM2` / `VM3` | no | Pin source VMs (auto-picked from mtv-func* when unset) |
| `MODE` | no | `kc` or `compare` (default `kc`, shell override) |
| `SKIP_CLEANUP` | no | Keep namespace after exit (default `true`) |
| `KEEP_IMAGE_SETTING` | no | Do not restore `virt_v2v_image_fqin` (default `true`) |
| `DISABLE_WAIT_FOR_REBOOT` | no | Set `feature_windows_wait_for_reboot=false` (default `true`) |
| `MEM_INTERVAL` | no | Seconds between metric samples (default `10`) |
| `INTERVAL` | no | Seconds between plan status polls (default `10`) |
| `MAX_ATTEMPTS` | no | Max poll attempts (~30 min for the combined plan, default `180`) |
