Offline conversion helpers for guests that use **systemd-networkd** as the primary network stack (Amazon Linux 2023, and other distros where NetworkManager is masked or disabled).

On EC2, `amazon-ec2-net-utils` generates ephemeral `/run/systemd/network/*.network` files from IMDS. After migration to KubeVirt virtio NICs need persistent DHCP configuration.

| Function | Role |
|----------|------|
| `Detect` | True for vendor `80-ec2.network`, `ID=amzn` with `VERSION_ID=2023` in os-release (unless both networkd and NM enabled), or networkd enabled without active NetworkManager. Amazon Linux 2 (`VERSION_ID=2`) is not matched unconditionally — it falls through to the networkd-enabled check |
| `InstallKubeVirtNetworking` | Writes virtio DHCP `.network` file and wait-online drop-in |
| `InstallDHCP` | Writes `10-kc-virtio.network` (DHCP on virtio `enp*` / `eth*`) |
| `WriteStaticNetworks` | Writes MAC-matched `10-kc-static-*.network` files from plan static IPs |
| `InstallWaitOnlineDropIn` | 30s `--any` override for `systemd-networkd-wait-online` |

`Detect` is used by the [`networkd` network handler](../handlers/networkd/). When that handler is selected via [`network.Select()`](../network.go), block 11b runs `InstallKubeVirtNetworking` and block 15 uses `WriteStaticNetworks` instead of `nicnaming` + firstboot. EC2 net-hook masking is handled by [`systemd.DisableEC2NetHooks`](../../systemd/) from the EC2 hypervisor plugin.
