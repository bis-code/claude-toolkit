package workspace

import (
	"os"
	"path/filepath"
	"strings"
)

// Detect scans a directory for git repositories and identifies their tech stacks.
// It returns a workspace Config based on what it finds.
func Detect(parentDir string) (*Config, error) {
	entries, err := os.ReadDir(parentDir)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		Name:  filepath.Base(parentDir),
		Repos: []Repo{},
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
