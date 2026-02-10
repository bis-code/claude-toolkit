#!/bin/bash
# Tech stack, package manager, and command detection

detect_tech_stack() {
  local dir="$1"
  local stack=()
  [ -f "$dir/go.mod" ]         && stack+=("go")
  [ -f "$dir/package.json" ]   && stack+=("node")
  [ -f "$dir/Cargo.toml" ]     && stack+=("rust")
  [ -f "$dir/requirements.txt" ] || [ -f "$dir/pyproject.toml" ] && stack+=("python")
  [ -f "$dir/Makefile" ]       && stack+=("make")
  [ -d "$dir/unity" ] || [ -d "$dir/Assets" ] && stack+=("unity")

  # Check for .csproj/.sln files
  local csproj_count
  csproj_count=$(find "$dir" -maxdepth 2 -name "*.csproj" 2>/dev/null | head -5 | wc -l | tr -d ' ')
  [ "$csproj_count" -gt 0 ] && stack+=("dotnet")

  # Check for Solidity/Hardhat/Foundry
  [ -f "$dir/hardhat.config.ts" ] || [ -f "$dir/hardhat.config.js" ] && stack+=("solidity")
  [ -f "$dir/foundry.toml" ] && stack+=("solidity")

  # Check for Java
  [ -f "$dir/pom.xml" ] || [ -f "$dir/build.gradle" ] || [ -f "$dir/build.gradle.kts" ] && stack+=("java")

  # Check for Docker
  [ -f "$dir/Dockerfile" ] || [ -f "$dir/docker-compose.yml" ] || [ -f "$dir/docker-compose.yaml" ] && stack+=("docker")

  # Check for frontend frameworks
  if [ -f "$dir/package.json" ]; then
    if grep -q '"react"' "$dir/package.json" 2>/dev/null; then
      stack+=("react")
    elif grep -q '"vue"' "$dir/package.json" 2>/dev/null; then
      stack+=("vue")
    elif grep -q '"svelte"' "$dir/package.json" 2>/dev/null; then
      stack+=("svelte")
    fi
  fi

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

# Deep domain detection — scans files for domain-specific signals
# Returns space-separated list of domain directory names
detect_deep_domains() {
  local dir="$1"
  local seen=""
  local domains=()

  _has_dd() { [[ " $seen " == *" $1 "* ]]; }
  _add_dd() { domains+=("$1"); seen="$seen $1"; }

  # GraphQL: .graphql files OR graphql in deps
  if find "$dir" -maxdepth 3 -name "*.graphql" 2>/dev/null | head -1 | grep -q .; then
    _add_dd graphql
  elif [ -f "$dir/package.json" ] && grep -q '"graphql"' "$dir/package.json" 2>/dev/null; then
    _add_dd graphql
  elif [ -f "$dir/go.mod" ] && grep -q 'gqlgen\|graphql' "$dir/go.mod" 2>/dev/null; then
    _add_dd graphql
  fi

  # AI/LLM: openai, anthropic, langchain in deps
  local ai_detected=false
  if [ -f "$dir/package.json" ] && grep -qE '"(openai|anthropic|langchain|@langchain)' "$dir/package.json" 2>/dev/null; then
    ai_detected=true
  elif [ -f "$dir/go.mod" ] && grep -qE 'openai|anthropic|langchain' "$dir/go.mod" 2>/dev/null; then
    ai_detected=true
  elif [ -f "$dir/requirements.txt" ] && grep -qEi 'openai|anthropic|langchain' "$dir/requirements.txt" 2>/dev/null; then
    ai_detected=true
  elif [ -f "$dir/pyproject.toml" ] && grep -qEi 'openai|anthropic|langchain' "$dir/pyproject.toml" 2>/dev/null; then
    ai_detected=true
  fi
  [ "$ai_detected" = true ] && _add_dd ai

  # SaaS/Payments: stripe, paypal in deps
  local saas_detected=false
  if [ -f "$dir/package.json" ] && grep -qE '"(stripe|paypal)' "$dir/package.json" 2>/dev/null; then
    saas_detected=true
  elif [ -f "$dir/go.mod" ] && grep -qE 'stripe|paypal' "$dir/go.mod" 2>/dev/null; then
    saas_detected=true
  elif [ -f "$dir/requirements.txt" ] && grep -qEi 'stripe|paypal' "$dir/requirements.txt" 2>/dev/null; then
    saas_detected=true
  fi
  [ "$saas_detected" = true ] && _add_dd saas

  # Database: GORM, prisma, migrations dir, schema files
  local db_detected=false
  if [ -f "$dir/go.mod" ] && grep -q 'gorm' "$dir/go.mod" 2>/dev/null; then
    db_detected=true
  elif find "$dir" -maxdepth 2 -name "*.prisma" 2>/dev/null | head -1 | grep -q .; then
    db_detected=true
  elif [ -d "$dir/migrations" ] || [ -d "$dir/db/migrations" ]; then
    db_detected=true
  elif [ -f "$dir/package.json" ] && grep -qE '"(prisma|@prisma|typeorm|knex|drizzle)' "$dir/package.json" 2>/dev/null; then
    db_detected=true
  fi
  [ "$db_detected" = true ] && _add_dd database

  # Kubernetes: k8s manifests, helm charts
  local k8s_detected=false
  if [ -f "$dir/helmfile.yaml" ] || find "$dir" -maxdepth 3 -name "Chart.yaml" 2>/dev/null | head -1 | grep -q .; then
    k8s_detected=true
  elif find "$dir" -maxdepth 3 -name "*.yaml" -exec grep -l 'kind:\s*Deployment\|kind:\s*Service\|kind:\s*StatefulSet' {} \; 2>/dev/null | head -1 | grep -q .; then
    k8s_detected=true
  fi
  [ "$k8s_detected" = true ] && _add_dd kubernetes

  # Observability: prometheus, grafana, datadog configs
  local obs_detected=false
  if [ -f "$dir/prometheus.yml" ] || [ -f "$dir/prometheus.yaml" ]; then
    obs_detected=true
  elif [ -d "$dir/grafana" ]; then
    obs_detected=true
  elif [ -f "$dir/datadog.yaml" ] || [ -f "$dir/datadog.yml" ]; then
    obs_detected=true
  elif [ -f "$dir/package.json" ] && grep -qE '"(prom-client|dd-trace|@opentelemetry)' "$dir/package.json" 2>/dev/null; then
    obs_detected=true
  fi
  [ "$obs_detected" = true ] && _add_dd observability

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
    globs=$(grep -E "^\s*-\s*'" "$dir/pnpm-workspace.yaml" 2>/dev/null | sed "s/.*'\(.*\)'.*/\1/")
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
