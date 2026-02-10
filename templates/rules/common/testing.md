# Testing

TDD is mandatory. No production code without tests.

## Test-Driven Development

1. **Red** — write a failing test that defines the expected behavior
2. **Green** — write the minimum code to make the test pass
3. **Refactor** — clean up while keeping tests green

Never skip steps. Never write production code before the test exists.

## Test Pyramid

```
     /  E2E  \        Few — critical user flows only
    /----------\
   / Integration \    Moderate — boundaries, APIs, DB
  /----------------\
 /    Unit Tests    \  Many — fast, isolated, exhaustive
/____________________\
```

- Unit tests: ALWAYS required for every change
- Integration tests: required when DB, external services, or cross-module calls are involved
- E2E tests: required for critical user flows, auth, and revenue paths

## Test Quality

- Test names describe behavior: `should return 404 when user not found` not `test getUserById`
- Cover happy path AND unhappy paths — invalid input, edge cases, error conditions
- Every bug fix requires a regression test that reproduces the bug first
- Tests must be deterministic — no flaky tests, no time-dependent assertions, no order dependency
- Tests must be independent — each test sets up and tears down its own state
- Tests must be fast — mock external dependencies, use in-memory stores where possible

## What to Test

| Change Type | Required Tests |
|-------------|----------------|
| New feature | Unit + Integration (+ E2E if critical flow) |
| Bug fix | Regression test that fails without the fix |
| Refactor | Existing tests must pass before AND after |
| API endpoint | Request/response validation, auth, error codes |
| Database change | Migration up/down, data integrity |

## What NOT to Test

- Framework internals or third-party library behavior
- Private implementation details — test through public interfaces
- Trivial getters/setters with no logic
- Generated code

## Test Execution

- Run the full relevant test suite after every change
- Fix failures before proceeding — never leave tests red
- If a test cannot be written, explain why and propose alternative validation
