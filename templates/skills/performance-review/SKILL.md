---
name: performance-review
description: "Review code changes for performance anti-patterns — N+1 queries, unbounded results, blocking ops."
---

# /performance-review

Spawns the `performance-reviewer` agent to detect performance anti-patterns in your code changes.

## Steps

1. **Gather context** — detect the scope of changes to review:
   - If arguments are provided (file paths or PR number), use those as scope
   - Otherwise, use `git diff` to determine what changed
   - Identify files touching database queries, API endpoints, or data processing

2. **Spawn the agent** — use the Task tool:
   ```
   Task tool with subagent_type="performance-reviewer"
   ```
   Pass in the prompt:
   - The diff output and list of changed files
   - The project's database ORM and framework context
   - Any performance budgets or constraints from project rules

3. **Present findings** — relay the agent's performance report:
   - Group by severity: CRITICAL > WARNING
   - Include specific code locations and estimated impact
   - Show recommended fixes with before/after examples

4. **Offer follow-up actions**:
   - "Fix critical issues?" — apply recommended performance fixes
   - "Add benchmarks?" — create performance tests for flagged code paths
   - "Review again?" — re-run after changes
