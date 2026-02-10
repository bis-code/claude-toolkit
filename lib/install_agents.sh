#!/bin/bash
# Agents installation logic

# Install agent definitions into project .claude/agents/
# Usage: install_agents <project_dir> <templates_dir> [domains]
#   domains: space-separated list of domain directory names (e.g., "blockchain react golang")
install_agents() {
  local project_dir="$1"
  local templates_dir="$2"
  local domains="$3"
  local agents_src="$templates_dir/agents"
  local agents_dest="$project_dir/.claude/agents"

  if [ ! -d "$agents_src" ]; then
    warn "No agent templates found"
    return
  fi

  mkdir -p "$agents_dest"

  # 1. Always install generic agents (top-level *.md files)
  for agent_file in "$agents_src"/*.md; do
    [ -f "$agent_file" ] || continue
    local filename
    filename="$(basename "$agent_file")"
    if [ ! -f "$agents_dest/$filename" ] || [ "${FORCE:-false}" = true ]; then
      cp "$agent_file" "$agents_dest/$filename"
    fi
  done

  # 2. Install domain-specific agents based on detected domains
  for domain in $domains; do
    local domain_src="$agents_src/domain/$domain"
    [ -d "$domain_src" ] || continue
    for agent_file in "$domain_src"/*.md; do
      [ -f "$agent_file" ] || continue
      local filename
      filename="$(basename "$agent_file")"
      if [ ! -f "$agents_dest/$filename" ] || [ "${FORCE:-false}" = true ]; then
        cp "$agent_file" "$agents_dest/$filename"
      fi
    done
  done
}
