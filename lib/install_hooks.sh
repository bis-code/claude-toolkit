#!/bin/bash
# Hooks installation logic

# Install hooks into project
# Usage: install_hooks <project_dir> <templates_dir>
install_hooks() {
  local project_dir="$1"
  local templates_dir="$2"
  local hooks_src="$templates_dir/hooks/hooks.json"
  local hooks_dest="$project_dir/.claude/hooks/hooks.json"
  local scripts_src="$templates_dir/hooks/scripts"
  local scripts_dest="$project_dir/.claude/hooks/scripts"

  if [ ! -f "$hooks_src" ]; then
    warn "No hooks template found"
    return
  fi

  # Merge hooks.json (preserves user hooks via _toolkit tag)
  merge_hooks_json "$hooks_dest" "$hooks_src"

  # Copy hook scripts directory
  if [ -d "$scripts_src" ]; then
    mkdir -p "$scripts_dest/lib"

    # Copy all script files
    for script in "$scripts_src"/*.js; do
      if [ -f "$script" ]; then
        local filename
        filename=$(basename "$script")
        _tracked_copy "$script" "$scripts_dest/$filename" ".claude/hooks/scripts/$filename"
      fi
    done

    # Copy lib/ utilities
    for lib_file in "$scripts_src"/lib/*.js; do
      if [ -f "$lib_file" ]; then
        local filename
        filename=$(basename "$lib_file")
        _tracked_copy "$lib_file" "$scripts_dest/lib/$filename" ".claude/hooks/scripts/lib/$filename"
      fi
    done
  fi
}
