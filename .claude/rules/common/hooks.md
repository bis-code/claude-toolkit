# Hooks

Hooks run automatically at defined points in the Claude agent lifecycle. They enforce constraints that prompts and rules cannot — because prompts are advisory, hooks are executed.

## Hook Types

| Hook | Trigger | Common Use |
|------|---------|-----------|
| `SessionStart` | Before the first user turn in a session | Load context, validate env, print reminders |
| `PreToolUse` | Before any tool is called | Block dangerous operations, enforce naming conventions |
| `PostToolUse` | After any tool returns | Log side effects, trigger follow-up actions |
| `Stop` | When the agent produces a final response | Validate output format, enforce checklists |
| `PreCompact` | Before context is compacted | Save critical state to files before truncation |

## Configuration

Hooks are defined in `.claude/hooks/hooks.json` (managed by the toolkit):

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "scripts/hooks/pre-bash.sh"
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "scripts/hooks/pre-stop.sh"
          }
        ]
      }
    ]
  }
}
```

## What Hooks Enforce That Prompts Forget

Prompts degrade over long conversations. Hooks do not. Use hooks for:

- **Dangerous command blocking** — prevent `git push --force`, `rm -rf`, `DROP TABLE` without confirmation
- **Branch naming enforcement** — reject commits to `main` directly
- **Secret scanning** — block file writes containing patterns that look like API keys
- **Test gate** — refuse to commit if the test command exits non-zero
- **Output format validation** — ensure every Stop response includes a file path list

## Example: Block Direct Commits to Main

```javascript
// .claude/hooks/scripts/block-main-commit.js
const { parseJSON, log } = require('./lib/utils');

function run(rawInput) {
  const payload = parseJSON(rawInput);
  const command = (payload.tool_input || {}).command || '';

  if (/git commit/.test(command)) {
    const { execSync } = require('child_process');
    const branch = execSync('git branch --show-current', { encoding: 'utf8' }).trim();
    if (branch === 'main') {
      log('[Toolkit] Blocked: direct commit to main');
      return JSON.stringify({ error: 'Direct commit to main is not allowed. Create a feature branch first.' });
    }
  }
  return rawInput;
}
module.exports = { run };
```

## Example: Test Gate Before Commit

```javascript
// .claude/hooks/scripts/test-gate.js
const { parseJSON, log } = require('./lib/utils');
const { execSync } = require('child_process');

function run(rawInput) {
  const payload = parseJSON(rawInput);
  const command = (payload.tool_input || {}).command || '';

  if (/git commit/.test(command)) {
    log('[Toolkit] Running tests before commit...');
    try {
      execSync('npm test --silent', { timeout: 30000 });
    } catch {
      return JSON.stringify({ error: 'Tests must pass before committing. Fix failures first.' });
    }
  }
  return rawInput;
}
module.exports = { run };
```

## Best Practices

- Hooks must be fast. A hook that blocks for 30 seconds on every tool call degrades the agent experience significantly.
- Hooks should print a clear reason when they block. "Permission denied" is not enough — tell the agent what to do instead.
- Keep hook scripts in `scripts/hooks/` and check them into version control. Hooks that live only on one machine are not team constraints.
- Use `PreToolUse` matchers to scope hooks to specific tools — do not run expensive checks on every Read call.
- Prefer `exit 1` with a message over silent failure. The agent needs the reason to correct its approach.

## What Hooks Cannot Do

- Hooks can modify stdout output (which Claude Code reads as the hook result). Use this to block by returning `{ "error": "reason" }`.
- Hooks cannot inject new tool calls into the agent's plan.
- Hooks that exit non-zero abort the tool call — the agent sees the hook's stderr as an error message.
