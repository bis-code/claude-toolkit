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
