# qemu backend

A guest-disk backend that boots **our own** minimal appliance directly with
`qemu-system-*` (no libvirt, no libguestfs), attaches the guest disks as
virtio-blk drives, and drives a tiny in-guest agent over a unix socket.

The appliance exposes only **primitive** operations (exec, file I/O, raw device
I/O, stat/statfs). **All conversion logic stays host-side** in this package:
partition discovery, LVM activation, LUKS/Clevis unlock, mount planning,
fs-checks, and chrooted guest commands are composed on the host out of those
primitives, then executed inside the appliance via the agent.

## Why it exists

- **We own it.** Unlike `guestfs`, there is no dependency on the libguestfs /
  supermin stack or its appliance — just a `qemu-system-*` binary and an image
  we build ([`build/kc-appliance`](../../../../build/kc-appliance)).
- **It runs on macOS.** The host code only launches qemu and speaks a socket, so
  this is a first-class backend on Apple Silicon (HVF), not just Linux (KVM).

## Transport

```text
host unix socket  <──virtio-serial──>  /dev/virtio-ports/org.kc-utils.agent
   (QEMU -chardev socket,server=on)            (agent protocol)

host debug.sock   <──virtio-serial──>  /dev/virtio-ports/org.kc-utils.debug
   (same private dir as agent.sock)            (interactive bash PTY)
```

Wire protocol (agent port): length-prefixed JSON frames, `pkg/qemuagent/proto`.
One request → one response, serialized by a mutex in `client.go`.

The debug port is a raw byte channel, not JSON. Attach while the appliance is
up with `socat` / `nc`; see
[Interactive debug shell](../../../../docs/architecture/qemu-appliance.md#interactive-debug-shell).

## Acceleration

`accelFor(GOOS, hasAccel)` picks **KVM** on Linux (`/dev/kvm`), **HVF** on macOS
(Hypervisor.framework), else **TCG** emulation. With acceleration `-cpu host`;
under TCG a concrete CPU model (`cortex-a72` / `max`). An arm64 appliance boots
near-native on an arm64 Mac; an x86_64 appliance runs under TCG there.

`RunCommand` chroots into the guest and runs the guest's *own* binaries, so
converting an x86_64 guest needs an x86_64 appliance (TCG on Apple Silicon).

## Cross-stage VM sharing

Like `guestfs`, a multi-stage pipeline shares one appliance (mounts live inside
the VM). `kc-v2v` boots it via `StartSharedListener(disks)` and exports
`KC_QEMU_SOCK` / `KC_QEMU_PID` / `KC_QEMU_DEBUG_SOCK`; each stage subprocess
adopts it (`adoptVMSession`) instead of booting its own. A standalone
single-stage run boots its own VM and remounts from the recorded disk infos
(`remountFromDiskInfos`).

## Files

| File | Responsibility |
|------|----------------|
| `register.go` | plugin factory + `SharedListenerPlugin`; `Requirements{QEMU}` (TCG fallback ⇒ no `Accel`) |
| `launch.go`   | pure launch helpers: arch→binary/machine/cpu/accel, `appliancePaths`, `qemuArgs` |
| `session.go`  | `vmSession`: qemu process + agent client; boot/adopt/restart/kill; env sharing |
| `client.go`   | `agentClient`: one method per primitive over the socket |
| `backend.go`  | `Backend`, `Setup`, disk→`/dev/vd*` mapping, `NewMounted` re-attach |
| `discover.go` | partition (`lsblk`) + LVM (`pvscan`/`vgchange`/`lvs`) discovery; mount ordering |
| `mount.go`    | eager `Mount`/`UnmountAll`/`FSTrim`, host↔appliance path rebasing |
| `fs.go`       | guest file ops, `Upload`/`Download`, `Glob` |
| `device.go`   | `DeviceRead`/`DeviceWrite` via `PRead`/`PWrite` |
| `fscheck.go`  | `FSType`/`BlkidAttr`/`FSCheck` (fsck command mapping) |
| `crypt.go`    | `Decrypt`, `UnlockClevis`, `CloseCrypt`, `RescanBlock` |
| `run.go`      | `RunCommand`: bind `/proc /sys /dev`, chroot, exec |
| `probe.go`    | `ProbeMount`/`ProbeUnmount`: RO-mount + copy OS markers to the host |
| `teardown.go` | `UnmountFilesystems`, `ReleaseDevices`, `Teardown*` |

Pure helpers (`qemuArgs`, `accelFor`, `parseLsblkPartitions`, `parseLVPaths`,
`orderMountPlans`, `fsckArgv`, `detectImageFormat`, …) are unit-tested in
`*_test.go` without launching a VM.

## Environment

| Var | Purpose |
|-----|---------|
| `KC_APPLIANCE_DIR`  | appliance image dir (default `/usr/lib/kc-utils/appliance`) |
| `KC_APPLIANCE_ARCH` | override appliance arch (default host `GOARCH`) |
| `KC_QEMU_BINARY`    | override the `qemu-system-*` binary |
| `KC_QEMU_SOCK` / `KC_QEMU_PID` | set by the parent to share a booted VM across stages |
| `KC_QEMU_DEBUG_SOCK` | debug-channel unix socket (`debug.sock` next to the agent socket) |
| `V2V_memSize` / `V2V_smp` | appliance RAM (MiB) / vCPUs |

See [docs/architecture/qemu-appliance.md](../../../../docs/architecture/qemu-appliance.md)
for the full protocol and host/guest logic split.
