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
  --value quay.io/yaacov/kc-v2v:devel-amd64
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
  --type warm \
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

## Related

- [docs/kc-v2v.md](kc-v2v.md) — kc-v2v orchestrator reference
- [docs/virt-v2v-vs-kc-v2v.md](virt-v2v-vs-kc-v2v.md) — Benchmark comparison
- [build/kc-v2v/README.md](../build/kc-v2v/README.md) — Container image and Forklift Plan config
