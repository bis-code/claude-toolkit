package workspace_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/bis-code/claude-toolkit/server/internal/workspace"
)

// makeTestRepo creates a fake git repository under parentDir/name with optional marker files.
func makeTestRepo(t *testing.T, dir, name string, markers ...string) string {
	t.Helper()
	repoDir := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Join(repoDir, ".git"), 0o755); err != nil {
		t.Fatalf("makeTestRepo: MkdirAll .git: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, ".git", "HEAD"), []byte("ref: refs/heads/main\n"), 0o644); err != nil {
		t.Fatalf("makeTestRepo: write HEAD: %v", err)
	}
	for _, m := range markers {
		if err := os.WriteFile(filepath.Join(repoDir, m), []byte(""), 0o644); err != nil {
			t.Fatalf("makeTestRepo: write marker %s: %v", m, err)
		}
	}
	return repoDir
}

// makeSharedDir creates a plain directory (no .git) under parentDir/name.
func makeSharedDir(t *testing.T, dir, name string) string {
	t.Helper()
	d := filepath.Join(dir, name)
	if err := os.MkdirAll(d, 0o755); err != nil {
		t.Fatalf("makeSharedDir: %v", err)
	}
	return d
}

// ---------------------------------------------------------------------------
// Detect tests
// ---------------------------------------------------------------------------

func TestDetect_FindsGitRepos(t *testing.T) {
	dir := t.TempDir()
	makeTestRepo(t, dir, "repo-a")
	makeTestRepo(t, dir, "repo-b")

	cfg, err := workspace.Detect(dir)
	if err != nil {
		t.Fatalf("Detect returned error: %v", err)
	}

	if len(cfg.Repos) != 2 {
		t.Errorf("expected 2 repos, got %d", len(cfg.Repos))
	}
}

func TestDetect_DetectsGoStack(t *testing.T) {
	dir := t.TempDir()
	makeTestRepo(t, dir, "my-go-repo", "go.mod")

	cfg, err := workspace.Detect(dir)
	if err != nil {
		t.Fatalf("Detect returned error: %v", err)
	}

	if len(cfg.Repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(cfg.Repos))
	}
	if cfg.Repos[0].Type != "go" {
		t.Errorf("type = %q, want %q", cfg.Repos[0].Type, "go")
	}
}

func TestDetect_DetectsPythonStack_RequirementsTxt(t *testing.T) {
	dir := t.TempDir()
	makeTestRepo(t, dir, "my-py-repo", "requirements.txt")

	cfg, err := workspace.Detect(dir)
	if err != nil {
		t.Fatalf("Detect returned error: %v", err)
	}

	if len(cfg.Repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(cfg.Repos))
	}
	if cfg.Repos[0].Type != "python" {
		t.Errorf("type = %q, want %q", cfg.Repos[0].Type, "python")
	}
}

func TestDetect_DetectsPythonStack_PyprojectToml(t *testing.T) {
	dir := t.TempDir()
	makeTestRepo(t, dir, "my-py2-repo", "pyproject.toml")

	cfg, err := workspace.Detect(dir)
	if err != nil {
		t.Fatalf("Detect returned error: %v", err)
	}

	if len(cfg.Repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(cfg.Repos))
	}
	if cfg.Repos[0].Type != "python" {
		t.Errorf("type = %q, want %q", cfg.Repos[0].Type, "python")
	}
}

func TestDetect_DetectsTypeScriptStack(t *testing.T) {
	dir := t.TempDir()
	makeTestRepo(t, dir, "my-ts-repo", "package.json", "tsconfig.json")

	cfg, err := workspace.Detect(dir)
	if err != nil {
		t.Fatalf("Detect returned error: %v", err)
	}

	if len(cfg.Repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(cfg.Repos))
	}
	if cfg.Repos[0].Type != "typescript" {
		t.Errorf("type = %q, want %q", cfg.Repos[0].Type, "typescript")
	}
}

func TestDetect_DetectsJavaScriptStack_PackageJsonOnly(t *testing.T) {
	dir := t.TempDir()
	// package.json without tsconfig.json → javascript
	makeTestRepo(t, dir, "my-js-repo", "package.json")

	cfg, err := workspace.Detect(dir)
	if err != nil {
		t.Fatalf("Detect returned error: %v", err)
	}

	if len(cfg.Repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(cfg.Repos))
	}
	// package.json alone is still reported as "typescript" per spec (or "javascript")
	// The spec says: package.json → "typescript" (or "javascript")
	// We check it's one of the two; the exact value is an implementation detail.
	typ := cfg.Repos[0].Type
	if typ != "typescript" && typ != "javascript" {
		t.Errorf("type = %q, want 'typescript' or 'javascript'", typ)
	}
}

func TestDetect_DetectsRustStack(t *testing.T) {
	dir := t.TempDir()
	makeTestRepo(t, dir, "my-rust-repo", "Cargo.toml")

	cfg, err := workspace.Detect(dir)
	if err != nil {
		t.Fatalf("Detect returned error: %v", err)
	}

	if len(cfg.Repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(cfg.Repos))
	}
	if cfg.Repos[0].Type != "rust" {
		t.Errorf("type = %q, want %q", cfg.Repos[0].Type, "rust")
	}
}

func TestDetect_DetectsCSharpStack_Csproj(t *testing.T) {
	dir := t.TempDir()
	makeTestRepo(t, dir, "my-cs-repo", "MyApp.csproj")

	cfg, err := workspace.Detect(dir)
	if err != nil {
		t.Fatalf("Detect returned error: %v", err)
	}

	if len(cfg.Repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(cfg.Repos))
	}
	if cfg.Repos[0].Type != "csharp" {
		t.Errorf("type = %q, want %q", cfg.Repos[0].Type, "csharp")
	}
}

func TestDetect_IgnoresNonGitDirs(t *testing.T) {
	dir := t.TempDir()
	makeTestRepo(t, dir, "repo-with-git")
	makeSharedDir(t, dir, "shared-docs")

	cfg, err := workspace.Detect(dir)
	if err != nil {
		t.Fatalf("Detect returned error: %v", err)
	}

	if len(cfg.Repos) != 1 {
		t.Errorf("expected 1 repo, got %d (non-git dirs must not be in repos)", len(cfg.Repos))
	}

	// The shared dir should appear in Shared.
	found := false
	for _, s := range cfg.Shared {
		if s == "shared-docs" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'shared-docs' in Shared, got %v", cfg.Shared)
	}
}

func TestDetect_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	cfg, err := workspace.Detect(dir)
	if err != nil {
		t.Fatalf("Detect returned error: %v", err)
	}

	if len(cfg.Repos) != 0 {
		t.Errorf("expected 0 repos in empty dir, got %d", len(cfg.Repos))
	}
}

func TestDetect_DetectsBranch(t *testing.T) {
	dir := t.TempDir()
	repoDir := makeTestRepo(t, dir, "my-repo")
	// Override HEAD with a feature branch.
	if err := os.WriteFile(filepath.Join(repoDir, ".git", "HEAD"), []byte("ref: refs/heads/feature/my-feature\n"), 0o644); err != nil {
		t.Fatalf("failed to write HEAD: %v", err)
	}

	cfg, err := workspace.Detect(dir)
	if err != nil {
		t.Fatalf("Detect returned error: %v", err)
	}

	if len(cfg.Repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(cfg.Repos))
	}
	if cfg.Repos[0].Branch != "feature/my-feature" {
		t.Errorf("branch = %q, want %q", cfg.Repos[0].Branch, "feature/my-feature")
	}
}

func TestDetect_DetectsBranch_DetachedHEAD(t *testing.T) {
	dir := t.TempDir()
	repoDir := makeTestRepo(t, dir, "detached-repo")
	// Write a commit hash directly (detached HEAD).
	if err := os.WriteFile(filepath.Join(repoDir, ".git", "HEAD"), []byte("abc123def456abc123def456abc123def456abc1\n"), 0o644); err != nil {
		t.Fatalf("failed to write HEAD: %v", err)
	}

	cfg, err := workspace.Detect(dir)
	if err != nil {
		t.Fatalf("Detect returned error: %v", err)
	}

	if len(cfg.Repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(cfg.Repos))
	}
	// Detached HEAD → default to "main".
	if cfg.Repos[0].Branch != "main" {
		t.Errorf("branch = %q, want %q (detached HEAD should default to main)", cfg.Repos[0].Branch, "main")
	}
}

func TestDetect_WorkspaceName_FromDirBasename(t *testing.T) {
	// Create a temp dir with a meaningful name.
	parent := t.TempDir()
	dir := filepath.Join(parent, "my-monorepo")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	cfg, err := workspace.Detect(dir)
	if err != nil {
		t.Fatalf("Detect returned error: %v", err)
	}

	if cfg.Name != "my-monorepo" {
		t.Errorf("name = %q, want %q", cfg.Name, "my-monorepo")
	}
}

func TestDetect_RepoPath_IsRelative(t *testing.T) {
	dir := t.TempDir()
	makeTestRepo(t, dir, "my-service")

	cfg, err := workspace.Detect(dir)
	if err != nil {
		t.Fatalf("Detect returned error: %v", err)
	}

	if len(cfg.Repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(cfg.Repos))
	}
	// Path should be relative (just the directory name, not an absolute path).
	if filepath.IsAbs(cfg.Repos[0].Path) {
		t.Errorf("repo path should be relative, got absolute: %q", cfg.Repos[0].Path)
	}
	if cfg.Repos[0].Path != "my-service" {
		t.Errorf("repo path = %q, want %q", cfg.Repos[0].Path, "my-service")
	}
}

// ---------------------------------------------------------------------------
// LoadConfig / SaveConfig tests
// ---------------------------------------------------------------------------

func TestLoadConfig_ValidJSON(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".claude-workspace.json")

	raw := `{
		"name": "my-workspace",
		"planning_repo": "planning",
		"repos": [
			{"path": "api", "type": "go", "branch": "main"},
			{"path": "web", "type": "typescript"}
		],
		"shared": ["docs"],
		"domain_labels": ["backend", "frontend"]
	}`
	if err := os.WriteFile(cfgPath, []byte(raw), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := workspace.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if cfg.Name != "my-workspace" {
		t.Errorf("name = %q, want %q", cfg.Name, "my-workspace")
	}
	if cfg.PlanningRepo != "planning" {
		t.Errorf("planning_repo = %q, want %q", cfg.PlanningRepo, "planning")
	}
	if len(cfg.Repos) != 2 {
		t.Errorf("repos count = %d, want 2", len(cfg.Repos))
	}
	if cfg.Repos[0].Type != "go" {
		t.Errorf("repos[0].type = %q, want %q", cfg.Repos[0].Type, "go")
	}
	if len(cfg.Shared) != 1 || cfg.Shared[0] != "docs" {
		t.Errorf("shared = %v, want [docs]", cfg.Shared)
	}
	if len(cfg.DomainLabels) != 2 {
		t.Errorf("domain_labels count = %d, want 2", len(cfg.DomainLabels))
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".claude-workspace.json")
	os.WriteFile(cfgPath, []byte("not valid json {{{"), 0o644)

	_, err := workspace.LoadConfig(cfgPath)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := workspace.LoadConfig("/nonexistent/path/.claude-workspace.json")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestSaveConfig_WritesJSON(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".claude-workspace.json")

	original := &workspace.Config{
		Name:         "saved-workspace",
		PlanningRepo: "planner",
		Repos: []workspace.Repo{
			{Path: "api", Type: "go", Branch: "main"},
		},
		Shared:       []string{"infra"},
		DomainLabels: []string{"backend"},
	}

	if err := workspace.SaveConfig(cfgPath, original); err != nil {
		t.Fatalf("SaveConfig returned error: %v", err)
	}

	// Re-read and verify round-trip.
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("ReadFile after save: %v", err)
	}

	var roundTrip workspace.Config
	if err := json.Unmarshal(data, &roundTrip); err != nil {
		t.Fatalf("saved file is not valid JSON: %v", err)
	}

	if roundTrip.Name != original.Name {
		t.Errorf("name = %q, want %q", roundTrip.Name, original.Name)
	}
	if roundTrip.PlanningRepo != original.PlanningRepo {
		t.Errorf("planning_repo = %q, want %q", roundTrip.PlanningRepo, original.PlanningRepo)
	}
	if len(roundTrip.Repos) != 1 {
		t.Errorf("repos count = %d, want 1", len(roundTrip.Repos))
	}
}

// ---------------------------------------------------------------------------
// LoadOrDetect tests
// ---------------------------------------------------------------------------

func TestLoadOrDetect_FallsBackToDetect(t *testing.T) {
	dir := t.TempDir()
	// No .claude-workspace.json, but a git repo exists.
	makeTestRepo(t, dir, "svc", "go.mod")

	cfg, err := workspace.LoadOrDetect(dir)
	if err != nil {
		t.Fatalf("LoadOrDetect returned error: %v", err)
	}

	if len(cfg.Repos) != 1 {
		t.Errorf("expected 1 detected repo, got %d", len(cfg.Repos))
	}
	if cfg.Repos[0].Type != "go" {
		t.Errorf("type = %q, want %q", cfg.Repos[0].Type, "go")
	}
}

func TestLoadOrDetect_PreferConfigFile(t *testing.T) {
	dir := t.TempDir()
	// A git repo that would be auto-detected.
	makeTestRepo(t, dir, "svc", "go.mod")

	// Also write a config file that overrides name and adds planning_repo.
	cfgRaw := `{
		"name": "override-name",
		"planning_repo": "planner",
		"repos": []
	}`
	if err := os.WriteFile(filepath.Join(dir, ".claude-workspace.json"), []byte(cfgRaw), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := workspace.LoadOrDetect(dir)
	if err != nil {
		t.Fatalf("LoadOrDetect returned error: %v", err)
	}

	// Name from config file wins.
	if cfg.Name != "override-name" {
		t.Errorf("name = %q, want %q", cfg.Name, "override-name")
	}
	// Config had empty repos → falls back to detected.
	if len(cfg.Repos) != 1 {
		t.Errorf("expected 1 detected repo (fallback), got %d", len(cfg.Repos))
	}
	// planning_repo from config.
	if cfg.PlanningRepo != "planner" {
		t.Errorf("planning_repo = %q, want %q", cfg.PlanningRepo, "planner")
	}
}

// ---------------------------------------------------------------------------
// merge tests
// ---------------------------------------------------------------------------

func TestMerge_UserOverridesName(t *testing.T) {
	dir := t.TempDir()
	makeTestRepo(t, dir, "svc")

	// Config with explicit name but no repos.
	cfgRaw := `{"name": "user-name", "repos": []}`
	os.WriteFile(filepath.Join(dir, ".claude-workspace.json"), []byte(cfgRaw), 0o644)

	cfg, err := workspace.LoadOrDetect(dir)
	if err != nil {
		t.Fatalf("LoadOrDetect: %v", err)
	}

	if cfg.Name != "user-name" {
		t.Errorf("name = %q, want %q (user config should override detected name)", cfg.Name, "user-name")
	}
}

func TestMerge_UserProvidesRepos_DetectedIgnored(t *testing.T) {
	dir := t.TempDir()
	// Two detectable repos.
	makeTestRepo(t, dir, "svc-a", "go.mod")
	makeTestRepo(t, dir, "svc-b", "Cargo.toml")

	// Config explicitly lists only one repo.
	cfgRaw := `{"name": "ws", "repos": [{"path": "svc-a", "type": "go"}]}`
	os.WriteFile(filepath.Join(dir, ".claude-workspace.json"), []byte(cfgRaw), 0o644)

	cfg, err := workspace.LoadOrDetect(dir)
	if err != nil {
		t.Fatalf("LoadOrDetect: %v", err)
	}

	// User provided repos → use those, not detected.
	if len(cfg.Repos) != 1 {
		t.Errorf("repos count = %d, want 1 (user-provided repos should override detected)", len(cfg.Repos))
	}
	if cfg.Repos[0].Path != "svc-a" {
		t.Errorf("repos[0].path = %q, want %q", cfg.Repos[0].Path, "svc-a")
	}
}

// ---------------------------------------------------------------------------
// Monorepo detection tests
// ---------------------------------------------------------------------------

// makeSubDir creates a subdirectory under parentDir/name with optional marker files.
func makeSubDir(t *testing.T, parentDir, name string, markers ...string) string {
	t.Helper()
	dir := filepath.Join(parentDir, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("makeSubDir: MkdirAll: %v", err)
	}
	for _, m := range markers {
		if err := os.WriteFile(filepath.Join(dir, m), []byte("{}"), 0o644); err != nil {
			t.Fatalf("makeSubDir: write marker %s: %v", m, err)
		}
	}
	return dir
}

func TestDetectMonorepo_Turborepo(t *testing.T) {
	dir := t.TempDir()
	// Create turbo.json at root
	os.WriteFile(filepath.Join(dir, "turbo.json"), []byte(`{"pipeline":{}}`), 0o644)
	// Create apps with tech markers
	makeSubDir(t, dir, "apps/web", "package.json", "tsconfig.json")
	makeSubDir(t, dir, "apps/api", "package.json")

	monorepoType, subProjects := workspace.DetectMonorepo(dir)

	if monorepoType != "turborepo" {
		t.Errorf("monorepoType = %q, want %q", monorepoType, "turborepo")
	}
	if len(subProjects) != 2 {
		t.Fatalf("expected 2 sub-projects, got %d", len(subProjects))
	}

	// Verify sub-project paths are relative
	for _, sp := range subProjects {
		if filepath.IsAbs(sp.Path) {
			t.Errorf("sub-project path should be relative, got: %q", sp.Path)
		}
	}
}

func TestDetectMonorepo_TurborepoAppsAndPackages(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "turbo.json"), []byte(`{"pipeline":{}}`), 0o644)
	makeSubDir(t, dir, "apps/web", "package.json", "tsconfig.json")
	makeSubDir(t, dir, "packages/ui", "package.json", "tsconfig.json")
	makeSubDir(t, dir, "packages/config", "package.json")

	monorepoType, subProjects := workspace.DetectMonorepo(dir)

	if monorepoType != "turborepo" {
		t.Errorf("monorepoType = %q, want %q", monorepoType, "turborepo")
	}
	if len(subProjects) != 3 {
		t.Fatalf("expected 3 sub-projects (1 app + 2 packages), got %d", len(subProjects))
	}

	// Verify paths include the parent dir prefix
	paths := make(map[string]bool)
	for _, sp := range subProjects {
		paths[sp.Path] = true
	}
	for _, expected := range []string{"apps/web", "packages/ui", "packages/config"} {
		if !paths[expected] {
			t.Errorf("expected sub-project %q not found in %v", expected, paths)
		}
	}
}

func TestDetectMonorepo_TurborepoNoApps(t *testing.T) {
	dir := t.TempDir()
	// turbo.json exists but no apps/ or packages/ directories
	os.WriteFile(filepath.Join(dir, "turbo.json"), []byte(`{"pipeline":{}}`), 0o644)

	monorepoType, subProjects := workspace.DetectMonorepo(dir)

	// No sub-projects found, so not treated as a monorepo
	if monorepoType != "" {
		t.Errorf("monorepoType = %q, want empty (no apps/packages)", monorepoType)
	}
	if len(subProjects) != 0 {
		t.Errorf("expected 0 sub-projects, got %d", len(subProjects))
	}
}

func TestDetectMonorepo_MixedLanguage(t *testing.T) {
	dir := t.TempDir()
	makeSubDir(t, dir, "backend", "go.mod")
	makeSubDir(t, dir, "frontend", "package.json", "tsconfig.json")

	monorepoType, subProjects := workspace.DetectMonorepo(dir)

	if monorepoType != "mixed" {
		t.Errorf("monorepoType = %q, want %q", monorepoType, "mixed")
	}
	if len(subProjects) != 2 {
		t.Fatalf("expected 2 sub-projects, got %d", len(subProjects))
	}

	// Verify types
	types := make(map[string]string)
	for _, sp := range subProjects {
		types[sp.Path] = sp.Type
	}
	if types["backend"] != "go" {
		t.Errorf("backend type = %q, want %q", types["backend"], "go")
	}
	if types["frontend"] != "typescript" {
		t.Errorf("frontend type = %q, want %q", types["frontend"], "typescript")
	}
}

func TestDetectMonorepo_MixedSkipsCommonDirs(t *testing.T) {
	dir := t.TempDir()
	makeSubDir(t, dir, "backend", "go.mod")
	makeSubDir(t, dir, "frontend", "package.json", "tsconfig.json")
	// These should be skipped
	makeSubDir(t, dir, "node_modules/some-pkg", "package.json")
	makeSubDir(t, dir, "vendor/some-dep", "go.mod")
	makeSubDir(t, dir, ".git")

	monorepoType, subProjects := workspace.DetectMonorepo(dir)

	if monorepoType != "mixed" {
		t.Errorf("monorepoType = %q, want %q", monorepoType, "mixed")
	}
	// Only backend and frontend, not node_modules or vendor
	if len(subProjects) != 2 {
		t.Fatalf("expected 2 sub-projects (skipping common dirs), got %d", len(subProjects))
	}
}

func TestDetectMonorepo_SingleSubdirNotMixed(t *testing.T) {
	dir := t.TempDir()
	makeSubDir(t, dir, "backend", "go.mod")
	makeSubDir(t, dir, "docs") // no tech markers

	monorepoType, subProjects := workspace.DetectMonorepo(dir)

	// Only 1 subdir with tech stack — not enough for "mixed"
	if monorepoType != "" {
		t.Errorf("monorepoType = %q, want empty (single subdir is not a monorepo)", monorepoType)
	}
	if len(subProjects) != 0 {
		t.Errorf("expected 0 sub-projects, got %d", len(subProjects))
	}
}

func TestDetectTestCommand(t *testing.T) {
	tests := []struct {
		techType string
		want     string
	}{
		{"go", "go test ./..."},
		{"typescript", "npm test"},
		{"python", "pytest"},
		{"rust", "cargo test"},
		{"csharp", ""},
		{"", ""},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.techType, func(t *testing.T) {
			got := workspace.DetectTestCommand(tt.techType)
			if got != tt.want {
				t.Errorf("DetectTestCommand(%q) = %q, want %q", tt.techType, got, tt.want)
			}
		})
	}
}

func TestDetect_MonorepoInWorkspace(t *testing.T) {
	dir := t.TempDir()
	// Create a repo that is also a turborepo monorepo
	repoDir := makeTestRepo(t, dir, "myapp", "package.json")
	os.WriteFile(filepath.Join(repoDir, "turbo.json"), []byte(`{"pipeline":{}}`), 0o644)
	makeSubDir(t, repoDir, "apps/web", "package.json", "tsconfig.json")
	makeSubDir(t, repoDir, "apps/api", "package.json")

	cfg, err := workspace.Detect(dir)
	if err != nil {
		t.Fatalf("Detect returned error: %v", err)
	}

	if len(cfg.Repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(cfg.Repos))
	}

	repo := cfg.Repos[0]
	if repo.MonorepoType != "turborepo" {
		t.Errorf("MonorepoType = %q, want %q", repo.MonorepoType, "turborepo")
	}
	if len(repo.SubProjects) != 2 {
		t.Errorf("expected 2 sub-projects, got %d", len(repo.SubProjects))
	}
}

func TestDetect_SingleRepoWithMonorepo(t *testing.T) {
	// When Detect is called on a directory that IS a git repo (has .git at root),
	// it should detect it as a single-repo workspace with monorepo sub-projects.
	dir := t.TempDir()

	// Make dir itself a git repo with turbo.json (Turborepo monorepo).
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".git", "HEAD"), []byte("ref: refs/heads/main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "turbo.json"), []byte(`{"pipeline":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"mono"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create apps/web and apps/api.
	for _, app := range []string{"apps/web", "apps/api"} {
		appDir := filepath.Join(dir, app)
		if err := os.MkdirAll(appDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(appDir, "package.json"), []byte(`{}`), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(appDir, "tsconfig.json"), []byte(`{}`), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	cfg, err := workspace.Detect(dir)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}

	if len(cfg.Repos) != 1 {
		t.Fatalf("expected 1 repo (self), got %d", len(cfg.Repos))
	}

	repo := cfg.Repos[0]
	if repo.Path != "." {
		t.Errorf("expected path '.', got %q", repo.Path)
	}
	if repo.MonorepoType != "turborepo" {
		t.Errorf("expected MonorepoType 'turborepo', got %q", repo.MonorepoType)
	}
	if len(repo.SubProjects) < 2 {
		t.Errorf("expected at least 2 sub-projects, got %d", len(repo.SubProjects))
	}
	if repo.Branch != "main" {
		t.Errorf("expected branch 'main', got %q", repo.Branch)
	}
}
