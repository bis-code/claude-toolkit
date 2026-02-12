#!/bin/bash
# Shared utility functions for the Claude Toolkit installer

# Colors (only set if not already defined)
: "${RED:=\033[0;31m}"
: "${GREEN:=\033[0;32m}"
: "${YELLOW:=\033[1;33m}"
: "${BLUE:=\033[0;34m}"
: "${BOLD:=\033[1m}"
: "${NC:=\033[0m}"

info()    { echo -e "    ${GREEN}✓${NC} $1"; }
warn()    { echo -e "    ${YELLOW}⚠${NC} $1"; }
error()   { echo -e "    ${RED}✗${NC} $1"; }
header()  { echo -e "\n${BOLD}[$1]${NC} $2"; }

check_cmd() {
  if command -v "$1" &> /dev/null; then
    info "$1"
    return 0
  else
    error "$1 — $2"
    return 1
  fi
}

# Merge JSON key into .mcp.json without overwriting existing servers
merge_mcp_server() {
  local file="$1" name="$2" config="$3"
  if [ ! -f "$file" ]; then
    echo "{\"mcpServers\":{\"$name\":$config}}" | jq '.' > "$file"
  elif jq -e ".mcpServers.\"$name\"" "$file" &>/dev/null; then
    return 0  # Already exists — do not overwrite
  else
    jq --arg name "$name" --argjson config "$config" '.mcpServers[$name] = $config' "$file" > "$file.tmp" && mv "$file.tmp" "$file"
  fi
}

# Append lines to .gitignore if not already present
append_gitignore() {
  local gitignore="$1" entries_file="$2"
  if [ ! -f "$gitignore" ]; then
    cp "$entries_file" "$gitignore"
    return
  fi
  while IFS= read -r line; do
    [ -z "$line" ] && continue
    [[ "$line" == \#* ]] && continue
    if ! grep -qF "$line" "$gitignore" 2>/dev/null; then
      echo "$line" >> "$gitignore"
    fi
  done < "$entries_file"
}

# Merge hooks.json: add new hook events without overwriting existing ones
merge_hooks_json() {
  local target="$1" source_file="$2"

  if [ ! -f "$target" ]; then
    mkdir -p "$(dirname "$target")"
    cp "$source_file" "$target"
    return
  fi

  # For each hook event in source, only add if the event doesn't already have
  # an entry with the same matcher
  local tmp_file
  tmp_file="$(mktemp)"

  jq -s '
    .[0] as $existing |
    .[1] as $new |
    $existing | .hooks as $eh |
    reduce ($new.hooks | to_entries[]) as $entry (
      $existing;
      if ($eh[$entry.key] // [] | map(.matcher) | index($entry.value[0].matcher)) then
        .
      else
        .hooks[$entry.key] = (($eh[$entry.key] // []) + $entry.value)
      end
    )
  ' "$target" "$source_file" > "$tmp_file" && mv "$tmp_file" "$target"
}

# Update .claude-toolkit.json during --update: bump version and re-detected fields,
# preserve user's commands, qa, ralph, and mcpServers sections.
update_toolkit_config() {
  local config_file="$1" version="$2" stack_json="$3" lang_json="$4" pkg_mgr="$5"
  jq --arg v "$version" \
     --argjson stack "$stack_json" \
     --argjson langs "$lang_json" \
     --arg pkg "$pkg_mgr" \
     '.version = $v |
      .project.techStack = $stack |
      .project.languages = $langs |
      .project.packageManager = (if $pkg == "" then null else $pkg end)' \
     "$config_file" > "$config_file.tmp" && mv "$config_file.tmp" "$config_file"
}

# ─────────────────────────────────────────────
# Update tracking: counters, managed files, deprecation
# ─────────────────────────────────────────────

# Reset tracking globals. If config_file is non-empty and has managedFiles,
# load them into OLD_MANAGED_FILES for deprecation comparison.
init_update_tracking() {
  local config_file="$1"
  UPDATE_COUNT=0
  ADDED_COUNT=0
  MANAGED_FILES=()
  OLD_MANAGED_FILES=()

  if [ -n "$config_file" ] && [ -f "$config_file" ]; then
    local raw
    raw=$(jq -r '.managedFiles // [] | .[]' "$config_file" 2>/dev/null)
    if [ -n "$raw" ]; then
      while IFS= read -r path; do
        OLD_MANAGED_FILES+=("$path")
      done <<< "$raw"
    fi
  fi
}

# Copy src to dest, tracking the relative path and incrementing counters.
# Always records rel_path in MANAGED_FILES regardless of copy decision.
# Creates parent directories as needed.
_tracked_copy() {
  local src="$1" dest="$2" rel_path="$3"

  MANAGED_FILES+=("$rel_path")

  if [ ! -f "$dest" ]; then
    mkdir -p "$(dirname "$dest")"
    cp "$src" "$dest"
    ADDED_COUNT=$((ADDED_COUNT + 1))
  elif [ "${FORCE:-false}" = true ] || [ "${MODE:-install}" = "update" ]; then
    cp "$src" "$dest"
    UPDATE_COUNT=$((UPDATE_COUNT + 1))
  fi
  # else: file exists, normal install — skip
}

# Write sorted MANAGED_FILES array into config's .managedFiles key.
write_managed_files() {
  local config_file="$1"

  # Sort the array
  local sorted
  sorted=$(printf '%s\n' "${MANAGED_FILES[@]}" | sort)

  # Build JSON array
  local json_array="["
  local first=true
  while IFS= read -r path; do
    [ -z "$path" ] && continue
    [ "$first" = true ] && first=false || json_array+=","
    json_array+="\"$path\""
  done <<< "$sorted"
  json_array+="]"

  jq --argjson mf "$json_array" '.managedFiles = $mf' "$config_file" > "$config_file.tmp" \
    && mv "$config_file.tmp" "$config_file"
}

# Compare OLD_MANAGED_FILES vs MANAGED_FILES. For paths in old but not new,
# warn if file still exists on disk. No output on first install.
detect_deprecated_files() {
  local project_dir="$1"

  # No prior list means first install — nothing to compare
  [ "${#OLD_MANAGED_FILES[@]}" -eq 0 ] && return 0

  for old_path in "${OLD_MANAGED_FILES[@]}"; do
    # Check if still in current managed files
    local found=false
    for new_path in "${MANAGED_FILES[@]}"; do
      if [ "$old_path" = "$new_path" ]; then
        found=true
        break
      fi
    done

    if [ "$found" = false ] && [ -f "$project_dir/$old_path" ]; then
      warn "$old_path is no longer in toolkit templates (consider removing)"
    fi
  done
}

# Count .md files in .claude/{agents,rules,skills} that are NOT in MANAGED_FILES.
count_preserved_files() {
  local project_dir="$1"
  local count=0

  for dir in "$project_dir/.claude/agents" "$project_dir/.claude/rules" "$project_dir/.claude/skills"; do
    [ -d "$dir" ] || continue
    while IFS= read -r file; do
      [ -z "$file" ] && continue
      local rel_path="${file#"$project_dir/"}"
      local is_managed=false
      for mf in "${MANAGED_FILES[@]}"; do
        if [ "$rel_path" = "$mf" ]; then
          is_managed=true
          break
        fi
      done
      [ "$is_managed" = false ] && count=$((count + 1))
    done < <(find "$dir" -name "*.md" -type f 2>/dev/null)
  done

  echo "$count"
}

# Print a summary line: "Updated N files, added M new, preserved K user files"
print_update_summary() {
  local project_dir="$1"
  local preserved
  preserved=$(count_preserved_files "$project_dir")
  info "Updated $UPDATE_COUNT files, added $ADDED_COUNT new, preserved $preserved user files"
}

# Convert space-separated string to JSON array
to_json_array() {
  local input="$1"
  if [ -z "$input" ]; then
    echo "[]"
    return
  fi
  local result="["
  local first=true
  for item in $input; do
    [ "$first" = true ] && first=false || result+=","
    result+="\"$item\""
  done
  result+="]"
  echo "$result"
}
