---
name: code-review
description: >-
  Full code review for kc-utils: readability & maintainability, architecture
  compliance, docs completeness, and build/test infrastructure. Defaults to
  reviewing changed files vs main; pass "full" to review the entire codebase.
---

# kc-utils Code Review

Run a structured code review with four priority tiers. By default, review
only files changed vs the main branch. If the user passes "full" as an
argument, review all source files under `pkg/`, `cmd/`, `build/`, `tests/`,
and infrastructure files (`Makefile`, `.github/`, `.golangci.yml`).

## Step 0: Determine scope

Decide which files to review:

- **Default (changed files):** Collect paths from all of:
  `git diff --name-only main...HEAD`, `git diff --name-only`,
  `git diff --cached --name-only`, and
  `git ls-files --others --exclude-standard`. Deduplicate the combined
  list, then skip generated files, `vendor/`, log files (`*.log`), and
  binary artifacts (`bin/`, `build/kc-v2v/cache/`).
- **Full codebase:** Review all `.go` files under `pkg/` and `cmd/`, all
  `.sh` files under `build/` and `tests/`, the `Containerfile` files, the
  `Makefile`, `.github/workflows/ci.yml`, and `.golangci.yml`.

If the diff is empty (no changed, staged, or untracked reviewable files and
not "full" mode), tell the user there is nothing to review and stop.

## Step 1: Automated checks

Run the following and capture any output:

1. `GOOS=linux go vet ./pkg/... ./cmd/...`
2. `GOOS=linux make lint` (golangci-lint)
3. `GOOS=linux go test ./pkg/... ./cmd/...` (or `make test`)

Report any lint/vet/test findings verbatim at the top of the review under a
**Lint & Vet** heading before the priority sections. If clean, say so briefly.
Capture and report test results before producing the review verdict.

## Step 2: Read project standards

Before reviewing code, read these files with the Read tool to calibrate
your review against the project's own rules:

1. `community/architecture.md` — structural rules, block isolation, backend
   transparency, plugin system, priorities.
2. `community/CONTRIBUTING.md` — guest conversion conventions, symlink paths,
   firstboot safety, RunInGuest patterns, PR checklist.

## Step 3: Review — Priority 1: Readable Code & Maintainability

Read every in-scope Go file. For each, evaluate:

- **Naming:** Are function, variable, type, and package names clear and
  consistent with surrounding code? Do they describe *what*, not *how*?
- **Simplification:** Can any logic be reduced without losing clarity?
  Look for: dead code, unreachable branches, unnecessary abstractions,
  overly complex conditionals, duplicated logic within the same package.
- **Go idioms:** Proper error handling (no swallowed errors, no `_ = err`
  without justification), interface usage, struct design, receiver
  consistency, blank import discipline.
- **Function length:** Flag functions over ~60 lines that could be split
  into named helpers for readability.
- **Test coverage:** Are new or changed code paths covered by tests? Are
  existing tests still valid after the changes?

List findings under a **Priority 1: Readability & Maintainability** heading.

## Step 4: Review — Priority 2: Architecture

Check changed code against the architecture rules from Step 2:

- **Block isolation:** Does each block operate only on its typed inputs and
  outputs? No cross-block state leaks, no global mutable state shared
  between blocks.
- **Stage isolation:** No imports from `pkg/<stage-A>/` into `pkg/<stage-B>/`.
  Stages communicate only via JSON files and shared mount points.
- **Backend transparency:** No backend-specific logic (`mount`, `guestfish`,
  `chroot`, mode checks) outside `pkg/guest/`. Blocks use the `Guest`
  handle exclusively.
- **Plugin structure:** New pluggable behavior uses `init()` self-registration
  into a `plugin.Registry`. New plugins have their own file under `plugins/`.
  Blank imports are in the stage's `cmd/kc-*/main.go`.
- **Shared code placement:** Code shared across stages lives in `pkg/common/`
  or `pkg/guest/`, not in one stage imported by another.
- **Guest conversion conventions:** Symlinks use guest-absolute paths,
  firstboot scripts are append-safe, hypervisor cleanup uses shared helpers,
  guest-native tools are preferred via `RunInGuest`, offline removal is
  preferred over firstboot scheduling.

List findings under a **Priority 2: Architecture** heading.

## Step 5: Review — Priority 3: Docs Updates

Check whether changed code requires documentation updates:

- **`docs/kc-*.md`:** If a pipeline block was added, removed, renamed, or
  had its behavior changed, does the corresponding `docs/` file reflect it?
  Check CLI flags, block description tables, and behavioral notes.
- **`community/architecture.md`:** If structural patterns were changed
  (new shared package, new plugin registry, new backend method), does the
  architecture doc still hold?
- **`community/CONTRIBUTING.md`:** If build steps, test commands, or
  dependency requirements changed, is CONTRIBUTING updated?
- **Package READMEs:** If a `plugins/README.md` exists and plugins were
  added or removed, is the README current?
- **Code comments on public APIs:** Exported functions and types should have
  GoDoc comments. Flag exported symbols without doc comments.

List findings under a **Priority 3: Docs** heading.

## Step 6: Review — Priority 4: Build & Test Infrastructure

Review in-scope non-Go files: shell scripts, Containerfiles, Makefile,
CI workflow, and linter config.

### Shell scripts (`build/`, `tests/`)

- **Safety:** Scripts should use `set -euo pipefail` (or source
  `functions.sh` which sets up error traps). No unquoted variable
  expansions (`$VAR` should be `"$VAR"`). No unguarded `rm -rf` on
  variable paths.
- **Consistency:** Test scripts should source `tests/functions.sh` and
  use its helpers (`requires`, `requires_linux`, `skip_if_skipped`,
  `cleanup_fn`, `check_json_field`) rather than reimplementing them.
- **Naming:** Test script names follow `test-{linux,windows,root,kc,disk,dynamicscripts}-*.sh`.
  Helper scripts are clearly named (`functions.sh`, `build.sh`).
- **Cleanup:** Tests must register cleanup hooks via `cleanup_fn` for
  any temp files, mount points, or loop devices they create. No leaked
  state between tests.
- **Exit codes:** Tests exit 0 (pass), 77 (skip), or non-zero (fail).
  Skip conditions must be correct and documented.

### Containerfiles (`build/kc-v2v/Containerfile`, `tests/Containerfile`)

- **Layer ordering:** Place rarely-changing layers (base image, package
  installs) before frequently-changing layers (COPY source code) for
  cache efficiency.
- **Image size:** Multi-stage builds should not leak build-time tools
  into the runtime stage. Runtime stage installs only required packages.
- **Security:** No secrets, credentials, or tokens in any layer. Runtime
  containers should not run as root unless required (check USER/ENTRYPOINT).
- **Reproducibility:** Pin base image tags or digests where feasible.
  Package installs use `--setopt=install_weak_deps=False` and `dnf clean all`.

### Makefile

- **Target documentation:** Every public target should have a `## comment`
  line for the `help` target.
- **Consistency:** Variable naming, quoting, and pattern usage should be
  uniform across targets.
- **Correctness:** Dependencies between targets (e.g., `test-e2e: test-build`)
  should be accurate. Cross-platform variables (`CONTAINER_RUNTIME`,
  `PLATFORM`) should work on both Docker and Podman.

### CI (`.github/workflows/ci.yml`)

- **Job dependencies:** Verify `needs:` chains are correct (lint/test
  before build, build before e2e).
- **Go version:** Should match `go.mod`'s Go version.
- **golangci-lint version:** CI version should match the Makefile-pinned
  version (`GOLANGCI_LINT_VERSION`).
- **Caching:** Go modules and build cache should be handled (setup-go
  action does this).

### Linter config (`.golangci.yml`)

- **Linter set:** Are the enabled linters appropriate? Flag if important
  linters are disabled without reason or if noisy linters create false
  positives.

List findings under a **Priority 4: Build & Test Infrastructure** heading.

## Step 7: Output

Present the review as:

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

If a priority section has no findings, include it with "No issues found."

At the end, add a one-line **Verdict** summarizing the overall state:
how many findings per priority, and whether the change is ready to merge
or needs work.
