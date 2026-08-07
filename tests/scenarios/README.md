# Cluster scenario tests (MTV / kc-v2v)

Manual end-to-end tests that run against a live OpenShift cluster with MTV
(Forklift) and a vSphere source. Not part of GitHub CI.

## Prerequisites

- `oc` with the `mtv` plugin, `jq`
- MTV installed; `vddk_image` configured
- `GOVC_URL`, `GOVC_USERNAME`, `GOVC_PASSWORD` for vSphere with `mtv-func*` VMs
- A pushable/pullable kc-v2v image for smoke (`make build-kc-v2v-image push-kc-v2v-image`)

Integration reference: [docs/forklift-usage.md](../../docs/forklift-usage.md).

## Quick start

```bash
# Smoke: set virt_v2v_image_fqin to this build, migrate RHEL then Windows
make build-kc-v2v-image push-kc-v2v-image
make test-cluster-smoke

# Baseline benchmark (does NOT set the conversion image — set it yourself)
oc mtv settings set --setting virt_v2v_image_fqin --value "quay.io/you/kc-v2v:devel-amd64"
make test-cluster-baseline
```

`make test-cluster` is an alias for `test-cluster-smoke`.

## Scripts

| Script | Purpose |
|---|---|
| [test-mtv-kc-v2v.sh](test-mtv-kc-v2v.sh) | Smoke: RHEL then Windows cold migrate with image set/restore |
| [test-mtv-kc-v2v.md](test-mtv-kc-v2v.md) | Smoke test plan |
| [test-mtv-ref-baseline.sh](test-mtv-ref-baseline.sh) | Benchmark: sequential RHEL+Windows with mem/CPU/net sampling |
| [parse-kc-v2v-pod-phases.sh](parse-kc-v2v-pod-phases.sh) | Parse kc-v2v pod logs into phase durations |
| [common.sh](common.sh) | Shared preflight, plan wait, settings save/restore |
| [virt-v2v-vs-kc-v2v.md](virt-v2v-vs-kc-v2v.md) | Architecture and performance comparison narrative |

Published baseline results and dashboard live under
[docs/ref-baseline/](../../docs/ref-baseline/). After a baseline run, optionally
copy `test-mtv-ref-baseline-<ts>*` artifacts into `docs/ref-baseline/runs/` when
archiving a comparison.

## Environment variables (common)

| Variable | Used by | Description |
|---|---|---|
| `KC_V2V_IMAGE` | smoke | Conversion image FQIN (required for smoke) |
| `GOVC_*` | both | vSphere credentials |
| `RHEL_VM` / `WIN_VM` | both | Pin source VMs |
| `NS` | both | Test namespace |
| `SKIP_CLEANUP` | both | Keep namespace after exit |
| `KEEP_IMAGE_SETTING` | smoke | Do not restore `virt_v2v_image_fqin` |
| `DISABLE_WAIT_FOR_REBOOT` | both | Set `feature_windows_wait_for_reboot=false` (default true) |
