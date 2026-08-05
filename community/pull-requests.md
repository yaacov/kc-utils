# Pull Requests and Branch Naming

Guidelines for creating branches and writing pull requests in kc-utils.

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

Follow the same `<type>: <description>` format used in branch names, but with
a colon separator and sentence-style capitalization:

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

- [ ] `make check` passes
- [ ] Relevant e2e tests pass (list which ones)
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
- **Run CI locally first.** `make check` and relevant `make test-e2e*`
  before pushing.
- **Self-review before requesting review.** Read your own diff — catch
  leftover debug prints, TODO comments, or unintended file changes.

### Commit Messages Within a PR

Each commit in the branch should also follow `<type>: <description>` format.
Keep commits atomic — one logical change per commit. Squash fixup commits
before requesting review.
