---
name: tdd-workflow
description: "Enforce test-first discipline with red-green-refactor guidance and verification checkpoints."
---

# /tdd-workflow

Spawns the `tdd-guide` agent to enforce TDD discipline throughout an implementation task, including verification checkpoints at each step.

## Steps

1. **Gather context** — understand what needs TDD guidance:
   - If arguments are provided (feature description, bug to fix, files to refactor), use those
   - Otherwise, ask the user what they are implementing
   - Detect the test framework and test command from `.claude-toolkit.json` or auto-detection
   - Identify existing test files related to the scope

2. **Spawn the agent** — use the Task tool:
   ```
   Task tool with subagent_type="tdd-guide"
   ```
   Pass in the prompt:
   - The task description (feature, bug fix, or refactor)
   - The test command and framework
   - Existing test file paths for the affected modules
   - Any relevant source files for context

3. **Present guidance** — relay the agent's TDD plan:
   - Ordered list of test cases to write (red phase)
   - Minimum implementation for each test (green phase)
   - Refactoring suggestions after tests are green
   - Verification checkpoints between each cycle:
     - Type check / compile passes
     - Lint passes
     - All tests pass (no regressions)
     - Runtime verification (app starts, endpoints respond)

4. **Offer follow-up actions**:
   - "Start first test?" — write the first failing test
   - "Verify checkpoint?" — run the full verification loop
   - "Review coverage?" — check what is tested and what is missing
