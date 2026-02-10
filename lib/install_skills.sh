#!/bin/bash
# Skills installation logic

# Install skills into project
# Usage: install_skills <project_dir> <templates_dir>
install_skills() {
  local project_dir="$1"
  local templates_dir="$2"
  local skills_src="$templates_dir/skills"
  local skills_dest="$project_dir/.claude/skills"

  if [ ! -d "$skills_src" ]; then
    warn "No skills templates found"
    return
  fi

  for skill_dir in "$skills_src"/*/; do
    [ -d "$skill_dir" ] || continue
    local skill_name
    skill_name="$(basename "$skill_dir")"
    local dest_dir="$skills_dest/$skill_name"
    mkdir -p "$dest_dir"

    for file in "$skill_dir"*; do
      [ -f "$file" ] || continue
      local filename
      filename="$(basename "$file")"
      if [ ! -f "$dest_dir/$filename" ] || [ "${FORCE:-false}" = true ]; then
        cp "$file" "$dest_dir/$filename"
      fi
    done
  done
}
