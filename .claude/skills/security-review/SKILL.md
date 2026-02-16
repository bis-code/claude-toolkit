---
name: security-review
description: "Run an OWASP-focused security analysis on code changes."
---

# /security-review

Spawns the `security-reviewer` agent to perform a structured security analysis of your code changes.

## Steps

1. **Gather context** — detect the scope of the security review:
   - If arguments are provided (file paths or PR number), scope to those
   - Otherwise, use `git diff` to determine changed files
   - Identify files touching auth, API endpoints, input handling, or secrets

2. **Spawn the agent** — use the Task tool:
   ```
   Task tool with subagent_type="security-reviewer"
   ```
   Pass in the prompt:
   - The diff or file contents to review
   - The list of changed files with categories (auth, API, data handling)
   - The project's language and framework for context-specific checks

3. **Present findings** — relay the agent's security report:
   - Group by severity: CRITICAL > HIGH > MEDIUM > LOW
   - Include exploit scenarios for each finding
   - Show recommended fixes with code references
   - Report secrets detection and dependency audit results

4. **Offer follow-up actions**:
   - "Fix critical issues?" — apply recommended security fixes
   - "Create issues for non-critical findings?" — track as GitHub issues
   - "Run dependency audit?" — check for known CVEs in dependencies
