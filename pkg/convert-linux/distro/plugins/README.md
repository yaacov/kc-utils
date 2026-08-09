# distro plugins

`DistroHandler` interface — classify OS family, package format, and package manager.

Distro classification is the first step in the Linux conversion pipeline. The
matched handler tells downstream blocks which package format to use (RPM vs
DEB vs Zypper), how to invoke the package manager for guest-agent installation,
and which distro-specific quirks to apply (e.g. SELinux relabeling for RHEL
family, netplan for Ubuntu). Each plugin's `Matches` method inspects the
`InspectData` fields (distro ID, version) populated by kc-prepare and claims
the guest if it recognizes the distribution.

| Key | Package | Distros |
|-----|---------|---------|
| `rhel` | rhel/ | RHEL, CentOS, Rocky, Alma, Oracle Linux, Fedora, Amazon Linux (`amzn`) |
| `debian` | debian/ | Debian, Ubuntu |
| `suse` | suse/ | SLES, openSUSE |

## rhel

**What it does:** Matches Red Hat family distributions including RHEL, CentOS,
Rocky Linux, AlmaLinux, Oracle Linux, Fedora, and Amazon Linux.

**How it works:** `Matches` checks the inspect distro ID against a list of
known RHEL-family identifiers. Returns RPM as the package format and `yum` or
`dnf` as the package manager depending on the major version (dnf for RHEL 8+,
yum for older). Provides distro-specific console configuration hints used by
the bootconfig block.

## debian

**What it does:** Matches Debian and Ubuntu distributions.

**How it works:** `Matches` checks for `debian` or `ubuntu` distro IDs.
Returns DEB as the package format and `apt` as the package manager. Downstream
blocks use this to select the `deb` kernel scanner and netplan-based NIC
naming.

## suse

**What it does:** Matches SUSE Linux Enterprise Server (SLES) and openSUSE.

**How it works:** `Matches` checks for `sles` or `opensuse` distro IDs.
Returns RPM as the package format and `zypper` as the package manager. SUSE
guests use the `wicked` network backend for NIC naming preservation.
