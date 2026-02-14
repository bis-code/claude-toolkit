---
name: ralph
description: "Interactive feature builder with subagent delegation and approval gates."
---

# /ralph

Orchestrates feature development across multiple user stories. Spawns specialized subagents for exploration, planning, implementation, and review — with human approval gates between phases.

Replaces the old `ralph.sh` bash loop with an interactive, token-efficient approach that runs inside the user's session.

## Arguments: $ARGUMENTS

- `--issues N,N` — build from specific GitHub issues
- `--label X` — build from issues with that label
- `--prd path/to/prd.json` — use an existing PRD
- `--auto` — auto-approve gates (CI mode, skip approval prompts)
- No args — ask the user what to build

## Step 1: Initialize

### Parse arguments

Read `$ARGUMENTS` and determine the input mode.

### Generate or load PRD

**If no `prd.json` exists** (and `--prd` not provided):

1. Fetch issues via `gh` CLI:
   - `--issues N,N` → `gh issue view <N> --json number,title,body,labels,milestone`
   - `--label X` → `gh issue list --label "X" --state open --json number,title,body,labels`
   - No args → ask the user which issues or what to build
2. Decompose issues into user stories with strategy mapping:

   | Story type | Strategy |
   |------------|----------|
   | Simple feature, <= 3 criteria | `tdd-workflow` |
   | Database/schema changes | `migration-safety` |
   | Touches multiple modules | `cross-module` |
   | Payment/subscription logic | `billing-security` |
   | AI/LLM prompt changes | `ai-prompt-design` |
   | Multi-step user flow | `user-flow` |

3. Order stories: backend before frontend, migrations before code, core before UI
4. Write `prd.json` following `tools/ralph/prd.json.example` schema
5. **APPROVAL GATE**: Show the PRD to the user (story count, order, strategies). Wait for approval before proceeding. In `--auto` mode, skip this gate.

**If `prd.json` exists** (or `--prd` provided): read it directly.

### Project configuration

- Read `.claude-toolkit.json` for project config (test/lint commands, MCP servers)
- Scan `.claude/agents/` for available domain agents (filenames map to labels)
- Read `CLAUDE.md` for project conventions

### Branch and state

- Create a feature branch if on main: `ralph/<slug>`
- Initialize `progress.txt` if not present:
  ```
  # Progress Log
  Started: <date>
  ---
  ```

## Step 2: Story Loop

Process each story in priority order where `passes == false` and `stuck != true`.

For each story, run five phases:

---

### Phase 1: Understand

Explore the codebase area relevant to this story.

```
Task(subagent_type="planner"):
  "Explore the codebase to understand: <story title>.
   Focus on: <acceptance criteria summary>.
   Return: affected files, existing patterns, relevant tests, dependencies."
```

**Domain Agent Discovery**: If the story's labels match a filename in `.claude/agents/` (e.g., label `billing` matches `billing.md`), also spawn that domain agent:

```
Task(subagent_type=<matched-agent>):
  "Provide domain-specific context for: <story title>.
   Story details: <acceptance criteria>."
```

**Contract Truth Verification** (blockchain projects only): If the story has a `blockchain` or `solidity` label, or the project has a blockchain domain agent, spawn a verification step:

```
Task(subagent_type=<blockchain-domain-agent>):
  "List what the smart contract interface actually supports.
   Compare against the story requirements: <acceptance criteria>.
   Flag any mismatches — features the contract doesn't support,
   missing functions, or incompatible return types."
```

This prevents hallucinating features the contract doesn't support. Any mismatches must be resolved before proceeding to Phase 2.

Summarize findings in < 500 words. This summary is passed forward to Phase 2.

---

### Phase 2: Plan

Generate an implementation plan.

```
Task(subagent_type="planner"):
  "Plan implementation for: <story title>.
   Context from exploration: <Phase 1 summary>.
   Acceptance criteria: <list>.
   Constraints: TDD mandatory, conventional commits, one story only.
   Return: ordered steps, files to change, test strategy, risks."
```

Present the plan to the user.

**APPROVAL GATE**: Ask the user to approve, revise, or skip this story.
- **Approve** → proceed to Phase 3
- **Revise** → re-run Phase 2 with feedback
- **Skip** → mark story as skipped, move to next

In `--auto` mode: auto-approve.

---

### Phase 3: Implement

Execute the approved plan using TDD.

For standard stories:
```
Task(subagent_type="tdd-guide"):
  "Implement the following plan with strict TDD (test first):
   Plan: <approved plan from Phase 2>.
   Test command: <from .claude-toolkit.json or auto-detected>.
   Lint command: <from .claude-toolkit.json or auto-detected>.
   Acceptance criteria: <list>.
   IMPORTANT: Write failing tests first, then implement, then refactor.
   Commit with conventional format referencing issue #N."
```

For complex stories (cross-module, migration-safety, or stories that failed once with tdd-guide):
```
Task(subagent_type="senior-dev"):
  "Implement this plan. The story is complex — consider architecture implications.
   Plan: <approved plan from Phase 2>.
   Test/lint commands: <from config>.
   Acceptance criteria: <list>.
   TDD is mandatory. Commit with conventional format referencing issue #N."
```

If the agent reports failures after implementation, retry once with the error context. If still failing after 2 total attempts, mark the story as stuck.

---

### Phase 4: Review

Review the changes made in Phase 3.

```
Task(subagent_type="code-reviewer"):
  "Review the changes for story: <title>.
   Focus on: correctness, error handling, test coverage, style.
   Changed files: <list from Phase 3>."
```

If the story touches auth, payments, user data, or secrets:

```
Task(subagent_type="security-reviewer"):
  "Security review for story: <title>.
   Focus on: OWASP top 10, auth bypass, data exposure.
   Changed files: <list from Phase 3>."
```

Present review findings to the user.

**APPROVAL GATE**: Ask the user to approve, request fixes, or skip.
- **Approve** → proceed to Phase 5
- **Fix** → re-run Phase 3 with review feedback, then re-review
- **Skip** → mark story as skipped

In `--auto` mode: auto-approve if no CRITICAL findings. If CRITICAL findings exist, pause and present to user regardless.

---

### Phase 5: Commit & Update State

1. **Commit** (if not already committed by tdd-guide):
   ```bash
   git add <specific files>
   git commit -m "<type>(<scope>): <description>

   Closes #<issue>

   Co-Authored-By: Claude <noreply@anthropic.com>"
   ```

2. **Update `prd.json`**: Set `passes: true` for the completed story, increment `iterations`.

3. **Append to `progress.txt`**:
   ```
   ## [Date] - [Story ID]: [Title]
   - Implemented: <summary>
   - Files changed: <list>
   - Tests: <what was added>
   ---
   ```

4. **Save deep-think checkpoint** (if MCP available):
   ```
   checkpoint(operation="save", name="<story-id>")
   ```

**On stuck stories**: Set `stuck: true` and `stuckReason` in prd.json. Log the reason in progress.txt. Move to next story.

---

## Step 3: Completion

When all stories are either `passes: true` or `stuck: true`:

1. **Final QA**: Run the full test suite and linter.
2. **Print summary table**:
   ```
   Story    | Status    | Iterations
   ---------|-----------|----------
   US-001   | DONE      | 1
   US-002   | DONE      | 2
   US-003   | STUCK     | 2 (reason)
   ```
3. **Stuck stories**: Suggest creating GitHub issues for any stuck stories:
   ```
   gh issue create --title "<story title>" --body "<context>" --label "claude-ready"
   ```
4. **Final approval**: Ask user if they want to push to remote.

## Error Handling

- **Max 2 retries per story** before marking as stuck
- **prd.json corruption**: If prd.json becomes invalid JSON, warn and restore from the last known good state
- **Agent failures**: If a Task agent fails, log the error and retry once. If it fails again, mark story as stuck with the error context.
- **Test failures**: Never commit with failing tests. Either fix or mark stuck.

## State Management

| File | Purpose |
|------|---------|
| `prd.json` | Story definitions, status, strategy assignments |
| `progress.txt` | Human-readable log, codebase patterns section |
| Deep-think checkpoints | Reasoning state per story (if MCP available) |
