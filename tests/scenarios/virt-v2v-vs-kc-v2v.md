# Forklift conversion: virt-v2v vs kc-v2v

Cold migration comparison on OpenShift MTV, same source VMs and storage class.
The numbers below are a **historical sequential** baseline (RHEL then Windows).
The current runner (`test-mtv-benchmark.sh`) migrates three VMs in one plan and
samples the conversion pods in parallel.

| Converter | Image | Run id |
|---|---|---|
| **virt-v2v** (default Forklift) | operator default | `20260810T103043Z` (`runs/ref-20260810T103043Z*`) |
| **kc-v2v** | kc-v2v image under test | `20260810T103043Z` (`runs/kc-20260810T103043Z*`) |

Default SC: `ocs-storagecluster-ceph-rbd-virtualization`  
`feature_windows_wait_for_reboot=false` (fair conversion comparison)  
Canonical metrics: archived under [`docs/architecture/ref-baseline/runs/`](../../docs/architecture/ref-baseline/runs/) and summarized in [`docs/architecture/ref-baseline/README.md`](../../docs/architecture/ref-baseline/README.md). Runner: [`test-mtv-benchmark.sh`](test-mtv-benchmark.sh) (`MODE=compare`). Per-run Chart.js dashboards are written to [`runs/`](runs/) as `test-mtv-benchmark-<ts>.html`.

---

## Architecture and packaging

| | **virt-v2v** (Forklift default) | **kc-v2v** |
|---|---|---|
| Implementation | C / libguestfs stack (`virt-v2v`) | Pure Go |
| Architectures | **x86_64 only** | Compiles to **any Go target** (amd64, arm64, …) |
| vSphere disk copy | virt-v2v disk transfer | govmomi NFC export |
| Guest conversion | libguestfs / virt-v2v pipeline | kc-prepare / kc-finalize + guestfish |

**Highlight:** kc-v2v is a pure Go converter that can be built for any architecture Forklift targets. Default virt-v2v remains x86-centric.

---

## Speed (plan wall and pipeline)

Plan wall = Initialize → VirtualMachineCreation. Conversion work for kc-v2v is almost entirely inside `ImageConversion` (disk copy + guest convert); virt-v2v splits copy into a long `DiskTransferV2v` step after a shorter conversion.

### RHEL 9

| Metric | virt-v2v | kc-v2v | Delta (kc − ref) |
|---|---:|---:|---:|
| Plan wall | 16m 08s | 12m 27s | **−3m 41s** |
| ImageConversion | 5m 36s | 11m 28s | +5m 52s |
| DiskTransferV2v | 9m 42s | 2s | −9m 40s |
| ImageConversion + DiskTransfer | 15m 18s | 11m 30s | **−3m 48s** |

### Windows Server 2019

| Metric | virt-v2v | kc-v2v | Delta (kc − ref) |
|---|---:|---:|---:|
| Plan wall | 19m 37s | 13m 44s | **−5m 53s** |
| ImageConversion | 5m 44s | 12m 59s | +7m 15s |
| DiskTransferV2v | 13m 4s | 3s | −13m 1s |
| ImageConversion + DiskTransfer | 18m 48s | 13m 2s | **−5m 46s** |

**Takeaway:** On this archived baseline, both RHEL and Windows are clearly faster end-to-end with kc-v2v. Pipeline shape differs: virt-v2v spends most time in `DiskTransferV2v`; kc-v2v folds NFC copy + convert into `ImageConversion`.

---

## Memory (conversion pod cgroup RSS)

Peak cgroup RSS during the conversion pod lifetime:

| VM | virt-v2v peak | kc-v2v peak | Delta |
|---|---:|---:|---:|
| RHEL | 3497 Mi | **1976 Mi** | **−1521 Mi** (−43%) |
| Windows | 2834 Mi | **1434 Mi** | **−1400 Mi** (−49%) |

kc-v2v stays low through most of NFC copy, then rises for guestfish / guest customize. virt-v2v holds multi‑GiB for a larger fraction of the pod lifetime.

---

## CPU (conversion pod, metrics-server millicores)

Peak CPU samples during conversion:

| VM | virt-v2v peak | kc-v2v peak | Delta |
|---|---:|---:|---:|
| RHEL | 3527 m | **1080 m** | **−2447 m** |
| Windows | 1895 m | **1207 m** | **−688 m** |

kc-v2v peaks later (guest convert / finalize); virt-v2v shows higher spikes, especially on RHEL.

---

## Summary

| Dimension | Winner / note |
|---|---|
| Portability | **kc-v2v** — pure Go, multi-arch |
| RHEL wall clock | **kc-v2v** (~3.5 min faster) |
| Windows wall clock | **kc-v2v** (~6 min faster) |
| Peak memory | **kc-v2v** (lower on both) |
| Peak CPU | **kc-v2v** (lower) |

Live charts: [docs/architecture/ref-baseline/dashboard.html](../../docs/architecture/ref-baseline/dashboard.html).
