#!/usr/bin/env bats

load test_helper/common

setup() {
  setup_temp_project
  source_lib "detect.sh"
  source_lib "utils.sh"
  source_lib "install_agents.sh"
}

teardown() {
  teardown_temp_project
}

# ── map_stack_to_agent_domains ──

@test "map_stack_to_agent_domains: go maps to golang" {
  result=$(map_stack_to_agent_domains "go")
  [[ "$result" == *"golang"* ]]
}

@test "map_stack_to_agent_domains: node maps to react" {
  result=$(map_stack_to_agent_domains "node")
  [[ "$result" == *"react"* ]]
}

@test "map_stack_to_agent_domains: react maps to react" {
  result=$(map_stack_to_agent_domains "react")
  [[ "$result" == *"react"* ]]
}

@test "map_stack_to_agent_domains: vue maps to react" {
  result=$(map_stack_to_agent_domains "vue")
  [[ "$result" == *"react"* ]]
}

@test "map_stack_to_agent_domains: svelte maps to react" {
  result=$(map_stack_to_agent_domains "svelte")
  [[ "$result" == *"react"* ]]
}

@test "map_stack_to_agent_domains: dotnet maps to dotnet" {
  result=$(map_stack_to_agent_domains "dotnet")
  [[ "$result" == *"dotnet"* ]]
}

@test "map_stack_to_agent_domains: unity maps to unity and dotnet" {
  result=$(map_stack_to_agent_domains "unity")
  [[ "$result" == *"unity"* ]]
  [[ "$result" == *"dotnet"* ]]
}

@test "map_stack_to_agent_domains: solidity maps to blockchain" {
  result=$(map_stack_to_agent_domains "solidity")
  [[ "$result" == *"blockchain"* ]]
}

@test "map_stack_to_agent_domains: docker maps to docker" {
  result=$(map_stack_to_agent_domains "docker")
  [[ "$result" == *"docker"* ]]
}

@test "map_stack_to_agent_domains: unknown stack returns empty" {
  result=$(map_stack_to_agent_domains "make")
  [ -z "$result" ]
}

@test "map_stack_to_agent_domains: complex stack returns multiple domains" {
  result=$(map_stack_to_agent_domains "go node react solidity docker")
  [[ "$result" == *"golang"* ]]
  [[ "$result" == *"react"* ]]
  [[ "$result" == *"blockchain"* ]]
  [[ "$result" == *"docker"* ]]
}

@test "map_stack_to_agent_domains: deduplicates domains" {
  result=$(map_stack_to_agent_domains "node react vue svelte")
  count=$(echo "$result" | tr ' ' '\n' | grep -c "^react$")
  [ "$count" -eq 1 ]
}

# ── detect_deep_domains ──

@test "detect_deep_domains: .graphql files trigger graphql domain" {
  mkdir -p "$TEST_PROJECT_DIR/src"
  touch "$TEST_PROJECT_DIR/src/schema.graphql"
  result=$(detect_deep_domains "$TEST_PROJECT_DIR")
  [[ "$result" == *"graphql"* ]]
}

@test "detect_deep_domains: graphql dependency in package.json triggers graphql" {
  cat > "$TEST_PROJECT_DIR/package.json" <<'EOF'
{"dependencies":{"graphql":"^16.0.0","@apollo/server":"^4.0.0"}}
EOF
  result=$(detect_deep_domains "$TEST_PROJECT_DIR")
  [[ "$result" == *"graphql"* ]]
}

@test "detect_deep_domains: graphql dependency in go.mod triggers graphql" {
  cat > "$TEST_PROJECT_DIR/go.mod" <<'EOF'
module example.com/app

go 1.21

require github.com/99designs/gqlgen v0.17.0
EOF
  result=$(detect_deep_domains "$TEST_PROJECT_DIR")
  [[ "$result" == *"graphql"* ]]
}

@test "detect_deep_domains: openai dependency triggers ai domain" {
  cat > "$TEST_PROJECT_DIR/package.json" <<'EOF'
{"dependencies":{"openai":"^4.0.0"}}
EOF
  result=$(detect_deep_domains "$TEST_PROJECT_DIR")
  [[ "$result" == *"ai"* ]]
}

@test "detect_deep_domains: anthropic dependency triggers ai domain" {
  cat > "$TEST_PROJECT_DIR/requirements.txt" <<'EOF'
anthropic>=0.25.0
flask>=3.0.0
EOF
  result=$(detect_deep_domains "$TEST_PROJECT_DIR")
  [[ "$result" == *"ai"* ]]
}

@test "detect_deep_domains: langchain dependency triggers ai domain" {
  cat > "$TEST_PROJECT_DIR/package.json" <<'EOF'
{"dependencies":{"langchain":"^0.2.0"}}
EOF
  result=$(detect_deep_domains "$TEST_PROJECT_DIR")
  [[ "$result" == *"ai"* ]]
}

@test "detect_deep_domains: stripe dependency triggers saas domain" {
  cat > "$TEST_PROJECT_DIR/package.json" <<'EOF'
{"dependencies":{"stripe":"^14.0.0"}}
EOF
  result=$(detect_deep_domains "$TEST_PROJECT_DIR")
  [[ "$result" == *"saas"* ]]
}

@test "detect_deep_domains: stripe in go.mod triggers saas domain" {
  cat > "$TEST_PROJECT_DIR/go.mod" <<'EOF'
module example.com/app

go 1.21

require github.com/stripe/stripe-go/v76 v76.0.0
EOF
  result=$(detect_deep_domains "$TEST_PROJECT_DIR")
  [[ "$result" == *"saas"* ]]
}

@test "detect_deep_domains: gorm dependency triggers database domain" {
  cat > "$TEST_PROJECT_DIR/go.mod" <<'EOF'
module example.com/app

go 1.21

require gorm.io/gorm v1.25.0
EOF
  result=$(detect_deep_domains "$TEST_PROJECT_DIR")
  [[ "$result" == *"database"* ]]
}

@test "detect_deep_domains: prisma files trigger database domain" {
  mkdir -p "$TEST_PROJECT_DIR/prisma"
  touch "$TEST_PROJECT_DIR/prisma/schema.prisma"
  result=$(detect_deep_domains "$TEST_PROJECT_DIR")
  [[ "$result" == *"database"* ]]
}

@test "detect_deep_domains: migrations directory triggers database domain" {
  mkdir -p "$TEST_PROJECT_DIR/migrations"
  touch "$TEST_PROJECT_DIR/migrations/001_init.sql"
  result=$(detect_deep_domains "$TEST_PROJECT_DIR")
  [[ "$result" == *"database"* ]]
}

@test "detect_deep_domains: k8s manifests trigger kubernetes domain" {
  mkdir -p "$TEST_PROJECT_DIR/k8s"
  cat > "$TEST_PROJECT_DIR/k8s/deployment.yaml" <<'EOF'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
EOF
  result=$(detect_deep_domains "$TEST_PROJECT_DIR")
  [[ "$result" == *"kubernetes"* ]]
}

@test "detect_deep_domains: helm Chart.yaml triggers kubernetes domain" {
  mkdir -p "$TEST_PROJECT_DIR/charts/app"
  touch "$TEST_PROJECT_DIR/charts/app/Chart.yaml"
  result=$(detect_deep_domains "$TEST_PROJECT_DIR")
  [[ "$result" == *"kubernetes"* ]]
}

@test "detect_deep_domains: prometheus config triggers observability domain" {
  touch "$TEST_PROJECT_DIR/prometheus.yml"
  result=$(detect_deep_domains "$TEST_PROJECT_DIR")
  [[ "$result" == *"observability"* ]]
}

@test "detect_deep_domains: grafana directory triggers observability domain" {
  mkdir -p "$TEST_PROJECT_DIR/grafana/dashboards"
  result=$(detect_deep_domains "$TEST_PROJECT_DIR")
  [[ "$result" == *"observability"* ]]
}

@test "detect_deep_domains: datadog config triggers observability domain" {
  touch "$TEST_PROJECT_DIR/datadog.yaml"
  result=$(detect_deep_domains "$TEST_PROJECT_DIR")
  [[ "$result" == *"observability"* ]]
}

@test "detect_deep_domains: empty project returns empty" {
  result=$(detect_deep_domains "$TEST_PROJECT_DIR")
  [ -z "$result" ]
}

# ── monorepo-aware detection ──

@test "detect_deep_domains: scans monorepo sub-projects for deps" {
  # Simulate learnmeld: pnpm monorepo with go.mod + stripe in apps/api
  mkdir -p "$TEST_PROJECT_DIR/apps/api" "$TEST_PROJECT_DIR/apps/web"
  cat > "$TEST_PROJECT_DIR/pnpm-workspace.yaml" <<'EOF'
packages:
  - 'apps/*'
EOF
  cat > "$TEST_PROJECT_DIR/apps/api/go.mod" <<'EOF'
module example.com/api

go 1.21

require (
  gorm.io/gorm v1.25.0
  github.com/stripe/stripe-go/v76 v76.0.0
)
EOF
  cat > "$TEST_PROJECT_DIR/apps/web/package.json" <<'EOF'
{"dependencies":{"react":"^18.0.0","openai":"^4.0.0"}}
EOF
  result=$(detect_deep_domains "$TEST_PROJECT_DIR")
  [[ "$result" == *"database"* ]]
  [[ "$result" == *"saas"* ]]
  [[ "$result" == *"ai"* ]]
}

@test "detect_tech_stack: detects stack from monorepo sub-projects" {
  mkdir -p "$TEST_PROJECT_DIR/apps/api" "$TEST_PROJECT_DIR/apps/web"
  cat > "$TEST_PROJECT_DIR/pnpm-workspace.yaml" <<'EOF'
packages:
  - 'apps/*'
EOF
  touch "$TEST_PROJECT_DIR/apps/api/go.mod"
  cat > "$TEST_PROJECT_DIR/apps/web/package.json" <<'EOF'
{"dependencies":{"react":"^18.0.0"}}
EOF
  result=$(detect_tech_stack "$TEST_PROJECT_DIR")
  [[ "$result" == *"go"* ]]
  [[ "$result" == *"node"* ]]
  [[ "$result" == *"react"* ]]
}

@test "detect_deep_domains: monorepo with graphql in sub-project" {
  mkdir -p "$TEST_PROJECT_DIR/apps/api/graph"
  cat > "$TEST_PROJECT_DIR/pnpm-workspace.yaml" <<'EOF'
packages:
  - 'apps/*'
EOF
  touch "$TEST_PROJECT_DIR/apps/api/graph/schema.graphql"
  result=$(detect_deep_domains "$TEST_PROJECT_DIR")
  [[ "$result" == *"graphql"* ]]
}

@test "detect_deep_domains: monorepo with migrations in sub-project" {
  mkdir -p "$TEST_PROJECT_DIR/apps/api/migrations"
  cat > "$TEST_PROJECT_DIR/pnpm-workspace.yaml" <<'EOF'
packages:
  - 'apps/*'
EOF
  touch "$TEST_PROJECT_DIR/apps/api/migrations/001_init.sql"
  result=$(detect_deep_domains "$TEST_PROJECT_DIR")
  [[ "$result" == *"database"* ]]
}

@test "detect_deep_domains: multiple domains detected from complex project" {
  # GraphQL + AI + Database
  mkdir -p "$TEST_PROJECT_DIR/src"
  touch "$TEST_PROJECT_DIR/src/schema.graphql"
  mkdir -p "$TEST_PROJECT_DIR/migrations"
  touch "$TEST_PROJECT_DIR/migrations/001.sql"
  cat > "$TEST_PROJECT_DIR/package.json" <<'EOF'
{"dependencies":{"openai":"^4.0.0","graphql":"^16.0.0"}}
EOF
  result=$(detect_deep_domains "$TEST_PROJECT_DIR")
  [[ "$result" == *"graphql"* ]]
  [[ "$result" == *"ai"* ]]
  [[ "$result" == *"database"* ]]
}

# ── install_agents with domain detection ──

@test "install_agents: generic agents always installed" {
  local templates_dir="$TOOLKIT_ROOT/templates"
  install_agents "$TEST_PROJECT_DIR" "$templates_dir" ""
  assert_file_exists "$TEST_PROJECT_DIR/.claude/agents/code-reviewer.md"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/agents/security-reviewer.md"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/agents/tdd-guide.md"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/agents/planner.md"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/agents/architect-reviewer.md"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/agents/performance-reviewer.md"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/agents/incident-debugger.md"
}

@test "install_agents: domain agents installed when domains detected" {
  local templates_dir="$TOOLKIT_ROOT/templates"
  install_agents "$TEST_PROJECT_DIR" "$templates_dir" "blockchain"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/agents/blockchain-developer.md"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/agents/smart-contract-reviewer.md"
}

@test "install_agents: react domain agents installed" {
  local templates_dir="$TOOLKIT_ROOT/templates"
  install_agents "$TEST_PROJECT_DIR" "$templates_dir" "react"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/agents/frontend-developer.md"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/agents/ui-designer.md"
}

@test "install_agents: golang domain agent installed" {
  local templates_dir="$TOOLKIT_ROOT/templates"
  install_agents "$TEST_PROJECT_DIR" "$templates_dir" "golang"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/agents/go-backend-architect.md"
}

@test "install_agents: multiple domains install all relevant agents" {
  local templates_dir="$TOOLKIT_ROOT/templates"
  install_agents "$TEST_PROJECT_DIR" "$templates_dir" "blockchain react golang"
  # Generic always present
  assert_file_exists "$TEST_PROJECT_DIR/.claude/agents/code-reviewer.md"
  # Domain agents
  assert_file_exists "$TEST_PROJECT_DIR/.claude/agents/blockchain-developer.md"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/agents/smart-contract-reviewer.md"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/agents/frontend-developer.md"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/agents/ui-designer.md"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/agents/go-backend-architect.md"
}

@test "install_agents: no domain agents for empty domains" {
  local templates_dir="$TOOLKIT_ROOT/templates"
  install_agents "$TEST_PROJECT_DIR" "$templates_dir" ""
  # Generic present
  assert_file_exists "$TEST_PROJECT_DIR/.claude/agents/code-reviewer.md"
  # No domain agents (check a representative sample)
  [ ! -f "$TEST_PROJECT_DIR/.claude/agents/blockchain-developer.md" ]
  [ ! -f "$TEST_PROJECT_DIR/.claude/agents/frontend-developer.md" ]
  [ ! -f "$TEST_PROJECT_DIR/.claude/agents/go-backend-architect.md" ]
}

@test "install_agents: --force overwrites existing domain agents" {
  local templates_dir="$TOOLKIT_ROOT/templates"
  # Install blockchain agents
  install_agents "$TEST_PROJECT_DIR" "$templates_dir" "blockchain"
  # Modify a file
  echo "modified" > "$TEST_PROJECT_DIR/.claude/agents/blockchain-developer.md"
  # Reinstall without force — should not overwrite
  install_agents "$TEST_PROJECT_DIR" "$templates_dir" "blockchain"
  assert_file_contains "$TEST_PROJECT_DIR/.claude/agents/blockchain-developer.md" "modified"
  # Reinstall with force — should overwrite
  FORCE=true install_agents "$TEST_PROJECT_DIR" "$templates_dir" "blockchain"
  ! grep -q "modified" "$TEST_PROJECT_DIR/.claude/agents/blockchain-developer.md"
}

@test "install_agents: ai domain agents installed" {
  local templates_dir="$TOOLKIT_ROOT/templates"
  install_agents "$TEST_PROJECT_DIR" "$templates_dir" "ai"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/agents/ai-engineer.md"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/agents/prompt-engineer.md"
}

@test "install_agents: docker domain agent installed" {
  local templates_dir="$TOOLKIT_ROOT/templates"
  install_agents "$TEST_PROJECT_DIR" "$templates_dir" "docker"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/agents/cloud-architect.md"
}

@test "install_agents: kubernetes domain agent installed" {
  local templates_dir="$TOOLKIT_ROOT/templates"
  install_agents "$TEST_PROJECT_DIR" "$templates_dir" "kubernetes"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/agents/kubernetes-architect.md"
}

@test "install_agents: observability domain agent installed" {
  local templates_dir="$TOOLKIT_ROOT/templates"
  install_agents "$TEST_PROJECT_DIR" "$templates_dir" "observability"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/agents/observability-engineer.md"
}

@test "install_agents: saas domain agent installed" {
  local templates_dir="$TOOLKIT_ROOT/templates"
  install_agents "$TEST_PROJECT_DIR" "$templates_dir" "saas"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/agents/payment-integration.md"
}

@test "install_agents: graphql domain agent installed" {
  local templates_dir="$TOOLKIT_ROOT/templates"
  install_agents "$TEST_PROJECT_DIR" "$templates_dir" "graphql"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/agents/graphql-architect.md"
}

@test "install_agents: database domain agent installed" {
  local templates_dir="$TOOLKIT_ROOT/templates"
  install_agents "$TEST_PROJECT_DIR" "$templates_dir" "database"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/agents/database-architect.md"
}

@test "install_agents: dotnet domain agent installed" {
  local templates_dir="$TOOLKIT_ROOT/templates"
  install_agents "$TEST_PROJECT_DIR" "$templates_dir" "dotnet"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/agents/dotnet-architect.md"
}

@test "install_agents: unity domain agent installed" {
  local templates_dir="$TOOLKIT_ROOT/templates"
  install_agents "$TEST_PROJECT_DIR" "$templates_dir" "unity"
  assert_file_exists "$TEST_PROJECT_DIR/.claude/agents/unity-developer.md"
}
