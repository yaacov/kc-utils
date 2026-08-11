# kc-convert-linux — orchestrator

[`pipeline.go`](pipeline.go) is a thin orchestrator for kc-convert-linux. It
reads `PrepareOutput` JSON from kc-prepare, re-attaches to the already-mounted
guest filesystem, and runs 17 semantic blocks in order to convert a Linux guest
from its source hypervisor to KVM with virtio drivers.

**What it does:** Classifies the distribution, detects the bootloader, scans
and selects kernels, remaps device names, fixes UEFI boot entries, removes
source hypervisor tools, configures systemd-networkd virtio DHCP when
applicable, installs the QEMU guest agent, cleans stale guest state,
injects virtio modules into the initramfs, preserves NIC naming or writes
offline `.network` static IPs, runs an offline SELinux relabel, and builds
the GuestCaps output.

**How it works:** Each block is either pluggable (dispatched via a
`plugin.Registry` — distro, bootloader, remap, kernel scan, UEFI, hypervisor,
guestagent, nicnaming) or strict (fixed logic — bootconfig, guestcleanup,
initramfs, kernel select, guestcaps, SELinux, `network/networkd`). Pluggable
blocks iterate all registered plugins: `Detect`/`Matches` selects which run,
errors are recorded as warnings but do not abort the pipeline (except initramfs
failure, which is fatal since the VM will not boot without virtio drivers).

After hypervisor cleanup (block 11), the orchestrator calls `networkd.Detect`
once and passes the result to block 11b (KubeVirt virtio DHCP) and block 15
(static IP / NIC naming branch).

**Pluggable:** `pkg/convert-linux/<block>/plugins/` (distro, bootloader, remap, kernel,
uefi, hypervisor, guestagent, nicnaming).

**Strict:** `pkg/convert-linux/<block>/` (bootconfig, guestcleanup, initramfs, guestcaps, network/networkd).

**Stage helpers:** [`pkg/convert-linux/systemd/`](../convert-linux/systemd/) — systemd unit mask/disable (hypervisor plugins).

Writes `ConverterOutput` JSON on success.

Full block table: [`docs/apps/kc-convert-linux.md`](../../../docs/apps/kc-convert-linux.md).
