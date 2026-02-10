#!/usr/bin/env bats

load test_helper/common

setup() {
  setup_temp_project
  source_lib "detect.sh"
}

teardown() {
  teardown_temp_project
}

# ── detect_test_command ──

@test "detect_test_command: Makefile with test target" {
  create_makefile "$TEST_PROJECT_DIR" "test"
  result=$(detect_test_command "$TEST_PROJECT_DIR" "go make")
  [ "$result" = "make test" ]
}

@test "detect_test_command: Makefile with test-unit target" {
  create_makefile "$TEST_PROJECT_DIR" "test-unit"
  result=$(detect_test_command "$TEST_PROJECT_DIR" "go make")
  [ "$result" = "make test-unit" ]
}

@test "detect_test_command: go fallback" {
  result=$(detect_test_command "$TEST_PROJECT_DIR" "go")
  [ "$result" = "go test ./..." ]
}

@test "detect_test_command: node fallback" {
  result=$(detect_test_command "$TEST_PROJECT_DIR" "node")
  [ "$result" = "npm test" ]
}

@test "detect_test_command: dotnet fallback" {
  result=$(detect_test_command "$TEST_PROJECT_DIR" "dotnet")
  [ "$result" = "dotnet test" ]
}

@test "detect_test_command: python fallback" {
  result=$(detect_test_command "$TEST_PROJECT_DIR" "python")
  [ "$result" = "pytest" ]
}

@test "detect_test_command: rust fallback" {
  result=$(detect_test_command "$TEST_PROJECT_DIR" "rust")
  [ "$result" = "cargo test" ]
}

@test "detect_test_command: solidity fallback" {
  result=$(detect_test_command "$TEST_PROJECT_DIR" "solidity")
  [ "$result" = "npx hardhat test" ]
}

@test "detect_test_command: java with maven" {
  touch "$TEST_PROJECT_DIR/pom.xml"
  result=$(detect_test_command "$TEST_PROJECT_DIR" "java")
  [ "$result" = "mvn test" ]
}

@test "detect_test_command: java with gradle" {
  touch "$TEST_PROJECT_DIR/build.gradle"
  result=$(detect_test_command "$TEST_PROJECT_DIR" "java")
  [ "$result" = "./gradlew test" ]
}

@test "detect_test_command: unknown returns empty" {
  result=$(detect_test_command "$TEST_PROJECT_DIR" "unknown")
  [ -z "$result" ]
}

# ── detect_lint_command ──

@test "detect_lint_command: Makefile with lint target" {
  create_makefile "$TEST_PROJECT_DIR" "lint"
  result=$(detect_lint_command "$TEST_PROJECT_DIR" "go make")
  [ "$result" = "make lint" ]
}

@test "detect_lint_command: go fallback" {
  result=$(detect_lint_command "$TEST_PROJECT_DIR" "go")
  [ "$result" = "golangci-lint run" ]
}

@test "detect_lint_command: node fallback" {
  result=$(detect_lint_command "$TEST_PROJECT_DIR" "node")
  [ "$result" = "npm run lint" ]
}

@test "detect_lint_command: python fallback" {
  result=$(detect_lint_command "$TEST_PROJECT_DIR" "python")
  [ "$result" = "ruff check ." ]
}

@test "detect_lint_command: rust fallback" {
  result=$(detect_lint_command "$TEST_PROJECT_DIR" "rust")
  [ "$result" = "cargo clippy" ]
}

@test "detect_lint_command: dotnet fallback" {
  result=$(detect_lint_command "$TEST_PROJECT_DIR" "dotnet")
  [ "$result" = "dotnet format --verify-no-changes" ]
}

# ── detect_scan_categories ──

@test "detect_scan_categories: always includes base categories" {
  result=$(detect_scan_categories "go")
  [[ "$result" == *"tests"* ]]
  [[ "$result" == *"lint"* ]]
  [[ "$result" == *"missing-tests"* ]]
  [[ "$result" == *"todo-audit"* ]]
}

@test "detect_scan_categories: go includes module-boundaries" {
  result=$(detect_scan_categories "go")
  [[ "$result" == *"module-boundaries"* ]]
  [[ "$result" == *"security-scan"* ]]
}

@test "detect_scan_categories: react includes frontend categories" {
  result=$(detect_scan_categories "node react")
  [[ "$result" == *"accessibility"* ]]
  [[ "$result" == *"component-quality"* ]]
  [[ "$result" == *"browser-testing"* ]]
}

@test "detect_scan_categories: solidity includes blockchain categories" {
  result=$(detect_scan_categories "solidity")
  [[ "$result" == *"smart-contract-security"* ]]
  [[ "$result" == *"gas-optimization"* ]]
}
