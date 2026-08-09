# JSON and runnable examples

Copy-paste samples for each binary handoff. Paths use `/var/lib/kc/` as a
work directory; adjust for your environment.

## Prerequisites

- Linux host with root (for `kc-prepare` and `kc-finalize`)
- Built binaries: `make build` → `bin/kc-prepare`, etc.
- Runtime tools: see [External dependencies](../../../community/CONTRIBUTING.md#external-dependencies)
- For the runnable script: `jq`, loop device support, and optionally
  `libguestfs-tools` to create a test disk image

## Example files

| File | Used by | Description |
|------|---------|-------------|
| [prepare-input-linux.json](prepare-input-linux.json) | kc-prepare | Single-disk Linux guest |
| [prepare-input-windows.json](prepare-input-windows.json) | kc-prepare | Windows guest + static IPs |
| [prepare-input-multiboot.json](prepare-input-multiboot.json) | kc-prepare | Multiboot with `options.root=first` (also the default when omitted) |
| [prepare-input-luks.json](prepare-input-luks.json) | kc-prepare | LUKS keyfile mapping |
| [prepare-output-complete.json](prepare-output-complete.json) | converters, kc-finalize | Example `prepare` section of PipelineData |
| [prepare-output-error-multiboot.json](prepare-output-error-multiboot.json) | orchestrator | Explicit `options.root=single` multiboot failure + candidates |
| [convert-output-linux.json](convert-output-linux.json) | kc-finalize | Example `convert` section (Linux) of PipelineData |
| [convert-output-windows.json](convert-output-windows.json) | kc-finalize | Example `convert` section (Windows) of PipelineData |
| [target-meta.json](target-meta.json) | orchestrator | Example `target` section of PipelineData |

## Quick run (Linux test disk)

From the repository root, as root:

```bash
sudo docs/apps/examples/run-linux-disk.sh
```

This creates a phony RHEL disk image (via `tests/make-disk-linux.sh`), runs
prepare → convert → finalize, and prints paths to the JSON outputs.

Equivalent manual steps:

```bash
sudo mkdir -p /var/lib/kc /mnt/kc-guest
sudo cp docs/apps/examples/prepare-input-linux.json /var/lib/kc/prepare-input.json
# Edit disk path in prepare-input.json to your image path

sudo bin/kc-prepare \
  --input /var/lib/kc/prepare-input.json \
  --output /var/lib/kc/pipeline.json \
  --mount-root /mnt/kc-guest

CONVERTER=$(jq -r .converter /var/lib/kc/pipeline.json)
sudo bin/"$CONVERTER" \
  --input /var/lib/kc/pipeline.json \
  --output /var/lib/kc/pipeline.json \
  --mount-root /mnt/kc-guest \
  --offline

sudo bin/kc-finalize \
  --input /var/lib/kc/pipeline.json \
  --output /var/lib/kc/pipeline.json \
  --mount-root /mnt/kc-guest
```

## Multiboot

By default (or with `"options": {"root": "first"}`), prepare picks the first
discovered OS root. To fail when multiple roots exist, set `"root": "single"`;
the error output lists `root_candidates` so you can choose a device path:

```bash
jq '.error, .root_candidates[] | {device_path, product_name}' \
  /var/lib/kc/prepare-output.json
```

Then set `"root": "/dev/loop0p2"` (or `"first"`) and re-run `kc-prepare`. See
[prepare-output-error-multiboot.json](prepare-output-error-multiboot.json).

## Windows virtio-win drivers

`kc-convert-windows` does not take driver paths on the CLI or in JSON. Drivers
must already be on the **conversion host** at paths the built-in `DriverSource`
plugins scan.

### Driver tree

The kc-v2v container image downloads virtio-win ISOs from the public
[Fedora People archive](https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/archive-virtio/)
at build time and ships drivers under `/usr/share/virtio-win/drivers/by-os/`.

| Host path | Plugin | Role |
|-----------|--------|------|
| `/usr/share/virtio-win/drivers/by-os` | `directory` | Match guest arch and Windows version |

For local development without the container, extract a virtio-win ISO into that
path or install the `virtio-win` package on Fedora/RHEL (`dnf install -y virtio-win`).

### Linux guest packages (different feature)

Offline `qemu-guest-agent` for **RHEL-family** guests uses host RPMs (not VirtIO-Win):

```text
/usr/share/kc-packages/rpm/el8/x86_64/qemu-guest-agent-*.rpm
/usr/share/kc-packages/rpm/el9/x86_64/qemu-guest-agent-*.rpm
/usr/share/kc-packages/rpm/el10/x86_64/qemu-guest-agent-*.rpm
```

The kc-v2v image stages these via [`build/kc-v2v/stage-linux-packages.sh`](../../../build/kc-v2v/stage-linux-packages.sh).
For a local convert run without the image, populate that tree (or bind-mount it) and pass
`kc-convert-linux --offline`. See [kc-convert-linux.md](../kc-convert-linux.md).

More detail: [pkg/convert-windows/driversource/plugins/README.md](../../../pkg/convert-windows/driversource/plugins/README.md).

## Reference tests

End-to-end shell tests under `tests/` mirror these flows:

- `tests/kc-v2v.sh` — full chain orchestrator
- `tests/test-disk-linux-chain.sh` — Linux disk through all three stages
- `tests/test-disk-linux-prepare.sh` — prepare + finalize only
