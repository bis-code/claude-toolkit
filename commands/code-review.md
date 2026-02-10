---
description: Review recent changes for quality, correctness, and best practices
---

# /code-review — Code Quality Review

Review recent changes and provide structured feedback on quality, correctness, security, and test coverage.

## Arguments: $ARGUMENTS

Parse `$ARGUMENTS` for scope:
- `/code-review` — review unstaged + staged changes (git diff + git diff --cached)
- `/code-review --branch` — review all commits on current branch vs base
- `/code-review --last N` — review the last N commits
- `/code-review --file path/to/file` — review a specific file

## Step 1: Collect Changes

Based on scope, gather the diff:

```bash
# Default: working tree changes
git diff
git diff --cached

# Branch mode
git log --oneline main..HEAD
git diff main...HEAD

# Last N commits
git diff HEAD~N..HEAD
```

## Step 2: Analyze for Common Issues

Review the diff against these categories:

### Correctness
- Logic errors, off-by-one, null/undefined handling
- Missing return statements, unreachable code
- Incorrect type assertions or casts

### Error Handling
- Unhandled promise rejections or panics
- Missing error propagation
- Silent failures (empty catch blocks)

### Test Coverage
- Are there tests for the new/changed code?
- Do tests cover both happy and unhappy paths?
- Are edge cases tested?

### Security
- Hardcoded secrets or credentials
- User input used without validation
- SQL injection or XSS vectors
- Missing authentication/authorization checks

### Style and Maintainability
- Dead code, unused imports
- Overly complex functions (high cyclomatic complexity)
- Poor naming, missing documentation on public APIs

## Step 3: Rate and Summarize

Provide a structured review:

```
Code Review Summary
━━━━━━━━━━━━━━━━━━
Files reviewed: N
Lines changed:  +X / -Y

Issues Found:
  Critical: 0
  Warning:  2
  Nit:      3

Details:
  [WARN] src/auth.ts:42 — Missing error handling on token refresh
  [NIT]  src/utils.ts:15 — Unused import: lodash
  ...

Overall: Approve / Request Changes / Needs Discussion
```

## Step 4: Suggest Improvements

For each issue found, provide a concrete fix suggestion with the exact code change needed. Group by priority.
