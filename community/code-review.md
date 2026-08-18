# Code Review

Guidelines for reviewing kc-utils changes (humans and agents). Prefer Makefile
targets over raw Go commands. Do not run `make check` during review — it runs
`fmt` and writes files. Use `make vet`, `make lint`, and `make test` instead.
On non-Linux hosts, also run `make test-container` so `//go:build linux`
packages (e.g. `pkg/backend/plugins/guestfs`) are exercised — local `make test` skips them.

Read [architecture.md](architecture.md) and the "Guest conversion conventions"
section in [CONTRIBUTING.md](CONTRIBUTING.md) before judging structural or
guest-filesystem changes.

## Scope

- **Changed files (default):** Review the diff vs `main` plus unstaged, staged,
  and untracked reviewable files. Skip generated files, `vendor/`, `*.log`,
  `bin/`, and `build/kc-v2v/cache/`.
- **Full codebase:** Review `.go` under `pkg/` and `cmd/`, shell under `build/`
  and `tests/`, Containerfiles, `Makefile`, `.github/workflows/ci.yml`,
  `.golangci.yml`, markdown under `docs/` and `community/`, and root
  `README.md`. Same skip list as above.

## Priority 1: Readability and maintainability

- **Naming:** Clear names consistent with surrounding code; describe *what*,
  not *how*.
- **Simplification:** Dead code, unreachable branches, unnecessary
  abstractions, overly complex conditionals, duplicated logic in the same
  package.
- **Go idioms:** Proper error handling (no swallowed errors, no `_ = err`
  without justification), interfaces, struct design, receiver consistency,
  blank import discipline.
- **Function length:** Flag functions over ~60 lines that could be split into
  named helpers.
- **Test coverage:** New or changed paths covered; existing tests still valid.

## Priority 2: Architecture

Check against [architecture.md](architecture.md) and guest conventions in
[CONTRIBUTING.md](CONTRIBUTING.md):

- **Block isolation:** Typed inputs/outputs only; no cross-block state leaks.
- **Stage isolation:** No imports from `pkg/<stage-A>/` into `pkg/<stage-B>/`.
  Stages communicate via JSON files and shared mount points.
- **Backend transparency:** No backend-specific logic outside `pkg/guest/`.
  Blocks use the `Guest` handle exclusively.
- **Plugin structure:** `init()` self-registration into a `plugin.Registry`;
  new plugins in their own file under `plugins/`; blank imports in
  `cmd/kc-*/main.go`.
- **Shared code placement:** Cross-stage code in `pkg/common/` or `pkg/guest/`.
- **Guest conversion conventions:** Guest-absolute symlink targets, append-safe
  firstboot, shared hypervisor cleanup helpers, prefer `RunInGuest`, prefer
  offline removal over firstboot.

## Priority 3: Docs

- **Default:** Whether changed code needs doc updates; only review docs in
  scope or clearly impacted by the diff.
- **Full mode:** Accuracy of block tables, CLI flags, and architecture claims
  in `community/architecture.md` and `docs/apps/kc-*.md`.

In both modes:

- **`docs/apps/kc-*.md`:** Pipeline block add/remove/rename/behavior and CLI flags.
- **`community/architecture.md`:** Structural pattern changes still hold.
- **`community/CONTRIBUTING.md`:** Build, test, or dependency changes reflected.
- **Package READMEs:** Plugin add/remove keeps `plugins/README.md` current.
- **Public APIs:** Exported symbols should have GoDoc comments.

## Priority 4: Build and test infrastructure

### Shell scripts (`build/`, `tests/`)

- `set -euo pipefail` (or source `functions.sh` with error traps).
- Quote expansions (`"$VAR"`). No unguarded `rm -rf` on variable paths.
- Tests source `tests/functions.sh` and use its helpers.
- Names: `test-{linux,windows,root,kc,disk,dynamicscripts}-*.sh`.
- Register cleanup with `cleanup_fn`. Exit 0 / 77 / non-zero.

### Containerfiles

- Rarely changing layers before frequently changing ones.
- Multi-stage builds must not leak build tools into runtime.
- No secrets in layers; avoid root unless required.
- Pin tags/digests where feasible; `--setopt=install_weak_deps=False` and
  `dnf clean all`.

### Makefile

- Public targets have `##` help comments.
- Consistent variables/quoting; accurate target dependencies;
  `CONTAINER_RUNTIME` / `PLATFORM` work for Docker and Podman.

### CI (`.github/workflows/ci.yml`)

- Correct `needs:` chains; Go version matches `go.mod`; golangci-lint version
  matches `GOLANGCI_LINT_VERSION` in the Makefile; module/build cache handled.

### Linter config (`.golangci.yml`)

- Enabled linters appropriate; no important linters disabled without reason.

## Report shape

```markdown
## Lint & Vet
(automated tool output or "Clean — no issues.")

## Priority 1: Readability & Maintainability
- [file:line] Finding summary
  Details and suggested fix.

## Priority 2: Architecture
- [file:line] Finding summary
  Details and reference to violated rule.

## Priority 3: Docs
- [file:line or doc path] Finding summary
  What needs updating and why.

## Priority 4: Build & Test Infrastructure
- [file:line] Finding summary
  Details and suggested fix.
```

Include every priority section (use "No issues found." when empty). End with a
one-line **Verdict**: finding counts per priority, and ready to merge or needs
work.
