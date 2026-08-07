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

Format (subject + body — both required when drafting a commit message):

```text
<type>: <Description>

<Body paragraph(s).>
```

- Same type vocabulary as branches (`feat`, `fix`, `refactor`, `docs`, `test`, `ci`, `chore`).
- Sentence-style capitalization after the colon on the subject line.
- Subject under ~72 characters; one logical change per commit (atomic), per
  [community/pull-requests.md](../../../community/pull-requests.md).
- Blank line between subject and body.
- Body is required: short prose (1–3 sentences, wrap ~72 chars) covering
  **what** changed and **why**, in the same spirit as the PR “What” / “How”
  sections in `community/pull-requests.md`. Do not paste a file list or
  dump the full diff.
- Do not include secrets. Do not invent `Signed-off-by` (authors add that
  with `git commit -s` if needed).

Examples:

```text
feat: Add ARM64 virtio driver injection

Stage ARM64 VirtIO drivers from virtio-win so Windows guests on aarch64
can boot after conversion without a separate driver ISO.
```

```text
fix: Resolve LUKS unlock timeout in guestfs mode

Increase the guestfish wait for cryptsetup-open so slow NBDE unlocks do
not abort prepare before the volume is available to mount.
```

```text
chore: Switch container images from UBI to Go and Fedora

Use official golang and Fedora bases for upstream builds, and keep the
UBI Containerfile as an undocumented example under build/kc-v2v/ubi/.
```

## PR titles

Same `<type>: <Description>` shape as commit **subjects** only (no body).
Keep under ~72 characters.

## Output

Return only what the user asked for. If they want both a branch name and a
commit message, emit **two separate fenced code blocks** (one value each) so
each can be copied on its own. Label each block with a short plain-text line
above it (not inside the fence). Do not put branch and commit in the same
fence or on the same line.

If the user asks for only a branch name, only a commit message, or only a PR
title, return a single labeled fence with that value alone.

**Both branch and commit (default when the user asks for both / "suggest a
branch and commit msg"):**

Branch:

```text
<type>/<short-description>
```

Commit:

```text
<type>: <Description>

<Body paragraph(s).>
```

The commit fence must include the blank line and body so the user can paste
it into `git commit` / a HEREDOC unchanged.

**PR title only** (when asked):

PR title:

```text
<type>: <Description>
```

If the change mixes concerns, suggest a split instead of one overloaded name.
Keep any extra commentary outside the fences.
