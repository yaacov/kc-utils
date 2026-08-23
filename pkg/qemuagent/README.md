# qemuagent

The wire contract and in-guest agent for the [`qemu` backend](../backend/plugins/qemu/).
The backend boots a minimal appliance and drives it over a unix socket; this
package is what the appliance side speaks.

| Package | Role |
|---------|------|
| [`proto/`](proto/) | Wire protocol: length-prefixed JSON frames, `Request`/`Response`, `Op` constants, `ReadMsg`/`WriteMsg`. Shared by host and guest; no build tags. |
| [`server/`](server/) | The agent: `Serve(rw)` reads a `Request`, dispatches to a primitive handler (`os`/`os/exec`), writes a `Response`. Portable Go — handlers are testable on any OS over `net.Pipe`. |

The agent is compiled into `cmd/kc-guest-agent` and runs as `/init` inside the
appliance (see [`build/kc-appliance`](../../build/kc-appliance)). It exposes only
**primitive** operations — exec, file I/O, raw device I/O, stat/statfs — and never
imports `pkg/backend`: all conversion logic lives host-side in the backend.

Binary contract: [docs/apps/kc-guest-agent.md](../../docs/apps/kc-guest-agent.md).
Protocol and the host/guest split: [docs/architecture/qemu-appliance.md](../../docs/architecture/qemu-appliance.md).
