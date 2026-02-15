---
name: architect-review
description: "Review code changes for architectural issues — module boundaries, coupling, and dependency direction."
---

# /architect-review

Spawns the `architect-reviewer` agent to analyze your current changes at the system level — module boundaries, dependency direction, and pattern consistency.

## Steps

1. **Gather context** — detect the scope of changes to review:
   - If arguments are provided (file paths or PR number), use those as scope
   - Otherwise, use `git diff --name-only` to determine which modules are affected
   - Run `git log --oneline -5` for recent commit context

2. **Spawn the agent** — use the Task tool:
   ```
   Task tool with subagent_type="architect-reviewer"
   ```
   Pass in the prompt:
   - The diff output and list of changed files
   - The module/directory structure of the project
   - Any architectural conventions from `.claude/rules/` that apply

3. **Present findings** — relay the agent's architecture review:
   - Group by severity: VIOLATION > WARNING > SUGGESTION
   - Include module names and dependency chains
   - Show recommended fixes for boundary violations

4. **Offer follow-up actions**:
   - "Fix violations?" — restructure imports to respect boundaries
   - "Run code review too?" — spawn code-reviewer for line-level feedback
   - "Review again?" — re-run after changes
