# xml -- Forklift inspection XML writer

Generates a minimal virt-v2v-compatible inspection XML file from pipeline target metadata. The Forklift migration controller reads this XML to determine the guest OS identity (name, distro, osinfo ID, architecture) after conversion completes.

`WriteInspectionXML` takes a `TargetMeta` and an output path, constructs a `<v2v><operatingsystem>` XML document, and writes it to disk. If the `OsinfoID` field is empty, the `inferOsinfo` function derives it from the distro name and major version (e.g. `"rhel9"`, `"win2k22"`). Windows detection uses the product name to map to specific osinfo IDs. The architecture defaults to `x86_64` when not set in metadata. The output file's parent directory is created automatically.

## Key exports

| Symbol | Role |
|--------|------|
| `WriteInspectionXML` | Writes a virt-v2v inspection XML file from `*types.TargetMeta` to the given path |
