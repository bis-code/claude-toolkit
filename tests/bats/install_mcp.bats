#!/usr/bin/env bats

load test_helper/common

setup() {
  setup_temp_project
  source_lib "utils.sh"
  source_lib "install_mcp.sh"
}

teardown() {
  teardown_temp_project
}

@test "install_mcp: installs deep-think server" {
  install_mcp_config "$TEST_PROJECT_DIR" "deep-think"
  assert_file_exists "$TEST_PROJECT_DIR/.mcp.json"
  result=$(jq -r '.mcpServers["deep-think"].command' "$TEST_PROJECT_DIR/.mcp.json")
  [ "$result" = "mcp-deep-think" ]
}

@test "install_mcp: installs playwright server" {
  install_mcp_config "$TEST_PROJECT_DIR" "playwright"
  result=$(jq -r '.mcpServers.playwright.command' "$TEST_PROJECT_DIR/.mcp.json")
  [ "$result" = "npx" ]
  result=$(jq -r '.mcpServers.playwright.args[0]' "$TEST_PROJECT_DIR/.mcp.json")
  [ "$result" = "@playwright/mcp@latest" ]
}

@test "install_mcp: installs leann-server" {
  install_mcp_config "$TEST_PROJECT_DIR" "leann-server"
  result=$(jq -r '.mcpServers["leann-server"].command' "$TEST_PROJECT_DIR/.mcp.json")
  [ "$result" = "leann_mcp" ]
}

@test "install_mcp: installs context7 server" {
  install_mcp_config "$TEST_PROJECT_DIR" "context7"
  result=$(jq -r '.mcpServers.context7.command' "$TEST_PROJECT_DIR/.mcp.json")
  [ "$result" = "npx" ]
  result=$(jq -r '.mcpServers.context7.args[0]' "$TEST_PROJECT_DIR/.mcp.json")
  [ "$result" = "-y" ]
  result=$(jq -r '.mcpServers.context7.args[1]' "$TEST_PROJECT_DIR/.mcp.json")
  [ "$result" = "@upstash/context7-mcp@latest" ]
}

@test "install_mcp: multiple servers" {
  install_mcp_config "$TEST_PROJECT_DIR" "deep-think"
  install_mcp_config "$TEST_PROJECT_DIR" "context7"
  count=$(jq '.mcpServers | length' "$TEST_PROJECT_DIR/.mcp.json")
  [ "$count" -eq 2 ]
}

@test "install_mcp: does not overwrite existing server" {
  echo '{"mcpServers":{"deep-think":{"command":"custom-deep-think"}}}' | jq '.' > "$TEST_PROJECT_DIR/.mcp.json"
  install_mcp_config "$TEST_PROJECT_DIR" "deep-think"
  result=$(jq -r '.mcpServers["deep-think"].command' "$TEST_PROJECT_DIR/.mcp.json")
  [ "$result" = "custom-deep-think" ]
}
