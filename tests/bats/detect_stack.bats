#!/usr/bin/env bats

load test_helper/common

setup() {
  setup_temp_project
  source_lib "detect.sh"
}

teardown() {
  teardown_temp_project
}

# ── detect_tech_stack ──

@test "detect_tech_stack: empty directory returns empty" {
  result=$(detect_tech_stack "$TEST_PROJECT_DIR")
  [ -z "$result" ]
}

@test "detect_tech_stack: go.mod detects go" {
  create_go_mod "$TEST_PROJECT_DIR"
  result=$(detect_tech_stack "$TEST_PROJECT_DIR")
  [[ "$result" == *"go"* ]]
}

@test "detect_tech_stack: package.json detects node" {
  create_package_json "$TEST_PROJECT_DIR"
  result=$(detect_tech_stack "$TEST_PROJECT_DIR")
  [[ "$result" == *"node"* ]]
}

@test "detect_tech_stack: package.json with react detects react" {
  create_package_json "$TEST_PROJECT_DIR" '{"react": "^18.0.0"}'
  result=$(detect_tech_stack "$TEST_PROJECT_DIR")
  [[ "$result" == *"node"* ]]
  [[ "$result" == *"react"* ]]
}

@test "detect_tech_stack: package.json with vue detects vue" {
  create_package_json "$TEST_PROJECT_DIR" '{"vue": "^3.0.0"}'
  result=$(detect_tech_stack "$TEST_PROJECT_DIR")
  [[ "$result" == *"vue"* ]]
}

@test "detect_tech_stack: package.json with svelte detects svelte" {
  create_package_json "$TEST_PROJECT_DIR" '{"svelte": "^4.0.0"}'
  result=$(detect_tech_stack "$TEST_PROJECT_DIR")
  [[ "$result" == *"svelte"* ]]
}

@test "detect_tech_stack: Cargo.toml detects rust" {
  touch "$TEST_PROJECT_DIR/Cargo.toml"
  result=$(detect_tech_stack "$TEST_PROJECT_DIR")
  [[ "$result" == *"rust"* ]]
}

@test "detect_tech_stack: requirements.txt detects python" {
  touch "$TEST_PROJECT_DIR/requirements.txt"
  result=$(detect_tech_stack "$TEST_PROJECT_DIR")
  [[ "$result" == *"python"* ]]
}

@test "detect_tech_stack: pyproject.toml detects python" {
  touch "$TEST_PROJECT_DIR/pyproject.toml"
  result=$(detect_tech_stack "$TEST_PROJECT_DIR")
  [[ "$result" == *"python"* ]]
}

@test "detect_tech_stack: .csproj detects dotnet" {
  touch "$TEST_PROJECT_DIR/MyApp.csproj"
  result=$(detect_tech_stack "$TEST_PROJECT_DIR")
  [[ "$result" == *"dotnet"* ]]
}

@test "detect_tech_stack: Assets dir detects unity" {
  mkdir "$TEST_PROJECT_DIR/Assets"
  result=$(detect_tech_stack "$TEST_PROJECT_DIR")
  [[ "$result" == *"unity"* ]]
}

@test "detect_tech_stack: hardhat.config.ts detects solidity" {
  touch "$TEST_PROJECT_DIR/hardhat.config.ts"
  result=$(detect_tech_stack "$TEST_PROJECT_DIR")
  [[ "$result" == *"solidity"* ]]
}

@test "detect_tech_stack: foundry.toml detects solidity" {
  touch "$TEST_PROJECT_DIR/foundry.toml"
  result=$(detect_tech_stack "$TEST_PROJECT_DIR")
  [[ "$result" == *"solidity"* ]]
}

@test "detect_tech_stack: Makefile detects make" {
  create_makefile "$TEST_PROJECT_DIR" "test"
  result=$(detect_tech_stack "$TEST_PROJECT_DIR")
  [[ "$result" == *"make"* ]]
}

@test "detect_tech_stack: multiple stacks detected" {
  create_go_mod "$TEST_PROJECT_DIR"
  create_package_json "$TEST_PROJECT_DIR" '{"react": "^18.0.0"}'
  result=$(detect_tech_stack "$TEST_PROJECT_DIR")
  [[ "$result" == *"go"* ]]
  [[ "$result" == *"node"* ]]
  [[ "$result" == *"react"* ]]
}

@test "detect_tech_stack: docker-compose.yml detects docker" {
  touch "$TEST_PROJECT_DIR/docker-compose.yml"
  result=$(detect_tech_stack "$TEST_PROJECT_DIR")
  [[ "$result" == *"docker"* ]]
}

@test "detect_tech_stack: Dockerfile detects docker" {
  touch "$TEST_PROJECT_DIR/Dockerfile"
  result=$(detect_tech_stack "$TEST_PROJECT_DIR")
  [[ "$result" == *"docker"* ]]
}

@test "detect_tech_stack: angular.json detects angular" {
  touch "$TEST_PROJECT_DIR/angular.json"
  create_package_json "$TEST_PROJECT_DIR"
  result=$(detect_tech_stack "$TEST_PROJECT_DIR")
  [[ "$result" == *"angular"* ]]
}

# ── detect_package_manager ──

@test "detect_package_manager: pnpm-lock.yaml detects pnpm" {
  create_package_json "$TEST_PROJECT_DIR"
  touch "$TEST_PROJECT_DIR/pnpm-lock.yaml"
  result=$(detect_package_manager "$TEST_PROJECT_DIR")
  [ "$result" = "pnpm" ]
}

@test "detect_package_manager: yarn.lock detects yarn" {
  create_package_json "$TEST_PROJECT_DIR"
  touch "$TEST_PROJECT_DIR/yarn.lock"
  result=$(detect_package_manager "$TEST_PROJECT_DIR")
  [ "$result" = "yarn" ]
}

@test "detect_package_manager: bun.lockb detects bun" {
  create_package_json "$TEST_PROJECT_DIR"
  touch "$TEST_PROJECT_DIR/bun.lockb"
  result=$(detect_package_manager "$TEST_PROJECT_DIR")
  [ "$result" = "bun" ]
}

@test "detect_package_manager: package-lock.json detects npm" {
  create_package_json "$TEST_PROJECT_DIR"
  touch "$TEST_PROJECT_DIR/package-lock.json"
  result=$(detect_package_manager "$TEST_PROJECT_DIR")
  [ "$result" = "npm" ]
}

@test "detect_package_manager: no lockfile defaults to npm" {
  create_package_json "$TEST_PROJECT_DIR"
  result=$(detect_package_manager "$TEST_PROJECT_DIR")
  [ "$result" = "npm" ]
}

@test "detect_package_manager: no package.json returns empty" {
  result=$(detect_package_manager "$TEST_PROJECT_DIR")
  [ -z "$result" ]
}

# ── map_stack_to_languages ──

@test "map_stack_to_languages: go maps to golang" {
  result=$(map_stack_to_languages "go make")
  [[ "$result" == *"golang"* ]]
}

@test "map_stack_to_languages: node maps to typescript" {
  result=$(map_stack_to_languages "node")
  [[ "$result" == *"typescript"* ]]
}

@test "map_stack_to_languages: react maps to typescript" {
  result=$(map_stack_to_languages "node react")
  [[ "$result" == *"typescript"* ]]
}

@test "map_stack_to_languages: dotnet maps to csharp" {
  result=$(map_stack_to_languages "dotnet")
  [[ "$result" == *"csharp"* ]]
}

@test "map_stack_to_languages: unity maps to csharp" {
  result=$(map_stack_to_languages "unity")
  [[ "$result" == *"csharp"* ]]
}

@test "map_stack_to_languages: python maps to python" {
  result=$(map_stack_to_languages "python")
  [[ "$result" == *"python"* ]]
}

@test "map_stack_to_languages: solidity maps to solidity" {
  result=$(map_stack_to_languages "solidity")
  [[ "$result" == *"solidity"* ]]
}

@test "map_stack_to_languages: rust maps to rust" {
  result=$(map_stack_to_languages "rust")
  [[ "$result" == *"rust"* ]]
}

@test "map_stack_to_languages: docker maps to docker" {
  result=$(map_stack_to_languages "docker")
  [[ "$result" == *"docker"* ]]
}

@test "map_stack_to_languages: angular maps to typescript" {
  result=$(map_stack_to_languages "angular")
  [[ "$result" == *"typescript"* ]]
}

@test "map_stack_to_languages: complex stack deduplicates" {
  result=$(map_stack_to_languages "go node react make")
  # Should contain golang and typescript, no duplicates
  [[ "$result" == *"golang"* ]]
  [[ "$result" == *"typescript"* ]]
  # Count occurrences of typescript (should be exactly 1)
  count=$(echo "$result" | tr ' ' '\n' | grep -c "^typescript$")
  [ "$count" -eq 1 ]
}
