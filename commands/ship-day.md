---
description: End-of-day squash commits and create PR
---

# /ship-day — Ship Your Day's Work

Review all commits on the current branch, squash into logical units, and create a pull request with a comprehensive summary.

## Arguments: $ARGUMENTS

Parse `$ARGUMENTS` for options:
- `/ship-day` — default: squash and create PR against main
- `/ship-day --base develop` — PR against a different base branch
- `/ship-day --no-squash` — create PR without squashing
- `/ship-day --draft` — create as draft PR

## Step 1: Verify Branch State

```bash
git status
git log --oneline main..HEAD
```

Checks:
- Must NOT be on main/master (refuse to run if so)
- Must have at least one commit ahead of base
- Working tree should be clean (warn if dirty, suggest stash or commit)

## Step 2: Review Commits

List all commits on the branch with their diffs:

```bash
git log --oneline --stat main..HEAD
```

Categorize commits by type:
- Feature work
- Bug fixes
- Test additions
- Refactoring
- Documentation

## Step 3: Squash into Logical Units

Unless `--no-squash` is passed, propose squash groups:

```
Squash Plan:
  Group 1: feat(auth): add JWT token refresh
    - abc1234 add token refresh endpoint
    - def5678 add refresh token tests
    - ghi9012 fix refresh token expiry edge case

  Group 2: fix(api): correct pagination offset
    - jkl3456 fix off-by-one in pagination

Proceed? [Y/n/edit]
```

Wait for approval, then perform the squash using interactive rebase.

## Step 4: Push and Create PR

```bash
git push -u origin <branch-name>
```

Create the PR with a structured body:

```bash
gh pr create --title "<conventional commit title>" --body "..."
```

PR body includes:
- Summary of changes (bullet points)
- Test plan (what was tested and how)
- Related issues (closes #N)
- Screenshot links if UI changes

## Step 5: Confirm

```
PR Created
━━━━━━━━━━
URL:     https://github.com/owner/repo/pull/N
Title:   feat(auth): add JWT token refresh
Commits: 2 (squashed from 5)
Base:    main
Status:  Ready for review

Next: Wait for CI, then merge.
```
