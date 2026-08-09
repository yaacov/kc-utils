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

## 1. Configure the conversion image

```bash
oc mtv settings set --setting virt_v2v_image_fqin \
  --value quay.io/kubev2v/kc-v2v:devel-amd64
```

All subsequent migrations will use kc-v2v for guest conversion.

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
./tests/scenarios/test-mtv-benchmark-cleanup.sh          # delete NS + reset MTV settings
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
