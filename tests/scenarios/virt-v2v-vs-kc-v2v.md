# Forklift conversion: virt-v2v vs kc-v2v

Cold migration comparison on OpenShift MTV (cluster `qemtvd-07`), same VMs and storage class, sequential runs (RHEL then Windows).

| Converter | Image | Run id |
|---|---|---|
| **virt-v2v** (default Forklift) | operator default | `20260805T211538Z` (`runs/ref-20260805T211538Z*`) |
| **kc-v2v** | `quay.io/yaacov/kc-v2v:devel-amd64` | `20260805T204233Z` (`runs/kc-20260805T204233Z*`) |

VMs: `mtv-func-cold-rhel9-staticips`, `mtv-func-win2008`  
Default SC: `ocs-storagecluster-ceph-rbd-virtualization`  
`feature_windows_wait_for_reboot=false` (fair conversion comparison)  
Canonical metrics: archived under [`docs/ref-baseline/runs/`](../../docs/ref-baseline/runs/) and summarized in [`docs/ref-baseline/README.md`](../../docs/ref-baseline/README.md). Runner: [`test-mtv-benchmark.sh`](test-mtv-benchmark.sh) (`MODE=compare`). Per-run Chart.js dashboards are written to [`runs/`](runs/) as `test-mtv-benchmark-<ts>.html`.

---

## Architecture and packaging

| | **virt-v2v** (Forklift default) | **kc-v2v** |
|---|---|---|
| Implementation | C / libguestfs stack (`virt-v2v`) | Pure Go |
| Architectures | **x86_64 only** | Compiles to **any Go target** (amd64, arm64, …) |
| vSphere disk copy | Requires **proprietary VMware VDDK** | **No VDDK** — open copy path (e.g. NFC) |
| Guest conversion | libguestfs / virt-v2v pipeline | kc-prepare / kc-finalize + guestfish |
| Licensing / redistribution | VDDK redistributability and arch limits apply | No proprietary VDDK dependency |

**Highlight:** kc-v2v is a pure Go converter that can be built for any architecture Forklift targets, and it does **not** require the proprietary VDDK for disk copy. Default virt-v2v remains x86-centric and depends on VDDK for efficient vSphere transfers.

---

## Speed (plan wall and pipeline)

Plan wall = Initialize → VirtualMachineCreation. Conversion work for kc-v2v is almost entirely inside `ImageConversion` (disk copy + guest convert); virt-v2v splits copy into a long `DiskTransferV2v` step after a shorter conversion.

### RHEL 9

| Metric | virt-v2v | kc-v2v | Delta (kc − ref) |
|---|---:|---:|---:|
| Plan wall | 15m 21s | 12m 31s | **−2m 50s** |
| ImageConversion | 4m 43s | 11m 59s | +7m 16s |
| DiskTransferV2v | 9m 56s | 4s | −9m 52s |
| ImageConversion + DiskTransfer | 14m 39s | 12m 3s | **−2m 36s** |

### Windows Server 2008

| Metric | virt-v2v | kc-v2v | Delta (kc − ref) |
|---|---:|---:|---:|
| Plan wall | 14m 6s | 13m 32s | −34s |
| ImageConversion | 3m 24s | 12m 47s | +9m 23s |
| DiskTransferV2v | 9m 58s | 3s | −9m 55s |
| ImageConversion + DiskTransfer | 13m 22s | 12m 50s | −32s |

**Takeaway:** On this archived baseline, RHEL is clearly faster end-to-end with kc-v2v. Windows is modestly faster. Pipeline shape differs: virt-v2v spends most time in `DiskTransferV2v` (VDDK); kc-v2v folds NFC copy + convert into `ImageConversion`.

---

## Memory (conversion pod cgroup RSS)

Peak cgroup RSS during the conversion pod lifetime:

| VM | virt-v2v peak | kc-v2v peak | Delta |
|---|---:|---:|---:|
| RHEL | 1894 Mi | **1804 Mi** | −90 Mi (−5%) |
| Windows | 1526 Mi | **962 Mi** | **−564 Mi** (−37%) |

kc-v2v stays low through most of NFC copy, then rises for guestfish / guest customize. virt-v2v holds multi‑GiB for a larger fraction of the pod lifetime.

---

## CPU (conversion pod, metrics-server millicores)

Peak CPU samples during conversion:

| VM | virt-v2v peak | kc-v2v peak | Delta |
|---|---:|---:|---:|
| RHEL | 2851 m | **1029 m** | **−1822 m** |
| Windows | 1343 m | **493 m** | **−850 m** |

kc-v2v peaks later (guest convert / finalize); virt-v2v shows higher spikes, especially on RHEL.

---

## Summary

| Dimension | Winner / note |
|---|---|
| Portability | **kc-v2v** — pure Go, multi-arch |
| No proprietary VDDK | **kc-v2v** |
| RHEL wall clock | **kc-v2v** (~3 min faster) |
| Windows wall clock | **kc-v2v** (modestly faster) |
| Peak memory | **kc-v2v** (lower, especially Windows) |
| Peak CPU | **kc-v2v** (lower) |

Live charts: [docs/ref-baseline/dashboard.html](../../docs/ref-baseline/dashboard.html).
