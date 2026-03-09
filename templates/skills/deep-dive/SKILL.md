---
name: deep-dive
description: "Cross-repo investigation: research planning issues, break down into sub-issues, generate ralph session prompts per repo."
---

# /deep-dive

Comprehensive investigation workflow for planning issues that span multiple repositories. Produces actionable sub-issues and ralph session prompts ready to copy-paste into per-repo Claude sessions.

## Arguments: $ARGUMENTS

- `<number>` — planning issue number (e.g., `/deep-dive 9`)
- `<url>` — full GitHub issue URL
- No args — ask the user which planning issue to investigate

## Overview

```
/deep-dive 9
    ↓
Phase 1: Fetch & Understand the planning issue
    ↓
Phase 2: Parallel investigation (sub-agents per repo)
    ↓
Phase 3: Synthesize findings & identify sub-issues
    ↓
Phase 4: Create GitHub sub-issues in each affected repo
    ↓
Phase 5: Update planning issue with findings + ralph session guide
    ↓
Phase 6: Output ralph session prompts
```

---

## Workspace Context

This skill reads workspace configuration from `.claude-workspace.json` in the current directory (or auto-detects using the `toolkit__get_workspace` MCP tool).

### Workspace Discovery

1. Call `toolkit__get_workspace` with the current directory
2. This returns: repos (path, branch, type), planning_repo, dependency_order, domain_labels, shared dirs
3. For each repo, read its `.claude-toolkit.json` for test/lint commands and MCP server info

If no workspace config exists:
- Ask the user which repos are involved
- Fall back to manual configuration

---

## Phase 1: Fetch & Understand

1. Parse `$ARGUMENTS` to get the issue number or URL
2. Read workspace config via `toolkit__get_workspace`
3. Fetch the planning issue:
   ```bash
   gh issue view <number> --repo <planning_repo> --json number,title,body,labels,milestone
   ```
   If no `planning_repo` in config, ask the user which repo holds the planning issue.
4. Identify **affected repositories** from the issue body (look for repo links or "Affected repositories" section)
5. For each affected repo, read `.claude-toolkit.json`:
   ```bash
   cat <repo-path>/.claude-toolkit.json
   ```
6. Check for **in-flight branches** in each affected repo:
   ```bash
   cd <repo-path> && git branch -a | grep -i "<issue-slug-or-number>"
   ```
7. Present a brief summary to the user:
   - Issue title and current description quality
   - Repos identified + their tech stacks (from toolkit config)
   - Any in-flight work found
   - Ask: "Ready to investigate? Any specific areas to focus on?"

**APPROVAL GATE**: Wait for user confirmation before spawning agents.

---

## Phase 2: Parallel Investigation

Spawn one `Task(subagent_type="Explore")` agent per affected repository. Each agent gets:

1. The **issue context** (title, body, acceptance criteria)
2. The **repo's toolkit config** (test command, tech stack, LEANN indexes)
3. **Specific investigation questions**:
   - What is the current implementation state?
   - What code paths are involved?
   - Are there existing tests?
   - What patterns does similar working code follow?
   - Are there deployment/config prerequisites?

### Agent Prompt Template

```
## Investigation Task

**Planning Issue**: #<number> — <title>
**Repository**: <repo-name>
**Tech Stack**: <from .claude-toolkit.json or workspace config>
**Test Command**: <from .claude-toolkit.json>

**Issue Context**:
<issue body summary>

**Your mission**: Investigate the <repo-name> repository for this issue.

Answer these questions:
1. What is the current state of implementation for this feature?
2. Which files/classes are involved? List specific paths.
3. Are there existing tests covering this area?
4. What does a working similar feature look like? (for comparison/pattern reference)
5. Are there any deployment prerequisites (contracts deployed, configs, env vars)?
6. What are the specific bugs or gaps you can identify?

**Important**: Check for in-flight branches that may already address parts of this:
```bash
git branch -a | grep -i "<relevant-keywords>"
```

Return a structured report with sections for each question.
```

### Domain-Specific Checks

Domain checks are activated based on the repo's tech stack from `.claude-toolkit.json`:

| Tech Stack | Check |
|-----------|-------|
| `solidity` / `blockchain` | Check deployed contract addresses, compare contract ABI vs calling code |
| `unity` / `csharp` | Check for NuGet/package sync needs between repos |
| `typescript` / `react` | Check API endpoint compatibility with backend changes |
| `go` | Check protobuf/gRPC contract compatibility |
| `rust` | Check shared crate version compatibility |

If no domain-specific checks apply, skip this section.

---

## Phase 3: Synthesize Findings

After all agents return:

1. **Merge findings** into a unified picture:
   - Current state per repo
   - Root cause analysis (if bug)
   - Gap analysis (if feature)
   - Cross-repo dependencies

2. **Identify sub-issues** — break down into concrete, implementable units:
   - Each sub-issue should be completable in one ralph session
   - Group by repository
   - Order by dependency (use `dependency_order` from workspace config)
   - Each needs: title, description, acceptance criteria, affected files

3. **Determine dependency chain**:
   - Which repo must go first? (use `dependency_order` from config)
   - Are there package sync steps between repos?
   - Can any issues be parallelized?

4. **Check for in-flight work overlap**:
   - If branches exist, note what they already cover
   - Reduce sub-issue scope accordingly

5. Present findings to the user:
   - Root cause / gap summary
   - Proposed sub-issue breakdown
   - Dependency chain
   - Ask: "Does this breakdown look right? Any adjustments?"

**APPROVAL GATE**: Wait for user to approve the sub-issue breakdown.

---

## Phase 4: Create GitHub Sub-Issues

For each approved sub-issue, create it in the appropriate repository:

```bash
gh issue create \
  --repo <org>/<repo> \
  --title "<type>: <description>" \
  --body "$(cat <<'EOF'
## Context

Sub-issue of <planning_repo>#<parent-number>

<description with context from investigation>

## Acceptance Criteria

- [ ] Criterion 1
- [ ] Criterion 2

## Affected Files

- `path/to/file1`
- `path/to/file2`

## Technical Notes

<implementation hints from investigation>

## Dependencies

- <dependency on other sub-issues if any>
EOF
)" \
  --label "claude-ready"
```

Use `domain_labels` from workspace config for additional labels.

---

## Phase 5: Update Planning Issue

Update the parent planning issue with investigation results:

```bash
gh issue edit <number> --repo <planning_repo> --body "$(cat <<'EOF'
<original issue body>

---

## Investigation Results (<date>)

### Summary
<findings overview>

### Sub-Issues (Execution Order)

<grouped by repo, ordered by dependency_order>

### Dependency Chain
<visual dependency graph>
EOF
)"
```

---

## Phase 6: Output Ralph Session Prompts

Generate copy-paste-ready prompts for each repo's Claude session, **ordered by `dependency_order`** from workspace config.

### Prompt Structure

For each affected repo:

````
### <Repo Name> Ralph Session

**Run from**: `cd <repo-path>`
**Command**: `/ralph --issues <comma-separated-issue-numbers>`

<If this repo depends on another repo's output>
**Prerequisite**: <dependent-repo> PR must be merged first.
</If>

**Context prompt**:

```
You are working on <planning_repo>#<N>: <title>.

## Issues to implement (in order):

1. **#<issue-number>: <title>**
   - <acceptance criteria summary>
   - Key files: <files>

## Important context from investigation:
- <key finding 1>
- <key finding 2>

## Test command: <from .claude-toolkit.json>
## Lint command: <from .claude-toolkit.json>
```
````

---

## Error Handling

| Situation | Action |
|-----------|--------|
| No workspace config found | Ask user which repos are involved |
| Planning issue not found | Ask user for correct repo/number |
| `.claude-toolkit.json` missing in a repo | Fall back to workspace config for tech stack |
| Agent returns insufficient info | Run targeted follow-up search |
| In-flight branch conflicts with plan | Note overlap, reduce sub-issue scope |

## Checklist (Internal — verify before completing)

- [ ] Workspace config loaded (auto-detect or .claude-workspace.json)
- [ ] All affected repos investigated
- [ ] `.claude-toolkit.json` read for each repo
- [ ] Root cause or gap clearly identified
- [ ] Sub-issues created with acceptance criteria and `claude-ready` label
- [ ] Planning issue updated with findings + session guide
- [ ] Ralph prompts output for each repo (in dependency order)
- [ ] Dependency chain is clear and documented
