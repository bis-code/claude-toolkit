#!/usr/bin/env bats

load test_helper/common

setup() {
  setup_temp_project
  source_lib "utils.sh"
  source_lib "detect.sh"
  source_lib "install_agents.sh"
  source_lib "install_rules.sh"
  source_lib "install_skills.sh"
}

teardown() {
  teardown_temp_project
}

# ── Agents: update mode ──

@test "update mode overwrites managed agents" {
  local templates_dir="$TOOLKIT_ROOT/templates"
  # Initial install
  install_agents "$TEST_PROJECT_DIR" "$templates_dir" ""
  assert_file_exists "$TEST_PROJECT_DIR/.claude/agents/code-reviewer.md"
  # Modify installed agent
  echo "user modification" > "$TEST_PROJECT_DIR/.claude/agents/code-reviewer.md"
  assert_file_contains "$TEST_PROJECT_DIR/.claude/agents/code-reviewer.md" "user modification"
  # Update mode — should overwrite because template exists
  MODE=update install_agents "$TEST_PROJECT_DIR" "$templates_dir" ""
  ! grep -q "user modification" "$TEST_PROJECT_DIR/.claude/agents/code-reviewer.md"
}

@test "update mode does not delete user-created agents" {
  local templates_dir="$TOOLKIT_ROOT/templates"
  # Initial install
  install_agents "$TEST_PROJECT_DIR" "$templates_dir" ""
  # Add a custom agent file (no matching template)
  echo "my custom agent" > "$TEST_PROJECT_DIR/.claude/agents/my-custom-agent.md"
  # Update mode
  MODE=update install_agents "$TEST_PROJECT_DIR" "$templates_dir" ""
  # Custom agent survives
  assert_file_exists "$TEST_PROJECT_DIR/.claude/agents/my-custom-agent.md"
  assert_file_contains "$TEST_PROJECT_DIR/.claude/agents/my-custom-agent.md" "my custom agent"
}

@test "update mode overwrites managed domain agents" {
  local templates_dir="$TOOLKIT_ROOT/templates"
  # Initial install with blockchain domain
  install_agents "$TEST_PROJECT_DIR" "$templates_dir" "blockchain"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/agents/blockchain-developer.md"
  # Modify domain agent
  echo "modified" > "$TEST_PROJECT_DIR/.claude/agents/blockchain-developer.md"
  # Update mode — should overwrite
  MODE=update install_agents "$TEST_PROJECT_DIR" "$templates_dir" "blockchain"
  ! grep -q "modified" "$TEST_PROJECT_DIR/.claude/agents/blockchain-developer.md"
}

@test "update mode picks up new domain agents" {
  local templates_dir="$TOOLKIT_ROOT/templates"
  # Initial install without blockchain
  install_agents "$TEST_PROJECT_DIR" "$templates_dir" ""
  [ ! -f "$TEST_PROJECT_DIR/.claude/agents/blockchain-developer.md" ]
  # Update with blockchain domain now detected
  MODE=update install_agents "$TEST_PROJECT_DIR" "$templates_dir" "blockchain"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/agents/blockchain-developer.md"
}

# ── Rules: update mode ──

@test "update mode overwrites managed rules" {
  local templates_dir="$TOOLKIT_ROOT/templates"
  # Initial install
  install_rules "$TEST_PROJECT_DIR" "golang" "$templates_dir"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/rules/common/coding-style.md"
  # Modify installed rule
  echo "user edit" > "$TEST_PROJECT_DIR/.claude/rules/common/coding-style.md"
  assert_file_contains "$TEST_PROJECT_DIR/.claude/rules/common/coding-style.md" "user edit"
  # Update mode — should overwrite
  MODE=update install_rules "$TEST_PROJECT_DIR" "golang" "$templates_dir"
  ! grep -q "user edit" "$TEST_PROJECT_DIR/.claude/rules/common/coding-style.md"
}

@test "update mode preserves user-created rule files" {
  local templates_dir="$TOOLKIT_ROOT/templates"
  install_rules "$TEST_PROJECT_DIR" "golang" "$templates_dir"
  # Add custom rule file in common dir
  echo "my custom rule" > "$TEST_PROJECT_DIR/.claude/rules/common/my-project-conventions.md"
  # Update mode
  MODE=update install_rules "$TEST_PROJECT_DIR" "golang" "$templates_dir"
  # Custom rule survives
  assert_file_exists "$TEST_PROJECT_DIR/.claude/rules/common/my-project-conventions.md"
  assert_file_contains "$TEST_PROJECT_DIR/.claude/rules/common/my-project-conventions.md" "my custom rule"
}

@test "update mode overwrites language-specific rules" {
  local templates_dir="$TOOLKIT_ROOT/templates"
  install_rules "$TEST_PROJECT_DIR" "golang" "$templates_dir"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/rules/golang/coding-standards.md"
  echo "modified" > "$TEST_PROJECT_DIR/.claude/rules/golang/coding-standards.md"
  MODE=update install_rules "$TEST_PROJECT_DIR" "golang" "$templates_dir"
  ! grep -q "modified" "$TEST_PROJECT_DIR/.claude/rules/golang/coding-standards.md"
}

# ── Skills: update mode ──

@test "update mode overwrites managed skills" {
  local templates_dir="$TOOLKIT_ROOT/templates"
  install_skills "$TEST_PROJECT_DIR" "$templates_dir"
  # Find a skill file to modify
  local skill_file
  skill_file=$(find "$TEST_PROJECT_DIR/.claude/skills" -name "*.md" -type f | head -1)
  [ -n "$skill_file" ] || skip "No skill files found"
  echo "modified skill" > "$skill_file"
  assert_file_contains "$skill_file" "modified skill"
  # Update mode — should overwrite
  MODE=update install_skills "$TEST_PROJECT_DIR" "$templates_dir"
  ! grep -q "modified skill" "$skill_file"
}

@test "update mode preserves user-created skill files" {
  local templates_dir="$TOOLKIT_ROOT/templates"
  install_skills "$TEST_PROJECT_DIR" "$templates_dir"
  # Find a skill directory and add a custom file
  # Note: use unique var name because install_skills leaks 'skill_dir' (no local)
  local test_skill_dir
  test_skill_dir=$(find "$TEST_PROJECT_DIR/.claude/skills" -mindepth 1 -maxdepth 1 -type d | head -1)
  [ -n "$test_skill_dir" ] || skip "No skill dirs found"
  echo "my custom skill config" > "$test_skill_dir/custom-config.md"
  MODE=update install_skills "$TEST_PROJECT_DIR" "$templates_dir"
  # Custom file survives (install_skills only copies files from template, doesn't delete extras)
  assert_file_exists "$test_skill_dir/custom-config.md"
  assert_file_contains "$test_skill_dir/custom-config.md" "my custom skill config"
}

# ── .claude-toolkit.json: update mode ──

@test "update_toolkit_config preserves user config, updates version" {
  # Create a config with user customizations
  cat > "$TEST_PROJECT_DIR/.claude-toolkit.json" <<'EOF'
{
  "version": "1.0.0",
  "project": {
    "name": "test-project",
    "type": "repository",
    "techStack": ["go"],
    "languages": ["golang"],
    "packageManager": null
  },
  "commands": {
    "test": "make test-custom",
    "lint": "golangci-lint run ./..."
  },
  "qa": {
    "scanCategories": ["security", "tests"],
    "maxFixLines": 50,
    "worktreeFromBranch": "develop"
  },
  "ralph": {
    "maxLoops": 15,
    "stuckThreshold": 5
  },
  "mcpServers": {
    "required": ["deep-think"],
    "installed": ["deep-think", "context7"]
  }
}
EOF

  update_toolkit_config "$TEST_PROJECT_DIR/.claude-toolkit.json" "2.0.0" '["go","node"]' '["golang","typescript"]' "pnpm"

  # Version updated
  local version
  version=$(jq -r '.version' "$TEST_PROJECT_DIR/.claude-toolkit.json")
  [ "$version" = "2.0.0" ]

  # Tech stack updated
  local stack
  stack=$(jq -r '.project.techStack | join(",")' "$TEST_PROJECT_DIR/.claude-toolkit.json")
  [ "$stack" = "go,node" ]

  # Languages updated
  local langs
  langs=$(jq -r '.project.languages | join(",")' "$TEST_PROJECT_DIR/.claude-toolkit.json")
  [ "$langs" = "golang,typescript" ]

  # Package manager updated
  local pkg
  pkg=$(jq -r '.project.packageManager' "$TEST_PROJECT_DIR/.claude-toolkit.json")
  [ "$pkg" = "pnpm" ]

  # User's QA config preserved
  local test_cmd
  test_cmd=$(jq -r '.commands.test' "$TEST_PROJECT_DIR/.claude-toolkit.json")
  [ "$test_cmd" = "make test-custom" ]

  local max_fix
  max_fix=$(jq -r '.qa.maxFixLines' "$TEST_PROJECT_DIR/.claude-toolkit.json")
  [ "$max_fix" = "50" ]

  local branch
  branch=$(jq -r '.qa.worktreeFromBranch' "$TEST_PROJECT_DIR/.claude-toolkit.json")
  [ "$branch" = "develop" ]

  # Ralph config preserved
  local max_loops
  max_loops=$(jq -r '.ralph.maxLoops' "$TEST_PROJECT_DIR/.claude-toolkit.json")
  [ "$max_loops" = "15" ]

  # MCP servers preserved
  local mcps
  mcps=$(jq -r '.mcpServers.installed | join(",")' "$TEST_PROJECT_DIR/.claude-toolkit.json")
  [ "$mcps" = "deep-think,context7" ]
}

@test "update_toolkit_config handles null package manager" {
  cat > "$TEST_PROJECT_DIR/.claude-toolkit.json" <<'EOF'
{
  "version": "1.0.0",
  "project": {
    "name": "test-project",
    "type": "repository",
    "techStack": ["go"],
    "languages": ["golang"],
    "packageManager": "npm"
  },
  "commands": { "test": "go test", "lint": null },
  "qa": { "scanCategories": [], "maxFixLines": 30, "worktreeFromBranch": "main" },
  "ralph": { "maxLoops": 30, "stuckThreshold": 3 },
  "mcpServers": { "required": ["deep-think"], "installed": ["deep-think"] }
}
EOF

  update_toolkit_config "$TEST_PROJECT_DIR/.claude-toolkit.json" "2.0.0" '["go"]' '["golang"]' ""

  local pkg
  pkg=$(jq -r '.project.packageManager' "$TEST_PROJECT_DIR/.claude-toolkit.json")
  [ "$pkg" = "null" ]
}

# ── Normal install does NOT overwrite without force ──

@test "normal install does not overwrite existing agents" {
  local templates_dir="$TOOLKIT_ROOT/templates"
  install_agents "$TEST_PROJECT_DIR" "$templates_dir" ""
  echo "customized" > "$TEST_PROJECT_DIR/.claude/agents/code-reviewer.md"
  # Reinstall without MODE=update and without FORCE
  install_agents "$TEST_PROJECT_DIR" "$templates_dir" ""
  assert_file_contains "$TEST_PROJECT_DIR/.claude/agents/code-reviewer.md" "customized"
}

@test "normal install does not overwrite existing rules" {
  local templates_dir="$TOOLKIT_ROOT/templates"
  install_rules "$TEST_PROJECT_DIR" "golang" "$templates_dir"
  echo "customized" > "$TEST_PROJECT_DIR/.claude/rules/common/coding-style.md"
  install_rules "$TEST_PROJECT_DIR" "golang" "$templates_dir"
  assert_file_contains "$TEST_PROJECT_DIR/.claude/rules/common/coding-style.md" "customized"
}

# ══════════════════════════════════════════════
# _tracked_copy + update tracking
# ══════════════════════════════════════════════

@test "_tracked_copy: creates new file and increments ADDED_COUNT" {
  init_update_tracking ""
  mkdir -p "$TEST_PROJECT_DIR/src"
  echo "source content" > "$TEST_PROJECT_DIR/src/file.md"

  _tracked_copy "$TEST_PROJECT_DIR/src/file.md" "$TEST_PROJECT_DIR/dest/file.md" "dest/file.md"

  assert_file_exists "$TEST_PROJECT_DIR/dest/file.md"
  assert_file_contains "$TEST_PROJECT_DIR/dest/file.md" "source content"
  [ "$ADDED_COUNT" -eq 1 ]
  [ "$UPDATE_COUNT" -eq 0 ]
}

@test "_tracked_copy: overwrites in update mode, increments UPDATE_COUNT" {
  init_update_tracking ""
  mkdir -p "$TEST_PROJECT_DIR/src" "$TEST_PROJECT_DIR/dest"
  echo "new content" > "$TEST_PROJECT_DIR/src/file.md"
  echo "old content" > "$TEST_PROJECT_DIR/dest/file.md"

  MODE=update _tracked_copy "$TEST_PROJECT_DIR/src/file.md" "$TEST_PROJECT_DIR/dest/file.md" "dest/file.md"

  assert_file_contains "$TEST_PROJECT_DIR/dest/file.md" "new content"
  [ "$UPDATE_COUNT" -eq 1 ]
  [ "$ADDED_COUNT" -eq 0 ]
}

@test "_tracked_copy: skips existing file in normal install" {
  init_update_tracking ""
  mkdir -p "$TEST_PROJECT_DIR/src" "$TEST_PROJECT_DIR/dest"
  echo "new content" > "$TEST_PROJECT_DIR/src/file.md"
  echo "old content" > "$TEST_PROJECT_DIR/dest/file.md"

  _tracked_copy "$TEST_PROJECT_DIR/src/file.md" "$TEST_PROJECT_DIR/dest/file.md" "dest/file.md"

  assert_file_contains "$TEST_PROJECT_DIR/dest/file.md" "old content"
  [ "$UPDATE_COUNT" -eq 0 ]
  [ "$ADDED_COUNT" -eq 0 ]
}

@test "_tracked_copy: always records path in MANAGED_FILES" {
  init_update_tracking ""
  mkdir -p "$TEST_PROJECT_DIR/src" "$TEST_PROJECT_DIR/dest"
  echo "content" > "$TEST_PROJECT_DIR/src/file.md"
  echo "existing" > "$TEST_PROJECT_DIR/dest/file.md"

  # Even when skipping copy, path is tracked
  _tracked_copy "$TEST_PROJECT_DIR/src/file.md" "$TEST_PROJECT_DIR/dest/file.md" "dest/file.md"
  _tracked_copy "$TEST_PROJECT_DIR/src/file.md" "$TEST_PROJECT_DIR/dest/file2.md" "dest/file2.md"

  [ "${#MANAGED_FILES[@]}" -eq 2 ]
  [[ " ${MANAGED_FILES[*]} " == *" dest/file.md "* ]]
  [[ " ${MANAGED_FILES[*]} " == *" dest/file2.md "* ]]
}

# ── write_managed_files ──

@test "write_managed_files: writes sorted array to config" {
  local config="$TEST_PROJECT_DIR/.claude-toolkit.json"
  echo '{"version":"2.0.0"}' > "$config"
  init_update_tracking ""
  MANAGED_FILES=(".claude/rules/common/b.md" ".claude/agents/a.md" ".claude/rules/common/a.md")

  write_managed_files "$config"

  local first second third count
  count=$(jq '.managedFiles | length' "$config")
  first=$(jq -r '.managedFiles[0]' "$config")
  second=$(jq -r '.managedFiles[1]' "$config")
  third=$(jq -r '.managedFiles[2]' "$config")
  [ "$count" -eq 3 ]
  [ "$first" = ".claude/agents/a.md" ]
  [ "$second" = ".claude/rules/common/a.md" ]
  [ "$third" = ".claude/rules/common/b.md" ]
}

# ── detect_deprecated_files ──

@test "detect_deprecated_files: warns about removed templates" {
  local config="$TEST_PROJECT_DIR/.claude-toolkit.json"
  cat > "$config" <<'EOF'
{"version":"2.0.0","managedFiles":[".claude/agents/old-agent.md",".claude/agents/current.md"]}
EOF
  init_update_tracking "$config"
  # Simulate current install only tracking "current.md"
  MANAGED_FILES=(".claude/agents/current.md")
  # Create the deprecated file on disk
  mkdir -p "$TEST_PROJECT_DIR/.claude/agents"
  echo "old" > "$TEST_PROJECT_DIR/.claude/agents/old-agent.md"

  local output
  output=$(detect_deprecated_files "$TEST_PROJECT_DIR")

  [[ "$output" == *"old-agent.md"* ]]
}

@test "detect_deprecated_files: silent on first install" {
  init_update_tracking ""
  MANAGED_FILES=(".claude/agents/code-reviewer.md")

  local output
  output=$(detect_deprecated_files "$TEST_PROJECT_DIR")

  [ -z "$output" ]
}

@test "detect_deprecated_files: ignores already-deleted files" {
  local config="$TEST_PROJECT_DIR/.claude-toolkit.json"
  cat > "$config" <<'EOF'
{"version":"2.0.0","managedFiles":[".claude/agents/deleted.md",".claude/agents/current.md"]}
EOF
  init_update_tracking "$config"
  MANAGED_FILES=(".claude/agents/current.md")
  mkdir -p "$TEST_PROJECT_DIR/.claude/agents"
  # deleted.md does NOT exist on disk

  local output
  output=$(detect_deprecated_files "$TEST_PROJECT_DIR")

  [ -z "$output" ]
}

# ── count_preserved_files ──

@test "count_preserved_files: counts non-managed files" {
  init_update_tracking ""
  MANAGED_FILES=(".claude/agents/managed.md" ".claude/rules/common/managed.md")
  mkdir -p "$TEST_PROJECT_DIR/.claude/agents" "$TEST_PROJECT_DIR/.claude/rules/common" "$TEST_PROJECT_DIR/.claude/skills/qa"
  echo "m" > "$TEST_PROJECT_DIR/.claude/agents/managed.md"
  echo "u" > "$TEST_PROJECT_DIR/.claude/agents/user-custom.md"
  echo "m" > "$TEST_PROJECT_DIR/.claude/rules/common/managed.md"
  echo "u" > "$TEST_PROJECT_DIR/.claude/rules/common/user-rule.md"
  echo "u" > "$TEST_PROJECT_DIR/.claude/skills/qa/user-skill.md"

  local count
  count=$(count_preserved_files "$TEST_PROJECT_DIR")

  [ "$count" -eq 3 ]
}

# ── print_update_summary ──

@test "print_update_summary: shows correct counts" {
  init_update_tracking ""
  UPDATE_COUNT=5
  ADDED_COUNT=2
  MANAGED_FILES=(".claude/agents/managed.md")
  mkdir -p "$TEST_PROJECT_DIR/.claude/agents"
  echo "m" > "$TEST_PROJECT_DIR/.claude/agents/managed.md"
  echo "u" > "$TEST_PROJECT_DIR/.claude/agents/user.md"

  local output
  output=$(print_update_summary "$TEST_PROJECT_DIR")

  [[ "$output" == *"Updated 5"* ]]
  [[ "$output" == *"added 2 new"* ]]
  [[ "$output" == *"preserved 1 user"* ]]
}
