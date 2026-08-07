---
name: commits
description: >-
  Draft commit messages for kc-utils using community/commits.md: type-prefixed
  subject plus required what/why body, atomic commits. Use when the user asks
  for a commit message. Read-only git in the working tree; do not create
  commits.
---

# kc-utils Commit Messages

Draft messages only. Do not create commits in the user's working tree.

**Git under this skill:** read-only inspect commands only (`status`, `diff`,
`log`, `show`, `branch` list, `rev-parse`). Do not mutate the working tree,
index, or refs.

When drafting a commit message:

1. Read [community/commits.md](../../../community/commits.md) with the Read tool.
2. Inspect the change with read-only git and conversation context as needed.
3. Propose a subject and body that follow that doc.
4. Prefer one concern per commit; do not invent ticket numbers.

Return a single labeled fenced block the user can paste into `git commit`
unchanged (subject, blank line, body). For branch names or PR text, use the
pull-requests skill and [community/pull-requests.md](../../../community/pull-requests.md).
