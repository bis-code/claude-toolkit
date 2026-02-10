---
description: Run verification loop (tests, lint, type-check, report results)
---

# /verify — Verification Loop

Run the full verification suite: tests, lint, and type-check. Report a pass/fail summary.

## Arguments: $ARGUMENTS

## Step 1: Read Project Configuration

Load `.claude-toolkit.json` from the project root to get configured commands:

```json
{
  "commands": {
    "test": "...",
    "lint": "...",
    "typecheck": "..."
  }
}
```

If `.claude-toolkit.json` does not exist, fall back to common conventions:
- **Test**: `npm test`, `go test ./...`, `pytest`, `cargo test`
- **Lint**: `npm run lint`, `golangci-lint run`, `ruff check .`, `cargo clippy`
- **Type-check**: `npx tsc --noEmit`, `mypy .`

## Step 2: Run Test Suite

Execute the test command. Capture stdout and stderr.

- If tests fail, collect the failure output for the summary
- Do NOT attempt to fix failures — only report them
- If no test command is configured, report "No test command configured" as a warning

## Step 3: Run Linter

Execute the lint command. Capture output.

- Count the number of lint warnings and errors separately
- If no lint command is configured, skip and note in summary

## Step 4: Run Type Checker (if available)

Execute the type-check command if configured or detectable.

- For TypeScript projects: `npx tsc --noEmit`
- For Python with mypy: `mypy .`
- If not applicable, skip

## Step 5: Report Summary

Present a clear pass/fail summary:

```
Verification Results
━━━━━━━━━━━━━━━━━━━
Tests:      PASS (42 passed, 0 failed)
Lint:       WARN (0 errors, 3 warnings)
Type-check: PASS (no errors)
━━━━━━━━━━━━━━━━━━━
Overall:    PASS with warnings
```

If any step failed, list the specific failures with file locations so the user can act on them.
