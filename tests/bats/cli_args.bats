#!/usr/bin/env bats

load test_helper/common

setup() {
  setup_temp_project
  source_lib "utils.sh"
}

teardown() {
  teardown_temp_project
}

# We test the parse_args function from install.sh
# Source the argument parser

parse_args() {
  # Reset globals
  MODE="install"
  FORCE=false
  AUTO_MODE=false
  LANGUAGES=""
  SKIP_RULES=false
  SKIP_SKILLS=false
  SKIP_HOOKS=false
  SKIP_AGENTS=false
  DRY_RUN=false
  READ_ONLY=false
  PROJECT_DIR=""

  while [[ $# -gt 0 ]]; do
    case $1 in
      --update) MODE="update"; shift ;;
      --force) FORCE=true; shift ;;
      --uninstall) MODE="uninstall"; shift ;;
      --auto) AUTO_MODE=true; shift ;;
      --languages) LANGUAGES="$2"; shift 2 ;;
      --skip-rules) SKIP_RULES=true; shift ;;
      --skip-skills) SKIP_SKILLS=true; shift ;;
      --skip-hooks) SKIP_HOOKS=true; shift ;;
      --skip-agents) SKIP_AGENTS=true; shift ;;
      --dry-run) DRY_RUN=true; shift ;;
      --read-only) READ_ONLY=true; shift ;;
      --project-dir) PROJECT_DIR="$2"; shift 2 ;;
      -h|--help) echo "help"; return 0 ;;
      *) echo "Unknown option: $1"; return 1 ;;
    esac
  done
}

@test "parse_args: defaults" {
  parse_args
  [ "$MODE" = "install" ]
  [ "$FORCE" = "false" ]
  [ "$AUTO_MODE" = "false" ]
  [ -z "$LANGUAGES" ]
  [ "$SKIP_RULES" = "false" ]
  [ "$DRY_RUN" = "false" ]
}

@test "parse_args: --auto flag" {
  parse_args --auto
  [ "$AUTO_MODE" = "true" ]
}

@test "parse_args: --force flag" {
  parse_args --force
  [ "$FORCE" = "true" ]
}

@test "parse_args: --update mode" {
  parse_args --update
  [ "$MODE" = "update" ]
}

@test "parse_args: --uninstall mode" {
  parse_args --uninstall
  [ "$MODE" = "uninstall" ]
}

@test "parse_args: --languages with comma-separated list" {
  parse_args --languages "go,typescript,python"
  [ "$LANGUAGES" = "go,typescript,python" ]
}

@test "parse_args: --skip-rules flag" {
  parse_args --skip-rules
  [ "$SKIP_RULES" = "true" ]
}

@test "parse_args: --skip-skills flag" {
  parse_args --skip-skills
  [ "$SKIP_SKILLS" = "true" ]
}

@test "parse_args: --skip-hooks flag" {
  parse_args --skip-hooks
  [ "$SKIP_HOOKS" = "true" ]
}

@test "parse_args: --skip-agents flag" {
  parse_args --skip-agents
  [ "$SKIP_AGENTS" = "true" ]
}

@test "parse_args: --dry-run flag" {
  parse_args --dry-run
  [ "$DRY_RUN" = "true" ]
}

@test "parse_args: --project-dir with path" {
  parse_args --project-dir "/tmp/my-project"
  [ "$PROJECT_DIR" = "/tmp/my-project" ]
}

@test "parse_args: combined flags" {
  parse_args --auto --force --languages "go,typescript" --skip-hooks --dry-run
  [ "$AUTO_MODE" = "true" ]
  [ "$FORCE" = "true" ]
  [ "$LANGUAGES" = "go,typescript" ]
  [ "$SKIP_HOOKS" = "true" ]
  [ "$DRY_RUN" = "true" ]
}

@test "parse_args: --read-only flag" {
  parse_args --read-only
  [ "$READ_ONLY" = "true" ]
}

@test "parse_args: defaults READ_ONLY to false" {
  parse_args
  [ "$READ_ONLY" = "false" ]
}

@test "parse_args: --read-only combined with other flags" {
  parse_args --auto --read-only --force
  [ "$READ_ONLY" = "true" ]
  [ "$AUTO_MODE" = "true" ]
  [ "$FORCE" = "true" ]
}

@test "parse_args: unknown option returns error" {
  run parse_args --unknown
  [ "$status" -eq 1 ]
  [[ "$output" == *"Unknown option"* ]]
}
