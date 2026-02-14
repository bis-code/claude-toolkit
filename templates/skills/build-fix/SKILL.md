---
name: build-fix
description: "Diagnose and fix build errors with minimal changes."
---

# /build-fix

Spawns the `build-error-resolver` agent to diagnose build failures and apply targeted fixes.

## Steps

1. **Gather context** — determine the build command and current error state:
   - Check `.claude-toolkit.json` for `commands.build`
   - Fall back to auto-detection: `package.json` scripts, `Makefile` targets, language defaults
   - If arguments are provided (e.g., specific error output), pass those directly

2. **Spawn the agent** — use the Task tool:
   ```
   Task tool with subagent_type="build-error-resolver"
   ```
   Pass in the prompt:
   - The build command to run
   - Any captured error output (if available)
   - The project's language and framework context

3. **Present findings** — relay the agent's fix report:
   - Show initial error count vs final error count
   - List each fix applied with file path and description
   - Flag any remaining errors that need manual review

4. **Offer follow-up actions**:
   - "Run tests?" — verify fixes did not break test suite
   - "Review changes?" — spawn code-reviewer on the fixes
