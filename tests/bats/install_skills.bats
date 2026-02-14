#!/usr/bin/env bats

load test_helper/common

setup() {
  setup_temp_project
  source_lib "utils.sh"
  source_lib "install_skills.sh"
}

teardown() {
  teardown_temp_project
}

# ── Expected skill set ──

EXPECTED_SKILLS=(
  "build-fix"
  "code-review"
  "docs"
  "plan"
  "refactor-clean"
  "security-review"
  "tdd-workflow"
)

DELETED_SKILLS=(
  "coding-standards"
  "continuous-learning"
  "verification-loop"
)

# Maps skill name → paired agent name
declare -A SKILL_AGENT_MAP
SKILL_AGENT_MAP=(
  ["build-fix"]="build-error-resolver"
  ["code-review"]="code-reviewer"
  ["docs"]="doc-updater"
  ["plan"]="planner"
  ["refactor-clean"]="refactor-cleaner"
  ["security-review"]="security-reviewer"
  ["tdd-workflow"]="tdd-guide"
)

# ── Installation: each skill installs correctly ──

@test "install_skills: installs all 7 expected skills" {
  local templates_dir="$TOOLKIT_ROOT/templates"
  install_skills "$TEST_PROJECT_DIR" "$templates_dir"

  for skill in "${EXPECTED_SKILLS[@]}"; do
    assert_dir_exists "$TEST_PROJECT_DIR/.claude/skills/$skill"
    assert_file_exists "$TEST_PROJECT_DIR/.claude/skills/$skill/SKILL.md"
  done
}

@test "install_skills: installs exactly 7 skills (no extras)" {
  local templates_dir="$TOOLKIT_ROOT/templates"
  install_skills "$TEST_PROJECT_DIR" "$templates_dir"

  local count
  count=$(find "$TEST_PROJECT_DIR/.claude/skills" -mindepth 1 -maxdepth 1 -type d | wc -l | tr -d ' ')
  [ "$count" -eq 7 ]
}

# ── Deleted skills no longer ship ──

@test "install_skills: coding-standards skill does not exist in templates" {
  [ ! -d "$TOOLKIT_ROOT/templates/skills/coding-standards" ]
}

@test "install_skills: continuous-learning skill does not exist in templates" {
  [ ! -d "$TOOLKIT_ROOT/templates/skills/continuous-learning" ]
}

@test "install_skills: verification-loop skill does not exist in templates" {
  [ ! -d "$TOOLKIT_ROOT/templates/skills/verification-loop" ]
}

@test "install_skills: deleted skills are not installed" {
  local templates_dir="$TOOLKIT_ROOT/templates"
  install_skills "$TEST_PROJECT_DIR" "$templates_dir"

  for skill in "${DELETED_SKILLS[@]}"; do
    [ ! -d "$TEST_PROJECT_DIR/.claude/skills/$skill" ]
  done
}

# ── Frontmatter: each SKILL.md has name field ──

@test "each SKILL.md has frontmatter with name field" {
  for skill in "${EXPECTED_SKILLS[@]}"; do
    local skill_file="$TOOLKIT_ROOT/templates/skills/$skill/SKILL.md"
    assert_file_exists "$skill_file"
    # Check for YAML frontmatter with name field
    grep -q "^name:" "$skill_file" || {
      echo "Missing 'name:' in frontmatter of $skill/SKILL.md" >&2
      return 1
    }
  done
}

# ── Agent pairing: each SKILL.md references its agent ──

@test "each SKILL.md references its paired agent" {
  for skill in "${EXPECTED_SKILLS[@]}"; do
    local agent="${SKILL_AGENT_MAP[$skill]}"
    local skill_file="$TOOLKIT_ROOT/templates/skills/$skill/SKILL.md"
    assert_file_exists "$skill_file"
    grep -q "$agent" "$skill_file" || {
      echo "$skill/SKILL.md does not reference agent '$agent'" >&2
      return 1
    }
  done
}

# ── Skill template pattern: each SKILL.md mentions Task tool ──

@test "each SKILL.md instructs spawning via Task tool" {
  for skill in "${EXPECTED_SKILLS[@]}"; do
    local skill_file="$TOOLKIT_ROOT/templates/skills/$skill/SKILL.md"
    grep -qi "task" "$skill_file" || {
      echo "$skill/SKILL.md does not mention Task tool for agent spawning" >&2
      return 1
    }
  done
}

# ── Frontmatter name matches directory name ──

# ── Update mode: deprecated skills removed ──

@test "update mode removes deprecated skill directories" {
  local templates_dir="$TOOLKIT_ROOT/templates"
  # Simulate old installation with deprecated skills
  install_skills "$TEST_PROJECT_DIR" "$templates_dir"
  mkdir -p "$TEST_PROJECT_DIR/.claude/skills/coding-standards"
  echo "old" > "$TEST_PROJECT_DIR/.claude/skills/coding-standards/SKILL.md"
  mkdir -p "$TEST_PROJECT_DIR/.claude/skills/verification-loop"
  echo "old" > "$TEST_PROJECT_DIR/.claude/skills/verification-loop/SKILL.md"

  # Update mode should clean them up
  MODE=update install_skills "$TEST_PROJECT_DIR" "$templates_dir"

  [ ! -d "$TEST_PROJECT_DIR/.claude/skills/coding-standards" ]
  [ ! -d "$TEST_PROJECT_DIR/.claude/skills/verification-loop" ]
  # Valid skills still exist
  assert_dir_exists "$TEST_PROJECT_DIR/.claude/skills/code-review"
  assert_dir_exists "$TEST_PROJECT_DIR/.claude/skills/tdd-workflow"
}

@test "normal install does not remove extra skill directories" {
  local templates_dir="$TOOLKIT_ROOT/templates"
  install_skills "$TEST_PROJECT_DIR" "$templates_dir"
  # Simulate a user-created skill
  mkdir -p "$TEST_PROJECT_DIR/.claude/skills/my-custom-skill"
  echo "custom" > "$TEST_PROJECT_DIR/.claude/skills/my-custom-skill/SKILL.md"

  # Normal install (not update) should not touch user skills
  install_skills "$TEST_PROJECT_DIR" "$templates_dir"
  assert_dir_exists "$TEST_PROJECT_DIR/.claude/skills/my-custom-skill"
}

# ── Frontmatter name matches directory name ──

@test "each SKILL.md frontmatter name matches its directory" {
  for skill in "${EXPECTED_SKILLS[@]}"; do
    local skill_file="$TOOLKIT_ROOT/templates/skills/$skill/SKILL.md"
    local name_value
    name_value=$(grep "^name:" "$skill_file" | sed 's/^name:[[:space:]]*//' | tr -d '"' | tr -d "'")
    [ "$name_value" = "$skill" ] || {
      echo "$skill/SKILL.md: name '$name_value' does not match directory '$skill'" >&2
      return 1
    }
  done
}
