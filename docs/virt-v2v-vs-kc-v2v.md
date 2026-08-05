# Forklift conversion: virt-v2v vs kc-v2v

Cold migration comparison on OpenShift MTV, same VMs and storage class, sequential runs (RHEL then Windows).

| Converter | Image |
|---|---|---|
| **virt-v2v** (default Forklift) | `quay.io/yaacov/forklift-virt-v2v@sha256:6b6e471c…` |
| **kc-v2v** | `quay.io/yaacov/kc-v2v@sha256:083008ae…` |

VMs: `mtv-func-cold-rhel9-staticips`, `mtv-func-win2019`  
Default SC: `ocs-storagecluster-ceph-rbd-virtualization`  
`feature_windows_wait_for_reboot=false` (fair conversion comparison)  

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
| Plan wall | 11m 34s | 11m 43s | +9s |
| ImageConversion | 3m 29s | 10m 54s | +7m 25s |
| DiskTransferV2v | 7m 5s | 3s | −7m 2s |
| ImageConversion + DiskTransfer | 10m 34s | 10m 57s | +23s |

### Windows Server 2019

| Metric | virt-v2v | kc-v2v | Delta (kc − ref) |
|---|---:|---:|---:|
| Plan wall | 18m 10s | 13m 15s | **−4m 55s** |
| ImageConversion | 6m 22s | 12m 14s | +5m 52s |
| DiskTransferV2v | 11m 8s | 4s | −11m 4s |
| ImageConversion + DiskTransfer | 17m 30s | 12m 18s | **−5m 12s** |

**Takeaway:** End-to-end RHEL is roughly parity. Windows is clearly faster with kc-v2v on this cluster. Pipeline shape differs: virt-v2v spends most time in `DiskTransferV2v` (VDDK); kc-v2v folds NFC copy + convert into `ImageConversion`.

---

## Memory (conversion pod cgroup RSS)

Peak cgroup RSS during the conversion pod lifetime:

| VM | virt-v2v peak | kc-v2v peak | Delta |
|---|---:|---:|---:|
| RHEL | 2657 Mi | **1899 Mi** | **−758 Mi** (−29%) |
| Windows | 2076 Mi | **2024 Mi** | −52 Mi (−3%) |

kc-v2v stays low (~30–60 Mi) through most of NFC copy, then rises for guestfish / guest customize. virt-v2v holds multi‑GiB for a large fraction of the pod lifetime.

---

## CPU (conversion pod, metrics-server millicores)

Peak CPU samples during conversion:

| VM | virt-v2v peak | kc-v2v peak | Delta |
|---|---:|---:|---:|
| RHEL | 3552 m | **1061 m** | **−2491 m** |
| Windows | 1488 m | **1143 m** | −345 m |

kc-v2v peaks later (guest convert / finalize); virt-v2v shows higher spikes, especially on RHEL.

---

## Summary

| Dimension | Winner / note |
|---|---|
| Portability | **kc-v2v** — pure Go, multi-arch |
| No proprietary VDDK | **kc-v2v** |
| RHEL wall clock | Roughly **tie** (~12 min) |
| Windows wall clock | **kc-v2v** (~5 min faster) |
| Peak memory | **kc-v2v** (lower, especially RHEL) |
| Peak CPU | **kc-v2v** (lower) |

