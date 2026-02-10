---
name: tdd-workflow
description: "Complete TDD methodology. Use when writing new features, fixing bugs, or refactoring."
---

# TDD Workflow

Test-Driven Development is mandatory for all production code changes.

## When to Use

- Writing a new feature or endpoint
- Fixing a bug (write the regression test first)
- Refactoring existing code (tests must exist before and after)

## Red-Green-Refactor Cycle

### 1. Red: Write a Failing Test

- Write the **smallest possible test** that describes the desired behavior
- Run it and confirm it **fails for the right reason** (not a syntax error)
- The test name should read like a specification: `should reject expired tokens`

### 2. Green: Make It Pass

- Write the **minimum code** to make the test pass
- Do not generalize, optimize, or clean up yet
- Resist the urge to write more code than the test demands

### 3. Refactor: Clean Up

- Improve structure, naming, and duplication -- without changing behavior
- Run tests after every refactor step to confirm nothing broke

## Test-First Checklist

- [ ] Test file exists or is created
- [ ] Test describes **behavior**, not implementation details
- [ ] Test covers the **happy path**
- [ ] Test covers at least one **unhappy path** (invalid input, error state)
- [ ] Test runs and **fails** before implementation begins
- [ ] Test is **deterministic** (no flaky dependencies on time, network, randomness)

## Common Pitfalls

| Pitfall | Fix |
|---------|-----|
| Writing tests after code | Stop. Delete the code. Write the test first. |
| Testing implementation details | Test inputs and outputs, not internal methods |
| Tests that depend on order | Each test must set up its own state |
| Overly broad tests | One assertion per behavior; split large tests |
| Mocking everything | Mock boundaries (DB, HTTP), not internal logic |
| Skipping the refactor step | Green is not done -- refactored green is done. |

## Bug Fix Flow

1. Reproduce the bug with a failing test
2. Confirm the test fails on the current code
3. Fix the code (minimum change)
4. Confirm the test passes and no other tests broke

## New Feature Flow

1. Write test for the simplest case, implement just enough to pass
2. Write test for the next case (edge case, error, boundary), implement
3. Refactor once all cases are covered
4. Run full test suite before committing

## Test Quality Signals

- Tests run in **under 5 seconds** (unit) or **under 30 seconds** (integration)
- Test names form a **readable specification** of the module
- Deleting any line of production code causes **at least one test to fail**
- Tests can run in **any order** and still pass
