---
name: docs
description: "Update documentation to reflect recent code changes."
---

# /docs

Spawns the `doc-updater` agent to find and update documentation affected by recent changes.

## Steps

1. **Gather context** — detect what changed and what docs may be stale:
   - Run `git diff --name-only HEAD~1..HEAD` (or broader range if specified)
   - Identify changed function signatures, API endpoints, config keys
   - Locate documentation files: README.md, API docs, inline comments, CHANGELOG

2. **Spawn the agent** — use the Task tool:
   ```
   Task tool with subagent_type="doc-updater"
   ```
   Pass in the prompt:
   - The list of changed files and a summary of what changed
   - The commit messages for context
   - Paths to known documentation files in the project

3. **Present findings** — relay the agent's update report:
   - List each documentation file updated with a description of the change
   - Flag docs that may need updating but require human judgment
   - Note any docs that are already up to date

4. **Offer follow-up actions**:
   - "Review changes?" — show the diff of documentation updates
   - "Commit docs separately?" — stage and commit doc changes independently
