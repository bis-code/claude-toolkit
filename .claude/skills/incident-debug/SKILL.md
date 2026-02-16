---
name: incident-debug
description: "Diagnose runtime issues through structured hypothesis-driven debugging."
---

# /incident-debug

Spawns the `incident-debugger` agent to systematically diagnose runtime or production issues.

## Steps

1. **Gather context** — collect information about the incident:
   - If arguments are provided (error messages, log snippets, stack traces), pass those directly
   - Otherwise, ask the user for: symptoms, reproduction steps, and when it started
   - Run `git log --oneline -20` to identify recent changes that may be related

2. **Spawn the agent** — use the Task tool:
   ```
   Task tool with subagent_type="incident-debugger"
   ```
   Pass in the prompt:
   - All symptom information (error messages, stack traces, logs)
   - Recent git changes and affected files
   - The project's language, framework, and runtime context

3. **Present findings** — relay the agent's diagnosis:
   - Show the ranked hypotheses with evidence for/against each
   - Present the confirmed root cause with the code path
   - Show the proposed fix and regression test

4. **Offer follow-up actions**:
   - "Apply fix?" — implement the proposed minimal fix
   - "Run tests?" — verify the fix and check for regressions
   - "Create issue for systemic problems?" — track broader issues as GitHub issues
