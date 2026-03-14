---
name: rule-creator
description: "Interactive rule builder: guides the user through defining a new rule as a constraint doc, then generates the rule .md and installs it."
---

# /rule-creator

Build a new toolkit rule interactively. Rules are always-on constraints that shape Claude's behavior — coding conventions, patterns to follow, anti-patterns to avoid, and domain-specific guidelines.

## Arguments: $ARGUMENTS

- `<rule-name>` — name for the new rule (e.g., `/rule-creator unreal-cpp-conventions`)
- `domain:<tech>` — create rules for a tech stack (e.g., `/rule-creator domain:unreal`)
- No args — ask the user what rule they want to create

---

## Goal

Generate a production-ready rule `.md` file that Claude loads as always-on context, install it into the project's `.claude/rules/` directory.

---

## Dependencies

### Tools

- `Read` — read existing rules for pattern reference
- `Write` — create the rule .md file
- `Glob` — find existing rules to study structure and avoid duplication

### Connectors

- None required (fully local)

---

## Context

Rules are **constraint documents** — they tell Claude what to do and what NOT to do, always. They're loaded into context at conversation start and apply to every response.

**Rule structure:**

```
# <Title>

<Brief purpose — one sentence>

## <Section>

- Directive 1
- Directive 2

## <Section>

| Pattern | Do This | Not This |
|---------|---------|----------|
```

**Key characteristics:**
- Rules are prescriptive — "always do X", "never do Y"
- Rules use tables for quick scanning (do/don't, pattern/example)
- Rules are concise — Claude reads ALL rules every conversation, so brevity matters
- Rules don't have phases or gates — they're reference material, not procedures

**Rule categories:**

| Category | Location | Example |
|----------|----------|---------|
| Common | `.claude/rules/common/` | `coding-style.md`, `testing.md`, `security.md` |
| Language | `.claude/rules/<lang>/` | `golang/error-handling.md`, `typescript/types.md` |
| Conditional | `.claude/rules/conditional/` | `read-only.md` (only when flag set) |

**Existing common rules** (avoid duplicating these):
- `coding-style.md` — naming, functions, files, readability, DRY
- `testing.md` — TDD, test pyramid, test quality
- `security.md` — input validation, data access, secrets, OWASP
- `performance.md` — DB queries, API, caching, frontend
- `git-workflow.md` — commits, branches, PRs
- `search-strategy.md` — LEANN, Grep, Glob priority
- `agents.md` — when/how to use agents
- `hooks.md` — hook system behavior
- `patterns.md` — ECC patterns
- `development-workflow.md` — workflow phases
- `context-efficiency.md` — context window management

---

## Process

### Phase 1: Intake — Understand the Rule

1. Parse `$ARGUMENTS`. If empty, ask:
   > "What rule do you want to create? Describe the conventions or constraints in 1-2 sentences."

2. Scan existing rules to avoid duplication:
   ```
   Glob: .claude/rules/**/*.md
   ```
   Read any rules that overlap with the user's intent. If overlap exists, suggest updating the existing rule instead.

3. Ask the user:

   > **Rule Definition Questionnaire:**
   >
   > 1. **What should Claude always do?** (3-10 directives)
   > 2. **What should Claude never do?** (anti-patterns)
   > 3. **Is this for a specific tech stack?** (determines location: common/ vs <lang>/)
   > 4. **Can you give do/don't examples?** (concrete patterns)
   > 5. **Is this conditional?** (only applies in certain modes/contexts)

4. If domain-specific (`domain:<tech>`), research the tech stack:
   - Ask for key conventions (naming, file structure, macros, patterns)
   - Ask for common mistakes to prevent
   - Ask for framework-specific best practices

**APPROVAL GATE**: Present the interpreted rule scope. Wait for confirmation.

---

### Phase 2: Design — Structure the Rule

1. Read 2-3 existing rules for tone and density reference:
   ```
   Read: .claude/rules/common/coding-style.md
   Read: .claude/rules/common/security.md
   ```

2. Determine rule category and location:
   - Universal convention → `.claude/rules/common/<name>.md`
   - Language/framework specific → `.claude/rules/<tech>/<name>.md`
   - Context-dependent → `.claude/rules/conditional/<name>.md`

3. Structure the rule sections:
   - Group directives by topic (naming, structure, patterns, anti-patterns)
   - Use **tables** for do/don't comparisons
   - Use **bullet lists** for directives
   - Use **code blocks** for concrete examples
   - Keep it **scannable** — Claude reads this every conversation

4. Estimate rule size:
   - Target: 30-100 lines (rules should be concise)
   - If >150 lines: suggest splitting into multiple rules
   - If <10 lines: may be too thin — suggest merging with an existing rule

5. Present the outline to the user.

**APPROVAL GATE**: Wait for user to approve the structure.

---

### Phase 3: Generate — Write the Rule

Generate the rule `.md` following this pattern:

```markdown
# <Title>

<One-sentence purpose>

## <Topic Section>

- Directive using imperative voice ("Use X", "Avoid Y", "Always Z")
- Directive with rationale where non-obvious

## <Patterns Section>

| Pattern | Do This | Not This |
|---------|---------|----------|
| <case> | `good example` | `bad example` |

## <Anti-Patterns Section>

- Never <anti-pattern> — <why>
- Avoid <anti-pattern> — <consequence>

## <Framework-Specific Section> (if domain rule)

### <Subsystem>

- Convention 1
- Convention 2

```

**Writing principles:**
- Imperative voice: "Use", "Avoid", "Always", "Never" — not "You should"
- One directive per bullet — scannable, not paragraph-dense
- Tables for comparisons — faster to parse than prose
- Code examples for anything non-obvious
- No filler — every line earns its place in the context window

Present the generated rule to the user.

**APPROVAL GATE**: User reviews the full rule. Accept edits before writing.

---

### Phase 4: Install — Deploy the Rule

1. Determine install path:
   - Common: `.claude/rules/common/<name>.md`
   - Language: `.claude/rules/<tech>/<name>.md`
   - Conditional: `.claude/rules/conditional/<name>.md`

2. Create the directory if needed and write the file.

3. If this is a new language/tech directory, ask:
   > "Want to add this to the toolkit templates so future projects get these rules automatically?"

   If yes, also write to `templates/rules/<tech>/<name>.md` (if in toolkit repo).

4. Verify:
   ```
   Glob: .claude/rules/**/<name>.md
   ```

5. Report:
   > "Rule installed:
   >   - `.claude/rules/<category>/<name>.md`
   >   - Type: <common/language/conditional>
   >   - Directives: <count>
   >
   > This rule is now active in every Claude session in this project."

---

## Error Handling

| Situation | Action |
|-----------|--------|
| Rule name already exists | Warn, offer to update or rename |
| Overlaps with existing rule | Show overlap, suggest merging |
| Rule too long (>150 lines) | Suggest splitting by topic |
| Rule too short (<10 lines) | Suggest merging with related rule |
| User gives vague conventions | Ask for concrete do/don't examples |
| Tech stack not in installer | Offer to add detection + rule templates |

## Checklist (Internal — verify before completing)

- [ ] No duplication with existing rules
- [ ] Correct directory (common/ vs <tech>/ vs conditional/)
- [ ] Imperative voice throughout ("Use X" not "You should use X")
- [ ] Tables for do/don't comparisons
- [ ] Concise — 30-100 lines, every line earns its context window space
- [ ] Code examples for non-obvious patterns
- [ ] File written and verified
- [ ] Toolkit templates updated (if user requested)
