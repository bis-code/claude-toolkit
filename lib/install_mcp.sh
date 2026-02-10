#!/bin/bash
# MCP server configuration installation

# Install a specific MCP server config into project .mcp.json
install_mcp_config() {
  local project_dir="$1" server_name="$2"
  local mcp_file="$project_dir/.mcp.json"

  case "$server_name" in
    deep-think)
      merge_mcp_server "$mcp_file" "deep-think" '{"command":"mcp-deep-think","args":[]}'
      ;;
    playwright)
      merge_mcp_server "$mcp_file" "playwright" '{"command":"npx","args":["@playwright/mcp@latest","--headless","--isolated"]}'
      ;;
    leann-server)
      merge_mcp_server "$mcp_file" "leann-server" '{"command":"leann_mcp","args":[]}'
      ;;
    context7)
      merge_mcp_server "$mcp_file" "context7" '{"command":"npx","args":["-y","@upstash/context7-mcp@latest"]}'
      ;;
    *)
      error "Unknown MCP server: $server_name"
      return 1
      ;;
  esac
}
