# network — guest network configuration blocks

Pipeline blocks that configure guest networking during offline conversion.

| Package | Pipeline blocks | Role |
|---------|-----------------|------|
| [`networkd/`](networkd/) | 11b, 15 (when `Detect`) | systemd-networkd virtio DHCP, wait-online drop-in, MAC static `.network` files |
| [`staticip/`](staticip/) | 15 (when not `Detect`) | macToIP mapping file + firstboot `nmcli`/`ip` commands for NetworkManager/legacy guests |

Guests that do not match `networkd.Detect` use [`nicnaming/`](../nicnaming/) to preserve interface names and [`staticip/`](staticip/) for firstboot static IP configuration (block 15).

EC2 net-hook masking (`DisableEC2NetHooks`) lives in [`systemd/`](../systemd/) and runs from the EC2 hypervisor cleanup plugin.
