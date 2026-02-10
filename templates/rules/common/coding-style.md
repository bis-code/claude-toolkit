# Coding Style

Universal style rules for readability, consistency, and maintainability.

## Naming

- Use descriptive names that reveal intent: `remainingRetries` not `r`, `fetchUserById` not `getData`
- Boolean variables/functions start with `is`, `has`, `can`, `should`: `isActive`, `hasPermission`
- Constants use UPPER_SNAKE_CASE: `MAX_RETRIES`, `DEFAULT_TIMEOUT`
- Avoid abbreviations unless universally understood (`id`, `url`, `http` are fine; `usr`, `mgr`, `svc` are not)

## Functions

- One function, one responsibility — if you need "and" to describe it, split it
- Maximum 30 lines per function (excluding tests) — extract when larger
- Maximum 4 parameters — use an options/config object beyond that
- Pure functions over side effects whenever possible
- Early returns over deep nesting — guard clauses first, happy path last

## Files

- One concept per file — a file should have a single reason to change
- Keep files under 300 lines — split when larger
- Group related files by feature/domain, not by type (prefer `user/handler.ts` over `handlers/user.ts`)
- Consistent file naming within the project: pick one convention and enforce it

## Readability

- Readability over cleverness — write code your future self can understand at 2am
- No magic numbers or strings — extract to named constants with context
- Prefer explicit over implicit — don't rely on language quirks or obscure operator behavior
- Comments explain **why**, never **what** — if the code needs a "what" comment, refactor the code
- Delete dead code — version control remembers it for you

## DRY & Abstraction

- Duplicate code is acceptable up to 2 occurrences — extract on the third (Rule of Three)
- Wrong abstraction is worse than duplication — don't force unrelated code into a shared function
- Extract when the **reason for change** is the same, not when the code **looks similar**

## Formatting

- Use the project's formatter — never manually format code
- Consistent indentation (spaces or tabs — match the project)
- One blank line between logical sections, two between top-level declarations
- Imports sorted and grouped: stdlib, external, internal
