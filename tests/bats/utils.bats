#!/usr/bin/env bats

load test_helper/common

setup() {
  setup_temp_project
  source_lib "utils.sh"
}

teardown() {
  teardown_temp_project
}

# ── merge_mcp_server ──

@test "merge_mcp_server: creates new .mcp.json" {
  local file="$TEST_PROJECT_DIR/.mcp.json"
  merge_mcp_server "$file" "deep-think" '{"command":"mcp-deep-think","args":[]}'
  [ -f "$file" ]
  result=$(jq -r '.mcpServers["deep-think"].command' "$file")
  [ "$result" = "mcp-deep-think" ]
}

@test "merge_mcp_server: adds to existing .mcp.json" {
  local file="$TEST_PROJECT_DIR/.mcp.json"
  echo '{"mcpServers":{"existing":{"command":"existing-cmd"}}}' | jq '.' > "$file"
  merge_mcp_server "$file" "deep-think" '{"command":"mcp-deep-think","args":[]}'
  # Both should exist
  result_existing=$(jq -r '.mcpServers.existing.command' "$file")
  result_new=$(jq -r '.mcpServers["deep-think"].command' "$file")
  [ "$result_existing" = "existing-cmd" ]
  [ "$result_new" = "mcp-deep-think" ]
}

@test "merge_mcp_server: does not overwrite existing server" {
  local file="$TEST_PROJECT_DIR/.mcp.json"
  echo '{"mcpServers":{"deep-think":{"command":"custom-cmd"}}}' | jq '.' > "$file"
  merge_mcp_server "$file" "deep-think" '{"command":"mcp-deep-think","args":[]}'
  # Should keep original
  result=$(jq -r '.mcpServers["deep-think"].command' "$file")
  [ "$result" = "custom-cmd" ]
}

# ── append_gitignore ──

@test "append_gitignore: creates gitignore from entries" {
  local gitignore="$TEST_PROJECT_DIR/.gitignore"
  local entries="$TEST_PROJECT_DIR/entries.txt"
  echo -e "prd.json\nprogress.txt" > "$entries"
  append_gitignore "$gitignore" "$entries"
  [ -f "$gitignore" ]
  grep -q "prd.json" "$gitignore"
  grep -q "progress.txt" "$gitignore"
}

@test "append_gitignore: appends to existing gitignore" {
  local gitignore="$TEST_PROJECT_DIR/.gitignore"
  local entries="$TEST_PROJECT_DIR/entries.txt"
  echo "node_modules/" > "$gitignore"
  echo -e "prd.json\nprogress.txt" > "$entries"
  append_gitignore "$gitignore" "$entries"
  grep -q "node_modules/" "$gitignore"
  grep -q "prd.json" "$gitignore"
}

@test "append_gitignore: does not duplicate entries" {
  local gitignore="$TEST_PROJECT_DIR/.gitignore"
  local entries="$TEST_PROJECT_DIR/entries.txt"
  echo -e "prd.json\nprogress.txt" > "$gitignore"
  echo -e "prd.json\nnew-entry" > "$entries"
  append_gitignore "$gitignore" "$entries"
  count=$(grep -c "prd.json" "$gitignore")
  [ "$count" -eq 1 ]
  grep -q "new-entry" "$gitignore"
}

# ── merge_hooks_json ──

@test "merge_hooks_json: creates new hooks.json" {
  local target="$TEST_PROJECT_DIR/.claude/hooks/hooks.json"
  local source_hooks='{"hooks":{"PreToolUse":[{"matcher":"Bash","hooks":[{"type":"command","command":"echo test"}]}]}}'
  local source_file="$TEST_PROJECT_DIR/source-hooks.json"
  echo "$source_hooks" > "$source_file"
  merge_hooks_json "$target" "$source_file"
  [ -f "$target" ]
  result=$(jq -r '.hooks.PreToolUse[0].matcher' "$target")
  [ "$result" = "Bash" ]
}

@test "merge_hooks_json: merges into existing hooks.json" {
  local target="$TEST_PROJECT_DIR/.claude/hooks/hooks.json"
  mkdir -p "$(dirname "$target")"
  echo '{"hooks":{"PreToolUse":[{"matcher":"Write","hooks":[{"type":"command","command":"echo existing"}]}]}}' | jq '.' > "$target"

  local source_file="$TEST_PROJECT_DIR/source-hooks.json"
  echo '{"hooks":{"PostToolUse":[{"matcher":"Edit","hooks":[{"type":"command","command":"echo new"}]}]}}' > "$source_file"
  merge_hooks_json "$target" "$source_file"

  # Both should exist
  pre_count=$(jq '.hooks.PreToolUse | length' "$target")
  post_count=$(jq '.hooks.PostToolUse | length' "$target")
  [ "$pre_count" -eq 1 ]
  [ "$post_count" -eq 1 ]
}

@test "merge_hooks_json: does not duplicate existing hook events" {
  local target="$TEST_PROJECT_DIR/.claude/hooks/hooks.json"
  mkdir -p "$(dirname "$target")"
  echo '{"hooks":{"PreToolUse":[{"matcher":"Bash","hooks":[{"type":"command","command":"echo existing"}]}]}}' | jq '.' > "$target"

  local source_file="$TEST_PROJECT_DIR/source-hooks.json"
  echo '{"hooks":{"PreToolUse":[{"matcher":"Bash","hooks":[{"type":"command","command":"echo new"}]}]}}' > "$source_file"
  merge_hooks_json "$target" "$source_file"

  # Should still only have one PreToolUse entry (existing preserved, new skipped since same matcher)
  pre_count=$(jq '.hooks.PreToolUse | length' "$target")
  [ "$pre_count" -eq 1 ]
  # Original command preserved
  result=$(jq -r '.hooks.PreToolUse[0].hooks[0].command' "$target")
  [ "$result" = "echo existing" ]
}

# ── to_json_array ──

@test "to_json_array: converts space-separated to JSON array" {
  result=$(to_json_array "go node react")
  expected='["go","node","react"]'
  [ "$result" = "$expected" ]
}

@test "to_json_array: empty string returns empty array" {
  result=$(to_json_array "")
  [ "$result" = "[]" ]
}

@test "to_json_array: single item" {
  result=$(to_json_array "go")
  [ "$result" = '["go"]' ]
}
