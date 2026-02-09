---
description: Autonomous feature builder (Ralph + deep-think loop)
---

# /ralph — Autonomous Feature Builder

Build features autonomously using the Ralph pattern: fresh Claude instances per story with persistent state, structured reasoning via deep-think, and browser verification via Playwright.

## Phase 0: PRD Generation (this command)

Generate a `prd.json` from GitHub issues, then hand off to `ralph.sh` for autonomous execution.

### Usage

```
/ralph --issues 42,43        # Build from specific issues
/ralph --label enhancement   # Build from labeled issues
/ralph --prd path/to/prd.json  # Use existing PRD
```

### Arguments: $ARGUMENTS

## Step 1: Determine Input

Parse `$ARGUMENTS`:
- `--issues N,N` -> fetch those specific GitHub issues
- `--label X` -> fetch open issues with that label: `gh issue list --label "X" --state open --json number,title,body,labels`
- `--prd path` -> skip to Step 4 (use existing PRD)
- No args -> ask the user what to build

## Step 2: Fetch & Analyze Issues

For each issue:
```bash
gh issue view <number> --json number,title,body,labels,milestone
```

If deep-think MCP is available, use it to analyze complexity:
```
strategize(operation="set", strategy="convergent")
think(thought="Analyzing issues for story decomposition...", ...)
```

Consider:
- How many discrete stories does this break into?
- What's the dependency order?
- Which strategy fits each story?
- Are there database/schema changes?
- Frontend, backend, or both?

## Step 3: Generate prd.json

Create `prd.json` in the project root. Reference `tools/ralph/prd.json.example` for the schema.

**Strategy mapping:**
| Story type | Strategy |
|------------|----------|
| Simple feature, <= 3 criteria | `tdd-workflow` |
| Database/schema changes | `migration-safety` |
| Touches multiple modules | `cross-module` |
| Payment/subscription logic | `billing-security` |
| AI/LLM prompt changes | `ai-prompt-design` |
| Multi-step user flow | `user-flow` |
| Smart contract changes | `smart-contract-safety` |
| Infrastructure/deployment | `infrastructure` |

**Ordering rules:**
- Backend before frontend
- Migrations before code that uses new columns/tables
- Core logic before UI
- Each story should be independently testable

## Step 4: Show PRD & Get Approval

Display the generated PRD to the user:
- Story count and order
- Strategy per story
- Estimated complexity

Ask for approval before proceeding.

## Step 5: Create Branch & Launch

```bash
# Create feature branch (if not already on one)
git checkout -b ralph/<slug>

# Initialize progress file
echo "# Progress Log\nStarted: $(date)\n---" > progress.txt
```

Tell the user to run the orchestrator:

```bash
./tools/ralph/ralph.sh
```

Or for limited iterations:
```bash
./tools/ralph/ralph.sh --max-loops 10
```

## What Happens Next

`ralph.sh` spawns fresh Claude instances per story, each following `RALPH.md`:
1. Read state (prd.json + progress.txt + Tasks)
2. Reason (deep-think, conditional on complexity)
3. Implement (TDD — test first)
4. Commit (conventional commits)
5. Update state (prd.json + progress.txt + Tasks)
6. Final QA pass when all stories done
