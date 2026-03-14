#!/bin/bash
# Cross-platform install.sh integration test
# Exit 0: all checks passed
# Exit 1: one or more checks failed

set -euo pipefail

TOOLKIT_DIR="/toolkit"
TEST_DIR="/test-project"

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
    printf "  ${GREEN}PASS${NC} %s\n" "$desc"
    PASS=$((PASS + 1))
  else
    printf "  ${RED}FAIL${NC} %s\n" "$desc"
    FAIL=$((FAIL + 1))
  fi
}

printf "${BOLD}Claude Toolkit — Cross-Platform Install Test${NC}\n"
printf "Platform: %s\n" "$(uname -s -r)"
printf "═══════════════════════════════════════════\n\n"

# ── Step 1: Run installer ────────────────────────────────────────────────────
printf "${BOLD}Running install.sh --auto ...${NC}\n"

# install.sh sources lib/ relative to itself; HOME needs to be writable for
# global ~/.claude writes. We point HOME at a temp dir to keep the container clean.
export HOME="/tmp/test-home"
mkdir -p "$HOME"

cd "$TEST_DIR"

if bash "$TOOLKIT_DIR/install.sh" --auto --project-dir "$TEST_DIR" 2>&1; then
  printf "  install.sh completed\n\n"
else
  # install.sh may exit non-zero on partial failures (e.g., no MCP binary).
  # We still check the artifacts that should have been written.
  printf "  install.sh exited non-zero — checking artifacts anyway\n\n"
fi

# ── Step 2: Verify .claude/ structure ───────────────────────────────────────
printf "${BOLD}Checking: .claude/ directory structure${NC}\n"
assert ".claude/ directory exists"          test -d "$TEST_DIR/.claude"
assert ".claude/rules/ exists"             test -d "$TEST_DIR/.claude/rules"
assert ".claude/rules/common/ exists"      test -d "$TEST_DIR/.claude/rules/common"
assert ".claude/skills/ exists"            test -d "$TEST_DIR/.claude/skills"
assert ".claude/agents/ exists"            test -d "$TEST_DIR/.claude/agents"

# ── Step 3: Verify rules files ──────────────────────────────────────────────
printf "\n${BOLD}Checking: Rules files${NC}\n"
assert "common/coding-style.md exists"     test -f "$TEST_DIR/.claude/rules/common/coding-style.md"
assert "common/git-workflow.md exists"     test -f "$TEST_DIR/.claude/rules/common/git-workflow.md"
assert "common/testing.md exists"          test -f "$TEST_DIR/.claude/rules/common/testing.md"
assert "common/security.md exists"         test -f "$TEST_DIR/.claude/rules/common/security.md"

# ── Step 4: Verify config files ─────────────────────────────────────────────
printf "\n${BOLD}Checking: Config files${NC}\n"
assert ".claude-toolkit.json exists"       test -f "$TEST_DIR/.claude-toolkit.json"
assert ".deep-think.json exists"           test -f "$TEST_DIR/.deep-think.json"
assert ".mcp.json or ~/.claude.json exists" test -f "$TEST_DIR/.mcp.json" -o -f "$HOME/.claude.json"

# ── Step 5: Verify .claude-toolkit.json has expected keys ───────────────────
printf "\n${BOLD}Checking: .claude-toolkit.json content${NC}\n"
if command -v jq >/dev/null 2>&1 && test -f "$TEST_DIR/.claude-toolkit.json"; then
  assert "version key present"             jq -e '.version'        "$TEST_DIR/.claude-toolkit.json"
  assert "project.name key present"        jq -e '.project.name'   "$TEST_DIR/.claude-toolkit.json"
  assert "techStack is non-empty array"    jq -e '.project.techStack | length > 0' "$TEST_DIR/.claude-toolkit.json"
else
  printf "  SKIP jq not available or config missing\n"
fi

# ── Step 6: Verify hooks/scripts if present ──────────────────────────────────
printf "\n${BOLD}Checking: Hooks (if installed)${NC}\n"
# Hooks are optional — record if present, do not fail if absent
if test -d "$TEST_DIR/.claude/hooks"; then
  printf "  INFO .claude/hooks/ found\n"
  HOOK_COUNT=$(find "$TEST_DIR/.claude/hooks" -type f | wc -l | tr -d ' ')
  printf "  INFO %s hook file(s) installed\n" "$HOOK_COUNT"
else
  printf "  INFO .claude/hooks/ not present (hooks may be skipped without git remote)\n"
fi

# ── Summary ──────────────────────────────────────────────────────────────────
printf "\n═══════════════════════════════════════════\n"
printf "Results: ${GREEN}%d passed${NC}, ${RED}%d failed${NC}\n" "$PASS" "$FAIL"

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi

exit 0
