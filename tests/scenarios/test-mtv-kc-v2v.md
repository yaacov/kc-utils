# Test Plan: MTV-kc-v2v — RHEL then Windows cold smoke

## Objective

Verify cold migrations of one `mtv-func*` RHEL VM and one Windows VM from
vSphere to OpenShift Virtualization using MTV with the **kc-v2v** conversion
image (`virt_v2v_image_fqin`), on a self-contained namespace with freshly
created providers.

Pass/fail only — no mem/CPU sampling (see [ref-baseline](../../docs/ref-baseline/README.md)
for benchmarks).

## Prerequisites

- `oc` with `mtv` plugin and `jq` installed
- MTV installed on the cluster with VDDK image configured
- Environment variables: `GOVC_URL`, `GOVC_USERNAME`, `GOVC_PASSWORD`
- `KC_V2V_IMAGE` set to a cluster-pullable kc-v2v FQIN
  (or use `make test-cluster-smoke`, which sets it from `REGISTRY` / `REGISTRY_ORG` / `REGISTRY_TAG`)
- At least one inventory VM matching `mtv-func*` with a RHEL guest and one with Windows
- Optional overrides:
  - `RHEL_VM` / `WIN_VM` — pin source VMs
  - `SKIP_CLEANUP=true` — keep namespace after the run
  - `KEEP_IMAGE_SETTING=true` — leave `virt_v2v_image_fqin` (and reboot flag) after exit

Warm / preflight caveats: [docs/forklift-usage.md](../../docs/forklift-usage.md).

## Test Steps

1. Preflight: verify `oc`, MTV settings, and VDDK image
2. Save current `virt_v2v_image_fqin` and `feature_windows_wait_for_reboot`; set image to `KC_V2V_IMAGE` and `feature_windows_wait_for_reboot=false`
3. Create namespace `mtv-kc-v2v-test`
4. Create vSphere provider (`vsphere-test`) and OpenShift provider (`host`); wait Ready
5. Select one mtv-func RHEL VM and one Windows VM from inventory
6. Create cold plan `plan-smoke-rhel` with `--run-preflight-inspection false`; start; wait for terminal status
7. On RHEL success: delete RHEL plan (free memory), then cold plan `plan-smoke-win` the same way
8. Restore prior MTV settings (unless `KEEP_IMAGE_SETTING=true`); cleanup namespace (unless `SKIP_CLEANUP=true`)

## Pass Criteria

- Both plans reach `Completed` or `Succeeded`

## Fail Criteria

- Providers do not become Ready
- Either plan ends in `Failed`, `Error`, `Cancelled`, or times out (~30 minutes each)
- RHEL failure skips Windows and fails the overall test

## How to Run

```bash
# Build, push, then smoke (recommended)
make build-kc-v2v-image push-kc-v2v-image
make test-cluster-smoke

# Direct
export KC_V2V_IMAGE=quay.io/you/kc-v2v:devel-amd64
export GOVC_URL=... GOVC_USERNAME=... GOVC_PASSWORD=...
./tests/scenarios/test-mtv-kc-v2v.sh

# Pin VMs / keep resources
RHEL_VM=mtv-func-rhel9 WIN_VM=mtv-func-win2019 SKIP_CLEANUP=true \
  ./tests/scenarios/test-mtv-kc-v2v.sh
```
