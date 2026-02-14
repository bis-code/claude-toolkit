#!/usr/bin/env bats

load test_helper/common

setup() {
  setup_temp_project
}

teardown() {
  teardown_temp_project
}

# ── SKILL.md exists with correct frontmatter ──

@test "ralph SKILL.md exists in templates" {
  assert_file_exists "$TOOLKIT_ROOT/templates/skills/ralph/SKILL.md"
}

@test "ralph SKILL.md has frontmatter with name: ralph" {
  local skill_file="$TOOLKIT_ROOT/templates/skills/ralph/SKILL.md"
  grep -q "^name: ralph" "$skill_file"
}

@test "ralph SKILL.md has description in frontmatter" {
  local skill_file="$TOOLKIT_ROOT/templates/skills/ralph/SKILL.md"
  grep -q "^description:" "$skill_file"
}

# ── Agent references ──

@test "ralph SKILL.md references planner agent" {
  assert_file_contains "$TOOLKIT_ROOT/templates/skills/ralph/SKILL.md" "planner"
}

@test "ralph SKILL.md references tdd-guide agent" {
  assert_file_contains "$TOOLKIT_ROOT/templates/skills/ralph/SKILL.md" "tdd-guide"
}

@test "ralph SKILL.md references code-reviewer agent" {
  assert_file_contains "$TOOLKIT_ROOT/templates/skills/ralph/SKILL.md" "code-reviewer"
}

@test "ralph SKILL.md references security-reviewer agent" {
  assert_file_contains "$TOOLKIT_ROOT/templates/skills/ralph/SKILL.md" "security-reviewer"
}

# ── Orchestration features ──

@test "ralph SKILL.md mentions approval gate" {
  assert_file_contains "$TOOLKIT_ROOT/templates/skills/ralph/SKILL.md" "[Aa]pproval"
}

@test "ralph SKILL.md mentions prd.json" {
  assert_file_contains "$TOOLKIT_ROOT/templates/skills/ralph/SKILL.md" "prd.json"
}

@test "ralph SKILL.md mentions --auto flag" {
  assert_file_contains "$TOOLKIT_ROOT/templates/skills/ralph/SKILL.md" "\-\-auto"
}

@test "ralph SKILL.md mentions domain agent discovery" {
  assert_file_contains "$TOOLKIT_ROOT/templates/skills/ralph/SKILL.md" "[Dd]omain [Aa]gent"
}

@test "ralph SKILL.md mentions progress.txt" {
  assert_file_contains "$TOOLKIT_ROOT/templates/skills/ralph/SKILL.md" "progress.txt"
}

@test "ralph SKILL.md mentions Task tool" {
  assert_file_contains "$TOOLKIT_ROOT/templates/skills/ralph/SKILL.md" "Task"
}

# ── Old files removed ──

@test "commands/ralph.md no longer exists" {
  [ ! -f "$TOOLKIT_ROOT/commands/ralph.md" ]
}

@test "ralph.sh no longer exists" {
  [ ! -f "$TOOLKIT_ROOT/tools/ralph/ralph.sh" ]
}

@test "RALPH.md no longer exists" {
  [ ! -f "$TOOLKIT_ROOT/tools/ralph/RALPH.md" ]
}
