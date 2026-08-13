# network — guest network configuration blocks

Pipeline blocks that configure guest networking during offline conversion.

## Handler selection

After hypervisor cleanup (block 11), the pipeline calls [`network.Select()`](network.go) once. Exactly **one** registered [`Handler`](network.go) runs for blocks 11b and 15:

1. Collect non-default handlers whose `Detect()` reports the guest's **active** network stack (systemd unit state, not stale config files alone).
2. If multiple match, pick the highest `Priority()`.
3. If none match, use the `default` handler.

**Never** run all handlers whose `Detect()` is true. Leftover NetworkManager profiles on a systemd-networkd-primary guest must not trigger the default path.

| Handler | Priority | Selected when |
|---------|----------|---------------|
| [`networkd`](handlers/networkd/) | 100 | Tier-2 shortcut (`80-ec2.network`, AL2023 os-release) or networkd enabled with NM masked/absent; not when both networkd and NM are enabled |
| [`default`](handlers/default/) | 0 | Fallback — NM/legacy guests |

NIC namer plugins ([`nicnaming/`](../nicnaming/)) run only inside the `default` handler.

## Packages

| Package | Role |
|---------|------|
| [`handlers/networkd/`](handlers/networkd/) | Handler wrapping [`networkd/`](networkd/) — virtio DHCP, wait-online drop-in, MAC static `.network` files |
| [`handlers/default/`](handlers/default/) | Handler wrapping [`nicnaming/`](../nicnaming/) + [`staticip/`](staticip/) firstboot path |
| [`networkd/`](networkd/) | systemd-networkd offline helpers (used by the networkd handler) |
| [`staticip/`](staticip/) | macToIP mapping + firstboot `nmcli`/`ip` commands (used by the default handler) |

EC2 net-hook masking (`DisableEC2NetHooks`) lives in [`systemd/`](../systemd/) and runs from the EC2 hypervisor cleanup plugin.
