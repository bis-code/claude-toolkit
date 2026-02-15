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

Builds features from GitHub issues using an interactive orchestrator with subagent delegation and approval gates.

```bash
# In Claude Code:
/ralph --issues 42,43     # Build from specific issues
/ralph --label feature    # Build from labeled issues
/ralph --auto             # Auto-approve gates (CI mode)
```

**How it works:**
1. `/ralph` fetches GitHub issues and generates a `prd.json` with user stories
2. For each story: Explore → Plan → **Approval Gate** → Implement (TDD) → Review → **Approval Gate** → Commit
3. Specialized subagents handle each phase (planner, tdd-guide, code-reviewer, security-reviewer)
4. Domain agents spawned when story labels match installed agents (e.g., blockchain verification)
5. Final QA pass runs the full test suite before completion

### /qa — Three-Phase QA Orchestrator

Scans, triages, and fixes quality issues using parallel subagents with interactive approval.

```bash
# In Claude Code:
/qa                       # Full scan with interactive triage
/qa --scope api           # Backend only
/qa --scan-only           # Report only, no fixes
/qa --auto                # Auto-fix critical+high, issue medium, skip low
/qa --focus "auth"        # Boost severity of auth-related findings
```

**How it works:**
1. **Phase 1 — Scan**: Launches 7+ specialized agents in parallel (code-reviewer, security-reviewer, refactor-cleaner, etc.)
2. **Phase 2 — Triage**: Merges findings into a severity-ranked table. User picks what to fix, issue, or skip.
3. **Phase 3 — Fix**: Spawns targeted agents for each approved fix. Each fix: apply → test → commit individually.

## What Gets Installed

### In your project

| File | Purpose |
|------|---------|
| `tools/ralph/prd.json.example` | PRD format reference |
| `.claude/skills/*/SKILL.md` | Skill definitions (qa, ralph, etc.) |
| `.claude-toolkit.json` | Project config (test commands, QA categories) |
| `.mcp.json` | MCP server config (merged, not overwritten) |
| `.deep-think.json` | Reasoning strategies |

### Globally (~/.claude/)

| File | Purpose |
|------|---------|
| `commands/*.md` | Slash commands (verify, search, ship-day, etc.) |

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
    "maxFixLines": 30
  }
}
```

Edit this file to customize QA behavior for your project.

### Project Types

| Type | When | Behavior |
|------|------|----------|
| `repository` | Git repo detected | Full git integration (branches, commits, PRs) |
| `workspace` | No git repo | Runs in-place, skips .gitignore |

## Non-Git Workspaces

The toolkit works in directories that aren't git repos (e.g., `~/work/coding/` with multiple sub-projects).

```bash
cd ~/work/coding
~/.claude/toolkit/install.sh
```

**What's different in workspace mode:**
- `.claude-toolkit.json` has `"type": "workspace"`
- `.gitignore` modifications are skipped
- Tech stack detection scans the directory as-is

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

Global commands need to be installed in both config dirs:
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

Both `/ralph` and `/qa` are interactive skills with subagent delegation:

**`/ralph`** — Feature builder:
1. Runs inside the user's session (system prompt loaded once)
2. Spawns specialized subagents via Task tool (planner, tdd-guide, code-reviewer)
3. Human approval gates between plan and review phases
4. State: `prd.json` + `progress.txt` + deep-think checkpoints

**`/qa`** — Three-phase QA orchestrator:
1. Phase 1: Parallel scan — launches 7+ agents simultaneously
2. Phase 2: Interactive triage — severity-ranked findings with user approval
3. Phase 3: Guided fix — targeted agents apply fixes with individual commits

## License

MIT
