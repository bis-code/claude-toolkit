#!/bin/bash
# Commands installation logic

# Commands that were shipped in previous toolkit versions but are now skills or removed.
# Only these are cleaned up — user-created commands in ~/.claude/commands/ are never touched.
DEPRECATED_COMMANDS=(
  "ralph.md"
  "qa.md"
)

# Install slash commands to global ~/.claude/commands/
# Usage: install_commands <toolkit_dir>
install_commands() {
  local toolkit_dir="$1"
  local commands_src="$toolkit_dir/commands"
  local commands_dest="$HOME/.claude/commands"

  mkdir -p "$commands_dest"

  # Install current commands
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

  # Remove known deprecated commands (replaced by skills)
  for fname in "${DEPRECATED_COMMANDS[@]}"; do
    if [ -f "$commands_dest/$fname" ]; then
      rm -f "$commands_dest/$fname"
      warn "Removed deprecated command: /${fname%.md} (now a skill)"
    fi
  done
}
