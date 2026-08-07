---
name: pull-requests
description: >-
  Name PR branches and draft PR titles or bodies for kc-utils using
  community/pull-requests.md: type/short-description branches, PR title and
  What/How/Testing body. Use when the user asks for a branch name, PR title,
  or PR description. Read-only git in the working tree; do not create
  branches or PRs. Commit messages belong to the commits skill.
---

# kc-utils Pull Requests & Branches

Draft names and PR text only. Do not create branches or PRs in the user's
working tree.

**Git under this skill:** read-only inspect commands only (`status`, `diff`,
`log`, `show`, `branch` list, `rev-parse`). Do not mutate the working tree,
index, or refs.

When naming a branch or writing a PR title/body:

1. Read [community/pull-requests.md](../../../community/pull-requests.md) with the Read tool.
2. Inspect the change with read-only git and conversation context as needed.
3. Propose branch and/or PR text that follow that doc.
4. Prefer one concern per PR; do not invent ticket numbers.

Do **not** draft commit messages here — use the commits skill and
[community/commits.md](../../../community/commits.md).

Return only what the user asked for. If they want both a branch name and a PR
title, emit two separate labeled fenced code blocks. If the change mixes
concerns, suggest a split instead of one overloaded name.
