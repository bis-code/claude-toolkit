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
  _copy_rules_dir "$rules_src/common" "$rules_dest/common" ".claude/rules/common"

  # Install language-specific rules
  for lang in $languages; do
    if [ -d "$rules_src/$lang" ]; then
      _copy_rules_dir "$rules_src/$lang" "$rules_dest/$lang" ".claude/rules/$lang"
    else
      warn "No rules found for language: $lang"
    fi
  done
}

# Copy a rules directory using _tracked_copy when available, else inline logic
_copy_rules_dir() {
  local src="$1" dest="$2" rel_prefix="$3"
  mkdir -p "$dest"
  for file in "$src"/*; do
    [ -f "$file" ] || continue
    local filename
    filename="$(basename "$file")"
    if declare -F _tracked_copy &>/dev/null; then
      _tracked_copy "$file" "$dest/$filename" "$rel_prefix/$filename"
    else
      if [ ! -f "$dest/$filename" ] || [ "${FORCE:-false}" = true ] || [ "${MODE:-install}" = "update" ]; then
        cp "$file" "$dest/$filename"
      fi
    fi
  done
}
