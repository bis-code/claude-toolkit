#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
PROMPT_FILE="$SCRIPT_DIR/QA_PROMPT.md"

# Defaults
MAX_LOOPS=20
SCOPE="all"
SCAN_ONLY=false
WORKTREE_BRANCH="main"
USE_WORKTREE=true
CUSTOM_PROMPT=""

# Read config from .claude-toolkit.json if available
if [ -f "$PROJECT_DIR/.claude-toolkit.json" ] && command -v jq &> /dev/null; then
  CONFIGURED_BRANCH=$(jq -r '.qa.worktreeFromBranch // empty' "$PROJECT_DIR/.claude-toolkit.json" 2>/dev/null)
  if [ -n "$CONFIGURED_BRANCH" ] && [ "$CONFIGURED_BRANCH" != "null" ]; then
    WORKTREE_BRANCH="$CONFIGURED_BRANCH"
  else
    USE_WORKTREE=false
  fi

  PROJECT_TYPE=$(jq -r '.project.type // "repository"' "$PROJECT_DIR/.claude-toolkit.json" 2>/dev/null)
  if [ "$PROJECT_TYPE" = "workspace" ]; then
    USE_WORKTREE=false
  fi
fi

# Also check if we're in a git repo
if ! git rev-parse --show-toplevel &>/dev/null; then
  USE_WORKTREE=false
fi

# Parse args
while [[ $# -gt 0 ]]; do
  case $1 in
    --max-loops) MAX_LOOPS="$2"; shift 2 ;;
    --max-loops=*) MAX_LOOPS="${1#*=}"; shift ;;
    --scope) SCOPE="$2"; shift 2 ;;
    --scope=*) SCOPE="${1#*=}"; shift ;;
    --scan-only) SCAN_ONLY=true; shift ;;
    --branch) WORKTREE_BRANCH="$2"; USE_WORKTREE=true; shift 2 ;;
    --branch=*) WORKTREE_BRANCH="${1#*=}"; USE_WORKTREE=true; shift ;;
    --no-worktree) USE_WORKTREE=false; shift ;;
    --prompt) CUSTOM_PROMPT="$2"; shift 2 ;;
    --prompt=*) CUSTOM_PROMPT="${1#*=}"; shift ;;
    -h|--help)
      echo "Usage: qa.sh [OPTIONS]"
      echo ""
      echo "Autonomous QA agent. Scans, fixes, and reports issues."
      echo "Uses a git worktree when in a git repo, runs in-place otherwise."
      echo ""
      echo "Options:"
      echo "  --max-loops N    Maximum iterations (default: 20)"
      echo "  --scope SCOPE    Scan scope: all, api, web, or monorepo project name (default: all)"
      echo "  --prompt TEXT    Custom focus prompt (e.g., 'focus on N+1 queries')"
      echo "  --scan-only      Report only, no fixes"
      echo "  --branch NAME    Branch to create worktree from (default: main)"
      echo "  --no-worktree    Run in-place even in git repos"
      echo "  -h, --help       Show this help"
      exit 0
      ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

# ─────────────────────────────────────────────
# Worktree mode (git repos)
# ─────────────────────────────────────────────
if [ "$USE_WORKTREE" = true ]; then
  BRANCH="qa/$(date +%Y-%m-%d-%H%M%S)"
  WORKTREE_DIR="/tmp/$(basename "$PROJECT_DIR")-qa-$$"
  WORK_DIR="$WORKTREE_DIR"

  echo "Setting up QA worktree from $WORKTREE_BRANCH..."
  cd "$PROJECT_DIR"

  # Fetch latest
  git fetch origin "$WORKTREE_BRANCH" 2>/dev/null || true

  # Create QA branch and add worktree
  git branch "$BRANCH" "origin/$WORKTREE_BRANCH" 2>/dev/null || git branch "$BRANCH" "$WORKTREE_BRANCH"
  git worktree add "$WORKTREE_DIR" "$BRANCH"

  # Copy prompt file into worktree
  cp "$PROMPT_FILE" "$WORKTREE_DIR/tools/qa/QA_PROMPT.md"

  # Cleanup on exit (worktree mode)
  cleanup() {
    rm -f "$WORK_DIR/.qa-active"
    cd "$WORK_DIR" 2>/dev/null || cd "$PROJECT_DIR"

    # Check if we made any commits on QA branch
    COMMITS=$(git log "$WORKTREE_BRANCH".."$BRANCH" --oneline 2>/dev/null | wc -l | tr -d ' ')
    if [ "$COMMITS" -gt 0 ]; then
      echo ""
      echo "QA branch has $COMMITS commit(s). Creating PR..."
      git push -u origin "$BRANCH" 2>&1
      gh pr create --title "qa: automated QA fixes ($(date +%Y-%m-%d))" \
        --body "$(cat <<EOF
## Summary
Automated QA fixes from \`qa.sh\` run.

## Findings
$(jq -r '.findings[] | "- [\(.status)] \(.category): \(.summary)"' "$STATE_FILE" 2>/dev/null || echo "See qa-state.json")

## Stats
- Fixed: $(jq -r '.fixedCount' "$STATE_FILE")
- Reported as issues: $(jq -r '.reportedCount' "$STATE_FILE")
- Iterations: $(jq -r '.iteration' "$STATE_FILE")

Generated with Claude Code
EOF
)" --label "from-qa-auto" 2>&1
    else
      echo "No fixes made."
    fi

    # Remove worktree and branch
    cd "$PROJECT_DIR"
    git worktree remove "$WORKTREE_DIR" --force 2>/dev/null || rm -rf "$WORKTREE_DIR"
    if [ "$COMMITS" -eq 0 ] 2>/dev/null; then
      git branch -D "$BRANCH" 2>/dev/null || true
    fi
  }
  trap cleanup EXIT

# ─────────────────────────────────────────────
# In-place mode (non-git / workspace)
# ─────────────────────────────────────────────
else
  WORK_DIR="$PROJECT_DIR"

  echo "Running QA in-place (no worktree)..."

  # No cleanup needed for in-place mode
  cleanup() {
    rm -f "$WORK_DIR/.qa-active"
    echo ""
    echo "QA run complete. Review any changes with: git diff (if applicable)"
  }
  trap cleanup EXIT
fi

# ─────────────────────────────────────────────
# Common setup
# ─────────────────────────────────────────────
STATE_FILE="$WORK_DIR/tools/qa/qa-state.json"
PROGRESS_FILE="$WORK_DIR/tools/qa/qa-progress.txt"
MARKER_FILE="$WORK_DIR/.qa-active"

# ─────────────────────────────────────────────
# Resolve scope for monorepo projects
# ─────────────────────────────────────────────
SCOPE_DIR=""
if [ "$SCOPE" != "all" ]; then
  # Try to resolve scope to a directory path using monorepo detection
  TOOLKIT_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
  if [ -f "$TOOLKIT_LIB_DIR/.claude-toolkit.json" ]; then
    TOOLKIT_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && cd "$(jq -r '.project.toolkitDir // empty' "$TOOLKIT_LIB_DIR/.claude-toolkit.json" 2>/dev/null)" 2>/dev/null && pwd || echo "")"
  fi
  # Source detect.sh if available for monorepo resolution
  for try_dir in "$SCRIPT_DIR/../../lib" "$HOME/.claude/toolkit/lib"; do
    if [ -f "$try_dir/detect.sh" ]; then
      source "$try_dir/detect.sh"
      STRUCTURE=$(detect_project_structure "$PROJECT_DIR" 2>/dev/null || echo '{"type":"single"}')
      if echo "$STRUCTURE" | jq -e '.type == "monorepo"' &>/dev/null; then
        # Find matching project path
        MATCHED_PATH=$(echo "$STRUCTURE" | jq -r --arg s "$SCOPE" '.projects[] | select(test($s))' 2>/dev/null | head -1)
        if [ -n "$MATCHED_PATH" ]; then
          SCOPE_DIR="$MATCHED_PATH"
        fi
      fi
      break
    fi
  done
  # Fallback: check if scope is a direct directory
  if [ -z "$SCOPE_DIR" ] && [ -d "$PROJECT_DIR/$SCOPE" ]; then
    SCOPE_DIR="$SCOPE"
  fi
fi

# Initialize state file
mkdir -p "$(dirname "$STATE_FILE")"
STATE_JSON='{"scope":"'"$SCOPE"'","scanOnly":'"$SCAN_ONLY"',"findings":[],"fixedCount":0,"reportedCount":0,"iteration":0'
[ -n "$CUSTOM_PROMPT" ] && STATE_JSON+=',"customPrompt":"'"$(echo "$CUSTOM_PROMPT" | sed 's/"/\\"/g')"'"'
[ -n "$SCOPE_DIR" ] && STATE_JSON+=',"scopeDir":"'"$SCOPE_DIR"'"'
STATE_JSON+='}'
echo "$STATE_JSON" | jq '.' > "$STATE_FILE"

# Initialize progress file
echo "# QA Progress Log" > "$PROGRESS_FILE"
echo "Started: $(date)" >> "$PROGRESS_FILE"
echo "Mode: $([ "$USE_WORKTREE" = true ] && echo "worktree ($BRANCH)" || echo "in-place")" >> "$PROGRESS_FILE"
echo "---" >> "$PROGRESS_FILE"

# Activate QA mode
echo "$([ "$USE_WORKTREE" = true ] && echo "$BRANCH" || echo "in-place")" > "$MARKER_FILE"

echo "Starting QA Agent — Scope: $SCOPE, Max loops: $MAX_LOOPS, Scan only: $SCAN_ONLY"
[ -n "$CUSTOM_PROMPT" ] && echo "Focus: $CUSTOM_PROMPT"
[ -n "$SCOPE_DIR" ] && echo "Scope resolved to: $SCOPE_DIR"
[ "$USE_WORKTREE" = true ] && echo "Branch: $BRANCH" && echo "Worktree: $WORKTREE_DIR"
[ "$USE_WORKTREE" = false ] && echo "Mode: in-place at $WORK_DIR"
echo ""

for i in $(seq 1 "$MAX_LOOPS"); do
  echo "============================================"
  echo "  QA Iteration $i of $MAX_LOOPS"
  echo "============================================"

  # Update iteration counter in state
  jq --argjson i "$i" '.iteration = $i' "$STATE_FILE" > "$STATE_FILE.tmp" && mv "$STATE_FILE.tmp" "$STATE_FILE"

  # Spawn fresh Claude instance in the work directory
  OUTPUT=$(cd "$WORK_DIR" && claude --dangerously-skip-permissions --print < "$WORK_DIR/tools/qa/QA_PROMPT.md" 2>&1 | tee /dev/stderr) || true

  # Check for completion signal
  if echo "$OUTPUT" | grep -q "<promise>COMPLETE</promise>"; then
    echo ""
    echo "QA agent: codebase is clean!"
    echo "Completed at iteration $i of $MAX_LOOPS"
    exit 0
  fi

  echo ""
  echo "Iteration $i complete. Continuing..."
  sleep 2
done

echo ""
echo "QA agent reached max iterations ($MAX_LOOPS)."
exit 1
