---
name: verification-loop
description: "Continuous verification loop. Use when implementing multi-step features to ensure correctness."
---

# Verification Loop

Systematic checkpoint and verification strategy for multi-step implementations.

## When to Use

- Implementing features that span multiple files or modules
- Multi-step refactoring where intermediate states must be valid
- Changes that touch both frontend and backend
- Any implementation requiring more than 3 commits

## Checkpoint Strategy

Break the implementation into discrete checkpoints. Each checkpoint must be:

- **Independently verifiable** -- tests pass, code compiles, no runtime errors
- **Committable** -- represents a coherent state (not a half-finished thought)
- **Reversible** -- if the next step fails, you can roll back to this checkpoint

## Verification Steps (Run at Every Checkpoint)

### 1. Type Check / Compile

- [ ] No type errors or compilation warnings
- [ ] New types consistent with existing interfaces

### 2. Lint

- [ ] Linter passes with zero warnings
- [ ] No disabled lint rules without justification

### 3. Unit Tests

- [ ] All existing tests pass (no regressions)
- [ ] New tests added for new behavior

### 4. Integration Tests

- [ ] Cross-module interactions verified
- [ ] External service boundaries tested with realistic mocks

### 5. Runtime Verification

- [ ] Application starts without errors
- [ ] Key endpoints respond correctly
- [ ] No console errors or unhandled rejections

## Recovery from Failures

**Test failure**: Read the message. Identify if regression or incomplete implementation. Review the diff.

**Compile/type failure**: Check the most recently changed file first. Verify imports and type definitions.

**Stuck after 3 attempts**: Revert to last green checkpoint. Re-read requirements. Try a different approach.

## State Management

- **Track verified checkpoints** -- do not re-verify passing ones
- **Log each result** in progress notes for session continuity
- **Commit at each green checkpoint** -- small commits are cheaper than lost work
- **Never skip verification** -- catching bugs late costs more

## Verification Cadence

| Change Size | Verify After |
|-------------|-------------|
| Single function | Run related test file |
| Single file | Type check + related tests |
| Multiple files | Full lint + test suite |
| Cross-module | Full suite + runtime check |
| Pre-commit | All verification steps |
