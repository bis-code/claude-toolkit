#!/bin/bash
set -e

TOOLKIT_VERSION="4.0.0"
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
READ_ONLY=false
WORKSPACE_MODE=false
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
    --read-only)   READ_ONLY=true; shift ;;
    --workspace)   WORKSPACE_MODE=true; shift ;;
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
      echo "  --read-only        Install read-only rule (Claude won't modify files unless asked)"
      echo "  --workspace        Generate .claude-workspace.json from auto-detection"
      echo "  --dry-run          Show what would be installed without doing it"
      echo "  --project-dir DIR  Target directory (default: current or git root)"
      echo "  -h, --help         Show this help"
      exit 0
      ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

# Export for use in lib/ modules
export FORCE MODE READ_ONLY

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
# Step 1b: Install/update Go binaries (MCP server + TUI dashboard)
# ─────────────────────────────────────────────

TOOLKIT_BIN_DIR="$HOME/.claude/toolkit/bin"
SERVER_BIN="$TOOLKIT_BIN_DIR/claude-toolkit-server"
TUI_BIN="$TOOLKIT_BIN_DIR/claude-toolkit-tui"
SERVER_REPO="bis-code/claude-toolkit"

install_go_binaries() {
  mkdir -p "$TOOLKIT_BIN_DIR"

  # Detect platform
  local os_name arch_name
  os_name="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch_name="$(uname -m)"

  case "$arch_name" in
    x86_64)  arch_name="amd64" ;;
    aarch64) arch_name="arm64" ;;
    arm64)   arch_name="arm64" ;;
    *)       arch_name="$arch_name" ;;
  esac

  local server_binary_name="claude-toolkit-server-${os_name}-${arch_name}"
  local tui_binary_name="claude-toolkit-tui-${os_name}-${arch_name}"

  # Try downloading from GitHub releases
  local downloaded_server=false
  local downloaded_tui=false
  if command -v gh &>/dev/null; then
    local latest_tag
    latest_tag=$(gh release list --repo "$SERVER_REPO" --limit 1 --json tagName -q '.[0].tagName' 2>/dev/null || echo "")
    if [ -n "$latest_tag" ]; then
      info "Downloading binaries ($os_name/$arch_name) from release $latest_tag..."
      if gh release download "$latest_tag" --repo "$SERVER_REPO" --pattern "$server_binary_name" --dir "$TOOLKIT_BIN_DIR" --clobber 2>/dev/null; then
        mv "$TOOLKIT_BIN_DIR/$server_binary_name" "$SERVER_BIN"
        chmod +x "$SERVER_BIN"
        downloaded_server=true
      fi
      if gh release download "$latest_tag" --repo "$SERVER_REPO" --pattern "$tui_binary_name" --dir "$TOOLKIT_BIN_DIR" --clobber 2>/dev/null; then
        mv "$TOOLKIT_BIN_DIR/$tui_binary_name" "$TUI_BIN"
        chmod +x "$TUI_BIN"
        downloaded_tui=true
      fi
      if $downloaded_server; then
        info "Server binary installed from release"
      fi
      if $downloaded_tui; then
        info "TUI binary installed from release"
      fi
      if $downloaded_server && $downloaded_tui; then
        return 0
      fi
      [ "$downloaded_server" = false ] && warn "Server binary not in release, trying source build..."
    fi
  fi

  # Fallback: build from source if Go is available
  if command -v go &>/dev/null; then
    local server_src="$TOOLKIT_DIR/server"
    if [ -d "$server_src" ]; then
      if [ "$downloaded_server" = false ]; then
        info "Building server from source..."
        if (cd "$server_src" && go build -o "$SERVER_BIN" ./cmd/server/) 2>&1; then
          info "Server binary built"
        else
          error "Failed to build server binary"
        fi
      fi
      if [ "$downloaded_tui" = false ]; then
        info "Building TUI from source..."
        if (cd "$server_src" && go build -o "$TUI_BIN" ./cmd/tui/) 2>&1; then
          info "TUI binary built"
        else
          warn "Failed to build TUI binary (optional)"
        fi
      fi
      return 0
    else
      warn "Server source not found at $server_src"
      return 1
    fi
  fi

  warn "Cannot install binaries (no release found and Go not available)"
  warn "Install Go (https://go.dev) or wait for a release with pre-built binaries"
  return 1
}

verify_server_health() {
  if [ ! -x "$SERVER_BIN" ]; then
    warn "Server binary not found, skipping health check"
    return 1
  fi

  info "Verifying server health..."
  local health_response
  health_response=$(echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"health-check","version":"1.0.0"}}}' | timeout 5 "$SERVER_BIN" 2>/dev/null | head -1 || echo "")

  if echo "$health_response" | grep -q '"result"' 2>/dev/null; then
    info "Server health check passed"
    return 0
  else
    warn "Server health check inconclusive (server may still work)"
    return 0
  fi
}

setup_path() {
  # Create symlinks in /usr/local/bin if writable, otherwise add to shell profile
  local link_dir="/usr/local/bin"

  if [ -w "$link_dir" ]; then
    # Symlink approach (preferred — no PATH changes needed)
    if [ -x "$SERVER_BIN" ]; then
      ln -sf "$SERVER_BIN" "$link_dir/claude-toolkit-server"
    fi
    if [ -x "$TUI_BIN" ]; then
      ln -sf "$TUI_BIN" "$link_dir/claude-toolkit-tui"
    fi
    info "Commands available: claude-toolkit-server, claude-toolkit-tui"
    return 0
  fi

  # Fallback: add bin dir to PATH via shell profile
  local path_line="export PATH=\"\$HOME/.claude/toolkit/bin:\$PATH\""
  local profile=""

  if [ -f "$HOME/.zshrc" ]; then
    profile="$HOME/.zshrc"
  elif [ -f "$HOME/.bashrc" ]; then
    profile="$HOME/.bashrc"
  elif [ -f "$HOME/.bash_profile" ]; then
    profile="$HOME/.bash_profile"
  fi

  if [ -n "$profile" ]; then
    if ! grep -q '.claude/toolkit/bin' "$profile" 2>/dev/null; then
      echo "" >> "$profile"
      echo "# Claude Toolkit" >> "$profile"
      echo "$path_line" >> "$profile"
      info "Added ~/.claude/toolkit/bin to PATH in $(basename "$profile")"
      info "Run 'source $profile' or open a new terminal to use: claude-toolkit-tui"
    else
      info "PATH already configured in $(basename "$profile")"
    fi
  else
    warn "Could not find shell profile — add manually: $path_line"
  fi
}

header "1b" "Go Binaries"
if [ -x "$SERVER_BIN" ] && [ -x "$TUI_BIN" ] && [ "$MODE" != "update" ]; then
  info "Server binary: $SERVER_BIN"
  info "TUI binary: $TUI_BIN"
else
  install_go_binaries || true  # Binaries are optional — continue without them
fi
verify_server_health || true
setup_path || true

# ─────────────────────────────────────────────
# Uninstall: remove all toolkit files and exit
# ─────────────────────────────────────────────

if [ "$MODE" = "uninstall" ]; then
  if [ -n "$PROJECT_DIR" ]; then
    PROJECT_DIR="$(cd "$PROJECT_DIR" && pwd)"
  elif git rev-parse --show-toplevel &>/dev/null; then
    PROJECT_DIR="$(git rev-parse --show-toplevel)"
  else
    PROJECT_DIR="$(pwd)"
  fi

  header "2" "Uninstalling from: $PROJECT_DIR"

  TOOLKIT_CONFIG="$PROJECT_DIR/.claude-toolkit.json"

  if [ -f "$TOOLKIT_CONFIG" ]; then
    # Remove managed files listed in config
    REMOVED=0
    while IFS= read -r rel_path; do
      [ -z "$rel_path" ] && continue
      if [ -f "$PROJECT_DIR/$rel_path" ]; then
        rm "$PROJECT_DIR/$rel_path"
        REMOVED=$((REMOVED + 1))
      fi
    done < <(jq -r '.managedFiles // [] | .[]' "$TOOLKIT_CONFIG" 2>/dev/null)
    info "Removed $REMOVED managed files"
  fi

  # Remove toolkit-generated config files
  for f in .claude-toolkit.json .deep-think.json; do
    if [ -f "$PROJECT_DIR/$f" ]; then
      rm "$PROJECT_DIR/$f"
      info "Removed $f"
    fi
  done

  # Remove project .mcp.json only if it contains exclusively toolkit-managed servers
  if [ -f "$PROJECT_DIR/.mcp.json" ]; then
    rm "$PROJECT_DIR/.mcp.json"
    info "Removed .mcp.json"
  fi

  # Remove toolkit servers from user-scope ~/.claude.json
  USER_MCP="$HOME/.claude.json"
  if [ -f "$USER_MCP" ]; then
    for srv in claude-toolkit deep-think leann-server context7; do
      if jq -e ".mcpServers.\"$srv\"" "$USER_MCP" &>/dev/null; then
        jq "del(.mcpServers.\"$srv\")" "$USER_MCP" > "$USER_MCP.tmp" && mv "$USER_MCP.tmp" "$USER_MCP"
      fi
    done
    REMAINING=$(jq -r '.mcpServers | keys | join(", ")' "$USER_MCP" 2>/dev/null)
    info "~/.claude.json — removed toolkit servers (remaining: $REMAINING)"
  fi

  # Remove hooks.json reference file
  if [ -f "$PROJECT_DIR/.claude/hooks/hooks.json" ]; then
    rm "$PROJECT_DIR/.claude/hooks/hooks.json"
    info "Removed .claude/hooks/hooks.json"
  fi

  # Remove toolkit hooks from .claude/settings.json
  local settings_file="$PROJECT_DIR/.claude/settings.json"
  if [ -f "$settings_file" ] && jq -e '.hooks' "$settings_file" &>/dev/null; then
    jq '
      .hooks |= with_entries(
        .value |= map(select(._toolkit != true))
      ) |
      .hooks |= with_entries(select(.value | length > 0)) |
      if (.hooks | length) == 0 then del(.hooks) else . end
    ' "$settings_file" > "${settings_file}.tmp" && mv "${settings_file}.tmp" "$settings_file"
    info "Removed toolkit hooks from settings.json"
  fi

  # Remove tools/ralph/ if it only contains toolkit files
  if [ -d "$PROJECT_DIR/tools/ralph" ]; then
    rm -f "$PROJECT_DIR/tools/ralph/prd.json.example"
    rmdir "$PROJECT_DIR/tools/ralph" 2>/dev/null && rmdir "$PROJECT_DIR/tools" 2>/dev/null || true
    info "Removed tools/ralph/"
  fi

  # Clean up empty directories under .claude/
  if [ -d "$PROJECT_DIR/.claude" ]; then
    find "$PROJECT_DIR/.claude" -type d -empty -delete 2>/dev/null
    # Remove .claude/ itself if empty
    rmdir "$PROJECT_DIR/.claude" 2>/dev/null || true
    if [ -d "$PROJECT_DIR/.claude" ]; then
      warn ".claude/ still has user files (preserved)"
    else
      info "Removed .claude/"
    fi
  fi

  # Remove binaries and symlinks
  local link_dir="/usr/local/bin"
  for bin_name in claude-toolkit-server claude-toolkit-tui; do
    rm -f "$HOME/.claude/toolkit/bin/$bin_name"
    # Remove symlink if it points to our binary
    if [ -L "$link_dir/$bin_name" ]; then
      local target
      target=$(readlink "$link_dir/$bin_name" 2>/dev/null || echo "")
      if echo "$target" | grep -q '.claude/toolkit/bin' 2>/dev/null; then
        rm -f "$link_dir/$bin_name" 2>/dev/null || true
      fi
    fi
  done
  rmdir "$HOME/.claude/toolkit/bin" 2>/dev/null || true
  info "Removed toolkit binaries"

  echo ""
  echo -e "${GREEN}${BOLD}Done!${NC} Toolkit uninstalled from $(basename "$PROJECT_DIR")."
  echo ""
  exit 0
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

echo ""
echo -e "  ${BOLD}Scanning project...${NC}"

TECH_STACK=$(detect_tech_stack "$PROJECT_DIR")
PKG_MANAGER=$(detect_package_manager "$PROJECT_DIR")

# Map stack to language rules
if [ -n "$LANGUAGES" ]; then
  DETECTED_LANGUAGES="${LANGUAGES//,/ }"
else
  DETECTED_LANGUAGES=$(map_stack_to_languages "$TECH_STACK")
fi

# Display detected stack as a tree
echo ""
if [ "$IS_GIT_REPO" = true ]; then
  echo -e "    ${GREEN}${BOLD}$PROJECT_NAME${NC} (git repository)"
else
  echo -e "    ${GREEN}${BOLD}$PROJECT_NAME${NC} (workspace)"
fi

# Show detection tree
_show_detection() {
  local marker="$1" desc="$2" prefix="$3"
  echo -e "    ${prefix}${BLUE}${marker}${NC} ${desc}"
}

STACK_ITEMS=($TECH_STACK)
TOTAL_STACK_ITEMS=${#STACK_ITEMS[@]}
STACK_IDX=0
for item in $TECH_STACK; do
  STACK_IDX=$((STACK_IDX + 1))
  TREE_PREFIX="├── "
  [ $STACK_IDX -eq $TOTAL_STACK_ITEMS ] && [ -z "$PKG_MANAGER" ] && TREE_PREFIX="└── "
  case "$item" in
    go)    _show_detection "go.mod" "Go rules + agents" "$TREE_PREFIX" ;;
    node)  _show_detection "package.json" "Node.js rules" "$TREE_PREFIX" ;;
    react) _show_detection "React" "TypeScript rules + frontend agents" "$TREE_PREFIX" ;;
    vue)   _show_detection "Vue" "TypeScript rules + frontend agents" "$TREE_PREFIX" ;;
    svelte) _show_detection "Svelte" "TypeScript rules + frontend agents" "$TREE_PREFIX" ;;
    angular) _show_detection "Angular" "TypeScript rules + frontend agents" "$TREE_PREFIX" ;;
    rust)  _show_detection "Cargo.toml" "Rust rules + agents" "$TREE_PREFIX" ;;
    python) _show_detection "Python" "Python rules + agents" "$TREE_PREFIX" ;;
    csharp) _show_detection ".csproj" "C# rules + agents" "$TREE_PREFIX" ;;
    java)  _show_detection "pom.xml" "Java rules + agents" "$TREE_PREFIX" ;;
    docker) _show_detection "Dockerfile" "Docker rules" "$TREE_PREFIX" ;;
    unity) _show_detection "Unity" "C# + Unity agents" "$TREE_PREFIX" ;;
    solidity) _show_detection "Solidity" "Smart contract rules + agents" "$TREE_PREFIX" ;;
    make)  _show_detection "Makefile" "Build automation" "$TREE_PREFIX" ;;
    *)     _show_detection "$item" "Detected" "$TREE_PREFIX" ;;
  esac
done

if [ -n "$PKG_MANAGER" ]; then
  echo -e "    └── ${BLUE}$PKG_MANAGER${NC} Package manager"
fi

if [ -z "$TECH_STACK" ]; then
  echo -e "    └── ${YELLOW}No tech stack detected${NC} (will use common rules only)"
fi

echo ""

if [ -f "$PROJECT_DIR/.mcp.json" ]; then
  EXISTING_MCPS=$(jq -r '.mcpServers | keys | join(", ")' "$PROJECT_DIR/.mcp.json" 2>/dev/null || echo "none")
fi

if [ -f "$PROJECT_DIR/.claude-toolkit.json" ]; then
  if [ "$MODE" = "install" ] && [ "$FORCE" = false ]; then
    warn ".claude-toolkit.json already exists (use --update to refresh)"
  fi
  # Restore readOnly flag from existing config on update
  if [ "$MODE" = "update" ] && [ "$READ_ONLY" = false ]; then
    if [ "$(jq -r '.readOnly // false' "$PROJECT_DIR/.claude-toolkit.json")" = "true" ]; then
      READ_ONLY=true
    fi
  fi
fi

# ─────────────────────────────────────────────
# Step 3: MCP Server Configuration
# ─────────────────────────────────────────────

header "3" "MCP Server Configuration"

# Auto-detect and install MCPs with prerequisite checking
INSTALL_PLAYWRIGHT=false
INSTALL_LEANN=false
INSTALL_CONTEXT7=false
INSTALLED_MCP_LIST=""

# Check prerequisites before deciding what to install
_check_mcp_prereq() {
  local server="$1"
  case "$server" in
    deep-think)
      if command -v mcp-deep-think &>/dev/null; then
        return 0
      elif command -v npm &>/dev/null; then
        npm install -g mcp-deep-think 2>/dev/null && return 0
      fi
      return 1
      ;;
    playwright)
      command -v npx &>/dev/null && return 0
      return 1
      ;;
    leann-server)
      command -v leann_mcp &>/dev/null && return 0
      return 1
      ;;
    context7)
      command -v npx &>/dev/null && return 0
      return 1
      ;;
  esac
  return 1
}

# deep-think is mandatory
if _check_mcp_prereq "deep-think"; then
  echo -e "    ${GREEN}✓${NC} deep-think — Structured reasoning"
  INSTALLED_MCP_LIST="deep-think"
else
  echo -e "    ${YELLOW}⚠${NC} deep-think — Skipped (npm not available)"
fi

# Auto-detect optional MCPs based on tech stack + available tools
if [[ "$TECH_STACK" == *react* ]] || [[ "$TECH_STACK" == *vue* ]] || [[ "$TECH_STACK" == *svelte* ]] || [[ "$TECH_STACK" == *angular* ]] || [[ "$TECH_STACK" == *node* ]]; then
  if _check_mcp_prereq "playwright"; then
    INSTALL_PLAYWRIGHT=true
    echo -e "    ${GREEN}✓${NC} playwright — Browser testing (frontend detected)"
    INSTALLED_MCP_LIST="$INSTALLED_MCP_LIST, playwright"
  else
    echo -e "    ${YELLOW}⚠${NC} playwright — Skipped (npx not available)"
  fi
fi

if _check_mcp_prereq "leann-server"; then
  INSTALL_LEANN=true
  echo -e "    ${GREEN}✓${NC} leann-server — Semantic code search"
  INSTALLED_MCP_LIST="$INSTALLED_MCP_LIST, leann-server"
fi

if _check_mcp_prereq "context7"; then
  INSTALL_CONTEXT7=true
  echo -e "    ${GREEN}✓${NC} context7 — Library documentation lookup"
  INSTALLED_MCP_LIST="$INSTALLED_MCP_LIST, context7"
fi

# On update, keep existing selection
if [ "$MODE" = "update" ]; then
  echo -e "    ${BLUE}→${NC} Update mode — preserving existing MCP configuration"
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
        ""|y|Y) ;;
        edit|Edit|EDIT)
          read -r -p "    Enter test command: " TEST_CMD ;;
        *) TEST_CMD="$ans" ;;
      esac
    fi

    if [ -n "$LINT_CMD" ]; then
      read -r -p "    Lint command: $LINT_CMD  [Y/n/edit]: " ans
      case "$ans" in
        n|N) LINT_CMD="" ;;
        ""|y|Y) ;;
        edit|Edit|EDIT)
          read -r -p "    Enter lint command: " LINT_CMD ;;
        *) LINT_CMD="$ans" ;;
      esac
    fi
  else
    [ -n "$TEST_CMD" ] && info "Test command: $TEST_CMD"
    [ -n "$LINT_CMD" ] && info "Lint command: $LINT_CMD"
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
  [ "$READ_ONLY" = true ] && info "Read-only mode: enabled"
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
_tracked_copy "$TEMPLATES/ralph/prd.json.example" "$PROJECT_DIR/tools/ralph/prd.json.example" "tools/ralph/prd.json.example"
# Clean up deprecated files (ralph.sh and RALPH.md replaced by /ralph skill)
rm -f "$PROJECT_DIR/tools/ralph/ralph.sh" "$PROJECT_DIR/tools/ralph/RALPH.md"
info "tools/ralph/ — 1 file (prd.json.example)"

# Clean up deprecated tools/qa/ (replaced by /qa skill)
if [ -d "$PROJECT_DIR/tools/qa" ]; then
  rm -rf "$PROJECT_DIR/tools/qa"
  info "tools/qa/ — removed (replaced by /qa skill)"
fi

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

# Read-only rule (conditional — only when --read-only flag or config says so)
if [ "$READ_ONLY" = true ] && [ "$SKIP_RULES" = false ]; then
  _tracked_copy "$TEMPLATES/rules/conditional/read-only.md" \
    "$PROJECT_DIR/.claude/rules/common/read-only.md" \
    ".claude/rules/common/read-only.md"
  info ".claude/rules/common/read-only.md — read-only mode enabled"
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

# ── User-scope MCP servers (~/.claude.json) ──
# Global servers that are the same everywhere go to user scope.
# Claude Code reads user MCPs from ~/.claude.json under .mcpServers key.
USER_MCP="$HOME/.claude.json"

if [ -x "$SERVER_BIN" ]; then
  merge_mcp_server "$USER_MCP" "claude-toolkit" "{\"command\":\"$SERVER_BIN\",\"args\":[]}"
fi
merge_mcp_server "$USER_MCP" "deep-think" '{"command":"mcp-deep-think","args":[]}'

if [ "$INSTALL_LEANN" = true ]; then
  merge_mcp_server "$USER_MCP" "leann-server" '{"command":"leann_mcp","args":[]}'
fi

if [ "$INSTALL_CONTEXT7" = true ]; then
  merge_mcp_server "$USER_MCP" "context7" '{"command":"npx","args":["-y","@upstash/context7-mcp@latest"]}'
fi

USER_MCPS=$(jq -r '.mcpServers | keys | join(", ")' "$USER_MCP" 2>/dev/null)
info "~/.claude.json (user scope) — $USER_MCPS"

# ── Project-scope MCP servers (.mcp.json) ──
# Only project-specific servers go here (e.g., playwright for UI projects).
if [ "$INSTALL_PLAYWRIGHT" = true ]; then
  install_mcp_config "$PROJECT_DIR" "playwright"
fi

# Report project-scope MCPs (may be empty if no project-specific servers)
if [ -f "$PROJECT_DIR/.mcp.json" ]; then
  PROJECT_MCPS=$(jq -r '.mcpServers | keys | join(", ")' "$PROJECT_DIR/.mcp.json" 2>/dev/null)
  [ -n "$PROJECT_MCPS" ] && info ".mcp.json (project scope) — $PROJECT_MCPS"
fi

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
  # Persist readOnly flag
  jq --argjson ro "$READ_ONLY" '.readOnly = $ro' "$TOOLKIT_CONFIG" > "$TOOLKIT_CONFIG.tmp" && mv "$TOOLKIT_CONFIG.tmp" "$TOOLKIT_CONFIG"
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
  "readOnly": $READ_ONLY,
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
    "maxFixLines": 30
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
# Step 5b: Workspace mode — generate .claude-workspace.json
# ─────────────────────────────────────────────

if [ "$WORKSPACE_MODE" = true ]; then
  header "5b" "Workspace configuration"

  WORKSPACE_CONFIG="$PROJECT_DIR/.claude-workspace.json"

  if [ -f "$WORKSPACE_CONFIG" ] && [ "$FORCE" = false ]; then
    warn ".claude-workspace.json already exists (use --force to regenerate)"
  else
    # Auto-detect repos in the directory
    WORKSPACE_NAME="$(basename "$PROJECT_DIR")"
    REPOS_JSON="["
    SHARED_JSON="["
    FIRST_REPO=true
    FIRST_SHARED=true

    for dir in "$PROJECT_DIR"/*/; do
      [ ! -d "$dir" ] && continue
      dir_name="$(basename "$dir")"

      if [ -d "$dir/.git" ]; then
        # It's a git repo — detect tech stack
        repo_type=""
        [ -f "$dir/go.mod" ] && repo_type="go"
        [ -f "$dir/Cargo.toml" ] && repo_type="rust"
        [ -f "$dir/pyproject.toml" ] || [ -f "$dir/requirements.txt" ] && repo_type="python"
        [ -f "$dir/tsconfig.json" ] && repo_type="typescript"
        [ -f "$dir/package.json" ] && [ -z "$repo_type" ] && repo_type="typescript"

        # Detect branch
        repo_branch="main"
        if [ -f "$dir/.git/HEAD" ]; then
          head_content=$(cat "$dir/.git/HEAD")
          case "$head_content" in
            ref:*) repo_branch="${head_content#ref: refs/heads/}" ;;
          esac
        fi

        [ "$FIRST_REPO" = false ] && REPOS_JSON+=","
        REPOS_JSON+="{\"path\":\"$dir_name\",\"branch\":\"$repo_branch\""
        [ -n "$repo_type" ] && REPOS_JSON+=",\"type\":\"$repo_type\""
        REPOS_JSON+="}"
        FIRST_REPO=false
        info "Repo: $dir_name ($repo_type, branch: $repo_branch)"
      else
        # Not a git repo — shared directory
        [ "$FIRST_SHARED" = false ] && SHARED_JSON+=","
        SHARED_JSON+="\"$dir_name/\""
        FIRST_SHARED=false
      fi
    done

    REPOS_JSON+="]"
    SHARED_JSON+="]"

    cat > "$WORKSPACE_CONFIG" <<EOFWS
{
  "name": "$WORKSPACE_NAME",
  "repos": $REPOS_JSON,
  "shared": $SHARED_JSON,
  "planning_repo": "",
  "cross_repo_rules": [],
  "dependency_order": [],
  "domain_labels": []
}
EOFWS
    jq '.' "$WORKSPACE_CONFIG" > "$WORKSPACE_CONFIG.tmp" && mv "$WORKSPACE_CONFIG.tmp" "$WORKSPACE_CONFIG"
    info ".claude-workspace.json generated"
    info "Edit it to set planning_repo, dependency_order, and domain_labels"
  fi
fi

# Load seed rules into server on first install
if [ -x "$SERVER_BIN" ] && [ -d "$TOOLKIT_DIR/templates/rules/seed" ]; then
  info "Seed rules available at: $TOOLKIT_DIR/templates/rules/seed/"
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
echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
if [ "$MODE" = "update" ]; then
  echo -e "  ${GREEN}${BOLD}Claude Toolkit v$TOOLKIT_VERSION updated!${NC}"
else
  echo -e "  ${GREEN}${BOLD}Claude Toolkit v$TOOLKIT_VERSION installed!${NC}"
fi
echo ""

# Count installed components
RULE_COUNT=$(find "$PROJECT_DIR/.claude/rules" -name "*.md" 2>/dev/null | wc -l | tr -d ' ')
HOOK_COUNT=$(find "$PROJECT_DIR/.claude/hooks/scripts" -name "*.js" -not -path "*/lib/*" 2>/dev/null | wc -l | tr -d ' ')
CMD_COUNT=$(ls "$TOOLKIT_DIR/commands/"*.md 2>/dev/null | wc -l | tr -d ' ')

[ "$SKIP_RULES" = false ]  && echo -e "    ${GREEN}✓${NC} ${RULE_COUNT} rules (common + ${DETECTED_LANGUAGES:-none})"
[ "$SKIP_SKILLS" = false ] && echo -e "    ${GREEN}✓${NC} ${SKILL_COUNT:-0} skills (/ralph, /qa, /plan, /code-review, ...)"
[ "$SKIP_AGENTS" = false ] && echo -e "    ${GREEN}✓${NC} ${TOTAL_AGENT_COUNT:-0} agents (${GENERIC_COUNT:-0} generic + ${DOMAIN_AGENT_COUNT:-0} domain)"
[ "$SKIP_HOOKS" = false ]  && echo -e "    ${GREEN}✓${NC} ${HOOK_COUNT} hooks (auto-telemetry, quality gate, secret detection)"
echo -e "    ${GREEN}✓${NC} ${CMD_COUNT} slash commands"
echo -e "    ${GREEN}✓${NC} MCP: ${INSTALLED_MCP_LIST:-none}"
[ -x "$SERVER_BIN" ] && echo -e "    ${GREEN}✓${NC} MCP server → Dashboard at ${BLUE}localhost:19280${NC}"
[ -x "$TUI_BIN" ]    && echo -e "    ${GREEN}✓${NC} TUI dashboard → run ${BLUE}claude-toolkit-tui${NC}"
[ "$READ_ONLY" = true ] && echo -e "    ${YELLOW}!${NC} Read-only mode"
echo ""

echo -e "  ${BOLD}Your sessions will automatically:${NC}"
echo -e "    ${BLUE}•${NC} Track tool usage and patterns (invisible)"
echo -e "    ${BLUE}•${NC} Detect when you're stuck (patrol)"
echo -e "    ${BLUE}•${NC} Score skill effectiveness (auto-eval)"
echo -e "    ${BLUE}•${NC} Learn your coding preferences over time"
echo ""

echo -e "  ${BOLD}Quick start:${NC}"
echo "    /ralph --issues 1,2    Build features from GitHub issues"
echo "    /qa                    Scan and fix quality issues"
echo "    /plan                  Plan implementation approach"
echo "    /code-review           Review recent changes"
echo ""
echo -e "  ${BOLD}Analysis & review:${NC}"
echo "    /deep-dive             Deep codebase exploration & analysis"
echo "    /security-review       OWASP-focused security scan"
echo "    /architect-review      Module boundaries & coupling analysis"
echo "    /performance-review    N+1 queries, blocking ops, resource leaks"
echo "    /incident-debug        Structured hypothesis-driven debugging"
echo ""
echo -e "  ${BOLD}Implementation:${NC}"
echo "    /tdd-workflow          Red-green-refactor with verification"
echo "    /refactor-clean        Dead code removal & consolidation"
echo "    /build-fix             Diagnose and fix build errors"
echo "    /docs                  Update docs after code changes"
echo ""
echo -e "  ${BOLD}Creators:${NC}"
echo "    /skill-creator         Build a new skill (SOP with human gates)"
echo "    /agent-creator         Build a new agent (behavioral spec)"
echo "    /rule-creator          Build a new rule (constraint doc)"
echo ""
echo -e "  ${BOLD}Utilities:${NC}"
echo "    /search                Semantic + grep codebase search"
echo "    /verify                Run tests, lint, type-check"
echo "    /checkpoint            Save progress state"
echo "    /learn                 Extract patterns from session"
echo "    /ship-day              Squash commits & create PR"
echo ""
echo -e "  Update:    ${BLUE}$TOOLKIT_DIR/install.sh --update${NC}"
echo -e "  Uninstall: ${BLUE}$TOOLKIT_DIR/install.sh --uninstall${NC}"
echo ""
