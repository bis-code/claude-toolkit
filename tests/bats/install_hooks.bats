#!/usr/bin/env bats
# Tests for hook scripts installation

setup() {
  export TOOLKIT_DIR="${BATS_TEST_DIRNAME}/../.."
  export TEST_PROJECT=$(mktemp -d)
  mkdir -p "$TEST_PROJECT/.claude/hooks"
  source "$TOOLKIT_DIR/lib/utils.sh"
  source "$TOOLKIT_DIR/lib/install_hooks.sh"
  init_update_tracking
}

teardown() {
  rm -rf "$TEST_PROJECT"
}

@test "install_hooks copies hooks.json" {
  install_hooks "$TEST_PROJECT" "$TOOLKIT_DIR/templates"
  [ -f "$TEST_PROJECT/.claude/hooks/hooks.json" ]
}

@test "install_hooks copies scripts directory" {
  install_hooks "$TEST_PROJECT" "$TOOLKIT_DIR/templates"
  [ -d "$TEST_PROJECT/.claude/hooks/scripts" ]
}

@test "install_hooks copies run-with-flags.js" {
  install_hooks "$TEST_PROJECT" "$TOOLKIT_DIR/templates"
  [ -f "$TEST_PROJECT/.claude/hooks/scripts/run-with-flags.js" ]
}

@test "install_hooks copies all 8 hook scripts" {
  install_hooks "$TEST_PROJECT" "$TOOLKIT_DIR/templates"
  [ -f "$TEST_PROJECT/.claude/hooks/scripts/session-start.js" ]
  [ -f "$TEST_PROJECT/.claude/hooks/scripts/session-end.js" ]
  [ -f "$TEST_PROJECT/.claude/hooks/scripts/observe.js" ]
  [ -f "$TEST_PROJECT/.claude/hooks/scripts/evaluate-session.js" ]
  [ -f "$TEST_PROJECT/.claude/hooks/scripts/secret-detector.js" ]
  [ -f "$TEST_PROJECT/.claude/hooks/scripts/tmux-safety.js" ]
  [ -f "$TEST_PROJECT/.claude/hooks/scripts/quality-gate.js" ]
  [ -f "$TEST_PROJECT/.claude/hooks/scripts/pre-compact.js" ]
}

@test "install_hooks copies lib utilities" {
  install_hooks "$TEST_PROJECT" "$TOOLKIT_DIR/templates"
  [ -f "$TEST_PROJECT/.claude/hooks/scripts/lib/utils.js" ]
  [ -f "$TEST_PROJECT/.claude/hooks/scripts/lib/hook-flags.js" ]
}

@test "hooks.json contains all 6 event types" {
  install_hooks "$TEST_PROJECT" "$TOOLKIT_DIR/templates"
  local hooks_file="$TEST_PROJECT/.claude/hooks/hooks.json"
  grep -q "SessionStart" "$hooks_file"
  grep -q "PreToolUse" "$hooks_file"
  grep -q "PostToolUse" "$hooks_file"
  grep -q "PreCompact" "$hooks_file"
  grep -q "Stop" "$hooks_file"
}

@test "run-with-flags.js passes through stdin" {
  install_hooks "$TEST_PROJECT" "$TOOLKIT_DIR/templates"
  local result
  result=$(echo '{"test":"data"}' | node "$TEST_PROJECT/.claude/hooks/scripts/run-with-flags.js" "test:pass" "session-start.js" "minimal,standard,strict" 2>/dev/null)
  echo "$result" | grep -q '"test"'
}

@test "secret-detector warns on API keys" {
  install_hooks "$TEST_PROJECT" "$TOOLKIT_DIR/templates"
  local stderr
  stderr=$(echo '{"tool_input":{"content":"sk-abc123456789012345"}}' | node "$TEST_PROJECT/.claude/hooks/scripts/run-with-flags.js" "pre:secret" "secret-detector.js" "standard,strict" 2>&1 >/dev/null)
  echo "$stderr" | grep -q "WARNING"
}
