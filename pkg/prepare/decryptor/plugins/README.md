# decryptor plugins

`Decryptor` interface — open encrypted partitions during prepare.

| Key | Package | Description |
|-----|---------|-------------|
| `keyfile` | keyfile/ | Host key-file LUKS unlock after `guest.Open`/LVM; device key `all` / `all-*` tries every partition/LV |
| `clevis` | clevis/ | Clevis/Tang NBDE via LUKS PIN metadata (also after LVM) |
