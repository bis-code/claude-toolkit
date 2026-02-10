#!/bin/bash
# Agents installation logic

# Install agent definitions into project .claude/agents/
# Usage: install_agents <project_dir> <templates_dir>
install_agents() {
  local project_dir="$1"
  local templates_dir="$2"
  local agents_src="$templates_dir/agents"
  local agents_dest="$project_dir/.claude/agents"

  if [ ! -d "$agents_src" ]; then
    warn "No agent templates found"
    return
  fi

  mkdir -p "$agents_dest"

  for agent_file in "$agents_src"/*.md; do
    [ -f "$agent_file" ] || continue
    local filename
    filename="$(basename "$agent_file")"
    if [ ! -f "$agents_dest/$filename" ] || [ "${FORCE:-false}" = true ]; then
      cp "$agent_file" "$agents_dest/$filename"
    fi
  done
}
