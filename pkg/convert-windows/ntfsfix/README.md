# ntfsfix -- NTFS boot sector heads fix

Patches the "number of heads" field in the NTFS boot sector of pre-Vista Windows guests (XP, Server 2003). When these older guests are moved between hypervisors the BIOS-reported disk geometry can change, and a mismatched heads value in the boot sector prevents Windows from booting.

The fix reads the first disk's total size and selects a heads value from a lookup table matching virt-v2v's `fix_ntfs_heads` thresholds. It then scans the first disk's partitions for NTFS filesystems, verifies the 8-byte "NTFS    " magic at offset 3, and overwrites the 16-bit heads field at byte offset 0x1A with the computed value. Only the first suitable NTFS partition is patched. The operation is skipped entirely when the version handler's `NeedsNTFSHeadsFix` returns false.

## Key exports

| Symbol | Role |
|--------|------|
| `Fix` | Patches the NTFS boot sector heads field on the first disk's first NTFS partition if `needsFix` is true |
