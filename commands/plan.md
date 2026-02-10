---
description: Implementation planning (analyze, propose, get approval before coding)
---

# /plan — Implementation Planning

Analyze requirements and produce a structured implementation plan. No code is written until the user approves.

## Arguments: $ARGUMENTS

Parse `$ARGUMENTS` for the feature or issue reference:
- `/plan Add user avatar upload` — plan from description
- `/plan --issue 42` — plan from GitHub issue
- `/plan` (no args) — ask the user what to plan

## Step 1: Gather Requirements

If `--issue N` is provided:
```bash
gh issue view N --json number,title,body,labels,milestone
```

Otherwise, use the provided description or ask the user.

## Step 2: Analyze Affected Code

Search the codebase to identify:
- **Files to modify** — list each with the expected change type (new, modify, delete)
- **Files to create** — new modules, tests, migrations
- **Dependencies** — other modules that depend on or are depended upon
- **Database changes** — new tables, columns, migrations needed

Use Glob and Grep to locate relevant code. Do not guess file paths.

## Step 3: Propose Test Strategy

Based on the change scope, recommend:

| Layer | Required | Reason |
|-------|----------|--------|
| Unit tests | Yes/No | ... |
| Integration tests | Yes/No | ... |
| E2E tests | Yes/No | ... |

List specific test cases for each layer.

## Step 4: Identify Risks and Assumptions

Document:
- **Assumptions** — what are we assuming about the codebase or requirements?
- **Risks** — what could go wrong? What are the edge cases?
- **Tech Lead challenge** — what would a senior reviewer push back on?

## Step 5: Present Plan for Approval

Format the plan as:

```
Implementation Plan: <title>
━━━━━━━━━━━━━━━━━━━━━━━━━━━
Scope: <N files modified, M files created>
Estimated steps: <count>

1. <step> — <files affected>
2. <step> — <files affected>
...

Test strategy: <summary>
Risks: <summary>

Proceed? [Y/n/edit]
```

Do NOT write any code until the user confirms.
