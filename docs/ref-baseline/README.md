# MTV Cold Migration — ref vs kc-v2v Baseline

Sequential cold-migration benchmark (RHEL then Windows) comparing the default
virt-v2v image (**ref**) against the kc-v2v replacement image (**kc-v2v**).

Each run migrates two VMware VMs to OpenShift Virtualization while sampling
CPU, memory, and network traffic of the conversion pod every ~10 s.

## Quick start

```bash
# kc-v2v run (set image override first)
oc mtv settings set --setting virt_v2v_image_fqin --value "quay.io/yaacov/kc-v2v:devel-amd64"
NS=mtv-kc-v2v-ref KEEP_BETWEEN_TESTS=true bash test-mtv-ref-baseline.sh

# ref run (unset image override to use the operator default)
oc mtv settings set --setting virt_v2v_image_fqin --value ""
NS=mtv-ref-baseline KEEP_BETWEEN_TESTS=true bash test-mtv-ref-baseline.sh
```

### Prerequisites

`oc`, `oc mtv`, `jq`, and `GOVC_URL` / `GOVC_USERNAME` / `GOVC_PASSWORD`
environment variables pointing to a vSphere with `mtv-func*` VMs.

### Environment variables

| Variable | Default | Description |
|---|---|---|
| `NS` | `mtv-ref-baseline` | Namespace to create for the test |
| `PROVIDER` | `vsphere-test` | vSphere provider name |
| `RHEL_VM` / `WIN_VM` | (auto-picked) | Source VM names; auto-discovered from `mtv-func*` inventory |
| `SKIP_CLEANUP` | `true` | Keep the namespace after the test |
| `KEEP_BETWEEN_TESTS` | `false` | Leave RHEL plan/pods while running Windows |
| `DISABLE_WAIT_FOR_REBOOT` | `true` | Set `feature_windows_wait_for_reboot=false` |
| `MEM_INTERVAL` | `10` | Seconds between metric samples |
| `INTERVAL` | `10` | Seconds between plan status polls |
| `MAX_ATTEMPTS` | `180` | Max poll attempts (180 × 10 s = 30 min timeout) |

### Output files

Each run produces:

- `test-mtv-ref-baseline-<timestamp>.log` — full run log
- `test-mtv-ref-baseline-<timestamp>-mem/` — per-VM CSVs with columns:
  `timestamp_utc, elapsed_s, pod, node, mem_working_set_mi, mem_rss_mi_cgroup,
  cpu_m, net_rx_bytes, net_tx_bytes, phase`

---

## Archived runs

| Run | Image | Timestamp | Directory |
|---|---|---|---|
| **ref** | operator default (virt-v2v) | 2026-08-05T21:15:38Z | `runs/ref-20260805T211538Z*` |
| **kc-v2v** | `quay.io/yaacov/kc-v2v:devel-amd64` | 2026-08-05T20:42:33Z | `runs/kc-20260805T204233Z*` |

Cluster: qemtvd-07 · cold migration · VDDK 8.0.0 · Ceph HEALTH_OK

---

## Results

### Summary

| VM | Image | Status | Wall time | Peak mem (cgroup) | Peak CPU | Net RX | Net TX |
|---|---|---|---|---|---|---|---|
| RHEL | ref | Succeeded | 15m 21s | 1894 Mi | 2851 m | 6343 Mi | 34 Mi |
| RHEL | kc-v2v | Succeeded | 12m 31s | 1804 Mi | 1029 m | 2795 Mi | 6 Mi |
| Windows | ref | Succeeded | 14m 6s | 1526 Mi | 1343 m | 11719 Mi | 49 Mi |
| Windows | kc-v2v | Succeeded | 13m 32s | 962 Mi | 493 m | 4257 Mi | 9 Mi |

### Pipeline block timings

#### RHEL

| Block | ref | kc-v2v | Delta |
|---|---|---|---|
| ImageConversion | 4m 43s | 11m 59s | +7m 16s |
| DiskTransferV2v | 9m 56s | 4s | −9m 52s |
| **Combined** | **14m 39s** | **12m 3s** | **−2m 36s** |
| Plan wall (all steps) | 15m 21s | 12m 31s | −2m 50s |

#### Windows

| Block | ref | kc-v2v | Delta |
|---|---|---|---|
| ImageConversion | 3m 24s | 12m 47s | +9m 23s |
| DiskTransferV2v | 9m 58s | 3s | −9m 55s |
| **Combined** | **13m 22s** | **12m 50s** | **−32s** |
| Plan wall (all steps) | 14m 6s | 13m 32s | −34s |

### Peak resource comparison

| VM | ref peak mem | kc peak mem | ref peak CPU | kc peak CPU |
|---|---|---|---|---|
| RHEL | 1894 Mi | 1804 Mi | 2851 m | 1029 m |
| Windows | 1526 Mi | 962 Mi | 1343 m | 493 m |

### Network totals

| VM | ref RX | kc RX | ref TX | kc TX |
|---|---|---|---|---|
| RHEL | 6343 Mi | 2795 Mi | 34 Mi | 6 Mi |
| Windows | 11719 Mi | 4257 Mi | 49 Mi | 9 Mi |

kc-v2v transfers **56 % less data for RHEL** and **64 % less for Windows**
because it performs disk conversion locally inside the pod (longer
ImageConversion) instead of streaming raw disk data to a separate
DiskTransferV2v step.

---

## Dashboard

Open `dashboard.html` in a browser for interactive charts comparing memory,
CPU, and network I/O over time.
