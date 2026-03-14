# Agents

Specialized agents handle tasks that require deep, focused expertise. Use them instead of relying on a generalist model for tasks they are designed for.

## When to Use an Agent

Use an agent when:
- The task requires a specific, repeatable workflow (code review, DB analysis, E2E generation)
- The task benefits from a constrained tool set that limits accidental side effects
- The task will recur across the project lifecycle

Do not use an agent for one-line questions, quick lookups, or tasks that take fewer than 3 steps.

## Agent Decision Tree

```
Is this a code review or diff analysis?         → code-reviewer
Is this a performance bottleneck investigation? → performance-reviewer
Is this a security audit or threat model?       → security-reviewer
Is this a database query or schema review?      → database-reviewer
Is this E2E test generation or flake diagnosis? → e2e-runner
Is this a TDD cycle for a new feature?          → tdd-guide
Is this architectural planning or ADR?          → architect-reviewer
Is this a production incident?                  → incident-debugger
Is this a refactor or cleanup task?             → refactor-cleaner
Is this documentation or changelog?             → doc-updater
Is this a build or CI failure?                  → build-error-resolver
```

## How to Invoke

Reference the agent by name in your task description. The agent's system prompt and tool permissions load automatically.

```
@database-reviewer review the migration in db/migrations/20240310_add_orders.sql
@code-reviewer check the diff on branch issue/45-payment-modal
@e2e-runner generate tests for the checkout flow
```

## Agent Scope Rules

- Agents are READ-ONLY by default unless they explicitly include Write or Edit tools.
- Never ask a reviewer agent to make changes — use a separate implementation agent or direct Claude for edits.
- Agents inherit project rules (coding style, git workflow, testing) unless their system prompt overrides them.

## Composing Agents

For complex tasks, chain agents sequentially:

1. `architect-reviewer` — validate the approach before building
2. `tdd-guide` — drive implementation with tests
3. `code-reviewer` — review the output before committing
4. `security-reviewer` — audit any auth or data-access changes

Each agent operates independently. Pass its output as context to the next.
