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
  local -A seen
  local languages=()

  for item in $stack; do
    case "$item" in
      go)
        if [ -z "${seen[golang]+x}" ]; then
          languages+=("golang")
          seen[golang]=1
        fi
        ;;
      node|react|vue|svelte)
        if [ -z "${seen[typescript]+x}" ]; then
          languages+=("typescript")
          seen[typescript]=1
        fi
        ;;
      python)
        if [ -z "${seen[python]+x}" ]; then
          languages+=("python")
          seen[python]=1
        fi
        ;;
      dotnet|unity)
        if [ -z "${seen[csharp]+x}" ]; then
          languages+=("csharp")
          seen[csharp]=1
        fi
        ;;
      solidity)
        if [ -z "${seen[solidity]+x}" ]; then
          languages+=("solidity")
          seen[solidity]=1
        fi
        ;;
      rust)
        if [ -z "${seen[rust]+x}" ]; then
          languages+=("rust")
          seen[rust]=1
        fi
        ;;
      java)
        if [ -z "${seen[java]+x}" ]; then
          languages+=("java")
          seen[java]=1
        fi
        ;;
      docker)
        if [ -z "${seen[docker]+x}" ]; then
          languages+=("docker")
          seen[docker]=1
        fi
        ;;
      # make, etc. — no language rules
    esac
  done

  echo "${languages[*]}"
}
