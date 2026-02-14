#!/bin/bash
set -e

# Integration test: full install into temp directory
# Usage: ./tests/validation/validate_install.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TOOLKIT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

RED='\033[0;31m'
GREEN='\033[0;32m'
BOLD='\033[1m'
NC='\033[0m'

PASS=0
FAIL=0

assert() {
  local desc="$1"
  shift
  if "$@" 2>/dev/null; then
    echo -e "  ${GREEN}✓${NC} $desc"
    ((PASS++))
  else
    echo -e "  ${RED}✗${NC} $desc"
    ((FAIL++))
  fi
}

echo -e "${BOLD}Validate Install — Integration Test${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Create temp project with Go + Node (React) stack
TEST_DIR="$(mktemp -d)"
trap "rm -rf $TEST_DIR" EXIT

echo -e "\n${BOLD}Setup:${NC} Creating test project at $TEST_DIR"
cd "$TEST_DIR"
git init -q
cat > go.mod <<'EOF'
module test-project
go 1.21
EOF
cat > package.json <<'EOF'
{
  "name": "test-frontend",
  "dependencies": { "react": "^18.0.0" }
}
EOF
cat > Makefile <<'EOF'
test:
	@echo "running tests"
lint:
	@echo "running lint"
EOF
mkdir -p Assets  # Unity detection

# Run installer in auto mode
echo -e "\n${BOLD}Running:${NC} install.sh --auto --project-dir $TEST_DIR"
"$TOOLKIT_ROOT/install.sh" --auto --project-dir "$TEST_DIR" 2>&1 || true

echo -e "\n${BOLD}Checking: Project files${NC}"
assert "tools/ralph/prd.json.example exists" test -f "$TEST_DIR/tools/ralph/prd.json.example"
assert "tools/ralph/ralph.sh removed" test ! -f "$TEST_DIR/tools/ralph/ralph.sh"
assert "tools/ralph/RALPH.md removed" test ! -f "$TEST_DIR/tools/ralph/RALPH.md"
assert "tools/qa/qa.sh exists" test -f "$TEST_DIR/tools/qa/qa.sh"
assert "tools/qa/QA_PROMPT.md exists" test -f "$TEST_DIR/tools/qa/QA_PROMPT.md"
assert "tools/qa/qa.sh is executable" test -x "$TEST_DIR/tools/qa/qa.sh"

echo -e "\n${BOLD}Checking: Config files${NC}"
assert ".mcp.json exists" test -f "$TEST_DIR/.mcp.json"
assert ".deep-think.json exists" test -f "$TEST_DIR/.deep-think.json"
assert ".claude-toolkit.json exists" test -f "$TEST_DIR/.claude-toolkit.json"

echo -e "\n${BOLD}Checking: .mcp.json servers${NC}"
assert "deep-think server configured" jq -e '.mcpServers["deep-think"]' "$TEST_DIR/.mcp.json"

echo -e "\n${BOLD}Checking: .claude-toolkit.json${NC}"
assert "version is set" jq -e '.version' "$TEST_DIR/.claude-toolkit.json"
assert "project name is set" jq -e '.project.name' "$TEST_DIR/.claude-toolkit.json"
assert "tech stack detected" jq -e '.project.techStack | length > 0' "$TEST_DIR/.claude-toolkit.json"
assert "test command detected" jq -e '.commands.test != null' "$TEST_DIR/.claude-toolkit.json"
assert "lint command detected" jq -e '.commands.lint != null' "$TEST_DIR/.claude-toolkit.json"

echo -e "\n${BOLD}Checking: Rules${NC}"
assert ".claude/rules/common/ exists" test -d "$TEST_DIR/.claude/rules/common"
assert "common/coding-style.md exists" test -f "$TEST_DIR/.claude/rules/common/coding-style.md"

echo -e "\n${BOLD}Checking: Skills${NC}"
assert ".claude/skills/ exists" test -d "$TEST_DIR/.claude/skills"

echo -e "\n${BOLD}Checking: Agents${NC}"
assert ".claude/agents/ exists" test -d "$TEST_DIR/.claude/agents"

echo -e "\n${BOLD}Checking: .gitignore${NC}"
assert ".gitignore exists" test -f "$TEST_DIR/.gitignore"
assert "prd.json in .gitignore" grep -q "prd.json" "$TEST_DIR/.gitignore"

echo -e "\n${BOLD}Checking: Global commands${NC}"
assert "~/.claude/commands/ralph.md" test -f "$HOME/.claude/commands/ralph.md"
assert "~/.claude/commands/qa.md" test -f "$HOME/.claude/commands/qa.md"

# Summary
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "Results: ${GREEN}$PASS passed${NC}, ${RED}$FAIL failed${NC}"

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
