#!/bin/bash
set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m'

TOOLKIT_VERSION="1.0.0"
TOOLKIT_DIR="${CLAUDE_TOOLKIT_DIR:-$HOME/.claude/toolkit}"
TOOLKIT_REPO="https://github.com/bis-code/claude-toolkit.git"

# ─────────────────────────────────────────────
# Helpers
# ─────────────────────────────────────────────

info()    { echo -e "    ${GREEN}✓${NC} $1"; }
warn()    { echo -e "    ${YELLOW}⚠${NC} $1"; }
error()   { echo -e "    ${RED}✗${NC} $1"; }
header()  { echo -e "\n${BOLD}[$1]${NC} $2"; }

check_cmd() {
  if command -v "$1" &> /dev/null; then
    info "$1"
    return 0
  else
    error "$1 — $2"
    return 1
  fi
}

# Merge JSON key into .mcp.json without overwriting existing servers
merge_mcp_server() {
  local file="$1" name="$2" config="$3"
  if [ ! -f "$file" ]; then
    echo "{\"mcpServers\":{\"$name\":$config}}" | jq '.' > "$file"
  elif jq -e ".mcpServers.\"$name\"" "$file" &>/dev/null; then
    return 0  # Already exists
  else
    jq --arg name "$name" --argjson config "$config" '.mcpServers[$name] = $config' "$file" > "$file.tmp" && mv "$file.tmp" "$file"
  fi
}

# Append lines to .gitignore if not already present
append_gitignore() {
  local gitignore="$1" entries_file="$2"
  if [ ! -f "$gitignore" ]; then
    cp "$entries_file" "$gitignore"
    return
  fi
  while IFS= read -r line; do
    [ -z "$line" ] && continue
    [[ "$line" == \#* ]] && {
      # Only add comment if next non-empty line will be added
      continue
    }
    if ! grep -qF "$line" "$gitignore" 2>/dev/null; then
      echo "$line" >> "$gitignore"
    fi
  done < "$entries_file"
}

detect_tech_stack() {
  local dir="$1"
  local stack=()
  [ -f "$dir/go.mod" ]         && stack+=("go")
  [ -f "$dir/package.json" ]   && stack+=("node")
  [ -f "$dir/Cargo.toml" ]     && stack+=("rust")
  [ -f "$dir/requirements.txt" ] || [ -f "$dir/pyproject.toml" ] && stack+=("python")
  [ -f "$dir/Makefile" ]       && stack+=("make")
  [ -d "$dir/unity" ] || [ -d "$dir/Assets" ] && stack+=("unity")

  # Check for .csproj/.sln files
  local csproj_count
  csproj_count=$(find "$dir" -maxdepth 2 -name "*.csproj" 2>/dev/null | head -5 | wc -l | tr -d ' ')
  [ "$csproj_count" -gt 0 ] && stack+=("dotnet")

  # Check for Solidity/Hardhat
  [ -f "$dir/hardhat.config.ts" ] || [ -f "$dir/hardhat.config.js" ] && stack+=("solidity")
  [ -f "$dir/foundry.toml" ] && stack+=("solidity")

  # Check for frontend frameworks
  if [ -f "$dir/package.json" ]; then
    if grep -q '"react"' "$dir/package.json" 2>/dev/null; then
      stack+=("react")
    elif grep -q '"vue"' "$dir/package.json" 2>/dev/null; then
      stack+=("vue")
    elif grep -q '"svelte"' "$dir/package.json" 2>/dev/null; then
      stack+=("svelte")
    fi
  fi

  echo "${stack[*]}"
}

detect_test_command() {
  local dir="$1" stack="$2"
  # Check Makefile first
  if [ -f "$dir/Makefile" ]; then
    local test_target
    test_target=$(grep -E '^test[a-zA-Z_-]*:' "$dir/Makefile" 2>/dev/null | head -1 | cut -d: -f1)
    if [ -n "$test_target" ]; then
      echo "make $test_target"
      return
    fi
  fi
  # Fallback by stack
  case "$stack" in
    *go*)      echo "go test ./..." ;;
    *node*)    echo "npm test" ;;
    *dotnet*)  echo "dotnet test" ;;
    *python*)  echo "pytest" ;;
    *rust*)    echo "cargo test" ;;
    *solidity*) echo "npx hardhat test" ;;
    *)         echo "" ;;
  esac
}

detect_lint_command() {
  local dir="$1" stack="$2"
  if [ -f "$dir/Makefile" ]; then
    local lint_target
    lint_target=$(grep -E '^lint[a-zA-Z_-]*:' "$dir/Makefile" 2>/dev/null | head -1 | cut -d: -f1)
    if [ -n "$lint_target" ]; then
      echo "make $lint_target"
      return
    fi
  fi
  case "$stack" in
    *go*)      echo "golangci-lint run" ;;
    *node*)    echo "npm run lint" ;;
    *dotnet*)  echo "dotnet format --verify-no-changes" ;;
    *python*)  echo "ruff check ." ;;
    *rust*)    echo "cargo clippy" ;;
    *)         echo "" ;;
  esac
}

detect_scan_categories() {
  local stack="$1"
  local categories=("tests" "lint" "missing-tests" "todo-audit")

  case "$stack" in
    *go*|*node*|*dotnet*|*python*|*rust*)
      categories+=("module-boundaries" "security-scan")
      ;;
  esac

  if [[ "$stack" == *react* ]] || [[ "$stack" == *vue* ]] || [[ "$stack" == *svelte* ]]; then
    categories+=("accessibility" "component-quality")
  fi

  if [[ "$stack" == *solidity* ]]; then
    categories+=("smart-contract-security" "gas-optimization")
  fi

  if [[ "$stack" == *node* ]] || [[ "$stack" == *react* ]]; then
    categories+=("browser-testing")
  fi

  echo "${categories[*]}"
}

# ─────────────────────────────────────────────
# Bootstrap: ensure toolkit is installed locally
# ─────────────────────────────────────────────

bootstrap_toolkit() {
  # Are we running from within the cloned toolkit repo?
  local script_dir
  script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

  if [ -f "$script_dir/templates/ralph/RALPH.md" ]; then
    # Running from within the repo — use it directly
    TOOLKIT_DIR="$script_dir"
    return 0
  fi

  # Check if toolkit is already installed
  if [ -d "$TOOLKIT_DIR" ] && [ -f "$TOOLKIT_DIR/templates/ralph/RALPH.md" ]; then
    return 0
  fi

  # Need to clone
  echo -e "${BOLD}Downloading Claude Toolkit...${NC}"
  git clone "$TOOLKIT_REPO" "$TOOLKIT_DIR" 2>/dev/null
  info "Installed to $TOOLKIT_DIR"
}

# ─────────────────────────────────────────────
# Main
# ─────────────────────────────────────────────

MODE="install"
FORCE=false

while [[ $# -gt 0 ]]; do
  case $1 in
    --update) MODE="update"; shift ;;
    --force) FORCE=true; shift ;;
    --uninstall) MODE="uninstall"; shift ;;
    -h|--help)
      echo "Claude Code Toolkit Installer v$TOOLKIT_VERSION"
      echo ""
      echo "Usage: install.sh [OPTIONS]"
      echo ""
      echo "Options:"
      echo "  --update      Update toolkit files in current project"
      echo "  --force       Overwrite existing files without asking"
      echo "  --uninstall   Remove toolkit from current project"
      echo "  -h, --help    Show this help"
      echo ""
      echo "Run from any project root to install /ralph and /qa."
      exit 0
      ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

echo ""
echo -e "${BOLD}Claude Code Toolkit Installer v$TOOLKIT_VERSION${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# ── Step 1: Bootstrap ──
bootstrap_toolkit

TEMPLATES="$TOOLKIT_DIR/templates"

# ── Step 2: Prerequisites ──
header "1" "Prerequisites"
PREREQS_OK=true
check_cmd "claude" "Install: https://docs.anthropic.com/en/docs/claude-code" || PREREQS_OK=false
check_cmd "jq"     "Install: brew install jq" || PREREQS_OK=false
check_cmd "git"    "Install: brew install git" || PREREQS_OK=false
check_cmd "gh"     "Install: brew install gh (optional, for /ralph issue fetching)" || true
check_cmd "npm"    "Install: brew install node (needed for MCP servers)" || PREREQS_OK=false

if [ "$PREREQS_OK" = false ]; then
  echo ""
  error "Missing required prerequisites. Install them and re-run."
  exit 1
fi

# ── Step 3: Detect project ──
PROJECT_DIR="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
PROJECT_NAME="$(basename "$PROJECT_DIR")"

header "2" "Project detection: $PROJECT_DIR"
info "Git repo: $PROJECT_NAME"

TECH_STACK=$(detect_tech_stack "$PROJECT_DIR")
if [ -n "$TECH_STACK" ]; then
  info "Tech stack: $TECH_STACK"
else
  warn "No tech stack detected (will rely on CLAUDE.md)"
fi

if [ -f "$PROJECT_DIR/.mcp.json" ]; then
  EXISTING_MCPS=$(jq -r '.mcpServers | keys | join(", ")' "$PROJECT_DIR/.mcp.json" 2>/dev/null || echo "none")
  info "Existing .mcp.json: $EXISTING_MCPS"
fi

if [ -f "$PROJECT_DIR/.claude-toolkit.json" ]; then
  if [ "$MODE" = "install" ] && [ "$FORCE" = false ]; then
    warn ".claude-toolkit.json already exists (use --update to refresh)"
  fi
fi

# ── Step 4: MCP Servers ──
header "3" "MCP Server Configuration"

echo -e "\n    ${BOLD}REQUIRED (always installed):${NC}"

# Install deep-think (mandatory)
if command -v mcp-deep-think &> /dev/null; then
  info "deep-think — Structured reasoning (already installed)"
else
  echo -e "    ${YELLOW}→${NC} Installing mcp-deep-think..."
  npm install -g mcp-deep-think 2>/dev/null && info "deep-think installed" || error "Failed to install mcp-deep-think"
fi

echo -e "\n    ${BOLD}OPTIONAL (toggle with number, Enter to confirm):${NC}"

# Determine auto-suggestions
SUGGEST_PLAYWRIGHT=false
if [[ "$TECH_STACK" == *react* ]] || [[ "$TECH_STACK" == *vue* ]] || [[ "$TECH_STACK" == *svelte* ]] || [[ "$TECH_STACK" == *node* ]]; then
  SUGGEST_PLAYWRIGHT=true
fi

# MCP selection
INSTALL_PLAYWRIGHT=false
INSTALL_LEANN=false

if [ "$SUGGEST_PLAYWRIGHT" = true ]; then
  INSTALL_PLAYWRIGHT=true
fi

# Interactive selection
while true; do
  PW_MARK=" "; [ "$INSTALL_PLAYWRIGHT" = true ] && PW_MARK="x"
  LE_MARK=" "; [ "$INSTALL_LEANN" = true ] && LE_MARK="x"

  PW_NOTE=""; [ "$SUGGEST_PLAYWRIGHT" = true ] && PW_NOTE=" (suggested: UI project detected)"
  LE_NOTE=""
  if command -v leann_mcp &> /dev/null; then
    LE_NOTE=" (binary found)"
  fi

  echo ""
  echo -e "    [${PW_MARK}] 1. playwright     Browser testing & UI verification${PW_NOTE}"
  echo -e "    [${LE_MARK}] 2. leann-server   Semantic code search${LE_NOTE}"
  echo ""
  read -r -p "    Toggle [1-2], Enter to confirm: " choice

  case "$choice" in
    1) [ "$INSTALL_PLAYWRIGHT" = true ] && INSTALL_PLAYWRIGHT=false || INSTALL_PLAYWRIGHT=true ;;
    2) [ "$INSTALL_LEANN" = true ] && INSTALL_LEANN=false || INSTALL_LEANN=true ;;
    "") break ;;
    *) echo "    Invalid choice" ;;
  esac
done

# ── Step 5: QA Configuration ──
header "4" "QA Configuration"

TEST_CMD=$(detect_test_command "$PROJECT_DIR" "$TECH_STACK")
LINT_CMD=$(detect_lint_command "$PROJECT_DIR" "$TECH_STACK")
SCAN_CATS=$(detect_scan_categories "$TECH_STACK")

if [ -n "$TEST_CMD" ]; then
  read -r -p "    Test command: $TEST_CMD  [Y/n/edit]: " ans
  case "$ans" in
    n|N) TEST_CMD="" ;;
    "") ;;  # Accept default
    *) TEST_CMD="$ans" ;;
  esac
fi

if [ -n "$LINT_CMD" ]; then
  read -r -p "    Lint command: $LINT_CMD  [Y/n/edit]: " ans
  case "$ans" in
    n|N) LINT_CMD="" ;;
    "") ;;
    *) LINT_CMD="$ans" ;;
  esac
fi

# Default branch detection
DEFAULT_BRANCH=$(git symbolic-ref refs/remotes/origin/HEAD 2>/dev/null | sed 's@^refs/remotes/origin/@@' || echo "main")
read -r -p "    QA worktree branch: $DEFAULT_BRANCH  [Y/n/edit]: " ans
case "$ans" in
  n|N) DEFAULT_BRANCH="main" ;;
  "") ;;
  *) DEFAULT_BRANCH="$ans" ;;
esac

# ── Step 6: Install files ──
header "5" "Installing"

# tools/ralph/
mkdir -p "$PROJECT_DIR/tools/ralph"
if [ ! -f "$PROJECT_DIR/tools/ralph/ralph.sh" ] || [ "$FORCE" = true ] || [ "$MODE" = "update" ]; then
  cp "$TEMPLATES/ralph/ralph.sh" "$PROJECT_DIR/tools/ralph/ralph.sh"
  cp "$TEMPLATES/ralph/RALPH.md" "$PROJECT_DIR/tools/ralph/RALPH.md"
  cp "$TEMPLATES/ralph/prd.json.example" "$PROJECT_DIR/tools/ralph/prd.json.example"
  chmod +x "$PROJECT_DIR/tools/ralph/ralph.sh"
  info "tools/ralph/ — 3 files"
else
  warn "tools/ralph/ already exists (use --force to overwrite)"
fi

# tools/qa/
mkdir -p "$PROJECT_DIR/tools/qa"
if [ ! -f "$PROJECT_DIR/tools/qa/qa.sh" ] || [ "$FORCE" = true ] || [ "$MODE" = "update" ]; then
  cp "$TEMPLATES/qa/qa.sh" "$PROJECT_DIR/tools/qa/qa.sh"
  cp "$TEMPLATES/qa/QA_PROMPT.md" "$PROJECT_DIR/tools/qa/QA_PROMPT.md"
  chmod +x "$PROJECT_DIR/tools/qa/qa.sh"
  info "tools/qa/ — 2 files"
else
  warn "tools/qa/ already exists (use --force to overwrite)"
fi

# .mcp.json — merge servers
merge_mcp_server "$PROJECT_DIR/.mcp.json" "deep-think" '{"command":"mcp-deep-think","args":[]}'

if [ "$INSTALL_PLAYWRIGHT" = true ]; then
  merge_mcp_server "$PROJECT_DIR/.mcp.json" "playwright" '{"command":"npx","args":["@playwright/mcp@latest","--headless","--isolated"]}'
fi

if [ "$INSTALL_LEANN" = true ]; then
  merge_mcp_server "$PROJECT_DIR/.mcp.json" "leann-server" '{"command":"leann_mcp","args":[]}'
fi

INSTALLED_MCPS=$(jq -r '.mcpServers | keys | join(", ")' "$PROJECT_DIR/.mcp.json" 2>/dev/null)
info ".mcp.json — servers: $INSTALLED_MCPS"

# .deep-think.json
if [ ! -f "$PROJECT_DIR/.deep-think.json" ] || [ "$FORCE" = true ] || [ "$MODE" = "update" ]; then
  cp "$TEMPLATES/deep-think.json" "$PROJECT_DIR/.deep-think.json"
  info ".deep-think.json — created with $(jq '.strategies | length' "$TEMPLATES/deep-think.json") strategies"
else
  warn ".deep-think.json already exists (use --force to overwrite)"
fi

# .claude-toolkit.json
TOOLKIT_CONFIG="$PROJECT_DIR/.claude-toolkit.json"
if [ ! -f "$TOOLKIT_CONFIG" ] || [ "$FORCE" = true ] || [ "$MODE" = "update" ]; then
  # Build scan categories as JSON array
  SCAN_JSON="["
  first=true
  for cat in $SCAN_CATS; do
    [ "$first" = true ] && first=false || SCAN_JSON+=","
    SCAN_JSON+="\"$cat\""
  done
  SCAN_JSON+="]"

  # Build installed MCPs list
  MCPS_JSON='["deep-think"'
  [ "$INSTALL_PLAYWRIGHT" = true ] && MCPS_JSON+=',"playwright"'
  [ "$INSTALL_LEANN" = true ] && MCPS_JSON+=',"leann-server"'
  MCPS_JSON+=']'

  # Build tech stack array
  STACK_JSON="["
  first=true
  for t in $TECH_STACK; do
    [ "$first" = true ] && first=false || STACK_JSON+=","
    STACK_JSON+="\"$t\""
  done
  STACK_JSON+="]"

  cat > "$TOOLKIT_CONFIG" <<EOFCONFIG
{
  "version": "$TOOLKIT_VERSION",
  "project": {
    "name": "$PROJECT_NAME",
    "techStack": $STACK_JSON
  },
  "commands": {
    "test": $([ -n "$TEST_CMD" ] && echo "\"$TEST_CMD\"" || echo "null"),
    "lint": $([ -n "$LINT_CMD" ] && echo "\"$LINT_CMD\"" || echo "null")
  },
  "qa": {
    "scanCategories": $SCAN_JSON,
    "maxFixLines": 30,
    "worktreeFromBranch": "$DEFAULT_BRANCH"
  },
  "ralph": {
    "maxLoops": 30,
    "stuckThreshold": 3
  },
  "mcpServers": {
    "required": ["deep-think"],
    "installed": $MCPS_JSON
  }
}
EOFCONFIG
  # Pretty-print
  jq '.' "$TOOLKIT_CONFIG" > "$TOOLKIT_CONFIG.tmp" && mv "$TOOLKIT_CONFIG.tmp" "$TOOLKIT_CONFIG"
  info ".claude-toolkit.json — project config created"
else
  warn ".claude-toolkit.json already exists (use --force to overwrite)"
fi

# .gitignore — append entries
if [ -f "$TEMPLATES/gitignore-entries.txt" ]; then
  append_gitignore "$PROJECT_DIR/.gitignore" "$TEMPLATES/gitignore-entries.txt"
  info ".gitignore — runtime entries added"
fi

# ── Step 7: Global commands ──
header "6" "Global Claude Code commands"

COMMANDS_DIR="$HOME/.claude/commands"
mkdir -p "$COMMANDS_DIR"

if [ ! -f "$COMMANDS_DIR/ralph.md" ] || [ "$FORCE" = true ] || [ "$MODE" = "update" ]; then
  cp "$TOOLKIT_DIR/commands/ralph.md" "$COMMANDS_DIR/ralph.md"
  info "/ralph command installed"
else
  info "/ralph command (already exists)"
fi

if [ ! -f "$COMMANDS_DIR/qa.md" ] || [ "$FORCE" = true ] || [ "$MODE" = "update" ]; then
  cp "$TOOLKIT_DIR/commands/qa.md" "$COMMANDS_DIR/qa.md"
  info "/qa command installed"
else
  info "/qa command (already exists)"
fi

# ── Done ──
echo ""
echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}${BOLD}Done!${NC}"
echo ""
echo "  Next steps:"
echo "    1. Start Claude Code in this project"
echo "    2. Run /ralph --issues 1,2 to build features from GitHub issues"
echo "    3. Run /qa to scan and fix quality issues"
echo ""
echo "  Update toolkit:  ~/.claude/toolkit/install.sh --update"
echo "  Toolkit repo:    $TOOLKIT_DIR"
echo ""
