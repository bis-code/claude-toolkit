#!/usr/bin/env bats

load test_helper/common

setup() {
  setup_temp_project
  source_lib "detect.sh"
}

teardown() {
  teardown_temp_project
}

# ── detect_project_structure ──

@test "detect_project_structure: pnpm workspace detected" {
  cat > "$TEST_PROJECT_DIR/pnpm-workspace.yaml" <<'EOF'
packages:
  - 'apps/*'
  - 'packages/*'
EOF
  mkdir -p "$TEST_PROJECT_DIR/apps/api" "$TEST_PROJECT_DIR/apps/web" "$TEST_PROJECT_DIR/packages/ui"
  result=$(detect_project_structure "$TEST_PROJECT_DIR")
  echo "$result" | jq -e '.type == "monorepo"'
  echo "$result" | jq -e '.projects | length > 0'
}

@test "detect_project_structure: turbo.json detected as monorepo" {
  echo '{"pipeline":{}}' > "$TEST_PROJECT_DIR/turbo.json"
  mkdir -p "$TEST_PROJECT_DIR/apps/api" "$TEST_PROJECT_DIR/apps/web"
  result=$(detect_project_structure "$TEST_PROJECT_DIR")
  echo "$result" | jq -e '.type == "monorepo"'
}

@test "detect_project_structure: lerna.json detected as monorepo" {
  echo '{"packages":["packages/*"]}' > "$TEST_PROJECT_DIR/lerna.json"
  mkdir -p "$TEST_PROJECT_DIR/packages/core" "$TEST_PROJECT_DIR/packages/cli"
  result=$(detect_project_structure "$TEST_PROJECT_DIR")
  echo "$result" | jq -e '.type == "monorepo"'
}

@test "detect_project_structure: single project detected" {
  create_package_json "$TEST_PROJECT_DIR"
  result=$(detect_project_structure "$TEST_PROJECT_DIR")
  echo "$result" | jq -e '.type == "single"'
}

@test "detect_project_structure: empty directory is single project" {
  result=$(detect_project_structure "$TEST_PROJECT_DIR")
  echo "$result" | jq -e '.type == "single"'
}

@test "detect_project_structure: multiple .csproj files detected as monorepo" {
  mkdir -p "$TEST_PROJECT_DIR/src/Api" "$TEST_PROJECT_DIR/src/Web"
  touch "$TEST_PROJECT_DIR/src/Api/Api.csproj" "$TEST_PROJECT_DIR/src/Web/Web.csproj"
  result=$(detect_project_structure "$TEST_PROJECT_DIR")
  echo "$result" | jq -e '.type == "monorepo"'
  echo "$result" | jq -e '.projects | length >= 2'
}

@test "detect_project_structure: docker-compose services detected" {
  cat > "$TEST_PROJECT_DIR/docker-compose.yml" <<'EOF'
services:
  api:
    build: ./api
  web:
    build: ./web
  worker:
    build: ./worker
EOF
  mkdir -p "$TEST_PROJECT_DIR/api" "$TEST_PROJECT_DIR/web" "$TEST_PROJECT_DIR/worker"
  result=$(detect_project_structure "$TEST_PROJECT_DIR")
  echo "$result" | jq -e '.type == "monorepo"'
}

@test "detect_project_structure: apps/ and packages/ detected" {
  mkdir -p "$TEST_PROJECT_DIR/apps/frontend" "$TEST_PROJECT_DIR/apps/backend" "$TEST_PROJECT_DIR/packages/shared"
  result=$(detect_project_structure "$TEST_PROJECT_DIR")
  echo "$result" | jq -e '.type == "monorepo"'
  echo "$result" | jq -e '.projects | length >= 2'
}

# ── scope mapping ──

@test "detect_project_structure: scope maps to correct path" {
  cat > "$TEST_PROJECT_DIR/pnpm-workspace.yaml" <<'EOF'
packages:
  - 'apps/*'
EOF
  mkdir -p "$TEST_PROJECT_DIR/apps/api" "$TEST_PROJECT_DIR/apps/web"
  result=$(detect_project_structure "$TEST_PROJECT_DIR")
  # projects should contain apps/api and apps/web
  echo "$result" | jq -e '.projects | map(select(test("api"))) | length > 0'
  echo "$result" | jq -e '.projects | map(select(test("web"))) | length > 0'
}
