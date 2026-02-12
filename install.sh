#!/bin/bash
set -e

TOOLKIT_VERSION="2.0.0"
TOOLKIT_DIR="${CLAUDE_TOOLKIT_DIR:-$HOME/.claude/toolkit}"
TOOLKIT_REPO="https://github.com/bis-code/claude-toolkit.git"

# ─────────────────────────────────────────────
# Bootstrap: source lib/ modules
# ─────────────────────────────────────────────

resolve_toolkit_dir() {
  local script_dir
  script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

  if [ -f "$script_dir/lib/utils.sh" ]; then
    TOOLKIT_DIR="$script_dir"
    return 0
  fi

  if [ -d "$TOOLKIT_DIR" ] && [ -f "$TOOLKIT_DIR/lib/utils.sh" ]; then
    return 0
  fi

  echo "Downloading Claude Toolkit..."
  git clone "$TOOLKIT_REPO" "$TOOLKIT_DIR" 2>/dev/null
}

resolve_toolkit_dir

source "$TOOLKIT_DIR/lib/utils.sh"
source "$TOOLKIT_DIR/lib/detect.sh"
source "$TOOLKIT_DIR/lib/install_rules.sh"
source "$TOOLKIT_DIR/lib/install_skills.sh"
source "$TOOLKIT_DIR/lib/install_hooks.sh"
source "$TOOLKIT_DIR/lib/install_commands.sh"
source "$TOOLKIT_DIR/lib/install_agents.sh"
source "$TOOLKIT_DIR/lib/install_mcp.sh"

TEMPLATES="$TOOLKIT_DIR/templates"

# ─────────────────────────────────────────────
# Parse arguments
# ─────────────────────────────────────────────

MODE="install"
FORCE=false
AUTO_MODE=false
LANGUAGES=""
SKIP_RULES=false
SKIP_SKILLS=false
SKIP_HOOKS=false
SKIP_AGENTS=false
DRY_RUN=false
PROJECT_DIR=""

while [[ $# -gt 0 ]]; do
  case $1 in
    --update)      MODE="update"; shift ;;
    --force)       FORCE=true; shift ;;
    --uninstall)   MODE="uninstall"; shift ;;
    --auto)        AUTO_MODE=true; shift ;;
    --languages)   LANGUAGES="$2"; shift 2 ;;
    --skip-rules)  SKIP_RULES=true; shift ;;
    --skip-skills) SKIP_SKILLS=true; shift ;;
    --skip-hooks)  SKIP_HOOKS=true; shift ;;
    --skip-agents) SKIP_AGENTS=true; shift ;;
    --dry-run)     DRY_RUN=true; shift ;;
    --project-dir) PROJECT_DIR="$2"; shift 2 ;;
    -h|--help)
      echo "Claude Code Toolkit Installer v$TOOLKIT_VERSION"
      echo ""
      echo "Usage: install.sh [OPTIONS]"
      echo ""
      echo "Modes:"
      echo "  --update           Update toolkit files in current project"
      echo "  --uninstall        Remove toolkit from current project"
      echo "  --auto             Non-interactive mode (auto-detect, install all)"
      echo ""
      echo "Options:"
      echo "  --force            Overwrite existing files without asking"
      echo "  --languages LANGS  Comma-separated: go,typescript,python,csharp,solidity,java,rust,docker"
      echo "  --skip-rules       Skip rules installation"
      echo "  --skip-skills      Skip skills installation"
      echo "  --skip-hooks       Skip hooks installation"
      echo "  --skip-agents      Skip agents installation"
      echo "  --dry-run          Show what would be installed without doing it"
      echo "  --project-dir DIR  Target directory (default: current or git root)"
      echo "  -h, --help         Show this help"
      exit 0
      ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

# Export for use in lib/ modules
export FORCE MODE

# ─────────────────────────────────────────────
# Step 1: Prerequisites
# ─────────────────────────────────────────────

echo ""
echo -e "${BOLD}Claude Code Toolkit Installer v$TOOLKIT_VERSION${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

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

# ─────────────────────────────────────────────
# Step 2: Detect project
# ─────────────────────────────────────────────

if [ -n "$PROJECT_DIR" ]; then
  PROJECT_DIR="$(cd "$PROJECT_DIR" && pwd)"
else
  if git rev-parse --show-toplevel &>/dev/null; then
    PROJECT_DIR="$(git rev-parse --show-toplevel)"
  else
    PROJECT_DIR="$(pwd)"
  fi
fi

IS_GIT_REPO=false
if git -C "$PROJECT_DIR" rev-parse --show-toplevel &>/dev/null 2>&1; then
  IS_GIT_REPO=true
fi

PROJECT_NAME="$(basename "$PROJECT_DIR")"
PROJECT_TYPE="repository"
[ "$IS_GIT_REPO" = false ] && PROJECT_TYPE="workspace"

header "2" "Project detection: $PROJECT_DIR"
if [ "$IS_GIT_REPO" = true ]; then
  info "Git repository: $PROJECT_NAME"
else
  info "Workspace (no git): $PROJECT_NAME"
  warn "QA will run in-place (no worktree isolation without git)"
fi

TECH_STACK=$(detect_tech_stack "$PROJECT_DIR")
if [ -n "$TECH_STACK" ]; then
  info "Tech stack: $TECH_STACK"
else
  warn "No tech stack detected (will rely on CLAUDE.md)"
fi

# Detect package manager
PKG_MANAGER=$(detect_package_manager "$PROJECT_DIR")
[ -n "$PKG_MANAGER" ] && info "Package manager: $PKG_MANAGER"

# Map stack to language rules
if [ -n "$LANGUAGES" ]; then
  # User specified languages via --languages flag (comma to space)
  DETECTED_LANGUAGES="${LANGUAGES//,/ }"
else
  DETECTED_LANGUAGES=$(map_stack_to_languages "$TECH_STACK")
fi
[ -n "$DETECTED_LANGUAGES" ] && info "Language rules: $DETECTED_LANGUAGES"

if [ -f "$PROJECT_DIR/.mcp.json" ]; then
  EXISTING_MCPS=$(jq -r '.mcpServers | keys | join(", ")' "$PROJECT_DIR/.mcp.json" 2>/dev/null || echo "none")
  info "Existing .mcp.json: $EXISTING_MCPS"
fi

if [ -f "$PROJECT_DIR/.claude-toolkit.json" ]; then
  if [ "$MODE" = "install" ] && [ "$FORCE" = false ]; then
    warn ".claude-toolkit.json already exists (use --update to refresh)"
  fi
fi

# ─────────────────────────────────────────────
# Step 3: MCP Server Configuration
# ─────────────────────────────────────────────

header "3" "MCP Server Configuration"

echo -e "\n    ${BOLD}REQUIRED (always installed):${NC}"

# Install deep-think (mandatory)
if command -v mcp-deep-think &> /dev/null; then
  info "deep-think — Structured reasoning (already installed)"
else
  echo -e "    ${YELLOW}→${NC} Installing mcp-deep-think..."
  npm install -g mcp-deep-think 2>/dev/null && info "deep-think installed" || error "Failed to install mcp-deep-think"
fi

# Determine auto-suggestions
SUGGEST_PLAYWRIGHT=false
if [[ "$TECH_STACK" == *react* ]] || [[ "$TECH_STACK" == *vue* ]] || [[ "$TECH_STACK" == *svelte* ]] || [[ "$TECH_STACK" == *node* ]]; then
  SUGGEST_PLAYWRIGHT=true
fi

INSTALL_PLAYWRIGHT=false
INSTALL_LEANN=false
INSTALL_CONTEXT7=false

if [ "$SUGGEST_PLAYWRIGHT" = true ]; then
  INSTALL_PLAYWRIGHT=true
fi

# Auto mode or update: install all / skip interactive selection
if [ "$AUTO_MODE" = true ]; then
  INSTALL_PLAYWRIGHT=$SUGGEST_PLAYWRIGHT
  INSTALL_LEANN=true
  INSTALL_CONTEXT7=true
elif [ "$MODE" = "update" ]; then
  # On update: skip interactive selection, keep existing MCP config.
  # install_mcp_config merges without overwriting, so existing servers are preserved.
  info "Update mode — keeping existing MCP server selection"
else
  # Interactive selection
  echo -e "\n    ${BOLD}OPTIONAL (toggle with number, Enter to confirm):${NC}"

  while true; do
    PW_MARK=" "; [ "$INSTALL_PLAYWRIGHT" = true ] && PW_MARK="x"
    LE_MARK=" "; [ "$INSTALL_LEANN" = true ] && LE_MARK="x"
    C7_MARK=" "; [ "$INSTALL_CONTEXT7" = true ] && C7_MARK="x"

    PW_NOTE=""; [ "$SUGGEST_PLAYWRIGHT" = true ] && PW_NOTE=" (suggested: UI project detected)"
    LE_NOTE=""
    command -v leann_mcp &> /dev/null && LE_NOTE=" (binary found)"

    echo ""
    echo -e "    [${PW_MARK}] 1. playwright     Browser testing & UI verification${PW_NOTE}"
    echo -e "    [${LE_MARK}] 2. leann-server   Semantic code search${LE_NOTE}"
    echo -e "    [${C7_MARK}] 3. context7       Live library documentation lookup"
    echo ""
    read -r -p "    Toggle [1-3], Enter to confirm: " choice

    case "$choice" in
      1) [ "$INSTALL_PLAYWRIGHT" = true ] && INSTALL_PLAYWRIGHT=false || INSTALL_PLAYWRIGHT=true ;;
      2) [ "$INSTALL_LEANN" = true ] && INSTALL_LEANN=false || INSTALL_LEANN=true ;;
      3) [ "$INSTALL_CONTEXT7" = true ] && INSTALL_CONTEXT7=false || INSTALL_CONTEXT7=true ;;
      "") break ;;
      *) echo "    Invalid choice" ;;
    esac
  done
fi

# ─────────────────────────────────────────────
# Step 4: QA Configuration
# ─────────────────────────────────────────────

header "4" "QA Configuration"

if [ "$MODE" = "update" ]; then
  # On update: preserve existing QA config from .claude-toolkit.json
  info "Update mode — keeping existing QA configuration"
else
  TEST_CMD=$(detect_test_command "$PROJECT_DIR" "$TECH_STACK")
  LINT_CMD=$(detect_lint_command "$PROJECT_DIR" "$TECH_STACK")
  SCAN_CATS=$(detect_scan_categories "$TECH_STACK")

  if [ "$AUTO_MODE" = false ]; then
    if [ -n "$TEST_CMD" ]; then
      read -r -p "    Test command: $TEST_CMD  [Y/n/edit]: " ans
      case "$ans" in
        n|N) TEST_CMD="" ;;
        "") ;;
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
  else
    [ -n "$TEST_CMD" ] && info "Test command: $TEST_CMD"
    [ -n "$LINT_CMD" ] && info "Lint command: $LINT_CMD"
  fi

  # Default branch detection (git only)
  DEFAULT_BRANCH=""
  if [ "$IS_GIT_REPO" = true ]; then
    DEFAULT_BRANCH=$(git -C "$PROJECT_DIR" symbolic-ref refs/remotes/origin/HEAD 2>/dev/null | sed 's@^refs/remotes/origin/@@' || echo "main")
    if [ "$AUTO_MODE" = false ]; then
      read -r -p "    QA worktree branch: $DEFAULT_BRANCH  [Y/n/edit]: " ans
      case "$ans" in
        n|N) DEFAULT_BRANCH="main" ;;
        "") ;;
        *) DEFAULT_BRANCH="$ans" ;;
      esac
    else
      info "QA worktree branch: $DEFAULT_BRANCH"
    fi
  else
    info "QA worktree: disabled (no git repo — QA will run in-place)"
  fi
fi

# ─────────────────────────────────────────────
# Dry-run summary
# ─────────────────────────────────────────────

if [ "$DRY_RUN" = true ]; then
  header "DRY RUN" "Summary of what would be installed"
  echo ""
  info "Project: $PROJECT_NAME ($PROJECT_TYPE)"
  info "Tech stack: ${TECH_STACK:-none detected}"
  info "Languages: ${DETECTED_LANGUAGES:-none}"
  info "Package manager: ${PKG_MANAGER:-none}"
  info "Test command: ${TEST_CMD:-none}"
  info "Lint command: ${LINT_CMD:-none}"
  echo ""
  info "Components to install:"
  [ "$SKIP_RULES" = false ]  && info "  Rules: common ${DETECTED_LANGUAGES}" || warn "  Rules: SKIPPED"
  [ "$SKIP_SKILLS" = false ] && info "  Skills: 5 progressive disclosure skills" || warn "  Skills: SKIPPED"
  [ "$SKIP_HOOKS" = false ]  && info "  Hooks: PreToolUse, PostToolUse" || warn "  Hooks: SKIPPED"
  [ "$SKIP_AGENTS" = false ] && info "  Agents: 7 generic + domain-specific (auto-detected)" || warn "  Agents: SKIPPED"
  info "  Commands: $(ls "$TOOLKIT_DIR/commands/"*.md 2>/dev/null | wc -l | tr -d ' ') slash commands"
  echo ""
  info "MCP servers:"
  info "  deep-think (required)"
  [ "$INSTALL_PLAYWRIGHT" = true ] && info "  playwright"
  [ "$INSTALL_LEANN" = true ]      && info "  leann-server"
  [ "$INSTALL_CONTEXT7" = true ]   && info "  context7"
  echo ""
  echo -e "${BOLD}No changes made (dry run).${NC}"
  exit 0
fi

# ─────────────────────────────────────────────
# Step 5: Install files
# ─────────────────────────────────────────────

TOOLKIT_CONFIG="$PROJECT_DIR/.claude-toolkit.json"
init_update_tracking "$TOOLKIT_CONFIG"

header "5" "Installing"

# tools/ralph/
mkdir -p "$PROJECT_DIR/tools/ralph"
_tracked_copy "$TEMPLATES/ralph/ralph.sh" "$PROJECT_DIR/tools/ralph/ralph.sh" "tools/ralph/ralph.sh"
_tracked_copy "$TEMPLATES/ralph/RALPH.md" "$PROJECT_DIR/tools/ralph/RALPH.md" "tools/ralph/RALPH.md"
_tracked_copy "$TEMPLATES/ralph/prd.json.example" "$PROJECT_DIR/tools/ralph/prd.json.example" "tools/ralph/prd.json.example"
chmod +x "$PROJECT_DIR/tools/ralph/ralph.sh"
info "tools/ralph/ — 3 files"

# tools/qa/
mkdir -p "$PROJECT_DIR/tools/qa"
_tracked_copy "$TEMPLATES/qa/qa.sh" "$PROJECT_DIR/tools/qa/qa.sh" "tools/qa/qa.sh"
_tracked_copy "$TEMPLATES/qa/QA_PROMPT.md" "$PROJECT_DIR/tools/qa/QA_PROMPT.md" "tools/qa/QA_PROMPT.md"
chmod +x "$PROJECT_DIR/tools/qa/qa.sh"
info "tools/qa/ — 2 files"

# Rules
if [ "$SKIP_RULES" = false ] && [ -n "$DETECTED_LANGUAGES" ]; then
  install_rules "$PROJECT_DIR" "$DETECTED_LANGUAGES" "$TEMPLATES"
  info ".claude/rules/ — common + $DETECTED_LANGUAGES"
elif [ "$SKIP_RULES" = false ]; then
  # No languages detected, install common only
  install_rules "$PROJECT_DIR" "" "$TEMPLATES"
  info ".claude/rules/ — common only (no languages detected)"
else
  warn "Rules: skipped"
fi

# Skills
if [ "$SKIP_SKILLS" = false ]; then
  install_skills "$PROJECT_DIR" "$TEMPLATES"
  SKILL_COUNT=$(ls -d "$PROJECT_DIR/.claude/skills/"*/ 2>/dev/null | wc -l | tr -d ' ')
  info ".claude/skills/ — $SKILL_COUNT skills"
else
  warn "Skills: skipped"
fi

# Hooks
if [ "$SKIP_HOOKS" = false ]; then
  install_hooks "$PROJECT_DIR" "$TEMPLATES"
  info ".claude/hooks/ — hook templates installed"
else
  warn "Hooks: skipped"
fi

# Agents (generic + domain-specific)
if [ "$SKIP_AGENTS" = false ]; then
  AGENT_DOMAINS=$(map_stack_to_agent_domains "$TECH_STACK")
  DEEP_DOMAINS=$(detect_deep_domains "$PROJECT_DIR")
  # Merge and deduplicate domains
  ALL_DOMAINS=""
  for d in $AGENT_DOMAINS $DEEP_DOMAINS; do
    [[ " $ALL_DOMAINS " == *" $d "* ]] || ALL_DOMAINS="$ALL_DOMAINS $d"
  done
  ALL_DOMAINS="${ALL_DOMAINS# }"

  install_agents "$PROJECT_DIR" "$TEMPLATES" "$ALL_DOMAINS"
  GENERIC_COUNT=$(ls "$TEMPLATES/agents/"*.md 2>/dev/null | wc -l | tr -d ' ')
  TOTAL_AGENT_COUNT=$(ls "$PROJECT_DIR/.claude/agents/"*.md 2>/dev/null | wc -l | tr -d ' ')
  DOMAIN_AGENT_COUNT=$((TOTAL_AGENT_COUNT - GENERIC_COUNT))
  if [ "$DOMAIN_AGENT_COUNT" -gt 0 ]; then
    info ".claude/agents/ — $GENERIC_COUNT generic + $DOMAIN_AGENT_COUNT domain-specific = $TOTAL_AGENT_COUNT total"
    info "  Domains: $ALL_DOMAINS"
  else
    info ".claude/agents/ — $TOTAL_AGENT_COUNT agent definitions"
  fi
else
  warn "Agents: skipped"
fi

# .mcp.json — merge servers
install_mcp_config "$PROJECT_DIR" "deep-think"

if [ "$INSTALL_PLAYWRIGHT" = true ]; then
  install_mcp_config "$PROJECT_DIR" "playwright"
fi

if [ "$INSTALL_LEANN" = true ]; then
  install_mcp_config "$PROJECT_DIR" "leann-server"
fi

if [ "$INSTALL_CONTEXT7" = true ]; then
  install_mcp_config "$PROJECT_DIR" "context7"
fi

INSTALLED_MCPS=$(jq -r '.mcpServers | keys | join(", ")' "$PROJECT_DIR/.mcp.json" 2>/dev/null)
info ".mcp.json — servers: $INSTALLED_MCPS"

# .deep-think.json
_tracked_copy "$TEMPLATES/deep-think.json" "$PROJECT_DIR/.deep-think.json" ".deep-think.json"
STRATEGY_COUNT=$(jq '.strategies | length' "$TEMPLATES/deep-think.json")
info ".deep-think.json — $STRATEGY_COUNT strategies"

# .claude-toolkit.json
if [ "$MODE" = "update" ] && [ -f "$TOOLKIT_CONFIG" ]; then
  # Update mode: bump version and re-detected fields, preserve user config
  STACK_JSON=$(to_json_array "$TECH_STACK")
  LANG_JSON=$(to_json_array "$DETECTED_LANGUAGES")
  update_toolkit_config "$TOOLKIT_CONFIG" "$TOOLKIT_VERSION" "$STACK_JSON" "$LANG_JSON" "$PKG_MANAGER"
  info ".claude-toolkit.json — version and stack updated (user config preserved)"
elif [ ! -f "$TOOLKIT_CONFIG" ] || [ "$FORCE" = true ]; then
  SCAN_JSON=$(to_json_array "$SCAN_CATS")
  STACK_JSON=$(to_json_array "$TECH_STACK")
  LANG_JSON=$(to_json_array "$DETECTED_LANGUAGES")

  # Build installed MCPs list
  MCPS_JSON='["deep-think"'
  [ "$INSTALL_PLAYWRIGHT" = true ] && MCPS_JSON+=',"playwright"'
  [ "$INSTALL_LEANN" = true ] && MCPS_JSON+=',"leann-server"'
  [ "$INSTALL_CONTEXT7" = true ] && MCPS_JSON+=',"context7"'
  MCPS_JSON+=']'

  cat > "$TOOLKIT_CONFIG" <<EOFCONFIG
{
  "version": "$TOOLKIT_VERSION",
  "project": {
    "name": "$PROJECT_NAME",
    "type": "$PROJECT_TYPE",
    "techStack": $STACK_JSON,
    "languages": $LANG_JSON,
    "packageManager": $([ -n "$PKG_MANAGER" ] && echo "\"$PKG_MANAGER\"" || echo "null")
  },
  "commands": {
    "test": $([ -n "$TEST_CMD" ] && echo "\"$TEST_CMD\"" || echo "null"),
    "lint": $([ -n "$LINT_CMD" ] && echo "\"$LINT_CMD\"" || echo "null")
  },
  "qa": {
    "scanCategories": $SCAN_JSON,
    "maxFixLines": 30,
    "worktreeFromBranch": $([ -n "$DEFAULT_BRANCH" ] && echo "\"$DEFAULT_BRANCH\"" || echo "null")
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
  jq '.' "$TOOLKIT_CONFIG" > "$TOOLKIT_CONFIG.tmp" && mv "$TOOLKIT_CONFIG.tmp" "$TOOLKIT_CONFIG"
  info ".claude-toolkit.json — project config created"
else
  warn ".claude-toolkit.json already exists (use --force to overwrite)"
fi

# Write managed files list + update summary
write_managed_files "$TOOLKIT_CONFIG"
if [ "$MODE" = "update" ]; then
  detect_deprecated_files "$PROJECT_DIR"
  print_update_summary "$PROJECT_DIR"
fi

# .gitignore — append entries (git repos only)
if [ "$IS_GIT_REPO" = true ] && [ -f "$TEMPLATES/gitignore-entries.txt" ]; then
  append_gitignore "$PROJECT_DIR/.gitignore" "$TEMPLATES/gitignore-entries.txt"
  info ".gitignore — runtime entries added"
elif [ "$IS_GIT_REPO" = false ]; then
  info ".gitignore — skipped (not a git repo)"
fi

# ─────────────────────────────────────────────
# Step 6: Global commands
# ─────────────────────────────────────────────

header "6" "Global Claude Code commands"
install_commands "$TOOLKIT_DIR"

# ─────────────────────────────────────────────
# Done
# ─────────────────────────────────────────────

echo ""
echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
if [ "$MODE" = "update" ]; then
  echo -e "${GREEN}${BOLD}Done!${NC} Toolkit v$TOOLKIT_VERSION updated."
else
  echo -e "${GREEN}${BOLD}Done!${NC} Toolkit v$TOOLKIT_VERSION installed."
fi
echo ""
echo "  Installed:"
[ "$SKIP_RULES" = false ]  && echo "    Rules:    common + ${DETECTED_LANGUAGES:-none}"
[ "$SKIP_SKILLS" = false ] && echo "    Skills:   ${SKILL_COUNT:-0} skills"
[ "$SKIP_HOOKS" = false ]  && echo "    Hooks:    PreToolUse, PostToolUse"
[ "$SKIP_AGENTS" = false ] && echo "    Agents:   ${TOTAL_AGENT_COUNT:-0} (${GENERIC_COUNT:-0} generic + ${DOMAIN_AGENT_COUNT:-0} domain)"
echo "    Commands: $(ls "$TOOLKIT_DIR/commands/"*.md 2>/dev/null | wc -l | tr -d ' ') slash commands"
echo "    MCP:      $INSTALLED_MCPS"
echo ""
echo "  Next steps:"
echo "    1. Start Claude Code in this project"
echo "    2. Run /ralph --issues 1,2 to build features from GitHub issues"
echo "    3. Run /qa to scan and fix quality issues"
echo "    4. Run /verify to check test + lint status"
echo ""
echo "  Update toolkit:  ~/.claude/toolkit/install.sh --update"
echo "  Toolkit repo:    $TOOLKIT_DIR"
echo ""
