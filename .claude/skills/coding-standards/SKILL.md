---
name: coding-standards
description: "Universal coding standards reference. Use when unsure about naming, structure, or patterns."
---

# Coding Standards

Language-agnostic standards for writing clean, maintainable, and reviewable code.

## Naming Conventions

- Names reveal intent: `remainingRetries` not `r`, `fetchUserById` not `getData`
- Booleans start with `is`, `has`, `can`, `should`: `isActive`, `hasPermission`
- Functions describe actions: `calculateTotal`, `validateInput`, `sendNotification`
- Constants use UPPER_SNAKE_CASE: `MAX_RETRIES`, `DEFAULT_TIMEOUT`
- Avoid abbreviations unless universal (`id`, `url`, `http` are fine; `usr`, `mgr` are not)
- One concept per file -- group by feature/domain, not by type

## Function Design

- **One responsibility** -- if you need "and" to describe it, split it
- **Maximum 30 lines** (excluding tests) -- extract when larger
- **Maximum 4 parameters** -- use an options/config object beyond that
- **Pure functions preferred** -- same input, same output, no side effects
- **Early returns** over deep nesting -- guard clauses first, happy path last

## Error Handling

- Handle errors at the appropriate level -- not too early, not too late
- Fail fast on invalid input -- validate at boundaries
- Never swallow errors silently -- log, rethrow, or handle explicitly
- Use custom error types for domain-specific failures
- Error messages include context: what failed, with what input, why
- Async errors are always caught (no unhandled promise rejections)

## Documentation

- Comments explain **why**, never **what** -- if code needs a "what" comment, refactor it
- TODO comments include a ticket reference: `// TODO(#123): migrate to new API`
- Public functions have a one-line description of behavior
- Delete dead code -- version control remembers it

## Code Organization

- Keep files under 300 lines -- split when larger
- Imports sorted and grouped: stdlib, external, internal
- Public API at the top of the file, private helpers below
- Modules expose a public interface -- internal details are private
- Dependencies flow in one direction -- no circular imports

## DRY and Abstraction

- Duplicate code acceptable up to 2 occurrences -- extract on the third
- Wrong abstraction is worse than duplication
- Extract when the **reason for change** is the same, not when code looks similar

## Code Review Readiness

- [ ] All tests pass (unit, integration, E2E as applicable)
- [ ] No commented-out code or debug statements
- [ ] No TODO comments without issue references
- [ ] Naming is consistent with project conventions
- [ ] Functions are under 30 lines, files under 300 lines
- [ ] Error handling is explicit and tested
- [ ] No hardcoded values that should be configurable
