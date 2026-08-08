# Pull Requests and Branch Naming

Guidelines for creating branches and writing pull requests in kc-utils.

## Commits vs pull requests

A **pull request** is the review unit: care about the **branch name**, **PR
title**, and **PR body**. A **commit** is the git history unit: care about the
subject and message body (see [commits.md](commits.md)).

When a change is one commit and one PR, the PR **title** often matches the
commit **subject**. The bodies still differ: the PR body uses What / How /
Testing for reviewers; the commit body is short what/why prose. Commits on
the branch follow [commits.md](commits.md).

## Branch Naming

Use the format `<type>/<short-description>` with lowercase words separated by
hyphens. The type prefix indicates the nature of the change:

| Prefix | Use For |
|--------|---------|
| `feat/` | New features or capabilities |
| `fix/` | Bug fixes |
| `refactor/` | Code restructuring without behavior changes |
| `docs/` | Documentation-only changes |
| `test/` | Adding or updating tests |
| `ci/` | CI/CD pipeline changes |
| `chore/` | Dependency updates, tooling, build tweaks |

### Examples

```
feat/arm64-virtio-injection
fix/guestfs-luks-unlock-timeout
refactor/extract-registry-helpers
docs/add-privilege-model
test/e2e-windows-uefi
ci/add-disk-image-job
chore/bump-govmomi
```

### Rules

- Keep it short (3–5 words max after the prefix).
- Use hyphens, not underscores or slashes beyond the type prefix.
- Describe *what* the branch does, not a ticket number alone.
- If tied to an issue, you may append it: `fix/luks-timeout-42`.

## Writing a Pull Request

### Title

Use `<type>: <Description>` with a colon separator and sentence-style
capitalization (same shape as a commit subject, not a branch name):

```
feat: Add ARM64 virtio driver injection
fix: Resolve LUKS unlock timeout in guestfs mode
refactor: Extract shared registry helpers to pkg/common
docs: Add privilege model documentation
```

Keep the title under ~72 characters.

### Body Structure

```markdown
## What

One or two sentences explaining what the PR does and why.

## How

Brief description of the approach — which packages/blocks were changed
and the key design decisions.

## Testing

- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] `make test-e2e-container` passes
- [ ] Manual verification steps (if applicable)

## Notes (optional)

Anything reviewers should know: migration steps, known limitations,
follow-up work planned.
```

### Good PR Practices

- **One concern per PR.** Don't mix a bug fix with an unrelated refactor.
- **Keep it small.** Aim for <400 lines changed. Split large features into
  stacked PRs (base branch → feature branch → sub-branches).
- **Reference issues.** Use `Fixes #N` or `Relates to #N` in the body to
  link to tracked issues.
- **Update docs.** If the change affects CLI flags, pipeline behavior, or
  architecture, update the corresponding `docs/` or `community/` file.
- **Run CI locally first.** `make lint`, `make test`, and
  `make test-e2e-container` before pushing.
- **Self-review before requesting review.** Read your own diff — catch
  leftover debug prints, TODO comments, or unintended file changes.
