# Development Workflow

Every non-trivial change follows this pipeline in order. Skipping steps is a process failure, not a time saving.

## Pipeline

```
Research → Plan → TDD → Review → Commit
```

### 1. Research

Goal: understand before changing anything.

Do:
- Read the relevant files — handler, service, repository, tests
- Find all callers of the function or module being changed
- Identify what tests already exist for this area

Do NOT:
- Open an editor
- Write any code
- Make assumptions about behavior without reading the source

Exit criteria: you can explain the current behavior, its callers, and its existing test coverage.

### 2. Plan

Goal: define what will change and why, before touching a file.

Do:
- State the problem in one sentence
- List the files that will change and what each change is
- Define the test strategy (unit, integration, E2E — and why)
- Identify one assumption that could be wrong

Do NOT:
- Write production code
- Write tests yet
- Start a new branch until the plan is confirmed

Exit criteria: the plan could be reviewed by another developer with no additional context.

### 3. TDD

Goal: implement with tests driving the design.

Do:
- Write one failing test (Red)
- Write minimum code to pass it (Green)
- Refactor while keeping tests green (Refactor)
- Repeat until all planned behavior is covered

Do NOT:
- Write multiple tests before implementing any
- Write production code without a failing test first
- Skip the refactor step — technical debt accumulates here

Exit criteria: all tests pass, coverage matches the plan's test strategy.

### 4. Review

Goal: catch issues before they reach version control.

Do:
- Run `git diff` and read the full diff as if you were the reviewer
- Check every changed function for missing error handling
- Verify tests cover at least one failure path per happy path
- Run the linter and formatter

Do NOT:
- Commit before the diff review
- Ignore linter warnings — fix or explicitly suppress with a comment

Exit criteria: you would approve this diff if someone else submitted it.

### 5. Commit

Goal: create a traceable, atomic unit of change.

Do:
- Stage specific files only — never `git add .`
- Write a conventional commit message with scope
- Include `Closes #<issue>` in the commit body if applicable

Do NOT:
- Combine unrelated changes in one commit
- Commit with failing tests
- Use "WIP", "fix", or "changes" as a commit message

Exit criteria: `git log --oneline` tells a clear story of what changed and why.

## Fast-Track (Allowed For)

The full pipeline is required for features, bug fixes, and refactors. Fast-track (Research + Commit only) is acceptable for:
- Documentation-only changes
- Dependency version bumps with no API changes
- Configuration or environment variable changes
- Formatting and lint-only fixes

If unsure, use the full pipeline. The cost of a skipped step is always higher than the time saved.
