#!/bin/bash
# Commands installation logic

# Install slash commands to global ~/.claude/commands/
# Usage: install_commands <toolkit_dir>
install_commands() {
  local toolkit_dir="$1"
  local commands_src="$toolkit_dir/commands"
  local commands_dest="$HOME/.claude/commands"

  mkdir -p "$commands_dest"

  for cmd_file in "$commands_src"/*.md; do
    [ -f "$cmd_file" ] || continue
    local filename
    filename="$(basename "$cmd_file")"
    if [ ! -f "$commands_dest/$filename" ] || [ "${FORCE:-false}" = true ] || [ "${MODE:-install}" = "update" ]; then
      cp "$cmd_file" "$commands_dest/$filename"
      info "/${filename%.md} command installed"
    else
      info "/${filename%.md} command (already exists)"
    fi
  done
}
