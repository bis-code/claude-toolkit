---
description: Dead code removal and consolidation with safe, incremental changes
---

# /refactor-clean — Dead Code Removal & Consolidation

Find and remove unused code, consolidate duplicates, and clean up the codebase. Every removal is verified safe before applying.

## Arguments: $ARGUMENTS

Parse `$ARGUMENTS` for scope:
- `/refactor-clean` — scan entire project
- `/refactor-clean --dir src/services` — scan a specific directory
- `/refactor-clean --dry-run` — report findings without applying changes
- `/refactor-clean --aggressive` — include less certain removals (still verified)

## Step 1: Find Unused Exports

Search for exported functions, classes, types, and constants that have no importers:

1. List all exports using Grep: `export (function|const|class|type|interface)`
2. For each export, search for imports or usages across the codebase
3. Mark as unused if no references found outside the defining file

**Exclusions:**
- Entry points (main files, route handlers, CLI commands)
- Public API surface (index.ts barrel exports)
- Test helpers imported only in test files (flag but do not remove)

## Step 2: Find Unused Imports

Search for imports that are not referenced in the file body:

1. List all import statements
2. Check if the imported name appears in the file body
3. Mark as unused if no references found

## Step 3: Identify Duplicate Logic

Look for functions with similar signatures and bodies:

1. Find functions with identical parameter types
2. Compare function bodies for structural similarity
3. Flag duplicates that could be consolidated into a shared utility

Report duplicates but do NOT auto-merge — suggest the consolidation.

## Step 4: Identify Dead Code Paths

Look for:
- Functions only called from other unused functions (transitive dead code)
- Unreachable branches (constant conditions, early returns)
- Commented-out code blocks (flag for removal)
- TODO/FIXME comments older than 90 days (flag, do not remove)

## Step 5: Apply Safe Removals

For each removal:
1. Verify no references exist (double-check with Grep)
2. Remove the code
3. Run the test suite to confirm nothing broke
4. If tests fail, revert the removal and flag it

**Apply in order:** unused imports first, then unused local code, then unused exports.

## Step 6: Report

```
Refactor-Clean Results
━━━━━━━━━━━━━━━━━━━━━
Files scanned:   N
Removals applied: M
Lines removed:    K

Removed:
  src/utils/legacy.ts — Entire file (0 importers)
  src/api/handler.ts  — Removed unused import: lodash
  src/types/old.ts    — Removed unused type: LegacyUser

Suggested Consolidation (manual review):
  src/services/auth.ts:validateToken + src/middleware/auth.ts:checkToken
  → Nearly identical logic, consider extracting to shared utility

Flagged (not removed):
  src/core/engine.ts:42 — TODO comment (180 days old)
  src/helpers/format.ts — Only used in tests (intentional?)
```
