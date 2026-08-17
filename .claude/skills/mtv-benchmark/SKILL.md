---
name: mtv-benchmark
description: >-
  Run or inspect the MTV cluster scenario benchmark (kc-v2v vs operator
  virt-v2v) via tests/scenarios/test-mtv-benchmark.sh. Use when the user asks
  to run the full benchmark, MODE=kc or MODE=compare, cluster scenario tests,
  or to clean up after a benchmark run.
---

# MTV scenario benchmark

Human docs are the source of truth. Read them before running anything:

1. [tests/scenarios/README.md](../../../tests/scenarios/README.md) — runner, `.env`, artifacts, cleanup
2. [tests/scenarios/test-mtv-benchmark.md](../../../tests/scenarios/test-mtv-benchmark.md) — test plan and pass/fail
3. [docs/apps/forklift-usage.md](../../../docs/apps/forklift-usage.md) — MTV integration / `KC_V2V_IMAGE`
4. [docs/architecture/ref-baseline/README.md](../../../docs/architecture/ref-baseline/README.md) — published compare archives (only when publishing)

## Agent protocol

- **full** / compare → `MODE=compare ./tests/scenarios/test-mtv-benchmark.sh` (shell `MODE` overrides `.env`; `.env` may say `kc`).
- Independent kc-v2v only → `MODE=kc ./tests/scenarios/test-mtv-benchmark.sh`.
- If `tests/scenarios/.env` is missing, stop and tell the user to copy `.env.example`. Do not invent credentials.
- Preflight: `oc` login, `oc mtv health`, VDDK via script preflight. If MTV is not installed, **stop** — do not install operators unless asked.
- Never print `GOVC_*`, `.env`, or other secrets. `GOVC_*` may already be in the shell even if omitted from `.env`.
- Compare is hours (ref RHEL+Windows, then kc). Background the runner; do not block the session on it.
- Do **not** `make build-kc-v2v-image` / `push-kc-v2v-image`, `clean-env.sh`, or `clean-runs.sh` unless the user asks.
- Defaults leave namespace and MTV settings in place (`SKIP_CLEANUP=true`, `KEEP_IMAGE_SETTING=true`).
- Do not duplicate the test plan in chat; point at the docs and report command, cluster, `MODE`, and artifact prefix.
