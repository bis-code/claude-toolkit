---
name: skill-creator
description: "Interactive skill builder: guides the user through defining a new skill as a structured SOP, then generates the SKILL.md and installs it."
---

# /skill-creator

Build a new toolkit skill interactively. Each skill is a mini SOP (Standard Operating Procedure) — scoped, self-contained, with clear inputs, process, human gates, and defined output.

## Arguments: $ARGUMENTS

- `<skill-name>` — name for the new skill (e.g., `/skill-creator api-designer`)
- No args — ask the user what skill they want to create

---

## Goal

Generate a production-ready `SKILL.md` file that follows the toolkit's SOP structure, install it into the project's `.claude/skills/` directory, and optionally register it as a global slash command.

---

## Dependencies

### Tools

- `Read` — read existing skills for pattern reference
- `Write` — create the SKILL.md file
- `Bash` — install to project, optionally register global command
- `Glob` — find existing skills to avoid name collisions

### Connectors

- None required (fully local)

---

## Context

Skills follow a strict SOP structure:

```
1. Name & Trigger    — skill-name + when Claude should activate
2. Goal              — one clear desired-outcome
3. Dependencies      — tools, connectors, MCP servers needed
4. Context           — background knowledge Claude needs
5. Process           — step-by-step with human checkpoints (APPROVAL GATEs)
6. Output            — where result goes + what the final deliverable is
```

**Critical rules:**
- Skills are NOT fully autonomous — they MUST have at least one human checkpoint
- Each phase should be a discrete, reviewable unit of work
- Error handling must be explicit (table of situations → actions)
- An internal checklist at the end ensures nothing is missed

---

## Process

### Phase 1: Intake — Understand the Skill

1. Parse `$ARGUMENTS` for skill name. If empty, ask:
   > "What skill do you want to create? Describe what it should do in 1-2 sentences."

2. Check for name collisions:
   ```
   Glob: .claude/skills/*/SKILL.md
   ```
   If a skill with the same name exists, warn and ask to proceed or rename.

3. Ask the user these questions (batch them, don't ask one at a time):

   > **Skill Definition Questionnaire:**
   >
   > 1. **What does this skill do?** (one sentence — this becomes the Goal)
   > 2. **When should it trigger?** (e.g., "when the user types /foo", "when reviewing PRs")
   > 3. **What inputs does it need?** (arguments, files, context)
   > 4. **What does it produce?** (file, PR, report, code changes)
   > 5. **Are there external tools needed?** (MCP servers, APIs, CLIs)
   > 6. **How many phases?** (suggest 3-6 based on complexity)
   > 7. **Where should the user approve/review?** (which phase boundaries)

4. If the user's description is vague, present 2-3 structured interpretations:
   > "I see a few ways to interpret this. Which fits best?"

**APPROVAL GATE**: Present the interpreted spec back to the user. Wait for confirmation.

---

### Phase 2: Design — Structure the Phases

1. Based on the intake answers, design the phase breakdown:
   - Each phase gets: name, numbered steps, and clear completion criteria
   - Insert `**APPROVAL GATE**` markers where the user indicated review points
   - At minimum: one gate after planning, one before final output

2. For each phase, determine:
   - What tools/commands are needed
   - What information flows from the previous phase
   - What can fail and how to handle it

3. Design the error handling table:
   | Situation | Action |
   |-----------|--------|

4. Design the internal checklist (verification steps before the skill completes)

5. Present the full phase outline to the user:
   > **Proposed Structure:**
   > - Phase 1: [name] — [what it does]
   >   - APPROVAL GATE after this phase
   > - Phase 2: [name] — [what it does]
   > - Phase 3: [name] — [what it does]
   >   - APPROVAL GATE after this phase
   > - Output: [deliverable] → [location]

**APPROVAL GATE**: Wait for user to approve the structure. Accept modifications.

---

### Phase 3: Generate — Write the SKILL.md

1. Read 1-2 existing skills for tone and formatting reference:
   ```
   Read: .claude/skills/plan/SKILL.md
   Read: .claude/skills/code-review/SKILL.md
   ```

2. Generate the full SKILL.md following this template:

   ```markdown
   ---
   name: <skill-name>
   description: "<one-line description>"
   ---

   # /<skill-name>

   <2-3 sentence overview>

   ## Arguments: $ARGUMENTS

   - `<arg>` — description
   - No args — default behavior

   ---

   ## Goal

   <One clear desired-outcome sentence>

   ---

   ## Dependencies

   ### Tools
   - `<Tool>` — why needed

   ### Connectors
   - `<MCP server or API>` — why needed
   - None required (if none)

   ---

   ## Context

   <Background knowledge Claude needs to execute this skill properly>

   ---

   ## Phase 1: <Name>

   1. Step one
   2. Step two
   3. Present findings to user

   **APPROVAL GATE**: <what the user reviews>

   ---

   ## Phase 2: <Name>

   ...

   ---

   ## Error Handling

   | Situation | Action |
   |-----------|--------|
   | <situation> | <action> |

   ## Checklist (Internal — verify before completing)

   - [ ] Item 1
   - [ ] Item 2
   ```

3. Present the generated SKILL.md content to the user for review.

**APPROVAL GATE**: User reviews the full SKILL.md. Accept edits before writing.

---

### Phase 4: Install — Deploy the Skill

1. Write the file:
   ```
   Write: .claude/skills/<skill-name>/SKILL.md
   ```

2. Ask the user if they also want a global slash command:
   > "Want to register `/<skill-name>` as a global command? (works across all projects)"

3. If yes, create the command file:
   ```bash
   # Template for ~/.claude/commands/<skill-name>.md
   Read and execute the skill defined in .claude/skills/<skill-name>/SKILL.md

   User arguments: $ARGUMENTS
   ```

4. Verify the install:
   ```
   Glob: .claude/skills/<skill-name>/SKILL.md
   ```

5. Report to user:
   > "Skill installed:
   >   - `.claude/skills/<skill-name>/SKILL.md`
   >   - Global command: `/<skill-name>` (if registered)
   >
   > Try it: `/<skill-name> <example-args>`"

---

## Error Handling

| Situation | Action |
|-----------|--------|
| Skill name already exists | Warn user, offer to overwrite or rename |
| User gives vague description | Present 2-3 interpretations to choose from |
| No `.claude/skills/` directory | Create it |
| Generated skill is too complex (>6 phases) | Suggest splitting into two skills |
| User wants to edit after install | Re-read, apply edits, re-write |

## Checklist (Internal — verify before completing)

- [ ] Skill name is unique (no collision)
- [ ] SKILL.md follows SOP structure (name, goal, deps, context, process, output)
- [ ] At least one APPROVAL GATE exists in the process
- [ ] Error handling table is populated
- [ ] Internal checklist is populated
- [ ] File written to `.claude/skills/<name>/SKILL.md`
- [ ] Global command registered (if user requested)
- [ ] User shown how to invoke the new skill
