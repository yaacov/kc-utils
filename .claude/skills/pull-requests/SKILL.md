---
name: pull-requests
description: >-
  Name PR branches and draft PR titles or bodies for kc-utils using
  community/pull-requests.md: type/short-description branches, PR title and
  What/How/Testing body. Use when the user asks for a branch name, PR title,
  or PR description. Read-only git only; never run mutating git or GitHub
  commands. Commit messages belong to the commits skill.
---

# kc-utils Pull Requests & Branches

Draft names and PR text only. **Never** create branches, push, or open PRs.
The user runs those steps manually.

## Git policy

**Allowed (run these):** read-only inspect commands — `status`, `diff`, `log`,
`show`, `branch` (list only), `rev-parse`, `describe`, `ls-files`, and similar.

**Forbidden (never run; suggest for the user instead):** `checkout`, `switch`,
`branch` (create/delete/rename), `push`, `pull`, `add`, `commit`, `merge`,
`rebase`, `reset`, and `gh pr create` / other GitHub actions that create or
mutate branches or pull requests.

When the user needs a branch or PR, suggest exact commands such as
`git checkout -b feat/…`, `git push -u origin HEAD`, and `gh pr create …` for
them to run. Full policy:
[community/CONTRIBUTING.md](../../../community/CONTRIBUTING.md#ai-agents-and-git).

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
