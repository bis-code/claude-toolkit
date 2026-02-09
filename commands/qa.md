---
description: Run autonomous QA agent (scan, fix, report)
---

# /qa — Autonomous QA Agent

Run a continuous QA loop that scans, fixes, and reports until the codebase is clean.
Uses the Ralph pattern: spawns fresh Claude instances per iteration with persistent state.
Runs in a git worktree from the default branch — does not affect your current branch.

## Usage

Run the orchestrator from your terminal (recommended):

```bash
./tools/qa/qa.sh                    # Full scan, up to 20 iterations
./tools/qa/qa.sh --max-loops 5      # Limit iterations
./tools/qa/qa.sh --scope api        # Backend only
./tools/qa/qa.sh --scope web        # Frontend only
./tools/qa/qa.sh --scan-only        # Report only, no fixes
./tools/qa/qa.sh --branch develop   # Worktree from develop instead of main
```

Or run within this Claude session:

Execute the QA orchestrator script. Pass through any arguments from $ARGUMENTS:

```bash
./tools/qa/qa.sh $ARGUMENTS
```

## What It Does

Each iteration (fresh Claude instance in a worktree):
1. **SCAN** — Run tests, lint, security checks (one category per iteration)
2. **REASON** — Analyze findings (deep-think for complex issues)
3. **TRIAGE** — Fix-now (< 30 lines) vs report-as-issue
4. **FIX** — Apply one small fix, re-run tests
5. **REPORT** — Create GitHub issue for large findings
6. **UPDATE** — Persist state for next iteration

Stops when: all tests pass + lint clean + no new findings + all reported.

## Configuration

QA behavior is configured via `.claude-toolkit.json`:
- `qa.scanCategories` — which categories to scan
- `qa.maxFixLines` — max lines for direct fixes (default: 30)
- `qa.worktreeFromBranch` — branch to create worktree from (default: main)
- `commands.test` — test command to run
- `commands.lint` — lint command to run

## State Files (in worktree)

- `tools/qa/qa-state.json` — findings tracker
- `tools/qa/qa-progress.txt` — cumulative log with patterns
