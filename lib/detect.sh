#!/bin/bash
# Tech stack, package manager, and command detection

# Find monorepo sub-project directories (lightweight — no JSON parsing)
# Returns newline-separated absolute paths to sub-project dirs
_find_subproject_dirs() {
  local dir="$1"

  # pnpm workspace
  if [ -f "$dir/pnpm-workspace.yaml" ]; then
    local globs
    # Match quoted entries: - 'apps/*' or - "apps/*"
    globs=$(grep -E "^[[:space:]]*-[[:space:]]*[\"']" "$dir/pnpm-workspace.yaml" 2>/dev/null | sed "s/.*[\"']\(.*\)[\"'].*/\1/")
    # Also handle unquoted entries like: - apps/*
    [ -z "$globs" ] && globs=$(grep -E '^[[:space:]]*-[[:space:]]+[^[:space:]]' "$dir/pnpm-workspace.yaml" 2>/dev/null | sed 's/^[[:space:]]*-[[:space:]]*//')
    for glob_pattern in $globs; do
      local base_dir="${glob_pattern%/\*}"
      if [ -d "$dir/$base_dir" ]; then
        for subdir in "$dir/$base_dir"/*/; do
          [ -d "$subdir" ] && echo "${subdir%/}"
        done
      fi
    done
    return
  fi

  # turbo.json
  if [ -f "$dir/turbo.json" ]; then
    for subdir in "$dir"/apps/*/ "$dir"/packages/*/; do
      [ -d "$subdir" ] && echo "${subdir%/}"
    done
    return
  fi

  # lerna.json
  if [ -f "$dir/lerna.json" ]; then
    for subdir in "$dir"/packages/*/; do
      [ -d "$subdir" ] && echo "${subdir%/}"
    done
    return
  fi
}

_detect_tech_stack_in_dir() {
  local dir="$1"
  [ -f "$dir/go.mod" ]         && echo "go"
  [ -f "$dir/package.json" ]   && echo "node"
  [ -f "$dir/Cargo.toml" ]     && echo "rust"
  { [ -f "$dir/requirements.txt" ] || [ -f "$dir/pyproject.toml" ]; } && echo "python"
  [ -f "$dir/Makefile" ]       && echo "make"
  { [ -d "$dir/unity" ] || [ -d "$dir/Assets" ]; } && echo "unity"

  local csproj_count
  csproj_count=$(find "$dir" -maxdepth 2 -name "*.csproj" 2>/dev/null | head -5 | wc -l | tr -d ' ')
  [ "$csproj_count" -gt 0 ] && echo "dotnet"

  { [ -f "$dir/hardhat.config.ts" ] || [ -f "$dir/hardhat.config.js" ]; } && echo "solidity"
  [ -f "$dir/foundry.toml" ] && echo "solidity"

  { [ -f "$dir/pom.xml" ] || [ -f "$dir/build.gradle" ] || [ -f "$dir/build.gradle.kts" ]; } && echo "java"

  { [ -f "$dir/Dockerfile" ] || [ -f "$dir/docker-compose.yml" ] || [ -f "$dir/docker-compose.yaml" ]; } && echo "docker"

  if [ -f "$dir/package.json" ]; then
    if grep -q '"react"' "$dir/package.json" 2>/dev/null; then
      echo "react"
    elif grep -q '"vue"' "$dir/package.json" 2>/dev/null; then
      echo "vue"
    elif grep -q '"svelte"' "$dir/package.json" 2>/dev/null; then
      echo "svelte"
    fi
  fi
}

detect_tech_stack() {
  local dir="$1"
  local seen=""
  local stack=()

  # Scan root
  while IFS= read -r item; do
    [ -z "$item" ] && continue
    if [[ " $seen " != *" $item "* ]]; then
      stack+=("$item")
      seen="$seen $item"
    fi
  done < <(_detect_tech_stack_in_dir "$dir")

  # Scan monorepo sub-projects
  while IFS= read -r subdir; do
    [ -z "$subdir" ] && continue
    while IFS= read -r item; do
      [ -z "$item" ] && continue
      if [[ " $seen " != *" $item "* ]]; then
        stack+=("$item")
        seen="$seen $item"
      fi
    done < <(_detect_tech_stack_in_dir "$subdir")
  done < <(_find_subproject_dirs "$dir")

  echo "${stack[*]}"
}

detect_package_manager() {
  local dir="$1"
  [ ! -f "$dir/package.json" ] && return

  if [ -f "$dir/pnpm-lock.yaml" ]; then
    echo "pnpm"
  elif [ -f "$dir/yarn.lock" ]; then
    echo "yarn"
  elif [ -f "$dir/bun.lockb" ]; then
    echo "bun"
  else
    echo "npm"
  fi
}

detect_test_command() {
  local dir="$1" stack="$2"
  # Check Makefile first
  if [ -f "$dir/Makefile" ]; then
    local test_target
    test_target=$(grep -E '^test[a-zA-Z_-]*:' "$dir/Makefile" 2>/dev/null | head -1 | cut -d: -f1)
    if [ -n "$test_target" ]; then
      echo "make $test_target"
      return
    fi
  fi
  # Java build tools
  if [[ "$stack" == *java* ]]; then
    if [ -f "$dir/pom.xml" ]; then
      echo "mvn test"
      return
    elif [ -f "$dir/build.gradle" ] || [ -f "$dir/build.gradle.kts" ]; then
      echo "./gradlew test"
      return
    fi
  fi
  # Fallback by stack
  case "$stack" in
    *go*)       echo "go test ./..." ;;
    *node*)     echo "npm test" ;;
    *dotnet*)   echo "dotnet test" ;;
    *python*)   echo "pytest" ;;
    *rust*)     echo "cargo test" ;;
    *solidity*) echo "npx hardhat test" ;;
    *)          echo "" ;;
  esac
}

detect_lint_command() {
  local dir="$1" stack="$2"
  if [ -f "$dir/Makefile" ]; then
    local lint_target
    lint_target=$(grep -E '^lint[a-zA-Z_-]*:' "$dir/Makefile" 2>/dev/null | head -1 | cut -d: -f1)
    if [ -n "$lint_target" ]; then
      echo "make $lint_target"
      return
    fi
  fi
  case "$stack" in
    *go*)      echo "golangci-lint run" ;;
    *node*)    echo "npm run lint" ;;
    *dotnet*)  echo "dotnet format --verify-no-changes" ;;
    *python*)  echo "ruff check ." ;;
    *rust*)    echo "cargo clippy" ;;
    *java*)    echo "mvn checkstyle:check" ;;
    *)         echo "" ;;
  esac
}

detect_scan_categories() {
  local stack="$1"
  local categories=("tests" "lint" "missing-tests" "todo-audit")

  case "$stack" in
    *go*|*node*|*dotnet*|*python*|*rust*|*java*)
      categories+=("module-boundaries" "security-scan")
      ;;
  esac

  if [[ "$stack" == *react* ]] || [[ "$stack" == *vue* ]] || [[ "$stack" == *svelte* ]]; then
    categories+=("accessibility" "component-quality")
  fi

  if [[ "$stack" == *solidity* ]]; then
    categories+=("smart-contract-security" "gas-optimization")
  fi

  if [[ "$stack" == *node* ]] || [[ "$stack" == *react* ]]; then
    categories+=("browser-testing")
  fi

  echo "${categories[*]}"
}

# Map detected tech stack items to language rule directories
map_stack_to_languages() {
  local stack="$1"
  local seen=""
  local languages=()

  _has_lang() { [[ " $seen " == *" $1 "* ]]; }
  _add_lang() { languages+=("$1"); seen="$seen $1"; }

  for item in $stack; do
    case "$item" in
      go)         _has_lang golang     || _add_lang golang ;;
      node|react|vue|svelte) _has_lang typescript || _add_lang typescript ;;
      python)     _has_lang python     || _add_lang python ;;
      dotnet|unity) _has_lang csharp   || _add_lang csharp ;;
      solidity)   _has_lang solidity   || _add_lang solidity ;;
      rust)       _has_lang rust       || _add_lang rust ;;
      java)       _has_lang java       || _add_lang java ;;
      docker)     _has_lang docker     || _add_lang docker ;;
      # make, etc. — no language rules
    esac
  done

  echo "${languages[*]}"
}

# Map detected tech stack items to agent domain directories
map_stack_to_agent_domains() {
  local stack="$1"
  local seen=""
  local domains=()

  _has_domain() { [[ " $seen " == *" $1 "* ]]; }
  _add_domain() { domains+=("$1"); seen="$seen $1"; }

  for item in $stack; do
    case "$item" in
      go)         _has_domain golang     || _add_domain golang ;;
      node|react|vue|svelte) _has_domain react || _add_domain react ;;
      dotnet)     _has_domain dotnet     || _add_domain dotnet ;;
      unity)      _has_domain unity      || _add_domain unity
                  _has_domain dotnet     || _add_domain dotnet ;;
      solidity)   _has_domain blockchain || _add_domain blockchain ;;
      docker)     _has_domain docker     || _add_domain docker ;;
      # make, java, rust, python — no domain agents yet
    esac
  done

  echo "${domains[*]}"
}

# Scan a single directory for deep domain signals
# Outputs detected domain names (one per line)
_detect_deep_domains_in_dir() {
  local dir="$1"

  # GraphQL: .graphql files OR graphql in deps
  if find "$dir" -maxdepth 3 -name "*.graphql" 2>/dev/null | head -1 | grep -q .; then
    echo "graphql"
  elif [ -f "$dir/package.json" ] && grep -q '"graphql"' "$dir/package.json" 2>/dev/null; then
    echo "graphql"
  elif [ -f "$dir/go.mod" ] && grep -q 'gqlgen\|graphql' "$dir/go.mod" 2>/dev/null; then
    echo "graphql"
  fi

  # AI/LLM: openai, anthropic, langchain in deps
  if [ -f "$dir/package.json" ] && grep -qE '"(openai|anthropic|langchain|@langchain)' "$dir/package.json" 2>/dev/null; then
    echo "ai"
  elif [ -f "$dir/go.mod" ] && grep -qE 'openai|anthropic|langchain' "$dir/go.mod" 2>/dev/null; then
    echo "ai"
  elif [ -f "$dir/requirements.txt" ] && grep -qEi 'openai|anthropic|langchain' "$dir/requirements.txt" 2>/dev/null; then
    echo "ai"
  elif [ -f "$dir/pyproject.toml" ] && grep -qEi 'openai|anthropic|langchain' "$dir/pyproject.toml" 2>/dev/null; then
    echo "ai"
  fi

  # SaaS/Payments: stripe, paypal in deps
  if [ -f "$dir/package.json" ] && grep -qE '"(stripe|paypal)' "$dir/package.json" 2>/dev/null; then
    echo "saas"
  elif [ -f "$dir/go.mod" ] && grep -qE 'stripe|paypal' "$dir/go.mod" 2>/dev/null; then
    echo "saas"
  elif [ -f "$dir/requirements.txt" ] && grep -qEi 'stripe|paypal' "$dir/requirements.txt" 2>/dev/null; then
    echo "saas"
  fi

  # Database: GORM, prisma, migrations dir, schema files
  if [ -f "$dir/go.mod" ] && grep -q 'gorm' "$dir/go.mod" 2>/dev/null; then
    echo "database"
  elif find "$dir" -maxdepth 2 -name "*.prisma" 2>/dev/null | head -1 | grep -q .; then
    echo "database"
  elif [ -d "$dir/migrations" ] || [ -d "$dir/db/migrations" ]; then
    echo "database"
  elif [ -f "$dir/package.json" ] && grep -qE '"(prisma|@prisma|typeorm|knex|drizzle)' "$dir/package.json" 2>/dev/null; then
    echo "database"
  fi

  # Kubernetes: k8s manifests, helm charts
  if [ -f "$dir/helmfile.yaml" ] || find "$dir" -maxdepth 3 -name "Chart.yaml" 2>/dev/null | head -1 | grep -q .; then
    echo "kubernetes"
  elif find "$dir" -maxdepth 3 -name "*.yaml" -exec grep -l 'kind:\s*Deployment\|kind:\s*Service\|kind:\s*StatefulSet' {} \; 2>/dev/null | head -1 | grep -q .; then
    echo "kubernetes"
  fi

  # Observability: prometheus, grafana, datadog configs
  if [ -f "$dir/prometheus.yml" ] || [ -f "$dir/prometheus.yaml" ]; then
    echo "observability"
  elif [ -d "$dir/grafana" ]; then
    echo "observability"
  elif [ -f "$dir/datadog.yaml" ] || [ -f "$dir/datadog.yml" ]; then
    echo "observability"
  elif [ -f "$dir/package.json" ] && grep -qE '"(prom-client|dd-trace|@opentelemetry)' "$dir/package.json" 2>/dev/null; then
    echo "observability"
  fi
}

# Deep domain detection — scans files for domain-specific signals
# Scans root + monorepo sub-projects. Returns space-separated list of domain directory names.
detect_deep_domains() {
  local dir="$1"
  local seen=""
  local domains=()

  # Scan root directory
  while IFS= read -r domain; do
    [ -z "$domain" ] && continue
    if [[ " $seen " != *" $domain "* ]]; then
      domains+=("$domain")
      seen="$seen $domain"
    fi
  done < <(_detect_deep_domains_in_dir "$dir")

  # Scan monorepo sub-projects
  while IFS= read -r subdir; do
    [ -z "$subdir" ] && continue
    while IFS= read -r domain; do
      [ -z "$domain" ] && continue
      if [[ " $seen " != *" $domain "* ]]; then
        domains+=("$domain")
        seen="$seen $domain"
      fi
    done < <(_detect_deep_domains_in_dir "$subdir")
  done < <(_find_subproject_dirs "$dir")

  echo "${domains[*]}"
}

# Detect project structure (monorepo vs single)
# Returns JSON: { "type": "monorepo"|"single", "projects": [...] }
detect_project_structure() {
  local dir="$1"
  local projects=()

  # Check pnpm workspace
  if [ -f "$dir/pnpm-workspace.yaml" ]; then
    # Parse workspace globs and resolve to actual directories
    local globs
    globs=$(grep -E "^[[:space:]]*-[[:space:]]*[\"']" "$dir/pnpm-workspace.yaml" 2>/dev/null | sed "s/.*[\"']\(.*\)[\"'].*/\1/")
    [ -z "$globs" ] && globs=$(grep -E '^[[:space:]]*-[[:space:]]+[^[:space:]]' "$dir/pnpm-workspace.yaml" 2>/dev/null | sed 's/^[[:space:]]*-[[:space:]]*//')
    for glob_pattern in $globs; do
      # Replace trailing /* with actual dirs
      local base_dir="${glob_pattern%/\*}"
      if [ -d "$dir/$base_dir" ]; then
        for subdir in "$dir/$base_dir"/*/; do
          [ -d "$subdir" ] && projects+=("${subdir#$dir/}")
        done
      fi
    done
  fi

  # Check turbo.json
  if [ -f "$dir/turbo.json" ] && [ ${#projects[@]} -eq 0 ]; then
    # Turbo monorepo — scan for apps/ and packages/
    for subdir in "$dir"/apps/*/ "$dir"/packages/*/; do
      [ -d "$subdir" ] && projects+=("${subdir#$dir/}")
    done
  fi

  # Check lerna.json
  if [ -f "$dir/lerna.json" ] && [ ${#projects[@]} -eq 0 ]; then
    for subdir in "$dir"/packages/*/; do
      [ -d "$subdir" ] && projects+=("${subdir#$dir/}")
    done
  fi

  # Check for multiple .csproj files
  if [ ${#projects[@]} -eq 0 ]; then
    local csproj_dirs=()
    while IFS= read -r csproj; do
      [ -n "$csproj" ] && csproj_dirs+=("$(dirname "${csproj#$dir/}")")
    done < <(find "$dir" -maxdepth 3 -name "*.csproj" 2>/dev/null)
    if [ ${#csproj_dirs[@]} -ge 2 ]; then
      projects=("${csproj_dirs[@]}")
    fi
  fi

  # Check docker-compose services with build contexts
  if [ ${#projects[@]} -eq 0 ]; then
    if [ -f "$dir/docker-compose.yml" ] || [ -f "$dir/docker-compose.yaml" ]; then
      local compose_file="$dir/docker-compose.yml"
      [ ! -f "$compose_file" ] && compose_file="$dir/docker-compose.yaml"
      local build_dirs
      build_dirs=$(grep -E '^\s+build:\s+\.' "$compose_file" 2>/dev/null | sed 's/.*build:\s*//' | tr -d ' ')
      for build_dir in $build_dirs; do
        [ -d "$dir/$build_dir" ] && projects+=("${build_dir#./}")
      done
      if [ ${#projects[@]} -ge 2 ]; then
        : # keep the projects
      else
        projects=()
      fi
    fi
  fi

  # Check for apps/ and packages/ pattern (generic monorepo)
  if [ ${#projects[@]} -eq 0 ]; then
    if [ -d "$dir/apps" ] || [ -d "$dir/packages" ]; then
      local app_count=0
      for subdir in "$dir"/apps/*/ "$dir"/packages/*/; do
        [ -d "$subdir" ] && { projects+=("${subdir#$dir/}"); app_count=$((app_count + 1)); }
      done
      if [ $app_count -lt 2 ]; then
        projects=()
      fi
    fi
  fi

  # Remove trailing slashes from project paths
  local clean_projects=()
  for p in "${projects[@]}"; do
    clean_projects+=("${p%/}")
  done

  # Build JSON output
  if [ ${#clean_projects[@]} -ge 2 ]; then
    local json_projects
    json_projects=$(printf '"%s",' "${clean_projects[@]}")
    json_projects="[${json_projects%,}]"
    echo "{\"type\":\"monorepo\",\"projects\":$json_projects}"
  else
    echo '{"type":"single","projects":[]}'
  fi
}
