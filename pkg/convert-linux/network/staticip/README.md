# staticip -- firstboot static IP configuration for NetworkManager/legacy guests

Offline conversion helper for guests on the [`default` network handler](../handlers/default/) (NetworkManager/legacy). Since interface names and network tooling (`nmcli`/`ip`) can't be relied on offline, static IP assignment is deferred to first boot: a MAC-to-IP mapping file is written into the guest, and firstboot shell commands read it back to configure each interface once the guest is running.

`WriteMacToIP` writes one line per static IP entry (formatted by `MacToIPLine`) to `/tmp/macToIP` inside the guest. `FirstbootCommands` returns shell commands that, at first boot, read that file, resolve each MAC to its live interface name, and configure the address via `nmcli` (preferred) or a plain `ip addr`/`ip route` fallback when NetworkManager isn't present.

## Key exports

| Symbol | Role |
|--------|------|
| `MacToIPLine` | Formats a single static IP entry (`mac:ip:ip,gateway,prefix,dns`) for the mapping file |
| `WriteMacToIP` | Writes the macToIP mapping file into the guest filesystem |
| `FirstbootCommands` | Returns shell commands that configure static IPs from the mapping file on first boot |

## Usage

Pipeline block 15 ([`pkg/cmd/convert-linux/pipeline.go`](../../../cmd/convert-linux/pipeline.go)) calls `network.Select()`; the `default` handler calls `WriteMacToIP` and hands `FirstbootCommands()` to the `systemd` firstboot handler. The `networkd` handler uses [`networkd.WriteStaticNetworks`](../networkd/) instead, writing offline `.network` files directly with no firstboot step required.
