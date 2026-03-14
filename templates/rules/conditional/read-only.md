# GitHub Read-Only Mode

GitHub write operations are blocked by default. Code editing, file changes, git commits, and all other local work are fully allowed.

## GitHub: Restricted (unless user explicitly asks)

- Do NOT create, edit, or close issues (`gh issue create`, `gh issue edit`, `gh issue close`)
- Do NOT create, merge, or close PRs (`gh pr create`, `gh pr merge`, `gh pr close`)
- Do NOT comment on issues or PRs (`gh issue comment`, `gh pr comment`)
- Do NOT review or approve PRs (`gh pr review`, `gh pr review --approve`)
- Do NOT add labels, assign, or edit metadata
- Do NOT use `gh api` with POST, PUT, PATCH, or DELETE methods

## GitHub: Always Allowed

- `gh issue list`, `gh issue view`, `gh issue status`
- `gh pr list`, `gh pr view`, `gh pr status`, `gh pr diff`, `gh pr checks`
- `gh repo view`, `gh search`
- Any `gh api` with GET method

## No Claude Attribution in GitHub Content

When the user grants permission for a GitHub write operation, NEVER include any of these in issue bodies, PR descriptions, comments, review text, or any GitHub-visible content:

- "Generated with Claude Code"
- "Co-Authored-By: Claude"
- "Claude" or "AI-generated" references
- Robot emoji (🤖) attribution lines

This applies to all GitHub-facing text. Git commits are separate — follow the project's commit rules.

## Override

When the user explicitly asks to perform a GitHub write action, proceed for that request only. Return to GitHub read-only mode after.
