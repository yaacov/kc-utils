# selinux -- offline SELinux relabel with setfiles

Performs an offline SELinux filesystem relabel of the converted guest using the `setfiles` utility. This avoids the slow boot-time relabel and automatic reboot that a `/.autorelabel` marker file would trigger.

`Relabel` first checks whether `/etc/selinux` exists and parses the SELinux config to determine the active policy type (e.g. `targeted`) and whether SELinux is disabled. If SELinux is active, it locates the `setfiles` binary in the guest (checking `/usr/sbin/setfiles` and `/sbin/setfiles`), builds the file-contexts spec path from the policy name, and runs `setfiles -r / <spec> <mountpoints...>` inside the guest via chroot. The function processes all guest mount points (defaulting to `/` if none are provided) because `setfiles` does not cross filesystem boundaries. On success it removes any existing `/.autorelabel` file and returns true; on failure the caller can fall back to creating `/.autorelabel`.

## Key exports

| Symbol | Role |
|--------|------|
| `Relabel` | Runs offline SELinux relabel via `setfiles` against all guest mount points; returns whether relabeling succeeded |
