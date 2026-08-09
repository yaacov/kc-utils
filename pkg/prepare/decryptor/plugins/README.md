# decryptor plugins

`Decryptor` interface — open encrypted partitions during prepare.

Some guest disks use LUKS full-disk encryption, which must be unlocked before
the prepare pipeline can mount and inspect the filesystems inside. Each
decryptor plugin implements a different unlock method. The prepare pipeline
tries each registered decryptor against encrypted devices found during
partition scanning. Once unlocked, the decrypted dm-crypt mapper device is
used transparently by all subsequent blocks (mount, inspect, convert,
finalize).

| Key | Package | Description |
|-----|---------|-------------|
| `keyfile` | keyfile/ | Host key-file LUKS unlock after `guest.Open`/LVM; device key `all` / `all-*` tries every partition/LV |
| `clevis` | clevis/ | Clevis/Tang NBDE via LUKS PIN metadata (also after LVM) |

## keyfile

**What it does:** Unlocks LUKS-encrypted block devices using a key file
provided on the conversion host (e.g. mounted into the pod at a known path).

**How it works:** Delegates to the active `guest.Guest` handle's `Decrypt`
method, passing the device path, key file path, and a sanitized mapper name
(prefixed with `v2v-luks-keyfile-`). When the device key is `all` or starts
with `all-`, the prepare pipeline iterates every discovered partition and
logical volume, attempting to unlock each one.

## clevis

**What it does:** Unlocks LUKS-encrypted block devices using Clevis
network-bound disk encryption (NBDE), typically backed by a Tang server or
TPM2 policy stored in the LUKS header metadata.

**How it works:** Delegates to the active `guest.Guest` handle's
`UnlockClevis` method, which reads the Clevis PIN metadata from the LUKS
header and uses it to derive the decryption key without requiring a separate
key file. The mapper name includes a SHA-256 suffix of the device path to
ensure uniqueness when multiple devices share the same basename.
