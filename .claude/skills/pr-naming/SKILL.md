---
name: pr-naming
description: >-
  Name PR branches and draft commit messages (and PR titles) for kc-utils
  using community/pull-requests.md conventions: type prefixes, hyphenated
  short descriptions, and atomic commits. Use when the user asks for a
  branch name, commit message, or PR title. Read-only git in the working
  tree; destructive git only in a /tmp clone, never in the workspace.
---

# kc-utils PR Branch & Commit Naming

Draft names and messages only. Do not create branches, commits, or PRs in
the user's working tree — propose text the user can apply.

**Git under this skill:**
- Allowed in the working tree (read-only): `git status`, `git diff`,
  `git log`, `git show`, `git branch` (list only), `git rev-parse`, and
  similar inspect commands.
- Forbidden in the working tree (destructive/write): `git add`,
  `git commit`, `git checkout`, `git switch`, `git branch`
  create/delete/rename, `git push`, `git pull`, `git merge`, `git rebase`,
  `git stash`, `git reset`, `git tag`, or any other command that mutates
  the working tree, index, or refs.
- If a destructive git operation is required (e.g. validating a branch
  name, dry-running a commit message, or other experiments):
  1. `git clone` the repo into a fresh directory under `/tmp`
     (e.g. `/tmp/kc-utils-pr-naming-$$`).
  2. Run all destructive commands only inside that clone.
  3. Never mutate the user's real working directory, index, or refs.
  4. Clean up the `/tmp` clone when done if practical.

When naming a branch, drafting a commit message, or writing a PR title:

1. Read [community/pull-requests.md](../../../community/pull-requests.md) with the Read tool.
2. Inspect the change with read-only git (`status`, `diff`, recent `log`) and conversation context as needed.
3. Propose names that follow the rules below.
4. Prefer one concern per branch/commit; do not invent ticket numbers.

## Branch names

Format: `<type>/<short-description>`

| Prefix | Use for |
|--------|---------|
| `feat/` | New features or capabilities |
| `fix/` | Bug fixes |
| `refactor/` | Restructuring without behavior changes |
| `docs/` | Documentation-only |
| `test/` | Adding or updating tests |
| `ci/` | CI/CD pipeline |
| `chore/` | Dependencies, tooling, build tweaks |

Rules:
- Lowercase; hyphens only (no underscores; no extra slashes after the type).
- 3–5 words after the prefix.
- Describe *what* the branch does, not a ticket alone.
- Optional issue suffix: `fix/luks-timeout-42`.

Examples:
```text
feat/arm64-virtio-injection
fix/guestfs-luks-unlock-timeout
refactor/extract-registry-helpers
docs/add-privilege-model
test/e2e-windows-uefi
ci/add-disk-image-job
chore/bump-govmomi
```

## Commit messages

Format: `<type>: <description>`

- Same type vocabulary as branches (`feat`, `fix`, `refactor`, `docs`, `test`, `ci`, `chore`).
- Sentence-style capitalization after the colon.
- One logical change per commit; keep the subject under ~72 characters.
- Focus on *why* when a short subject alone is ambiguous; otherwise a clear subject is enough.
- Do not include secrets or dump the full diff into the message.

Examples:
```text
feat: Add ARM64 virtio driver injection
fix: Resolve LUKS unlock timeout in guestfs mode
refactor: Extract shared registry helpers to pkg/common
docs: Add privilege model documentation
```

## PR titles

Same `<type>: <Description>` shape as commit subjects. Keep under ~72 characters.

## Output

Reply with:

```text
Branch: <type>/<short-description>
Commit: <type>: <Description>
PR title: <type>: <Description>
```

Omit lines the user did not ask for. If the change mixes concerns, suggest a split instead of one overloaded name.
