# pkg/

Block-centric libraries organized by utility. Each utility has one README here;
pluggable blocks document implementers in `<block>/plugins/README.md`.

| Utility | README | Role |
|---------|--------|------|
| **common** | [`common/`](common/) | Cross-cutting helpers: types, logger, plugin, configedit, registry, uefi, firstboot |
| **prepare** | [`prepare/`](prepare/) | All kc-prepare blocks |
| **convert-linux** | [`convert-linux/`](convert-linux/) | All kc-convert-linux blocks |
| **convert-windows** | [`convert-windows/`](convert-windows/) | All kc-convert-windows blocks |
| **finalize** | [`finalize/`](finalize/) | All kc-finalize blocks |
| **guest** | [`guest/`](guest/) | Privileged guest disk ops: direct (host mounts) and guestfs (libguestfs) backends behind a common `Backend` interface |
| **copy** | [`copy/`](copy/) | NFC disk copy via govmomi: VMDK stream-to-raw conversion and vSphere export |
| **v2v** | [`v2v/`](v2v/) | kc-v2v libraries (config, env, copy, vsphere, inspection); orchestrator in [`cmd/v2v/`](cmd/v2v/) |

Path pattern: `pkg/<utility>/<semantic-name>/` with optional `plugins/` for pluggable blocks.

Pipeline orchestration: [`pkg/cmd/`](cmd/). User-facing docs: [`docs/`](../docs/).
