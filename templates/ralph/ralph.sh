#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
PROMPT_FILE="$SCRIPT_DIR/RALPH.md"
PRD_FILE="$PROJECT_DIR/prd.json"
PROGRESS_FILE="$PROJECT_DIR/progress.txt"

# Defaults
MAX_LOOPS=30
STUCK_THRESHOLD=3

# Parse args
while [[ $# -gt 0 ]]; do
  case $1 in
    --max-loops) MAX_LOOPS="$2"; shift 2 ;;
    --max-loops=*) MAX_LOOPS="${1#*=}"; shift ;;
    --stuck-threshold) STUCK_THRESHOLD="$2"; shift 2 ;;
    --stuck-threshold=*) STUCK_THRESHOLD="${1#*=}"; shift ;;
    -h|--help)
      echo "Usage: ralph.sh [OPTIONS]"
      echo ""
      echo "Autonomous feature builder. Spawns fresh Claude Code instances"
      echo "per user story with persistent state and structured reasoning."
      echo ""
      echo "Options:"
      echo "  --max-loops N          Maximum iterations (default: 30)"
      echo "  --stuck-threshold N    Mark stuck after N iterations with no progress (default: 3)"
      echo "  -h, --help             Show this help"
      echo ""
      echo "Prerequisites:"
      echo "  - prd.json in project root (generate with /ralph command)"
      echo "  - claude CLI installed and authenticated"
      echo "  - jq installed"
      exit 0
      ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

# Validate prerequisites
if [ ! -f "$PRD_FILE" ]; then
  echo "ERROR: prd.json not found at $PRD_FILE"
  echo "Run /ralph first to generate a PRD from GitHub issues."
  exit 1
fi

if ! command -v claude &> /dev/null; then
  echo "ERROR: claude CLI not found in PATH"
  exit 1
fi

if ! command -v jq &> /dev/null; then
  echo "ERROR: jq not found in PATH (brew install jq)"
  exit 1
fi

# Initialize progress file if it doesn't exist
if [ ! -f "$PROGRESS_FILE" ]; then
  echo "# Progress Log" > "$PROGRESS_FILE"
  echo "Started: $(date)" >> "$PROGRESS_FILE"
  echo "---" >> "$PROGRESS_FILE"
fi

# Summary function
print_summary() {
  echo ""
  echo "============================================"
  echo "  RALPH SESSION SUMMARY"
  echo "============================================"

  local total passed stuck remaining
  total=$(jq '.userStories | length' "$PRD_FILE" 2>/dev/null || echo "0")
  passed=$(jq '[.userStories[] | select(.passes == true)] | length' "$PRD_FILE" 2>/dev/null || echo "0")
  stuck=$(jq '[.userStories[] | select(.stuck == true)] | length' "$PRD_FILE" 2>/dev/null || echo "0")
  remaining=$((total - passed - stuck))

  echo "  Total stories:     $total"
  echo "  Completed:         $passed"
  echo "  Stuck:             $stuck"
  echo "  Remaining:         $remaining"
  echo ""

  # Show stuck stories
  if [ "$stuck" -gt 0 ]; then
    echo "  STUCK STORIES:"
    jq -r '.userStories[] | select(.stuck == true) | "    - \(.id): \(.title) — \(.stuckReason)"' "$PRD_FILE" 2>/dev/null
    echo ""
  fi

  echo "============================================"
}

# Track iterations without progress for stuck detection
NO_PROGRESS_COUNT=0
LAST_PASSED_COUNT=$(jq '[.userStories[] | select(.passes == true)] | length' "$PRD_FILE" 2>/dev/null || echo "0")

echo "============================================"
echo "  RALPH — Autonomous Feature Builder"
echo "============================================"
echo "  PRD: $PRD_FILE"
echo "  Max loops: $MAX_LOOPS"
echo "  Stuck threshold: $STUCK_THRESHOLD iterations"
echo ""

# Show stories to implement
echo "  Stories:"
jq -r '.userStories[] | "    [\(if .passes then "DONE" elif .stuck then "STUCK" else "TODO" end)] \(.id): \(.title)"' "$PRD_FILE" 2>/dev/null
echo ""

for i in $(seq 1 "$MAX_LOOPS"); do
  # Check if all stories are done
  REMAINING=$(jq '[.userStories[] | select(.passes == false and .stuck != true)] | length' "$PRD_FILE" 2>/dev/null || echo "0")
  if [ "$REMAINING" -eq 0 ]; then
    echo "All stories completed or stuck!"
    print_summary
    exit 0
  fi

  echo "============================================"
  echo "  Ralph Iteration $i of $MAX_LOOPS"
  echo "  Remaining stories: $REMAINING"
  echo "============================================"

  # Backup prd.json before each iteration (corruption recovery)
  cp "$PRD_FILE" "$PRD_FILE.bak"

  # Spawn fresh Claude instance
  OUTPUT=$(cd "$PROJECT_DIR" && claude --dangerously-skip-permissions --print < "$PROMPT_FILE" 2>&1 | tee /dev/stderr) || true

  # Validate prd.json wasn't corrupted
  if ! jq empty "$PRD_FILE" 2>/dev/null; then
    echo "WARNING: prd.json corrupted! Restoring backup..."
    cp "$PRD_FILE.bak" "$PRD_FILE"
  fi

  # Check for completion signal
  if echo "$OUTPUT" | grep -q "<promise>COMPLETE</promise>"; then
    echo ""
    echo "Ralph: All stories complete and QA passed!"
    print_summary
    exit 0
  fi

  # Stuck detection: check if progress was made
  CURRENT_PASSED=$(jq '[.userStories[] | select(.passes == true)] | length' "$PRD_FILE" 2>/dev/null || echo "0")
  if [ "$CURRENT_PASSED" -eq "$LAST_PASSED_COUNT" ]; then
    NO_PROGRESS_COUNT=$((NO_PROGRESS_COUNT + 1))
    if [ "$NO_PROGRESS_COUNT" -ge "$STUCK_THRESHOLD" ]; then
      echo ""
      echo "WARNING: No progress for $NO_PROGRESS_COUNT iterations."

      # Find the current story and mark it stuck
      CURRENT_STORY_ID=$(jq -r '[.userStories[] | select(.passes == false and .stuck != true)] | sort_by(.priority) | .[0].id // empty' "$PRD_FILE" 2>/dev/null)
      if [ -n "$CURRENT_STORY_ID" ]; then
        echo "Marking $CURRENT_STORY_ID as stuck (no progress after $STUCK_THRESHOLD iterations)"
        jq --arg id "$CURRENT_STORY_ID" '(.userStories[] | select(.id == $id)) |= (.stuck = true | .stuckReason = "No progress after automatic stuck detection")' "$PRD_FILE" > "$PRD_FILE.tmp" && mv "$PRD_FILE.tmp" "$PRD_FILE"
        NO_PROGRESS_COUNT=0
      fi
    fi
  else
    NO_PROGRESS_COUNT=0
    LAST_PASSED_COUNT=$CURRENT_PASSED
  fi

  echo ""
  echo "Iteration $i complete. Continuing..."
  sleep 2
done

echo ""
echo "Ralph reached max iterations ($MAX_LOOPS)."
print_summary
exit 1
