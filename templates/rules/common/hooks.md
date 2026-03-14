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

Hooks are defined in `.claude/settings.json`:

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

```bash
#!/bin/bash
# scripts/hooks/pre-bash.sh
# Runs before every Bash tool call

COMMAND="$1"

if echo "$COMMAND" | grep -q "git commit" && git branch --show-current | grep -q "^main$"; then
  echo "ERROR: Direct commit to main is not allowed. Create a feature branch first."
  exit 1
fi
```

## Example: Test Gate Before Commit

```bash
#!/bin/bash
# Only runs when the Bash command looks like a commit

COMMAND="$1"

if echo "$COMMAND" | grep -q "git commit"; then
  echo "Running test suite before commit..."
  if ! npm test --silent; then
    echo "ERROR: Tests must pass before committing. Fix failures first."
    exit 1
  fi
fi
```

## Best Practices

- Hooks must be fast. A hook that blocks for 30 seconds on every tool call degrades the agent experience significantly.
- Hooks should print a clear reason when they block. "Permission denied" is not enough — tell the agent what to do instead.
- Keep hook scripts in `scripts/hooks/` and check them into version control. Hooks that live only on one machine are not team constraints.
- Use `PreToolUse` matchers to scope hooks to specific tools — do not run expensive checks on every Read call.
- Prefer `exit 1` with a message over silent failure. The agent needs the reason to correct its approach.

## What Hooks Cannot Do

- Hooks cannot modify the tool's input before it runs (they can only block or allow).
- Hooks cannot inject new tool calls into the agent's plan.
- Hooks that exit non-zero abort the tool call — the agent sees the hook's stderr as an error message.
