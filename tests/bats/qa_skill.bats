#!/usr/bin/env bats

load test_helper/common

setup() {
  setup_temp_project
}

teardown() {
  teardown_temp_project
}

# ── SKILL.md exists with correct frontmatter ──

@test "qa SKILL.md exists in templates" {
  assert_file_exists "$TOOLKIT_ROOT/templates/skills/qa/SKILL.md"
}

@test "qa SKILL.md has frontmatter with name: qa" {
  local skill_file="$TOOLKIT_ROOT/templates/skills/qa/SKILL.md"
  grep -q "^name: qa" "$skill_file"
}

@test "qa SKILL.md has description in frontmatter" {
  local skill_file="$TOOLKIT_ROOT/templates/skills/qa/SKILL.md"
  grep -q "^description:" "$skill_file"
}

# ── Agent references (6 scan agents) ──

@test "qa SKILL.md references refactor-cleaner agent" {
  assert_file_contains "$TOOLKIT_ROOT/templates/skills/qa/SKILL.md" "refactor-cleaner"
}

@test "qa SKILL.md references code-reviewer agent" {
  assert_file_contains "$TOOLKIT_ROOT/templates/skills/qa/SKILL.md" "code-reviewer"
}

@test "qa SKILL.md references security-reviewer agent" {
  assert_file_contains "$TOOLKIT_ROOT/templates/skills/qa/SKILL.md" "security-reviewer"
}

@test "qa SKILL.md references planner agent" {
  assert_file_contains "$TOOLKIT_ROOT/templates/skills/qa/SKILL.md" "planner"
}

@test "qa SKILL.md references doc-updater agent" {
  assert_file_contains "$TOOLKIT_ROOT/templates/skills/qa/SKILL.md" "doc-updater"
}

@test "qa SKILL.md references build-error-resolver agent" {
  assert_file_contains "$TOOLKIT_ROOT/templates/skills/qa/SKILL.md" "build-error-resolver"
}

# ── Three phases ──

@test "qa SKILL.md mentions Phase 1 / scan" {
  assert_file_contains "$TOOLKIT_ROOT/templates/skills/qa/SKILL.md" "Phase 1"
  assert_file_contains "$TOOLKIT_ROOT/templates/skills/qa/SKILL.md" "[Ss]can"
}

@test "qa SKILL.md mentions Phase 2 / triage" {
  assert_file_contains "$TOOLKIT_ROOT/templates/skills/qa/SKILL.md" "Phase 2"
  assert_file_contains "$TOOLKIT_ROOT/templates/skills/qa/SKILL.md" "[Tt]riage"
}

@test "qa SKILL.md mentions Phase 3 / fix" {
  assert_file_contains "$TOOLKIT_ROOT/templates/skills/qa/SKILL.md" "Phase 3"
  assert_file_contains "$TOOLKIT_ROOT/templates/skills/qa/SKILL.md" "[Ff]ix"
}

# ── Orchestration features ──

@test "qa SKILL.md mentions approval gate" {
  assert_file_contains "$TOOLKIT_ROOT/templates/skills/qa/SKILL.md" "[Aa]pproval"
}

@test "qa SKILL.md mentions severity levels" {
  assert_file_contains "$TOOLKIT_ROOT/templates/skills/qa/SKILL.md" "CRITICAL"
  assert_file_contains "$TOOLKIT_ROOT/templates/skills/qa/SKILL.md" "HIGH"
  assert_file_contains "$TOOLKIT_ROOT/templates/skills/qa/SKILL.md" "MEDIUM"
  assert_file_contains "$TOOLKIT_ROOT/templates/skills/qa/SKILL.md" "LOW"
}

@test "qa SKILL.md mentions --auto flag" {
  assert_file_contains "$TOOLKIT_ROOT/templates/skills/qa/SKILL.md" "\-\-auto"
}

@test "qa SKILL.md mentions --scan-only flag" {
  assert_file_contains "$TOOLKIT_ROOT/templates/skills/qa/SKILL.md" "\-\-scan-only"
}

@test "qa SKILL.md mentions --scope flag" {
  assert_file_contains "$TOOLKIT_ROOT/templates/skills/qa/SKILL.md" "\-\-scope"
}

@test "qa SKILL.md mentions --focus flag" {
  assert_file_contains "$TOOLKIT_ROOT/templates/skills/qa/SKILL.md" "\-\-focus"
}

@test "qa SKILL.md mentions Task tool" {
  assert_file_contains "$TOOLKIT_ROOT/templates/skills/qa/SKILL.md" "Task"
}

@test "qa SKILL.md mentions GitHub issue creation" {
  assert_file_contains "$TOOLKIT_ROOT/templates/skills/qa/SKILL.md" "gh issue create"
}

@test "qa SKILL.md mentions domain agent discovery" {
  assert_file_contains "$TOOLKIT_ROOT/templates/skills/qa/SKILL.md" "[Dd]omain agent"
}

@test "qa SKILL.md mentions qa-report.json output" {
  assert_file_contains "$TOOLKIT_ROOT/templates/skills/qa/SKILL.md" "qa-report.json"
}

@test "qa SKILL.md mentions branch isolation before fixes" {
  assert_file_contains "$TOOLKIT_ROOT/templates/skills/qa/SKILL.md" "qa/"
  assert_file_contains "$TOOLKIT_ROOT/templates/skills/qa/SKILL.md" "[Bb]ranch isolation"
}

# ── Old files removed ──

@test "commands/qa.md no longer exists" {
  [ ! -f "$TOOLKIT_ROOT/commands/qa.md" ]
}

@test "tools/qa/ directory no longer exists" {
  [ ! -d "$TOOLKIT_ROOT/tools/qa" ]
}

@test "templates/qa/ directory no longer exists" {
  [ ! -d "$TOOLKIT_ROOT/templates/qa" ]
}
