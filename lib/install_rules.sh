#!/bin/bash
# Rules installation logic

# Install rules for given languages into project
# Usage: install_rules <project_dir> <languages> <templates_dir>
# languages: space-separated list like "golang typescript"
install_rules() {
  local project_dir="$1"
  local languages="$2"
  local templates_dir="$3"
  local rules_src="$templates_dir/rules"
  local rules_dest="$project_dir/.claude/rules"

  # Always install common rules
  _copy_rules_dir "$rules_src/common" "$rules_dest/common"

  # Install language-specific rules
  for lang in $languages; do
    if [ -d "$rules_src/$lang" ]; then
      _copy_rules_dir "$rules_src/$lang" "$rules_dest/$lang"
    else
      warn "No rules found for language: $lang"
    fi
  done
}

# Copy a rules directory, respecting --force flag
_copy_rules_dir() {
  local src="$1" dest="$2"
  mkdir -p "$dest"
  for file in "$src"/*; do
    [ -f "$file" ] || continue
    local filename
    filename="$(basename "$file")"
    if [ ! -f "$dest/$filename" ] || [ "${FORCE:-false}" = true ]; then
      cp "$file" "$dest/$filename"
    fi
  done
}
