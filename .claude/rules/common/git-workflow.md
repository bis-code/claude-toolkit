# Git Workflow

Rules for clean, traceable version control.

## Commit Format

```
<type>(<scope>): <description>

[optional body]

Co-Authored-By: Claude <noreply@anthropic.com>
```

### Types

| Type | Use When |
|------|----------|
| `feat` | New feature or capability |
| `fix` | Bug fix |
| `refactor` | Code restructuring without behavior change |
| `test` | Adding or updating tests only |
| `docs` | Documentation changes |
| `chore` | Build, CI, dependency updates |

### Scope

Use the module, directory, or domain the change affects: `auth`, `api`, `ui`, `billing`.

## Commit Rules

- One logical change per commit — if the diff needs two explanations, it needs two commits
- Tests MUST pass before committing — no exceptions
- No WIP commits — every commit should leave the codebase in a working state
- Never commit generated files, secrets, or environment configs
- Stage specific files (`git add <file>`) — never use `git add -A` or `git add .`

## Branch Naming

- Feature branches: `issue/<number>-<short-slug>` (e.g., `issue/42-payment-modal`)
- Fresh branch from the default branch for each issue
- Never push directly to main/master

## Pull Requests

- One PR per issue — do not batch unrelated changes
- PR title matches conventional commit format
- PR body includes: summary, test plan, and linked issue
- CI must pass before merge

## Squash Policy

- Squash merge feature branches into main — keeps history clean
- Rebase is acceptable for small, linear changes
- Never force-push to shared branches
