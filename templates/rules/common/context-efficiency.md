# Context Efficiency

Context window is a finite resource. Wasting it on redundant reads, over-loaded rules, or unnecessary compaction degrades response quality for the rest of the session.

## Token Budget Awareness

| Session state | Action |
|---------------|--------|
| Early (<20% used) | Read freely, explore broadly |
| Mid (20-60% used) | Target reads only — confirm file path before reading |
| Late (60-80% used) | Stop exploratory reads; work only from already-loaded context |
| Critical (>80% used) | Trigger `/compact` before the next major task |

Avoid reading files "just in case." Read a file only when you have a specific question it answers.

## Lazy Loading Triggers

Load a file or rule only when a specific trigger is met:

| Trigger | Load |
|---------|------|
| Writing a migration | `database-migrations` skill or `database-reviewer` agent |
| Handling a payment webhook | `billing-security` strategy |
| Changing an auth flow | `security.md` rules + `security-reviewer` agent |
| Designing a new API endpoint | `patterns.md` (Repository + Envelope patterns) |
| Writing E2E tests | `e2e-runner` agent |

Do not pre-load all rules at session start. Load them when their trigger appears.

## Iterative Retrieval

When searching for code, use at most 3 cycles before stopping:

1. Semantic search (`leann_search`) — broad, finds relevant files
2. Targeted Grep — narrow, confirms the exact symbol or pattern
3. Focused Read — read only the confirmed file, only the relevant section

If 3 cycles yield no result, ask the user for the file path rather than continuing to search.

## Duplicate Rule Detection

Before adding a new rule to a project, check for overlap:

```bash
# Find rules that mention the same topic
grep -r "pagination" .claude/rules/
grep -r "error handling" .claude/rules/
```

Rules that conflict produce unpredictable behavior. Rules that duplicate each other waste tokens on every load. When two rules overlap, merge them or remove the less specific one.

## Rule Scope Hierarchy

Rules load from three scopes. Lower scopes override higher:

```
Global (~/.claude/rules/)        → applies to all projects
  Project (.claude/rules/)       → applies to this repo
    Agent (agent system prompt)  → applies to one agent session
```

Put rules at the lowest scope where they apply. A rule about PostgreSQL belongs at the project level, not global. A rule about a specific agent's output format belongs in the agent file.

## When to Compact

Compact the context when:
- A major task is complete and the next task is unrelated
- The context exceeds 70% and a new feature implementation is starting
- You notice the agent repeating information it already stated (a sign of degraded context quality)

Before compacting, save any critical state (plan decisions, open questions, file paths) to a scratchpad file so it survives the compaction.

## Anti-Patterns

- Reading the same file twice in the same session — check what is already loaded
- Loading all `.claude/rules/` files at session start — lazy load by trigger
- Asking for large files when only a function signature is needed — use Grep with context lines
- Running broad Glob searches that return 50+ files — add path constraints and `head_limit`
