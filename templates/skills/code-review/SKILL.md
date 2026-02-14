---
name: code-review
description: "Review code changes for correctness, security, test coverage, and style."
---

# /code-review

Spawns the `code-reviewer` agent to analyze your current changes and provide structured feedback.

## Steps

1. **Gather context** — detect the scope of changes to review:
   - If arguments are provided (file paths or PR number), use those as scope
   - Otherwise, use `git diff` (unstaged) and `git diff --cached` (staged) to determine what changed
   - Run `git log --oneline -5` for recent commit context

2. **Spawn the agent** — use the Task tool:
   ```
   Task tool with subagent_type="code-reviewer"
   ```
   Pass in the prompt:
   - The diff output (or file contents if reviewing specific files)
   - The list of changed files
   - Any project conventions from `.claude/rules/` that apply

3. **Present findings** — relay the agent's review to the user:
   - Group by severity: CRITICAL > WARNING > NIT > PRAISE
   - Include file paths and line numbers
   - Show suggested fixes for each finding

4. **Offer follow-up actions**:
   - "Fix critical issues?" — apply suggested fixes
   - "Run tests?" — verify fixes do not break anything
   - "Review again?" — re-run after changes
