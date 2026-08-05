# firmware plugins

`FirmwareDetector` interface — confidence-based winner selects BIOS vs UEFI.

| Key | Package | Description |
|-----|---------|-------------|
| `gpt-esp` | gptesp/ | Heuristic ESP detection (`vfat` + mount point / size / path); returns UEFI when an ESP-like partition is found |
