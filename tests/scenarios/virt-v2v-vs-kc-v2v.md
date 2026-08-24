# Forklift conversion: virt-v2v vs kc-v2v

Cold migration comparison on OpenShift MTV, same source VMs and storage class.
The runner (`test-mtv-benchmark.sh`) migrates three VMs in one plan and samples
the conversion pods in parallel.

| Converter | Image | Run id |
|---|---|---|
| **virt-v2v** (default Forklift) | operator default | `20260824T123342Z` (`runs/ref-20260824T123342Z*`) |
| **kc-v2v** | kc-v2v image under test | `20260824T123342Z` (`runs/kc-20260824T123342Z*`) |

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

See [`docs/architecture/ref-baseline/README.md`](../../docs/architecture/ref-baseline/README.md) for per-VM wall times and pipeline block tables from the latest archived run (`20260824T123342Z`).

**Takeaway:** kc-v2v is faster end-to-end on the three-VM plan (about 16% shorter plan lifetime). Pipeline shape differs: virt-v2v spends most time in `DiskTransferV2v`; kc-v2v folds NFC copy + convert into `ImageConversion`.

---

## Memory (conversion pod cgroup RSS)

Peak cgroup RSS during the conversion pod lifetime — see the
[baseline peak table](../../docs/architecture/ref-baseline/README.md#peak-resource-usage-conversion-pod-cgroup-rss--cpu)
and [dashboard](../../docs/architecture/ref-baseline/dashboard.html).

kc-v2v stays lower on the RHEL guests; Windows peak memory can be similar or
slightly higher depending on guest and timing. virt-v2v tends to hold
multi‑GiB RSS for a larger fraction of the pod lifetime.

---

## CPU (conversion pod, metrics-server millicores)

Peak CPU samples during conversion — see the baseline table and dashboard.
kc-v2v peak CPU is substantially lower on RHEL; virt-v2v shows higher spikes,
especially on the larger RHEL 9 guest.

---

## Network

kc-v2v pulls less cumulative RX on all three guests in the latest run (about
35–57% less) because conversion happens in-pod instead of streaming raw disk
data through `DiskTransferV2v`.

---

## Summary

| Dimension | Winner / note |
|---|---|
| Portability | **kc-v2v** — pure Go, multi-arch |
| Plan wall clock (3-VM plan) | **kc-v2v** (shorter) |
| Peak memory | **kc-v2v** on RHEL guests; mixed on Windows |
| Peak CPU | **kc-v2v** (lower on RHEL) |
| Network RX | **kc-v2v** (less on all three VMs) |

Live charts: [docs/architecture/ref-baseline/dashboard.html](../../docs/architecture/ref-baseline/dashboard.html).
