#!/bin/bash
# Common test helpers for bats tests

TOOLKIT_ROOT="$(cd "$(dirname "$BATS_TEST_FILENAME")/../.." && pwd)"
INSTALL_SCRIPT="$TOOLKIT_ROOT/install.sh"

# Create a temporary project directory for testing
setup_temp_project() {
  TEST_PROJECT_DIR="$(mktemp -d)"
  export TEST_PROJECT_DIR
}

# Clean up temporary project directory
teardown_temp_project() {
  [ -n "$TEST_PROJECT_DIR" ] && rm -rf "$TEST_PROJECT_DIR"
}

# Source only the lib/ modules (not the full install.sh which runs main logic)
source_lib() {
  local module="$1"
  source "$TOOLKIT_ROOT/lib/$module"
}

# Create a minimal package.json in test project
create_package_json() {
  local dir="${1:-$TEST_PROJECT_DIR}"
  local deps="${2:-{}}"
  cat > "$dir/package.json" <<EOF
{
  "name": "test-project",
  "dependencies": $deps
}
EOF
}

# Create a minimal go.mod in test project
create_go_mod() {
  local dir="${1:-$TEST_PROJECT_DIR}"
  cat > "$dir/go.mod" <<EOF
module test-project

go 1.21
EOF
}

# Create a minimal Makefile with optional targets
create_makefile() {
  local dir="${1:-$TEST_PROJECT_DIR}"
  shift
  local targets=("$@")
  for target in "${targets[@]}"; do
    echo -e "${target}:\n\t@echo running $target" >> "$dir/Makefile"
  done
}

# Assert file exists
assert_file_exists() {
  [ -f "$1" ] || {
    echo "Expected file to exist: $1" >&2
    return 1
  }
}

# Assert file contains string
assert_file_contains() {
  grep -q "$2" "$1" 2>/dev/null || {
    echo "Expected '$1' to contain '$2'" >&2
    return 1
  }
}

# Assert directory exists
assert_dir_exists() {
  [ -d "$1" ] || {
    echo "Expected directory to exist: $1" >&2
    return 1
  }
}
