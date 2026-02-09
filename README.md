# Claude Code Toolkit

Autonomous development tooling for [Claude Code](https://docs.anthropic.com/en/docs/claude-code). Install `/ralph` (feature builder) and `/qa` (quality agent) into any project with one command.

## Quick Start

```bash
# Install toolkit
git clone https://github.com/bis-code/claude-toolkit.git ~/.claude/toolkit

# Install into your project
cd ~/your-project
~/.claude/toolkit/install.sh
```

Or one-liner:
```bash
curl -fsSL https://raw.githubusercontent.com/bis-code/claude-toolkit/main/install.sh | bash
```

## What You Get

### /ralph — Autonomous Feature Builder

Builds features from GitHub issues using the Ralph pattern: fresh Claude Code instances per user story with persistent state.

```bash
# In Claude Code:
/ralph --issues 42,43     # Generate PRD from issues
/ralph --label feature    # Generate PRD from labeled issues

# Then run the orchestrator:
./tools/ralph/ralph.sh
```

**How it works:**
1. `/ralph` fetches GitHub issues and generates a `prd.json` with user stories
2. `ralph.sh` spawns fresh Claude instances per story (prevents context degradation)
3. Each iteration: deep-think reasoning → TDD implementation → commit → state update
4. Final iteration runs a full QA pass before declaring complete

### /qa — Autonomous QA Agent

Scans, fixes, and reports quality issues. Uses a git worktree for isolation (git repos) or runs in-place (workspaces).

```bash
# In Claude Code:
/qa                       # Full scan
/qa --scope api           # Backend only
/qa --scan-only           # Report only, no fixes
```

**How it works:**
1. Creates a git worktree from your default branch (or runs in-place for non-git dirs)
2. Spawns fresh Claude instances per scan category
3. Fixes small issues (< 30 lines), reports larger ones as GitHub issues
4. Creates a PR with all fixes when done (git repos only)

## What Gets Installed

### In your project

| File | Purpose |
|------|---------|
| `tools/ralph/ralph.sh` | Orchestrator script |
| `tools/ralph/RALPH.md` | Per-iteration prompt |
| `tools/ralph/prd.json.example` | PRD format reference |
| `tools/qa/qa.sh` | QA orchestrator script |
| `tools/qa/QA_PROMPT.md` | QA per-iteration prompt |
| `.claude-toolkit.json` | Project config (test commands, QA categories) |
| `.mcp.json` | MCP server config (merged, not overwritten) |
| `.deep-think.json` | Reasoning strategies |

### Globally (~/.claude/)

| File | Purpose |
|------|---------|
| `commands/ralph.md` | `/ralph` command |
| `commands/qa.md` | `/qa` command |

## MCP Servers

| Server | Tier | Purpose |
|--------|------|---------|
| `deep-think` | **Required** | Structured reasoning with strategies and reflection |
| `playwright` | Recommended | Browser testing (auto-suggested for UI projects) |
| `leann-server` | Optional | Semantic code search |

## Project Configuration

The installer generates `.claude-toolkit.json` with your project's settings:

```json
{
  "version": "1.0.0",
  "project": {
    "name": "my-project",
    "type": "repository",
    "techStack": ["go", "react"]
  },
  "commands": {
    "test": "make test",
    "lint": "make lint"
  },
  "qa": {
    "scanCategories": ["tests", "lint", "missing-tests", "todo-audit"],
    "maxFixLines": 30,
    "worktreeFromBranch": "main"
  }
}
```

Edit this file to customize QA behavior for your project.

### Project Types

| Type | When | QA Behavior |
|------|------|-------------|
| `repository` | Git repo detected | Uses worktree for isolated QA (default) |
| `workspace` | No git repo | Runs QA in-place, skips .gitignore |

## Non-Git Workspaces

The toolkit works in directories that aren't git repos (e.g., `~/work/coding/` with multiple sub-projects).

```bash
cd ~/work/coding
~/.claude/toolkit/install.sh
```

**What's different in workspace mode:**
- `.claude-toolkit.json` has `"type": "workspace"`
- `qa.sh` runs in-place (no worktree isolation)
- `.gitignore` modifications are skipped
- Tech stack detection scans the directory as-is

**`qa.sh` flags for workspaces:**
```bash
./tools/qa/qa.sh                  # Runs in-place automatically
./tools/qa/qa.sh --scan-only      # Report only, no fixes
./tools/qa/qa.sh --no-worktree    # Force in-place mode (even in git repos)
```

## Auto-Detection

The toolkit auto-detects your tech stack and suggests appropriate settings:

| Detected | Test Command | Lint Command | QA Categories |
|----------|-------------|-------------|---------------|
| Go | `go test ./...` | `golangci-lint run` | + module-boundaries, security |
| Node.js | `npm test` | `npm run lint` | + accessibility, browser-testing |
| .NET/C# | `dotnet test` | `dotnet format` | + module-boundaries, security |
| Python | `pytest` | `ruff check .` | + module-boundaries, security |
| Rust | `cargo test` | `cargo clippy` | + module-boundaries, security |
| Solidity | `npx hardhat test` | `npx solhint` | + smart-contract-security, gas |
| React/Vue | (from package.json) | (from package.json) | + accessibility, component-quality |

Makefile targets are preferred over raw commands when available.

## Multiple Claude Accounts

If you use different Claude accounts for different directories (e.g., personal vs work), use the `CLAUDE_CONFIG_DIR` environment variable with a shell function:

```bash
# Add to ~/.zshrc (same pattern as gh CLI account switching)
claude() {
  if [[ "$PWD" == "$HOME/work"* ]]; then
    CLAUDE_CONFIG_DIR="$HOME/.claude-work" command claude "$@"
  else
    command claude "$@"
  fi
}
```

Then authenticate each account:
```bash
# Personal (default ~/.claude)
claude login

# Work
CLAUDE_CONFIG_DIR=~/.claude-work claude login
```

Global commands (`/ralph`, `/qa`) need to be installed in both config dirs:
```bash
cp -r ~/.claude/commands ~/.claude-work/commands
```

## Prerequisites

- [Claude Code](https://docs.anthropic.com/en/docs/claude-code) CLI installed and authenticated
- [jq](https://jqlang.github.io/jq/) — `brew install jq`
- [git](https://git-scm.com/)
- [gh](https://cli.github.com/) — GitHub CLI (optional, for issue fetching)
- [npm](https://nodejs.org/) — for MCP server installation

## Updating

```bash
cd ~/.claude/toolkit && git pull
~/.claude/toolkit/install.sh --update
```

## How It Works

Both `/ralph` and `/qa` use the same core pattern:

1. **Bash orchestrator** (`ralph.sh` / `qa.sh`) spawns fresh Claude Code instances
2. **Per-iteration prompt** (`RALPH.md` / `QA_PROMPT.md`) guides each instance
3. **State files** (`prd.json` / `qa-state.json`) persist progress between iterations
4. **Deep-think MCP** provides structured reasoning for complex decisions
5. **Claude Code Tasks** (`TaskCreate`/`TaskUpdate`) survive context compactions

Fresh instances prevent context window degradation. State files ensure nothing is lost.

## License

MIT
