# Commit Messages

Guidelines for writing git commit messages in kc-utils.

## AI agents

AI agents **draft** commit messages only. They must **not** run `git add`,
`git commit`, or any other command that mutates the working tree, index, or
refs. The user stages and commits manually.

Agents may inspect changes with read-only git (`status`, `diff`, `log`, `show`,
`branch` list, `rev-parse`, and similar). They may suggest commands such as
`git add …` and `git commit -m "…"` for the user to run. See
[CONTRIBUTING.md](CONTRIBUTING.md#ai-agents-and-git) for the full git policy.

## Commits vs pull requests

A **commit** is the git history unit: care about the **subject** and **message
body**. A **pull request** is the review unit: care about the **branch name**
and the **PR description** (see [pull-requests.md](pull-requests.md)).

When a change is one commit and one PR, the commit **subject** often matches
the PR **title**. The bodies still differ: the commit body is short what/why
prose for `git log`; the PR body uses the What / How / Testing template for
reviewers. Branch names use slash form (`feat/…`) and never appear on the
commit subject.

## Format

Subject and body are both required:

```text
<type>: <Description>

<Body paragraph(s).>
```

| Type | Use for |
|------|---------|
| `feat` | New features or capabilities |
| `fix` | Bug fixes |
| `refactor` | Restructuring without behavior changes |
| `docs` | Documentation-only |
| `test` | Adding or updating tests |
| `ci` | CI/CD pipeline |
| `chore` | Dependencies, tooling, build tweaks |

### Rules

- Sentence-style capitalization after the colon on the subject line.
- Subject under ~72 characters.
- Blank line between subject and body.
- Body: short prose (1–3 sentences, wrap ~72 chars) covering **what** changed
  and **why**. Do not paste a file list or dump the full diff.
- One logical change per commit (atomic). Squash fixup commits before
  requesting review.
- Do not invent `Signed-off-by` in drafted messages (authors add that with
  `git commit -s` if needed).

## Examples

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
