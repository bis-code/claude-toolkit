#!/bin/bash
set -e

# Dry-run install against a real project to validate detection
# Usage: ./tests/validation/validate_project.sh <project-dir>

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TOOLKIT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

if [ -z "$1" ]; then
  echo "Usage: $0 <project-dir>"
  echo "Runs a dry-run install against a real project directory."
  exit 1
fi

PROJECT_DIR="$1"
if [ ! -d "$PROJECT_DIR" ]; then
  echo "Error: $PROJECT_DIR is not a directory"
  exit 1
fi

echo "Dry-run install for: $PROJECT_DIR"
echo ""

"$TOOLKIT_ROOT/install.sh" --auto --dry-run --project-dir "$PROJECT_DIR"
