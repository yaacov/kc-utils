# metadata -- TargetMeta JSON assembly

Builds customizer options from pipeline data and writes the final pipeline metadata JSON to disk. This metadata file is consumed by downstream components such as the Forklift migration controller.

`CustomizerOpts` extracts OS type, hostname, timezone, scripts directory, and SELinux state from `PipelineData` into a flat `map[string]string` suitable for the `Customizer.Apply` interface. `WriteTargetMeta` merges converter warnings and block-level errors into the `TargetMeta.Warnings` slice (surfacing partial failures like a failed initramfs rebuild), then serializes the full `PipelineData` as indented JSON.

## Key exports

| Symbol | Role |
|--------|------|
| `CustomizerOpts` | Builds a `map[string]string` of customizer options from `*types.PipelineData` |
| `WriteTargetMeta` | Merges warnings/errors and writes `PipelineData` JSON to the given file path |
