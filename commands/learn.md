---
description: Extract patterns and insights from the current session
---

# /learn — Extract Session Patterns

Review the current conversation and extract reusable patterns, anti-patterns, and insights. Persist them for future sessions.

## Arguments: $ARGUMENTS

Parse `$ARGUMENTS` for options:
- `/learn` — extract from current session
- `/learn --topic "error handling"` — focus extraction on a specific topic
- `/learn --format notes` — output as session notes (default)
- `/learn --format rule` — output as a Claude rule suggestion

## Step 1: Review Conversation

Analyze the current session for:

### Patterns Discovered
- Code patterns that worked well
- Architectural decisions made and their reasoning
- Debugging approaches that were effective
- Testing strategies applied

### Anti-Patterns Encountered
- Approaches that failed and why
- Common mistakes repeated in this session
- Assumptions that turned out to be wrong

### Codebase Insights
- Module relationships discovered
- Hidden dependencies found
- Performance characteristics observed
- Configuration quirks identified

## Step 2: Categorize Findings

Group findings into:

| Category | Description |
|----------|-------------|
| Architecture | System design, module boundaries, data flow |
| Testing | Test strategies, mocking approaches, coverage gaps |
| Debugging | Root cause analysis techniques, common failure modes |
| Workflow | Process improvements, tooling discoveries |
| Domain | Business logic insights, edge cases |

## Step 3: Persist Insights

### Option A: Append to progress.txt

Add a learning section to the project's `progress.txt`:

```
## Learnings — YYYY-MM-DD HH:MM
Topic: <topic or "general session review">

### Patterns
- <pattern 1>
- <pattern 2>

### Anti-Patterns
- <anti-pattern 1>

### Codebase Notes
- <insight 1>
---
```

### Option B: Suggest Rule File

If the insights are broadly applicable, suggest a new `.claude/rules/` file:

```markdown
# <Topic> Patterns

## Do
- <pattern>

## Don't
- <anti-pattern>
```

Present the suggestion — do not create the rule file without user approval.

## Step 4: Summary

Report what was learned and where it was saved:
- Count of patterns, anti-patterns, and insights extracted
- File(s) updated
- Suggested follow-up actions
