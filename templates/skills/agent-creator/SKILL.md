---
name: agent-creator
description: "Interactive agent builder: guides the user through defining a new agent as a behavioral spec, then generates the agent .md and installs it."
---

# /agent-creator

Build a new toolkit agent interactively. Agents are behavioral specifications — they define how an autonomous specialist thinks, analyzes, and reports. Not steps to execute, but a persona with responsibilities, analysis patterns, and output templates.

## Arguments: $ARGUMENTS

- `<agent-name>` — name for the new agent (e.g., `/agent-creator unreal-developer`)
- `domain:<tech>` — create a domain-specific agent (e.g., `/agent-creator domain:unreal`)
- No args — ask the user what agent they want to create

---

## Goal

Generate a production-ready agent `.md` file following the toolkit's behavioral spec structure, install it into the project's `.claude/agents/` directory (or `templates/agents/domain/<tech>/` for domain agents).

---

## Dependencies

### Tools

- `Read` — read existing agents for pattern reference
- `Write` — create the agent .md file
- `Glob` — find existing agents to avoid collisions and study patterns
- `Bash` — install to project

### Connectors

- None required (fully local)

---

## Context

Agents follow a behavioral specification structure — fundamentally different from skills:

```
Frontmatter:
  name, description, allowed_tools

Body:
  1. Role Definition     — who you ARE (one sentence persona)
  2. Core Responsibilities — what you OWN (3-5 areas, not steps)
  3. Analysis Process     — how to THINK (phases with concrete guidance)
  4. Output Format        — exact template for findings/reports
  5. Behavioral Constraints — scope limits, anti-patterns, hard rules
  6. ECC Enrichments      — optional advanced edge cases
```

**Key distinctions from skills:**
- Agents say "here's how you should think" — skills say "run these steps"
- Agents use `allowed_tools` for access control — skills use APPROVAL GATEs
- Agents output reports/findings — skills output artifacts (code, PRs, files)
- Agents are spawned as subagents — skills are invoked as `/commands`

**Agent categories and their tool access patterns:**

| Category | Tools | Writes Files? |
|----------|-------|---------------|
| Reviewer (code, security, perf) | Read, Glob, Grep, Bash | No |
| Planner (architecture, design) | Read, Glob, Grep, Bash | No |
| Implementer (build-fix, refactor) | Read, Glob, Grep, Bash, Edit, Write | Yes |
| Specialist (incident, domain) | Read, Glob, Grep, Bash + deep-think | Depends |

---

## Process

### Phase 1: Intake — Understand the Agent

1. Parse `$ARGUMENTS` for agent name and type. If empty, ask:
   > "What agent do you want to create? Describe the role in 1-2 sentences."

2. Check for name collisions:
   ```
   Glob: .claude/agents/*.md
   Glob: templates/agents/**/*.md
   ```

3. Ask the user these questions (batch them):

   > **Agent Definition Questionnaire:**
   >
   > 1. **What role does this agent play?** (e.g., "Unreal Engine C++ specialist")
   > 2. **What category?** (reviewer / planner / implementer / specialist)
   > 3. **What are its 3-5 core responsibilities?** (areas it owns)
   > 4. **Should it modify files or only analyze?** (determines tool access)
   > 5. **What does its output look like?** (report, checklist, diff suggestions)
   > 6. **Is it domain-specific?** (tied to a tech stack like Unreal, Unity, Rust)
   > 7. **Does it need deep-think?** (for complex reasoning, cross-module analysis)

4. If domain-specific (`domain:<tech>` argument or user says yes):
   - Ask what tech stack markers to detect (file extensions, config files)
   - Ask for domain-specific checklists or conventions

**APPROVAL GATE**: Present the interpreted agent spec. Wait for confirmation.

---

### Phase 2: Design — Define the Behavioral Spec

1. Read 2-3 existing agents in the same category for reference:
   - Reviewer? Read `code-reviewer.md`, `security-reviewer.md`
   - Implementer? Read `build-error-resolver.md`
   - Specialist? Read any domain agent in `templates/agents/domain/`
   - Planner? Read `planner.md`

2. Design the agent's sections:

   **Role Definition** — one sentence: "You are a [role]. Your job is to [purpose]."

   **Core Responsibilities** — 3-5 bullet points of what the agent owns.

   **allowed_tools** — based on category:
   - Read-only: `[Read, Glob, Grep, Bash]`
   - With LEANN: add `mcp__leann-server__leann_search`
   - With deep-think: add `mcp__deep-think__think`, `mcp__deep-think__reflect`, `mcp__deep-think__strategize`
   - With write access: add `Edit`, `Write`

   **Analysis Process** — 2-5 phases:
   - Phase 1 is always "Gather" (read code, understand context)
   - Middle phases are analysis-specific
   - Final phase is always "Report" (produce output)

   **Output Format** — design the exact report template:
   - Severity levels if reviewer (CRITICAL / HIGH / MEDIUM / LOW / PRAISE)
   - Sections matching the responsibilities
   - Concrete examples of good vs bad findings

   **Behavioral Constraints** — hard limits:
   - What the agent must NOT do
   - Scope boundaries
   - When to escalate vs. handle autonomously

   **ECC Enrichments** — if the agent needs advanced patterns:
   - Edge case handling
   - Framework-specific gotchas
   - Common false positives to avoid

3. Present the full spec outline to the user.

**APPROVAL GATE**: Wait for user to approve. Accept modifications.

---

### Phase 3: Generate — Write the Agent File

Generate the full agent `.md` file:

```markdown
---
name: <agent-name>
description: "<role in one line>"
allowed_tools:
  - Read
  - Glob
  - Grep
  - Bash
  - <additional tools>
---

# <Agent Title>

You are a <role>. Your job is to <purpose>.

## Core Responsibilities

1. <Responsibility 1>
2. <Responsibility 2>
3. <Responsibility 3>

## Analysis Process

### Phase 1: <Gather/Understand>

<steps with specific tool usage, grep patterns, file paths>

### Phase 2: <Analyze/Evaluate>

<domain-specific analysis guidance>

### Phase 3: <Report/Recommend>

<synthesis and output generation>

## Output Format

<exact template with severity levels, sections, metrics>

## Behavioral Constraints

- <constraint 1>
- <constraint 2>
- <scope boundary>

## ECC Enrichments

<advanced patterns, edge cases, special handling>
```

Present the generated content to the user.

**APPROVAL GATE**: User reviews the full agent definition. Accept edits before writing.

---

### Phase 4: Install — Deploy the Agent

1. Determine install location:
   - Generic agent: `.claude/agents/<agent-name>.md`
   - Domain agent: `.claude/agents/<agent-name>.md` (project) AND `templates/agents/domain/<tech>/<agent-name>.md` (toolkit source, if in toolkit repo)

2. Write the file:
   ```
   Write: .claude/agents/<agent-name>.md
   ```

3. If domain-specific, ask if the user wants to add it to the toolkit's detection:
   > "Want to add `<tech>` detection to the installer? This would auto-install this agent for all future `<tech>` projects."

4. Verify:
   ```
   Glob: .claude/agents/<agent-name>.md
   ```

5. Report:
   > "Agent installed:
   >   - `.claude/agents/<agent-name>.md`
   >   - Category: <reviewer/implementer/planner/specialist>
   >   - Tools: <allowed_tools summary>
   >
   > Claude will now use this agent when tasks match its responsibilities."

---

## Error Handling

| Situation | Action |
|-----------|--------|
| Agent name already exists | Warn, offer to overwrite or rename |
| User gives vague role description | Present 2-3 role interpretations |
| Too many responsibilities (>7) | Suggest splitting into two agents |
| Tool access too broad for category | Warn about principle of least privilege |
| Domain tech not in installer detection | Offer to add detection pattern |

## Checklist (Internal — verify before completing)

- [ ] Agent name is unique
- [ ] Frontmatter has `name`, `description`, `allowed_tools`
- [ ] Role definition is one clear sentence
- [ ] Core responsibilities are 3-5 items (areas, not steps)
- [ ] Analysis process has 2-5 phases with concrete guidance
- [ ] Output format is an exact template (not vague)
- [ ] Behavioral constraints define scope boundaries
- [ ] `allowed_tools` follows least-privilege for the category
- [ ] File written to `.claude/agents/<name>.md`
- [ ] Domain detection offered if domain-specific
