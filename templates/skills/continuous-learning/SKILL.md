---
name: continuous-learning
description: "Extract and apply patterns from sessions. Use at end of implementation to capture learnings."
---

# Continuous Learning

Extract reusable patterns, anti-patterns, and decisions from implementation sessions.

## When to Use

- At the end of a feature implementation
- After resolving a non-trivial bug
- After a refactoring session or failed approach that taught something valuable

## What to Capture

### Patterns (Reusable Solutions)

- Architectural patterns applied (e.g., "Repository pattern for DB access")
- Testing strategies that caught real bugs
- Error handling approaches that improved reliability
- Code organization decisions that improved readability

### Anti-Patterns (Mistakes to Avoid)

- Approaches that seemed correct but failed in practice
- Assumptions that turned out to be wrong
- Configurations that caused subtle bugs
- Test patterns that produced flaky results

### Decisions (Context-Dependent Choices)

- Why one approach was chosen over alternatives
- Trade-offs accepted and their justification
- What would trigger revisiting this decision

## Learning Format

```
### [Category]: [Short Title]

**Context:** When/where this applies
**Learning:** What was discovered
**Evidence:** What proved this (test result, error, metric)
**Action:** What to do differently next time
```

## Where to Store Learnings

**Project-level**: Add to `progress.txt` under `## Codebase Patterns` -- stack-specific patterns, framework gotchas, naming conventions.

**Session-level**: Append to `progress.txt` under the session entry -- files changed, problems encountered, how they were resolved.

**Knowledge base**: If Obsidian or similar is configured -- architecture decisions, language insights, and debugging techniques that apply across projects.

## How to Apply Learnings

1. **At session start**: Read `## Codebase Patterns` in `progress.txt` before implementing
2. **During implementation**: Note patterns immediately; reference known anti-patterns to avoid them
3. **At session end**: Run the extraction checklist below

## Extraction Checklist

- [ ] Did I discover a reusable pattern? Record it.
- [ ] Did I hit a problem that cost significant time? Record the anti-pattern.
- [ ] Did I make a non-obvious decision? Record the context and reasoning.
- [ ] Did I learn something about the framework or language? Record it.
- [ ] Would a future session benefit from knowing this? If yes, record it.

## Quality Criteria

Good learnings are:

- **Specific** -- includes the exact context where it applies
- **Actionable** -- tells you what to do, not just what happened
- **Evidenced** -- based on an observed outcome, not a guess
- **Discoverable** -- stored where future sessions will naturally find it
