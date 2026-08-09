# kc-convert-windows — orchestrator

[`pipeline.go`](pipeline.go) is a thin orchestrator for kc-convert-windows. It
reads `PrepareOutput` JSON from kc-prepare, re-attaches to the already-mounted
guest filesystem, and runs 15+ semantic blocks in order to convert a Windows
guest from its source hypervisor to KVM with VirtIO drivers.

**What it does:** Classifies the Windows version, locates VirtIO driver files
on the conversion host, detects antivirus and RTC mode, removes source
hypervisor software and services, copies VirtIO drivers into the guest,
registers them in the Windows registry for boot-time loading, disables crash
auto-reboot, generates PowerShell/batch firstboot scripts (driver install,
static IP, guest agent, cleanup), patches the NTFS boot sector for legacy
Windows, updates UEFI BCD entries, and builds the GuestCaps output.

**How it works:** The pipeline operates on the Windows registry hives (SYSTEM
and SOFTWARE) checked out to the host via `guest.Checkout` and opened with the
`hivex` registry editor. Pluggable blocks (driversource, hypervisor remove,
driver registrar, hypervisor service disable) iterate registered plugins.
Strict blocks (version classify, inspect, crashcontrol, driver copy, firstboot
generation, ntfsfix, UEFI BCD, output) run fixed logic. Changes are batched
into the registry hives and committed back to the guest filesystem at the end
of the pipeline.

**Pluggable:** `pkg/convert-windows/<block>/plugins/` (driversource, hypervisor, drivers).

**Strict:** `pkg/convert-windows/<block>/` (inspect, crashcontrol, firstboot, ntfsfix,
output).

Writes `ConverterOutput` JSON on success.

Full block table: [`docs/kc-convert-windows.md`](../../../docs/kc-convert-windows.md).
