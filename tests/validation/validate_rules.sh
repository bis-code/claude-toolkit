#!/bin/bash
set -e

# Validate rules installation for different stacks
# Usage: ./tests/validation/validate_rules.sh

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

source "$TOOLKIT_ROOT/lib/utils.sh"
source "$TOOLKIT_ROOT/lib/detect.sh"
source "$TOOLKIT_ROOT/lib/install_rules.sh"

echo -e "${BOLD}Validate Rules — Stack-Specific Installation${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Test each language mapping
test_language() {
  local lang="$1"
  local expected_dir="$2"
  shift 2
  local expected_files=("$@")

  local dir="$(mktemp -d)"
  trap "rm -rf $dir" RETURN

  echo -e "\n${BOLD}Language: $lang${NC}"
  FORCE=true install_rules "$dir" "$lang" "$TOOLKIT_ROOT/templates"

  assert "common/ installed" test -d "$dir/.claude/rules/common"
  assert "$expected_dir/ installed" test -d "$dir/.claude/rules/$expected_dir"

  for f in "${expected_files[@]}"; do
    assert "$expected_dir/$f exists" test -f "$dir/.claude/rules/$expected_dir/$f"
  done
}

test_language "golang" "golang" "coding-standards.md" "patterns.md" "testing.md"
test_language "typescript" "typescript" "coding-standards.md" "react-patterns.md" "testing.md"
test_language "python" "python" "coding-standards.md" "django-patterns.md" "testing.md"
test_language "csharp" "csharp" "coding-standards.md" "unity-patterns.md" "testing.md"
test_language "solidity" "solidity" "coding-standards.md" "security.md" "testing.md"
test_language "java" "java" "coding-standards.md" "springboot-patterns.md" "testing.md"
test_language "rust" "rust" "coding-standards.md" "patterns.md" "testing.md"
test_language "docker" "docker" "best-practices.md" "security.md"

# Summary
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "Results: ${GREEN}$PASS passed${NC}, ${RED}$FAIL failed${NC}"

[ "$FAIL" -gt 0 ] && exit 1
exit 0
