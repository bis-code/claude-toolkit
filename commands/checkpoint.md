---
description: Save current progress state (Tasks, progress.txt, deep-think checkpoint)
---

# /checkpoint — Save Progress State

Persist the current state of work so it survives context compactions and session restarts.

## Arguments: $ARGUMENTS

Parse `$ARGUMENTS` for an optional message: `/checkpoint "completed auth module"`

## Step 1: Determine Active Context

Check for active work indicators:
- `prd.json` — Ralph feature build in progress
- `tools/qa/qa-state.json` — QA session in progress
- Git branch name and recent commits
- Any open Tasks

## Step 2: Update Tasks

Use `TaskCreate` or `TaskUpdate` to record the current state:

```
Status: <in-progress|blocked|paused>
Current story: <story number if ralph is active>
Last completed: <brief description>
Next step: <what should happen next>
Blockers: <any known blockers>
```

If a Task already exists for this work, update it. Do not create duplicates.

## Step 3: Append to progress.txt

Append a timestamped entry to `progress.txt` in the project root:

```
## Checkpoint — YYYY-MM-DD HH:MM
Message: <user message or auto-generated summary>
Branch: <current branch>
Last commit: <short hash + message>
Open items: <count of remaining work>
---
```

Create `progress.txt` if it does not exist.

## Step 4: Deep-Think Checkpoint (if active)

If the deep-think MCP server is available and a reasoning session is active:

```
checkpoint(operation="save", label="checkpoint-<timestamp>")
```

This preserves the reasoning chain for the next session.

## Step 5: Confirm

Report what was saved:
- Task status
- Progress entry count
- Deep-think checkpoint (if saved)
- Suggested next command to resume work
