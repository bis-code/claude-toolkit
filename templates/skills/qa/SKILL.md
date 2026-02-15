---
name: qa
description: "Three-phase QA orchestrator: parallel scan, interactive triage, guided fix."
---

# /qa

Three-phase QA orchestrator that scans the codebase for quality issues using parallel subagents, presents findings for interactive triage, and applies guided fixes with individual commits.

Replaces the old `qa.sh` bash loop with an interactive, token-efficient approach that runs inside the user's session.

## Arguments: $ARGUMENTS

- `--scope SCOPE` — restrict scan to a directory/module (monorepo support)
- `--focus TEXT` — boost severity of findings matching this focus area
- `--scan-only` — report all findings, fix nothing
- `--auto` — fix CRITICAL+HIGH, create issues for MEDIUM, skip LOW
- No args — full scan with interactive triage

## Step 1: Initialize

### Parse arguments

Read `$ARGUMENTS` and determine the mode:
- Default: full scan + interactive triage
- `--scan-only`: scan and report, no fixes
- `--auto`: automated triage (no approval gates)

### Project configuration

- Read `.claude-toolkit.json` for:
  - `qa.scanCategories` — which scan dimensions to run
  - `commands.test` — test command
  - `commands.lint` — lint command
- Read `CLAUDE.md` for project conventions
- Scan `.claude/agents/` for domain-specific agent files (filenames map to domains)

### Scope resolution

If `--scope` is set:
1. Check if it matches a directory name in the project root
2. Check pnpm/npm/turbo workspace definitions for package name resolution
3. Set `SCOPE_DIR` to the resolved path — all scan agents will be restricted to this directory

## Step 2: Phase 1 — Autonomous Scan

Launch ALL scan agents in parallel via the Task tool. Each agent must return concise findings (< 200 words).

| Dimension | Agent (`subagent_type`) | Prompt Focus |
|-----------|------------------------|--------------|
| Dead code & unused exports | `refactor-cleaner` | Find unused exports, dead code, duplicate logic |
| Code quality & patterns | `code-reviewer` | Review code quality, anti-patterns, error handling |
| Security patterns | `security-reviewer` | OWASP top 10, auth bypass, data exposure |
| Test coverage gaps | `planner` | Identify untested code paths and missing test files |
| Documentation freshness | `doc-updater` | Check if docs match current code (read-only assessment) |
| Incomplete features | `planner` | Find TODOs, FIXMEs, partial implementations |
| Build health | `build-error-resolver` | Check for build/type errors, dependency issues |

**Domain agent discovery**: For each `.md` file in `.claude/agents/`, if the filename (minus extension) matches a relevant domain, also spawn that agent with a domain-specific scan prompt.

**Direct commands**: Also run test and lint commands directly via Bash tool (in parallel with agents):
- Test: run the configured test command
- Lint: run the configured lint command

If `--scope` is set, pass the scope directory to each agent and command.

Each Task call should include this instruction:
> "Scan the codebase for <dimension>. Return findings as a list: one line per finding with severity (CRITICAL/HIGH/MEDIUM/LOW) and a brief description. Maximum 200 words. Scope: <scope or 'entire project'>."

## Step 3: Phase 2 — Interactive Triage

### Merge findings

Collect all agent results and test/lint output. Merge into a unified findings table with columns:

| # | Severity | Category | Finding | Source |
|---|----------|----------|---------|--------|

### Severity definitions

| Severity | Criteria | Examples |
|----------|----------|---------|
| CRITICAL | Security vulnerabilities, data loss risk, broken production | SQL injection, auth bypass, data corruption |
| HIGH | Failing tests, missing auth checks, broken build | Test failures, unhandled errors, build errors |
| MEDIUM | Code quality gaps, coverage holes, lint violations | Missing tests, code smells, outdated deps |
| LOW | Style nits, doc staleness, minor TODOs | Formatting, stale comments, trivial TODOs |

### Focus boost

If `--focus` is set, boost the severity of any finding whose description matches the focus text (LOW→MEDIUM, MEDIUM→HIGH).

### Triage gate

**If `--scan-only`**: Print the full findings table. Done — skip Phase 3.

**If `--auto`**: Automatically triage:
- CRITICAL + HIGH → mark for fix
- MEDIUM → mark for GitHub issue creation
- LOW → skip

**Otherwise (interactive)**: Present the findings table to the user using `AskUserQuestion`. For each finding (or group), ask:
- **Fix** — apply a guided fix
- **Issue** — create a GitHub issue with label `claude-ready`
- **Skip** — ignore this finding

## Step 4: Phase 3 — Guided Fix

### Branch isolation

Before applying any fixes, create a dedicated branch to keep changes off the current branch:

1. Record the current branch name: `ORIGINAL_BRANCH=$(git branch --show-current)`
2. Create and switch to a QA branch: `git checkout -b qa/<date>` (e.g., `qa/2026-02-15`)
3. If the branch already exists (re-run), switch to it: `git checkout qa/<date>`

This ensures all fix commits land on a disposable branch. The user can review, cherry-pick, or merge at their discretion.

If not in a git repo, skip branch creation and apply fixes in-place.

### Apply fixes

For each finding marked "Fix", spawn the appropriate agent:

| Finding Category | Agent (`subagent_type`) |
|-----------------|------------------------|
| Security vulnerability | `security-reviewer` |
| Dead code / unused exports | `refactor-cleaner` |
| Test coverage gap | `tdd-guide` |
| Build / type error | `build-error-resolver` |
| Code quality issue | `code-reviewer` |
| Documentation staleness | `doc-updater` |

Each fix agent receives:
> "Fix the following issue: <finding description>. Apply the minimal fix. Run tests after fixing. If tests pass, stage and commit with: `fix(qa): <description>`. If tests fail, revert and report."

For each finding marked "Issue":
```bash
gh issue create --title "<finding summary>" --body "<details>" --label "claude-ready" --label "from-qa-auto"
```

Each fix follows the cycle: **apply → test → commit individually**.

## Step 5: Completion

Print a summary table:

```
QA Summary
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Fixed:   N items
  Issues:  N created
  Skipped: N items
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Details:
  # | Action  | Finding
  1 | FIXED   | Missing null check in auth handler
  2 | ISSUE   | Test coverage gap in payment module (#123)
  3 | SKIPPED | Formatting nit in README
```

If all tests pass and no CRITICAL/HIGH findings remain, report success. Otherwise, note remaining items.

### Write qa-report.json

Write a `qa-report.json` file to the project root with the full run results. This file persists across sessions and is useful for CI pipelines and re-runs.

```json
{
  "date": "2026-02-15T14:30:00Z",
  "scope": "all",
  "focus": null,
  "mode": "interactive",
  "branch": "qa/2026-02-15",
  "findings": [
    {
      "id": 1,
      "severity": "HIGH",
      "category": "security",
      "description": "Missing null check in auth handler",
      "source": "security-reviewer",
      "action": "fixed",
      "commitHash": "abc1234"
    },
    {
      "id": 2,
      "severity": "MEDIUM",
      "category": "test-coverage",
      "description": "Test coverage gap in payment module",
      "source": "planner",
      "action": "issue",
      "issueUrl": "https://github.com/org/repo/issues/123"
    },
    {
      "id": 3,
      "severity": "LOW",
      "category": "docs",
      "description": "Formatting nit in README",
      "source": "doc-updater",
      "action": "skipped"
    }
  ],
  "summary": {
    "total": 3,
    "fixed": 1,
    "issued": 1,
    "skipped": 1
  }
}
```

If `qa-report.json` already exists, overwrite it — each run produces a fresh report.

After the summary, if a QA branch was created:
1. Ask the user: **merge into original branch**, **keep as separate branch**, or **create a PR**.
2. If merge: `git checkout <ORIGINAL_BRANCH> && git merge qa/<date>`
3. If PR: `gh pr create --base <ORIGINAL_BRANCH> --head qa/<date>`
4. If keep: leave the branch as-is and switch back to `ORIGINAL_BRANCH`.

## Error Handling

- **Agent failures**: If a Task agent fails, log the error and skip that dimension. Do not retry — the user can re-run `/qa` to try again.
- **Test failures after fix**: Revert the fix (`git checkout -- .`) and create a GitHub issue instead.
- **No findings**: Report a clean bill of health. No Phase 3 needed.
