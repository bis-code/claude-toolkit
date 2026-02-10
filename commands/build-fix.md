---
description: Fix build/type errors with minimal diffs
---

# /build-fix — Fix Build Errors

Run the build, parse errors, and apply minimal fixes until the build is clean. Prioritize smallest possible changes.

## Arguments: $ARGUMENTS

Parse `$ARGUMENTS` for options:
- `/build-fix` — detect and run the project's build command
- `/build-fix --cmd "npm run build"` — use a specific build command
- `/build-fix --dry-run` — show what would be fixed without applying

## Step 1: Determine Build Command

Check in order:
1. `$ARGUMENTS` for explicit `--cmd`
2. `.claude-toolkit.json` for `commands.build`
3. `Makefile` for a `build` target
4. `package.json` for a `build` script
5. Language-specific defaults: `go build ./...`, `cargo build`, `dotnet build`

If no build command can be determined, ask the user.

## Step 2: Run Build

Execute the build command and capture all output (stdout + stderr).

If the build succeeds on the first run:
```
Build is already clean. No fixes needed.
```

## Step 3: Parse Errors

Extract structured error information:
- **File path** and **line number**
- **Error code** (e.g., TS2345, E0308)
- **Error message**
- **Related context** (expected type, got type)

Group errors by file to minimize edit passes.

## Step 4: Apply Minimal Fixes

For each error, apply the smallest possible fix:

| Error Type | Minimal Fix |
|------------|-------------|
| Missing import | Add the import |
| Type mismatch | Correct the type annotation |
| Unused variable | Remove or prefix with underscore |
| Missing return | Add return statement |
| Undefined reference | Check for typo, add import or declaration |

**Rules:**
- One fix at a time, re-run build after each batch per file
- Never refactor working code — only fix what is broken
- If a fix requires understanding business logic, flag it and skip
- Maximum 10 fix iterations before stopping and reporting remaining errors

## Step 5: Re-run and Verify

After each round of fixes:
1. Re-run the build command
2. If new errors appear, add them to the queue
3. If error count is not decreasing, stop and report

## Step 6: Report

```
Build Fix Results
━━━━━━━━━━━━━━━━━
Build command: <command>
Initial errors: N
Fixes applied:  M
Remaining:      K

Fixed:
  src/api/handler.ts:42 — Added missing import for UserService
  src/types/index.ts:15 — Corrected return type to Promise<void>

Remaining (manual fix needed):
  src/core/engine.ts:88 — Requires business logic decision
```
