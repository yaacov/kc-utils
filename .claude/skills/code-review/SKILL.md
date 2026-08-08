---
name: code-review
description: >-
  Full code review for kc-utils against community/code-review.md: readability,
  architecture, docs, and build/test infrastructure. Defaults to changed
  files vs main; pass "full" to review the entire codebase including docs.
---

# kc-utils Code Review

Run a structured code review. By default, review files changed vs main (plus
unstaged/staged/untracked). If the user passes "full", review the full
codebase scope described in the community doc.

1. Determine scope per [community/code-review.md](../../../community/code-review.md). If there is nothing to review and not "full" mode, say so and stop.
2. Run `make vet`, `make lint`, `make test`, and `make test-e2e-container` (do **not** run `make check` — it writes files via `fmt`). Capture results for the Lint & Vet section.
3. Read [community/code-review.md](../../../community/code-review.md) with the Read tool. Also read [community/architecture.md](../../../community/architecture.md) and the guest conversion conventions in [community/CONTRIBUTING.md](../../../community/CONTRIBUTING.md) when judging structural or guest-filesystem changes.
4. Review against that doc’s four priorities and present findings in its report shape, ending with a one-line Verdict.
