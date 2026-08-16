# Clevis / NBDE Unlock

kc-utils can unlock LUKS volumes bound to a Clevis pin (NBDE — Network-Bound
Disk Encryption) so encrypted guests convert without a manual passphrase. How
the unlock happens depends on the active [backend](README.md).

## Trigger (Forklift)

Forklift sets `V2V_NBDE_CLEVIS=true` on the conversion pod when Plan
`nbdeClevis` or Conversion `diskEncryption.type=Clevis` is configured. LUKS
passphrase secrets are mounted at `/etc/luks` (no Clevis env). **Clevis takes
precedence over keyfiles when both are present.** After unlock, prepare rescans
LVM and probes `/dev/mapper/*` devices as root candidates. Keyfile unlock uses
`cryptsetup-open` with `--keys-from-stdin`.

## Per backend

| Backend | Clevis unlock |
|---------|---------------|
| [direct](direct.md) | `clevis luks unlock` on the host (needs `CAP_SYS_ADMIN`) |
| [guestfs](guestfs.md) | guestfish `clevis-luks-unlock` inside the appliance (appliance-root, not host `CAP_SYS_ADMIN`) |
| [qemu](qemu.md) | `unlock_clevis` op in [`kc-agent`](kc-agent.md) inside the appliance |

## Appliance networking

Both appliance backends must reach the Tang server(s) from the Clevis pin over
the network, so appliance networking is enabled before the VM runs:

1. Appliance networking enabled before `run` (guestfs: `set-network true`),
   gated on Clevis via internal `KC_GUESTFS_NETWORK=1` (shared by the qemu
   backend, which adds a QEMU user-net device).
2. Tang servers from the Clevis pin tree must be reachable from the conversion
   pod network (QEMU user networking).
3. guestfs additionally requires the `clevisluks` libguestfs feature (clevis
   packages available to supermin); the qemu appliance ships `clevis` /
   `clevis-luks` in its initramfs (see [appliance.md](appliance.md)).

## See also

- [../apps/kc-v2v.md](../apps/kc-v2v.md) — `V2V_NBDE_CLEVIS` orchestrator wiring
- [../apps/forklift-usage.md](../apps/forklift-usage.md) — Forklift Plan fields
