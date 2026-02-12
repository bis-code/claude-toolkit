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
