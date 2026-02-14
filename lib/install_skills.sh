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

  # Build list of template skill names for cleanup
  local template_skills=()
  for skill_dir in "$skills_src"/*/; do
    [ -d "$skill_dir" ] || continue
    template_skills+=("$(basename "$skill_dir")")
  done

  # Install skills from templates
  for skill_name in "${template_skills[@]}"; do
    local src_dir="$skills_src/$skill_name"
    local dest_dir="$skills_dest/$skill_name"
    mkdir -p "$dest_dir"

    for file in "$src_dir"/*; do
      [ -f "$file" ] || continue
      local filename
      filename="$(basename "$file")"
      if declare -F _tracked_copy &>/dev/null; then
        _tracked_copy "$file" "$dest_dir/$filename" ".claude/skills/$skill_name/$filename"
      else
        if [ ! -f "$dest_dir/$filename" ] || [ "${FORCE:-false}" = true ] || [ "${MODE:-install}" = "update" ]; then
          cp "$file" "$dest_dir/$filename"
        fi
      fi
    done
  done

  # In update mode, remove skill directories that no longer have templates
  if [ "${MODE:-install}" = "update" ] && [ -d "$skills_dest" ]; then
    for installed_dir in "$skills_dest"/*/; do
      [ -d "$installed_dir" ] || continue
      local installed_name
      installed_name="$(basename "$installed_dir")"
      local found=false
      for tpl in "${template_skills[@]}"; do
        if [ "$tpl" = "$installed_name" ]; then
          found=true
          break
        fi
      done
      if [ "$found" = false ]; then
        warn "Removing deprecated skill: $installed_name"
        rm -rf "$installed_dir"
      fi
    done
  fi
}
