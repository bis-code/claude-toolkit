package workspace

import (
	"os"
	"path/filepath"
	"strings"
)

// Detect scans a directory for git repositories and identifies their tech stacks.
// If the directory itself is a git repo, it detects it as a single-repo workspace
// (with potential monorepo sub-projects). Otherwise, it scans child directories
// for git repos (multi-repo workspace mode).
func Detect(parentDir string) (*Config, error) {
	cfg := &Config{
		Name:  filepath.Base(parentDir),
		Repos: []Repo{},
	}

	// Check if parentDir itself is a git repo.
	if fileExists(filepath.Join(parentDir, ".git")) {
		repo := Repo{
			Path:   ".",
			Type:   detectTechStack(parentDir),
			Branch: detectBranch(parentDir),
		}
		monorepoType, subProjects := DetectMonorepo(parentDir)
		if monorepoType != "" {
			repo.MonorepoType = monorepoType
			repo.SubProjects = subProjects
		}
		cfg.Repos = append(cfg.Repos, repo)
		return cfg, nil
	}

	entries, err := os.ReadDir(parentDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		fullPath := filepath.Join(parentDir, name)

		// Check if it is a git repository.
		gitDir := filepath.Join(fullPath, ".git")
		info, err := os.Stat(gitDir)
		if err != nil || !info.IsDir() {
			// Not a git repo — treat as shared directory.
			cfg.Shared = append(cfg.Shared, name)
			continue
		}

		repo := Repo{
			Path:   name,
			Type:   detectTechStack(fullPath),
			Branch: detectBranch(fullPath),
		}

		monorepoType, subProjects := DetectMonorepo(fullPath)
		if monorepoType != "" {
			repo.MonorepoType = monorepoType
			repo.SubProjects = subProjects
		}

		cfg.Repos = append(cfg.Repos, repo)
	}

	return cfg, nil
}

// detectTechStack checks marker files in priority order to determine the tech stack.
func detectTechStack(repoPath string) string {
	type marker struct {
		file string
		tech string
	}

	simpleMarkers := []marker{
		{"go.mod", "go"},
		{"Cargo.toml", "rust"},
		{"pyproject.toml", "python"},
		{"requirements.txt", "python"},
		{"setup.py", "python"},
		{"Assets/Scenes", "unity"},
	}

	for _, m := range simpleMarkers {
		if fileExists(filepath.Join(repoPath, m.file)) {
			return m.tech
		}
	}

	// package.json: distinguish TypeScript from JavaScript by presence of tsconfig.json.
	if fileExists(filepath.Join(repoPath, "package.json")) {
		if fileExists(filepath.Join(repoPath, "tsconfig.json")) {
			return "typescript"
		}
		return "typescript" // default to typescript even without tsconfig per spec comment
	}

	// Glob-based detectors for C# (.csproj, .sln).
	if matchesGlob(repoPath, "*.csproj") || matchesGlob(repoPath, "*.sln") {
		return "csharp"
	}

	// Unreal Engine.
	if matchesGlob(repoPath, "*.uproject") {
		return "unreal"
	}

	return ""
}

// detectBranch reads the current branch from .git/HEAD.
// Returns "main" for detached HEAD or unreadable HEAD files.
func detectBranch(repoPath string) string {
	headPath := filepath.Join(repoPath, ".git", "HEAD")
	data, err := os.ReadFile(headPath)
	if err != nil {
		return "main"
	}

	content := strings.TrimSpace(string(data))

	// Symbolic ref: "ref: refs/heads/<branch>"
	if strings.HasPrefix(content, "ref: refs/heads/") {
		return strings.TrimPrefix(content, "ref: refs/heads/")
	}

	// Detached HEAD: content is a commit hash (40 hex chars).
	// Default to "main".
	return "main"
}

// fileExists returns true if path exists (file or directory).
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// matchesGlob returns true if any file in repoPath matches the glob pattern.
func matchesGlob(repoPath, pattern string) bool {
	matches, err := filepath.Glob(filepath.Join(repoPath, pattern))
	return err == nil && len(matches) > 0
}

// skipDirs is the set of directory names to ignore when scanning for mixed-language sub-projects.
var skipDirs = map[string]bool{
	"node_modules":  true,
	"vendor":        true,
	".git":          true,
	"dist":          true,
	"build":         true,
	"coverage":      true,
	".next":         true,
	"__pycache__":   true,
	".pytest_cache": true,
	"docs":          true,
	"scripts":       true,
	".github":       true,
	".vscode":       true,
	".idea":         true,
	"tmp":           true,
	"log":           true,
	"logs":          true,
	".turbo":        true,
	".nx":           true,
}

// DetectMonorepo checks whether a repository is a monorepo and returns its type
// and sub-projects. Returns ("", nil) if not a monorepo.
func DetectMonorepo(repoPath string) (string, []SubProject) {
	// Try Turborepo first.
	if fileExists(filepath.Join(repoPath, "turbo.json")) {
		subs := detectTurborepo(repoPath)
		if len(subs) > 0 {
			return "turborepo", subs
		}
		return "", nil
	}

	// Fall back to mixed-language detection.
	subs := detectMixedLanguage(repoPath)
	if len(subs) >= 2 {
		return "mixed", subs
	}

	return "", nil
}

// detectTurborepo scans apps/ and packages/ subdirectories for tech stacks.
func detectTurborepo(repoPath string) []SubProject {
	var subs []SubProject

	for _, parent := range []string{"apps", "packages"} {
		parentPath := filepath.Join(repoPath, parent)
		entries, err := os.ReadDir(parentPath)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			subPath := filepath.Join(parentPath, entry.Name())
			techType := detectTechStack(subPath)
			if techType == "" {
				continue
			}

			subs = append(subs, SubProject{
				Path:        filepath.Join(parent, entry.Name()),
				Type:        techType,
				TestCommand: DetectTestCommand(techType),
			})
		}
	}

	return subs
}

// detectMixedLanguage scans top-level subdirectories for different tech stacks.
// Returns sub-projects only if 2+ subdirectories have detected tech stacks.
func detectMixedLanguage(repoPath string) []SubProject {
	entries, err := os.ReadDir(repoPath)
	if err != nil {
		return nil
	}

	var subs []SubProject

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		if skipDirs[name] {
			continue
		}

		subPath := filepath.Join(repoPath, name)
		techType := detectTechStack(subPath)
		if techType == "" {
			continue
		}

		subs = append(subs, SubProject{
			Path:        name,
			Type:        techType,
			TestCommand: DetectTestCommand(techType),
		})
	}

	return subs
}

// DetectTestCommand returns the conventional test command for a given tech type.
func DetectTestCommand(techType string) string {
	switch techType {
	case "go":
		return "go test ./..."
	case "typescript":
		return "npm test"
	case "python":
		return "pytest"
	case "rust":
		return "cargo test"
	default:
		return ""
	}
}
