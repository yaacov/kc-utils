# nicnaming plugins

`NICNamer` interface — preserve NIC naming for static IP configurations across network backends.

| Key | Package | Backend |
|-----|---------|---------|
| `dhclient` | dhclient/ | dhclient lease-based NIC identification |
| `ifcfg` | ifcfg/ | ifcfg-style network scripts (legacy RHEL/CentOS) |
| `netplan` | netplan/ | Netplan YAML configs (Ubuntu/Debian) |
| `nm` | nm/ | NetworkManager connection profiles |
| `nmdhcp` | nmdhcp/ | NetworkManager DHCP lease files |
| `wicked` | wicked/ | Wicked network configs (SUSE) |
