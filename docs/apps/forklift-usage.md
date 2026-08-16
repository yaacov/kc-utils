# Using kc-v2v with Forklift (MTV)

kc-v2v is a drop-in replacement for the virt-v2v conversion image in
[Forklift / Migration Toolkit for Virtualization](https://github.com/kubev2v/forklift).
Point Forklift at the kc-v2v container image and run migrations as usual.

> **Note:** kc-v2v does not support warm preflight inspection. For warm
> vSphere plans, use `--run-preflight-inspection false`.

## Prerequisites

- `oc` CLI with the `mtv` plugin installed
- MTV (Forklift) installed on the cluster
- A source provider (e.g. vSphere) configured in Forklift

## Naming: Forklift vs kc-utils

Forklift keeps **virt-v2v** names for the conversion pod and cluster settings.
kc-utils replaces only the **container image contents** (entrypoint `kc-v2v`).

| Layer | Forklift / MTV name | kc-utils name | Notes |
|-------|---------------------|---------------|--------|
| Cluster setting (`oc mtv settings`) | `virt_v2v_image_fqin` | Image FQIN value, e.g. `quay.io/you/kc-v2v:tag` | There is **no** `kc-v2v` MTV setting |
| Controller env (synced from setting) | `VIRT_V2V_IMAGE` | Same FQIN string | Operator deployment, not conversion-pod env |
| Conversion pod label | `forklift.app=virt-v2v` | — | `oc get pods -l forklift.app=virt-v2v` |
| Pod container name | `virt-v2v` | Runs `/usr/bin/kc-v2v` when using this image |
| Conversion pod env | See [below](#environment-variables-by-app) | Read only by `kc-v2v` (`pkg/v2v/env`) | Stage binaries do not read Forklift env |
| Local tests / Makefile | — | `KC_V2V_IMAGE` | Test helper only; scripts set `virt_v2v_image_fqin` from it |

Upstream default image: `quay.io/kubev2v/forklift-virt-v2v`. To use kc-utils,
set **`virt_v2v_image_fqin`** to your **`kc-v2v`** image — the pod is still
called virt-v2v in MTV.

### Environment variables by app

Exact names. What `kc-v2v` loads:
[`pkg/v2v/config/config.go`](../../pkg/v2v/config/config.go).

#### Forklift controller (`konveyor-forklift` deployment)

Not on the conversion pod:

| Variable | Role |
|----------|------|
| `VIRT_V2V_IMAGE` | Synced from `virt_v2v_image_fqin`; image for the virt-v2v conversion pod |
| `VIRT_V2V_EXTRA_ARGS` | From `virt_v2v_extra_args` → pod `V2V_extra_args` |
| `VIRT_V2V_INSPECTOR_EXTRA_ARGS` | Inspector-only; not used by `kc-v2v` |

#### Conversion pod (`virt-v2v` container)

Forklift `AppConfig` env on a typical cold vSphere conversion. Credentials and
CA are files under `/etc/secret`, not env.

| Variable | Purpose | `kc-v2v` |
|----------|---------|---------|
| `V2V_source` | Source hypervisor (e.g. `vSphere`) | Respected |
| `V2V_libvirtURL` | vCenter URL (inventory + NFC); `no_verify=1` → insecure TLS | Respected |
| `V2V_vmName` | Source VM name | Respected |
| `V2V_fingerprint` | vCenter SSL thumbprint | Respected |
| `LOCAL_MIGRATION` | Serve Forklift HTTP API on `:8080` | Respected |
| `V2V_extra_args` | From `virt_v2v_extra_args` | Read, ignored (warning) |
| `V2V_inspector_extra_args` | Inspector args | Not read |
| `V2V_preserveStaticIPs` | From Plan `preserveStaticIPs` (controller sets; unused by pod, same as upstream virt-v2v) | Not read; feature via `V2V_staticIPs` |

Optional Plan / feature env Forklift may also set:

| Variable | Purpose | `kc-v2v` |
|----------|---------|---------|
| `V2V_inPlace` | Skip disk copy (pre-filled PVCs) | Respected |
| `V2V_staticIPs` | Static IP mapping (set when Plan `preserveStaticIPs` yields MAC→IP data) | Respected |
| `V2V_HOSTNAME` | Guest hostname | Respected |
| `V2V_NBDE_CLEVIS` | Clevis LUKS unlock | Respected |
| `V2V_NewName` | Destination VM name | Respected |
| `V2V_RootDisk` | Root disk policy | Respected |
| `V2V_firmware` | `uefi` / `bios` override | Respected |
| `V2V_diskPath` | Explicit VMDK paths | Respected |
| `V2V_multipleIPsPerNic` | Multiple IPs per NIC | Respected |
| `V2V_vsphereVmwareDriverRemoval` | Remove VMware tools/drivers | Respected |
| `V2V_windowsRegistryNetworkConfig` | Windows registry network config | Respected |
| `V2V_waitForGuestReboot` | Signal conversion done on COM1 | Respected |
| `V2V_overlayEnabled` | qcow2 overlay (default `true` if unset) | Respected |

Image / `kc-v2v`-only (not Forklift `AppConfig`; image defaults or overrides):

| Variable | Role |
|----------|------|
| `V2V_backend` | Guest disk backend (`direct`\|`guestfs`; image often sets `guestfs`) |
| `V2V_offline` | Pass `--offline` to converters |
| `V2V_copyConcurrency` | Max parallel NFC copies (default `4`) |
| `KC_BIN_DIR` | Stage binary dir (default `/usr/lib/kc-utils`) |

#### Stage binaries

`kc-copy`, `kc-prepare`, `kc-convert-linux`, `kc-convert-windows`, `kc-finalize`
do not read Forklift env. They take JSON/CLI from `kc-v2v`. In guestfs mode they
adopt the shared listener via `GUESTFISH_PID` / `KC_GUESTFISH_PID` (and
optionally `KC_GUESTFS_NETWORK=1` for Clevis).

#### Local MTV scenario tests (host shell — not pod env)

| Variable | Role |
|----------|------|
| `KC_V2V_IMAGE` | FQIN written to MTV `virt_v2v_image_fqin` by benchmark scripts |
| `GOVC_URL`, `GOVC_USERNAME`, `GOVC_PASSWORD` | govc / provider setup in `tests/scenarios/.env` |

Full flag/env reference: [kc-v2v.md](kc-v2v.md#configuration).

## 1. Configure the conversion image

```bash
oc mtv settings set --setting virt_v2v_image_fqin \
  --value quay.io/yaacov/kc-v2v:devel-amd64
```

All subsequent migrations run the **virt-v2v conversion pod** with your
**kc-v2v** container image (same MTV setting name as upstream virt-v2v).

## 2. Create a migration plan

Cold migration:

```bash
oc mtv create plan --name my-plan \
  --source my-vsphere-provider --target host \
  --vms my-vm \
  -n my-namespace
```

Warm migration (disable preflight inspection):

```bash
oc mtv create plan --name my-plan \
  --source my-vsphere-provider --target host \
  --migration-type warm \
  --vms my-vm \
  --run-preflight-inspection false \
  -n my-namespace
```

## 3. Start the migration

```bash
oc mtv start plan --name my-plan -n my-namespace
```

## 4. Monitor progress

```bash
oc mtv get plan --name my-plan -n my-namespace
oc mtv get plan --name my-plan -n my-namespace --vms
```

The plan status reaches `Completed` or `Succeeded` when the migration finishes.

## Configuration Reference

See [build/kc-v2v/README.md](../build/kc-v2v/README.md#forklift-configuration-for-kc-v2v)
for full details on cluster settings, Plan fields, Conversion CR types, and
features that kc-v2v does not handle.

## Cluster tests

Manual MTV scenario tests live under
[tests/scenarios/](../tests/scenarios/README.md) (not run in CI).

```bash
# Build, push, then benchmark (logs + mem/CPU/net + pod logs + HTML)
make build-kc-v2v-image push-kc-v2v-image
cp tests/scenarios/.env.example tests/scenarios/.env   # set KC_V2V_IMAGE, GOVC_*, etc.
MODE=kc ./tests/scenarios/test-mtv-benchmark.sh          # independent kc-v2v
MODE=compare ./tests/scenarios/test-mtv-benchmark.sh     # kc then operator default
./tests/scenarios/clean-env.sh          # delete NS + reset MTV settings
```

See [tests/scenarios/test-mtv-benchmark.md](../../tests/scenarios/test-mtv-benchmark.md)
and [docs/architecture/ref-baseline/README.md](../architecture/ref-baseline/README.md).

## Related

- [kc-v2v.md](kc-v2v.md) — kc-v2v orchestrator reference
- [../../tests/scenarios/README.md](../../tests/scenarios/README.md) — Cluster benchmark runner
- [../../tests/scenarios/virt-v2v-vs-kc-v2v.md](../../tests/scenarios/virt-v2v-vs-kc-v2v.md) — virt-v2v vs kc-v2v comparison
- [../architecture/ref-baseline/README.md](../architecture/ref-baseline/README.md) — Benchmark comparison (results, dashboard, archived runs)
- [Dashboard](https://htmlpreview.github.io/?https://github.com/yaacov/kc-utils/blob/main/docs/architecture/ref-baseline/dashboard.html) ([source](../architecture/ref-baseline/dashboard.html)) — Interactive charts (memory, CPU, network over time)
- [../../build/kc-v2v/README.md](../../build/kc-v2v/README.md) — Container image and Forklift Plan config
