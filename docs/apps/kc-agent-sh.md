# kc-agent-sh (appliance debug shell)

Host helper that attaches an interactive bash PTY to a **running** QEMU
appliance without stealing the conversion RPC socket. Unix only
(`//go:build unix`).

**Entry:** [`cmd/kc-agent-sh/main.go`](../../cmd/kc-agent-sh/main.go) —
orchestrator in [`pkg/cmd/agentsh/`](../../pkg/cmd/agentsh/).

This is not a pipeline stage: no JSON I/O, no `--backend` flag. It talks to
the second virtio-serial port (`org.kc-utils.shell` / sibling `shell.sock`)
documented in [../backends/kc-agent.md](../backends/kc-agent.md#debug-shell).

## Requirements

- A qemu-backend conversion with the appliance already up (`KC_AGENT_SOCK` set
  by a shared session, or a standalone prepare that started QEMU).
- An appliance image built **after** this helper was added (`make appliance`).
  Older initramfs images have no shell port; the RPC path still works.
- The host `kc-agent-sh` binary from `make build`. It is **not** in the
  guestfs `kc-v2v` conversion image (`V2V_backend=guestfs` has no shell port).

## CLI Flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--sock` | no | sibling of `$KC_AGENT_SOCK` | Shell Unix socket. If the path basename is `agent.sock`, the sibling `shell.sock` is used. |
| `--chroot` | no | | `chroot` into a path already mounted in the appliance (for example the guest `--mount-root`) |
| extra args | no | interactive `/bin/bash` | Command argv to exec instead of a login shell |

## Usage

```bash
# Shared qemu session: KC_AGENT_SOCK is already set
kc-agent-sh

# Explicit agent socket (derives shell.sock next to it)
kc-agent-sh --sock /tmp/kc-qemu-xxx/agent.sock

# Drop into the mounted guest root instead of the appliance
kc-agent-sh --chroot /tmp/kc-guest

# One-shot command
kc-agent-sh -- lsblk
```

The wire after a one-line JSON header is a raw PTY byte stream. Equivalent
with socat (send `{}` so the agent starts default bash):

```bash
( printf '%s\n' '{}'; cat ) | socat STDIO,rawer UNIX-CONNECT:/path/to/shell.sock
```

## Errors

If the shell socket is missing, the helper reports that the qemu appliance is
not running or the appliance was not rebuilt. Conversion RPC on `KC_AGENT_SOCK`
is independent and is not interrupted by this helper.
