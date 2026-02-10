#!/bin/bash
# Hooks installation logic

# Install hooks into project
# Usage: install_hooks <project_dir> <templates_dir>
install_hooks() {
  local project_dir="$1"
  local templates_dir="$2"
  local hooks_src="$templates_dir/hooks/hooks.json"
  local hooks_dest="$project_dir/.claude/hooks/hooks.json"

  if [ ! -f "$hooks_src" ]; then
    warn "No hooks template found"
    return
  fi

  merge_hooks_json "$hooks_dest" "$hooks_src"
}
