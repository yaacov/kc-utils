# nicnaming -- NIC naming via udev and systemd rules

Preserves network interface names across conversion by discovering MAC-to-interface-name bindings from the guest's network configuration and writing persistent naming rules. This ensures that static IP assignments and firewall rules that reference specific interface names continue to work after the VM moves to KubeVirt.

`Apply` takes the list of static IPs, queries all registered `NICNamer` plugins to resolve MAC-to-device-name bindings from the guest's network config backend (e.g. NetworkManager, ifcfg files), deduplicates the results (rejecting conflicting mappings for the same MAC), then writes both a udev `70-persistent-net.rules` file and systemd `.link` files under `/etc/systemd/network/`. The udev rules use `ATTR{address}` matching, while the `.link` files provide the same mapping for RHEL 9+ systems where predictable naming overrides raw udev `NAME=` rules.

**When not used:** Guests matching [`networkd.Detect`](../network/networkd/) skip `Apply` in pipeline block 15; static IPs are written as offline systemd `.network` files instead. Guests that use `Apply` also use [`staticip/`](../network/staticip/) in the same block to write a firstboot `nmcli`/`ip` mapping file.

## Key exports

| Symbol | Role |
|--------|------|
| `NICNamer` | Interface for resolving MAC-to-interface-name bindings from a guest network config backend (Detect, ResolveNames) |
| `Namers` | Global plugin registry of `NICNamer` implementations |
| `NamingRule` | Struct holding a MAC address and its target interface device name |
| `MacIPEntry` | Struct mapping a MAC address to an IP for lookup by NICNamer plugins |
| `Apply` | Discovers NIC naming rules and writes udev rules and systemd `.link` files |
