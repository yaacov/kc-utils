# firstboot plugins

`FirstBootHandler` interface — install commands to run on first guest boot (Linux).

The firstboot mechanism lets conversion blocks schedule shell commands that must
execute inside the guest on its first boot under KVM. The handler writes a
self-contained script and the init-system glue needed to run it exactly once,
then clean up after itself.

| Key | Package | Handler |
|-----|---------|---------|
| `systemd` | systemd/ | Systemd oneshot service with self-cleanup |

## systemd

**What it does:** Creates a systemd oneshot service unit and a companion shell
script under the guest filesystem. On the guest's first boot the service runs
the accumulated commands (e.g. static-IP application, qemu-guest-agent install)
and then removes both the unit file and the script so nothing persists.

**How it works:** `Install(mountRoot, commands)` writes two files into the
mounted guest:

1. A shell script at a well-known path containing all requested commands.
2. A systemd unit (`Type=oneshot`, `After=network-online.target`, `TimeoutStartSec=120`) that executes
   the script and deletes both files on completion.

The script header uses `RETRIES=2` and `DELAY=5` for transient command failures.

The unit is symlinked into `multi-user.target.wants` so systemd picks it up
automatically. No package installation is needed — the mechanism relies only on
systemd, which is present on all supported Linux distributions.
