# nicnaming plugins

`NICNamer` interface — preserve NIC naming for static IP configurations across network backends.

When a VM moves between hypervisors, the virtual NIC hardware changes (e.g.
from vmxnet3 to virtio-net), which causes the guest OS to assign new interface
names. If the guest had static IP addresses bound to the old interface names,
networking breaks after conversion. Each NIC namer plugin knows how to read
MAC-to-interface mappings from a specific network configuration backend and
rewrite them so the static IPs survive the NIC hardware change.

The block iterates all registered plugins, calls `Detect` to find which backend
is in use, and then calls `Apply` with the StaticIP list from the prepare
output to update the configuration files.

| Key | Package | Backend |
|-----|---------|---------|
| `dhclient` | dhclient/ | dhclient lease-based NIC identification |
| `ifcfg` | ifcfg/ | ifcfg-style network scripts (legacy RHEL/CentOS) |
| `netplan` | netplan/ | Netplan YAML configs (Ubuntu/Debian) |
| `nm` | nm/ | NetworkManager connection profiles |
| `nmdhcp` | nmdhcp/ | NetworkManager DHCP lease files |
| `wicked` | wicked/ | Wicked network configs (SUSE) |

## dhclient

**What it does:** Identifies NICs from dhclient lease files and updates
interface references for the new virtio NIC names.

**How it works:** Parses `/var/lib/dhclient/dhclient-*.leases` to extract
MAC-to-interface mappings. Updates lease file names and internal interface
references to match the post-conversion NIC naming.

## ifcfg

**What it does:** Preserves static IP configuration in legacy ifcfg-style
network scripts used by RHEL/CentOS 7 and earlier.

**How it works:** Reads `/etc/sysconfig/network-scripts/ifcfg-*` files, matches
`HWADDR` fields to the StaticIP MAC addresses, and rewrites `DEVICE` and
filename references to use the new interface names.

## netplan

**What it does:** Updates Netplan YAML configuration files used by Ubuntu and
modern Debian installations.

**How it works:** Parses `/etc/netplan/*.yaml` files, locates interface entries
by their `macaddress` match field, and updates the interface stanza keys and
any `set-name` directives to reflect the post-conversion NIC naming.

## nm

**What it does:** Rewrites NetworkManager connection profiles to preserve
static IP bindings.

**How it works:** Scans `/etc/NetworkManager/system-connections/*.nmconnection`
files, matches connections by `[ethernet] mac-address` to the StaticIP list,
and updates `[connection] interface-name` to the new NIC name.

## nmdhcp

**What it does:** Updates NetworkManager DHCP lease files to reflect the
new interface names.

**How it works:** Parses DHCP lease state files under
`/var/lib/NetworkManager/` and rewrites interface references found in lease
entries whose MAC address matches a known StaticIP.

## wicked

**What it does:** Preserves static IP configuration in Wicked network configs
used by SUSE distributions.

**How it works:** Reads `/etc/sysconfig/network/ifcfg-*` Wicked config files,
matches `LLADDR` (MAC address) fields to the StaticIP list, and renames the
files and updates internal interface references to use the post-conversion NIC
names.
