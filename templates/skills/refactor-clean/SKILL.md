---
name: refactor-clean
description: "Find and safely remove dead code, unused imports, and duplicate logic."
---

# /refactor-clean

Spawns the `refactor-cleaner` agent to identify unused code and safely remove it with test verification.

## Steps

1. **Gather context** — determine the scope of cleanup:
   - If arguments are provided (file paths or module names), scope to those
   - Otherwise, scan the full `src/` or project source directory
   - Check for a working test suite (cleanup requires test verification)

2. **Spawn the agent** — use the Task tool:
   ```
   Task tool with subagent_type="refactor-cleaner"
   ```
   Pass in the prompt:
   - The scope (specific files or full project)
   - The test command from `.claude-toolkit.json` or auto-detected
   - Any known entry points or public API files to protect

3. **Present findings** — relay the agent's cleanup report:
   - List items removed (unused imports, dead functions, unreferenced files)
   - List items flagged but not removed (uncertain usage)
   - Suggested consolidations (duplicate logic that could be merged)
   - Test results after cleanup

4. **Offer follow-up actions**:
   - "Review flagged items?" — investigate uncertain items
   - "Consolidate duplicates?" — merge suggested duplicate logic
   - "Run full test suite?" — verify nothing was missed
