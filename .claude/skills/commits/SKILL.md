---
name: commits
description: >-
  Draft commit messages for kc-utils using community/commits.md: type-prefixed
  subject plus required what/why body, atomic commits. Use when the user asks
  for a commit message. Read-only git only; never run mutating git commands.
---

# kc-utils Commit Messages

Draft messages only. **Never** run mutating git commands. The user stages and
commits manually.

## Git policy

**Allowed (run these):** read-only inspect commands — `status`, `diff`, `log`,
`show`, `branch` (list only), `rev-parse`, `describe`, `ls-files`, and similar.

**Forbidden (never run; suggest for the user instead):** `add`, `commit`,
`push`, `pull`, `checkout`, `switch`, `restore`, `branch` (create/delete/rename),
`tag`, `merge`, `rebase`, `reset`, `cherry-pick`, `stash`, and any command that
changes the working tree, index, or refs.

When the user needs to commit, suggest the exact `git add` and `git commit`
commands (or a paste-ready message block) for them to run. Full policy:
[community/CONTRIBUTING.md](../../../community/CONTRIBUTING.md#ai-agents-and-git).

When drafting a commit message:

1. Read [community/commits.md](../../../community/commits.md) with the Read tool.
2. Inspect the change with read-only git and conversation context as needed.
3. Propose a subject and body that follow that doc.
4. Prefer one concern per commit; do not invent ticket numbers.

Return a single labeled fenced block the user can paste into `git commit`
unchanged (subject, blank line, body). For branch names or PR text, use the
pull-requests skill and [community/pull-requests.md](../../../community/pull-requests.md).
