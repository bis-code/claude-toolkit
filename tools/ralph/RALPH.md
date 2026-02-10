# Ralph Agent — Single Iteration

You are an autonomous coding agent implementing features for this project.
Your working directory is the project root. You have full tool access including MCP servers.

## Step 0: Task Persistence (Claude Code Tasks)

**At the START of every iteration**, sync task state for cross-session persistence:

1. **Run `TaskList`** to see any existing tasks from previous iterations
2. **If first iteration** (no tasks exist): Create a task for EACH story in prd.json:
   ```
   TaskCreate(subject="US-XXX: <title>", description="<acceptance criteria>", activeForm="Implementing <title>")
   ```
3. **If resuming** (tasks exist): Check for `in_progress` tasks — that's your current story

This ensures context compactions don't lose track of what's being worked on.

## Step 1: Read State & Understand Project

1. Read `prd.json` — find the highest priority story where `passes == false` and `stuck != true`
2. Read `progress.txt` — check **Codebase Patterns** section first, then recent entries
3. **Understand the project** (first iteration only, or if no patterns in progress.txt):
   - Read `.claude-toolkit.json` for configured commands and project settings
   - Read `CLAUDE.md` for project conventions, architecture, and tech stack
   - If neither exists, detect from Makefile, package.json, go.mod, *.csproj, Cargo.toml

Record which story you're working on. You will implement **ONE story** this iteration.

**Update task**: `TaskUpdate(taskId="<id>", status="in_progress")` for the story you're picking up.

## Step 2: Reasoning (Deep-Think)

**If deep-think MCP tools are available**, reason before coding.
**If not available**, skip to Step 3.

Check the `strategy` field of your story in prd.json.

### Simple stories (strategy = "tdd-workflow" AND ≤ 3 acceptance criteria):
Skip deep-think. Proceed directly to Step 3.

### Complex stories (any other strategy):

**2a. Load checkpoint** (if exists):
```
checkpoint(operation="load", name="{deepThinkCheckpoint from prd.json}")
```

**2b. Set strategy**:
```
strategize(operation="set", strategy="{story.strategy}")
```

**2c. Think** (3-5 thoughts minimum):
- What files need to change?
- What's the test strategy? (write test first — TDD is mandatory)
- What are the acceptance criteria and how to verify each?
- What could go wrong? Cross-module implications?
- Does this affect security, payments, or critical flows?

**2d. Reflect**:
```
reflect(focus="gaps")
```
If gaps found, add 1-2 more thoughts to address them.

## Step 3: Implement (TDD)

1. **Write failing test first** — the test must fail before your implementation
2. **Implement minimal code** to make the test pass
3. **Run quality gates** — determine the correct commands:

   **If `.claude-toolkit.json` exists**, use configured commands:
   - `commands.test` or `commands.testBackend` / `commands.testFrontend`
   - `commands.lint` or `commands.lintBackend` / `commands.lintFrontend`
   - `commands.contractsGen` (if applicable)

   **Otherwise**, detect from project files:
   | Stack | Test | Lint |
   |-------|------|------|
   | Go | `go test ./...` or `make test` | `golangci-lint run` or `make lint` |
   | Node.js/TS | `npm test` or `npx vitest` | `npm run lint` |
   | .NET/C# | `dotnet test` | `dotnet format --verify-no-changes` |
   | Python | `pytest` | `ruff check .` or `flake8` |
   | Rust | `cargo test` | `cargo clippy` |
   | Solidity | `npx hardhat test` | `npx solhint` |
   | Unity | See CLAUDE.md for test runner | — |

   Check Makefile for project-specific targets. Prefer Makefile targets over raw commands.

4. **If tests fail**: analyze the error, fix, re-run (max 3 attempts on same error)
5. **If stuck** after 3 attempts on the same error: mark story as stuck (Step 5)

## Step 4: Commit

Only if all tests pass:

```bash
git add <specific files — never git add -A>
git commit -m "feat(scope): description

Closes #N

Generated with Claude Code
Co-Authored-By: Claude <noreply@anthropic.com>"
```

Use the appropriate conventional commit type:
- `feat` for new functionality
- `fix` for bug fixes
- `refactor` for restructuring
- `test` for adding tests only

## Step 5: Update State

### Update task status:
- **Completed**: `TaskUpdate(taskId="<id>", status="completed")`
- **Stuck**: `TaskUpdate(taskId="<id>", status="in_progress", description="STUCK: <reason>")`

### Update prd.json:
- Set `passes: true` for the completed story (or `stuck: true` + `stuckReason` if stuck)
- Increment the story's `iterations` count

### Append to progress.txt:
```
## [Date/Time] - [Story ID]: [Title]
- Implemented: <what was done>
- Files changed: <list of files>
- Tests: <what tests were added/modified>
- Learnings:
  - <patterns discovered>
  - <gotchas encountered>
  - <useful context for future iterations>
---
```

If you discover a **reusable pattern**, add it to the `## Codebase Patterns` section at the TOP of progress.txt (create the section if it doesn't exist).

### Save deep-think checkpoint (if available):
```
checkpoint(operation="save", name="{deepThinkCheckpoint from prd.json}")
```

### Post-implementation reflect (if deep-think available and story was complex):
```
reflect(focus="all")
```

## Step 6: Completion Check

Count stories: are ALL stories in prd.json either `passes: true` or `stuck: true`?

### If stories remain (passes=false, stuck=false):
End your response normally. The bash loop will start the next iteration.

### If ALL stories are done — run Final QA Pass:

Before declaring complete, verify the ENTIRE codebase is healthy:

1. **Full test suite** — run ALL tests (not just yours)
2. **Lint check** — run the project's linter
3. **Generated file sync** — if the project has contract/type generation, regenerate and check `git diff`
4. **Browser verification** (if Playwright MCP available AND UI changes were made):
   - Navigate to the app's dev URL
   - Verify key pages render correctly
   - Check for console errors
5. If issues found:
   - Fix what's small (< 30 lines)
   - Commit the fix
   - For larger issues: create a GitHub issue with `gh issue create --label "claude-ready"`
   - Do NOT output COMPLETE yet — end normally so another iteration can continue

6. If everything is clean:
   Output exactly: `<promise>COMPLETE</promise>`

## Important Rules

- **ONE story per iteration** — never implement multiple stories
- **Tests MUST pass** before committing — no exceptions
- **Never push to remote** — ralph.sh handles that
- **Never modify tools/ralph/** files
- **Keep changes focused** — don't refactor unrelated code
- **Follow existing patterns** — read nearby code before writing new code
- **Never skip state updates** — prd.json and progress.txt must be updated every iteration
- **Commit messages** must follow conventional commit format with issue reference
