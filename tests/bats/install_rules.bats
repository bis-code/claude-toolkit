#!/usr/bin/env bats

load test_helper/common

setup() {
  setup_temp_project
  source_lib "utils.sh"
  source_lib "detect.sh"
  source_lib "install_rules.sh"
}

teardown() {
  teardown_temp_project
}

@test "install_rules: always installs common rules" {
  install_rules "$TEST_PROJECT_DIR" "golang" "$TOOLKIT_ROOT/templates"
  assert_dir_exists "$TEST_PROJECT_DIR/.claude/rules/common"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/rules/common/coding-style.md"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/rules/common/git-workflow.md"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/rules/common/testing.md"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/rules/common/security.md"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/rules/common/performance.md"
}

@test "install_rules: installs golang rules for go stack" {
  install_rules "$TEST_PROJECT_DIR" "golang" "$TOOLKIT_ROOT/templates"
  assert_dir_exists "$TEST_PROJECT_DIR/.claude/rules/golang"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/rules/golang/coding-standards.md"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/rules/golang/patterns.md"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/rules/golang/testing.md"
}

@test "install_rules: installs typescript rules for typescript lang" {
  install_rules "$TEST_PROJECT_DIR" "typescript" "$TOOLKIT_ROOT/templates"
  assert_dir_exists "$TEST_PROJECT_DIR/.claude/rules/typescript"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/rules/typescript/coding-standards.md"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/rules/typescript/react-patterns.md"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/rules/typescript/testing.md"
}

@test "install_rules: installs python rules" {
  install_rules "$TEST_PROJECT_DIR" "python" "$TOOLKIT_ROOT/templates"
  assert_dir_exists "$TEST_PROJECT_DIR/.claude/rules/python"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/rules/python/coding-standards.md"
}

@test "install_rules: installs csharp rules" {
  install_rules "$TEST_PROJECT_DIR" "csharp" "$TOOLKIT_ROOT/templates"
  assert_dir_exists "$TEST_PROJECT_DIR/.claude/rules/csharp"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/rules/csharp/coding-standards.md"
}

@test "install_rules: installs solidity rules" {
  install_rules "$TEST_PROJECT_DIR" "solidity" "$TOOLKIT_ROOT/templates"
  assert_dir_exists "$TEST_PROJECT_DIR/.claude/rules/solidity"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/rules/solidity/coding-standards.md"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/rules/solidity/security.md"
}

@test "install_rules: installs java rules" {
  install_rules "$TEST_PROJECT_DIR" "java" "$TOOLKIT_ROOT/templates"
  assert_dir_exists "$TEST_PROJECT_DIR/.claude/rules/java"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/rules/java/coding-standards.md"
}

@test "install_rules: installs rust rules" {
  install_rules "$TEST_PROJECT_DIR" "rust" "$TOOLKIT_ROOT/templates"
  assert_dir_exists "$TEST_PROJECT_DIR/.claude/rules/rust"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/rules/rust/coding-standards.md"
}

@test "install_rules: installs docker rules" {
  install_rules "$TEST_PROJECT_DIR" "docker" "$TOOLKIT_ROOT/templates"
  assert_dir_exists "$TEST_PROJECT_DIR/.claude/rules/docker"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/rules/docker/best-practices.md"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/rules/docker/security.md"
}

@test "install_rules: multiple languages" {
  install_rules "$TEST_PROJECT_DIR" "golang typescript" "$TOOLKIT_ROOT/templates"
  assert_dir_exists "$TEST_PROJECT_DIR/.claude/rules/common"
  assert_dir_exists "$TEST_PROJECT_DIR/.claude/rules/golang"
  assert_dir_exists "$TEST_PROJECT_DIR/.claude/rules/typescript"
}

@test "install_rules: does not overwrite existing rules without force" {
  mkdir -p "$TEST_PROJECT_DIR/.claude/rules/common"
  echo "custom content" > "$TEST_PROJECT_DIR/.claude/rules/common/coding-style.md"
  FORCE=false install_rules "$TEST_PROJECT_DIR" "golang" "$TOOLKIT_ROOT/templates"
  result=$(cat "$TEST_PROJECT_DIR/.claude/rules/common/coding-style.md")
  [ "$result" = "custom content" ]
}
