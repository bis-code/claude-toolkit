# Read-Only Mode

This project is in read-only mode.

## Restricted (unless user explicitly asks)

- Do NOT create, modify, or delete source code files
- Do NOT run filesystem-modifying commands (mkdir, rm, cp, mv, npm install)
- Do NOT stage, commit, or push changes
- Do NOT modify configuration files

## Always Allowed

- Read and explore any files and directories
- Run read-only commands (git status, git log, git diff, test suites)
- Create and manage GitHub issues (gh issue create, gh issue edit)
- Review and comment on PRs (gh pr review, gh pr comment)
- Plan, analyze, and explain code
- Search the codebase with any tools

## Override

When the user explicitly asks to modify files or write code, proceed for that request only. Return to read-only mode after completing the requested modification.
