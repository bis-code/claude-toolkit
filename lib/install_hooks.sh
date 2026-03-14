#!/bin/bash
# Hooks installation logic

# Install hooks into project
# Usage: install_hooks <project_dir> <templates_dir>
install_hooks() {
  local project_dir="$1"
  local templates_dir="$2"
  local hooks_src="$templates_dir/hooks/hooks.json"
  local settings_file="$project_dir/.claude/settings.json"
  local scripts_src="$templates_dir/hooks/scripts"
  local scripts_dest="$project_dir/.claude/hooks/scripts"

  if [ ! -f "$hooks_src" ]; then
    warn "No hooks template found"
    return
  fi

  # Merge hooks into .claude/settings.json (where Claude Code reads them)
  merge_hooks_into_settings "$settings_file" "$hooks_src"

  # Also keep hooks.json as reference (not read by Claude Code)
  local hooks_dest="$project_dir/.claude/hooks/hooks.json"
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

# Merge toolkit hooks into .claude/settings.json
# Preserves existing settings (mcpServers, etc.) and user hooks
merge_hooks_into_settings() {
  local settings_file="$1"
  local hooks_src="$2"

  mkdir -p "$(dirname "$settings_file")"

  # Extract just the hooks object from the template, tagged with _toolkit
  local toolkit_hooks
  toolkit_hooks=$(jq '.hooks | with_entries(.value |= map(. + {"_toolkit": true}))' "$hooks_src")

  if [ ! -f "$settings_file" ]; then
    # No settings.json yet — create with hooks only
    echo "{\"hooks\": $toolkit_hooks}" | jq '.' > "$settings_file"
    return
  fi

  # settings.json exists — merge hooks into it
  local tmp_file
  tmp_file="$(mktemp)"

  jq --argjson new_hooks "$toolkit_hooks" '
    # Remove old toolkit hooks from existing settings
    .hooks //= {} |
    .hooks |= with_entries(
      .value |= (if type == "array" then map(select(._toolkit != true)) else . end)
    ) |
    .hooks |= with_entries(select(.value | length > 0)) |

    # Merge in new toolkit hooks
    reduce ($new_hooks | to_entries[]) as $entry (
      .;
      .hooks[$entry.key] = ((.hooks[$entry.key] // []) + $entry.value)
    )
  ' "$settings_file" > "$tmp_file"

  if jq empty "$tmp_file" 2>/dev/null; then
    mv "$tmp_file" "$settings_file"
  else
    rm -f "$tmp_file"
    warn "Failed to merge hooks into settings.json"
  fi
}
