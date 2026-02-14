---
name: plan
description: "Analyze requirements and produce a structured implementation plan."
---

# /plan

Spawns the `planner` agent to analyze a task and produce an actionable implementation plan before writing any code.

## Steps

1. **Gather context** — understand what needs to be planned:
   - If arguments are provided (issue number, feature description), use those
   - Otherwise, ask the user what they want to plan
   - Collect relevant context: affected files, existing patterns, related code

2. **Spawn the agent** — use the Task tool:
   ```
   Task tool with subagent_type="planner"
   ```
   Pass in the prompt:
   - The requirement or task description
   - Relevant codebase context (file structure, existing patterns)
   - Any constraints or preferences the user mentioned

3. **Present the plan** — relay the agent's structured output:
   - Requirement analysis and restated problem
   - Ordered implementation steps with file paths
   - Test strategy (unit, integration, E2E)
   - Risk assessment and assumptions
   - The one decision most likely to be challenged in review

4. **Offer follow-up actions**:
   - "Approve and implement?" — proceed to coding with the plan as guide
   - "Refine?" — adjust specific sections of the plan
   - "Create issue?" — turn the plan into a GitHub issue
