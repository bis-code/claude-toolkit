package workspace_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/bis-code/claude-toolkit/server/internal/workspace"
)

func TestIntegration_DetectTestWorkspace(t *testing.T) {
	// Get path to test/workspace/ relative to this source file.
	_, filename, _, _ := runtime.Caller(0)
	testWorkspace := filepath.Join(filepath.Dir(filename), "..", "..", "..", "test", "workspace")

	cfg, err := workspace.LoadOrDetect(testWorkspace)
	if err != nil {
		t.Fatalf("failed to detect workspace: %v", err)
	}

	// Should find 4 repos from the config file.
	if len(cfg.Repos) != 4 {
		t.Errorf("expected 4 repos, got %d", len(cfg.Repos))
	}

	// Verify each repo type.
	repoTypes := make(map[string]string)
	for _, r := range cfg.Repos {
		repoTypes[r.Path] = r.Type
	}

	expected := map[string]string{
		"api-go":      "go",
		"ml-python":   "python",
		"bff-ts":      "typescript",
		"worker-rust": "rust",
	}

	for path, expectedType := range expected {
		if got := repoTypes[path]; got != expectedType {
			t.Errorf("repo %s: type = %q, want %q", path, got, expectedType)
		}
	}

	// Verify workspace name.
	if cfg.Name != "test-workspace" {
		t.Errorf("name = %q, want 'test-workspace'", cfg.Name)
	}

	// Verify shared dirs.
	if len(cfg.Shared) != 1 || cfg.Shared[0] != "shared/" {
		t.Errorf("shared = %v, want ['shared/']", cfg.Shared)
	}
}

func TestIntegration_AutoDetectTestWorkspace(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	testWorkspace := filepath.Join(filepath.Dir(filename), "..", "..", "..", "test", "workspace")

	// Test auto-detection by calling Detect directly (ignores config file).
	cfg, err := workspace.Detect(testWorkspace)
	if err != nil {
		t.Fatalf("failed to auto-detect: %v", err)
	}

	// Should still find 4 repos (they have .git dirs).
	if len(cfg.Repos) != 4 {
		t.Errorf("expected 4 repos from auto-detect, got %d", len(cfg.Repos))
	}

	// Name should be derived from directory name.
	if cfg.Name != "workspace" {
		t.Errorf("auto-detected name = %q, want 'workspace'", cfg.Name)
	}
}

func TestIntegration_DetectTurborepoMonorepo(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	testDir := filepath.Join(filepath.Dir(filename), "..", "..", "..", "test", "workspace-turborepo")

	cfg, err := workspace.Detect(testDir)
	if err != nil {
		t.Fatalf("failed to detect workspace: %v", err)
	}

	if len(cfg.Repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(cfg.Repos))
	}

	repo := cfg.Repos[0]
	if repo.Path != "myapp" {
		t.Errorf("repo path = %q, want %q", repo.Path, "myapp")
	}
	if repo.MonorepoType != "turborepo" {
		t.Errorf("MonorepoType = %q, want %q", repo.MonorepoType, "turborepo")
	}

	// apps/web (ts), apps/api (ts), packages/ui (ts), packages/config (ts)
	if len(repo.SubProjects) != 4 {
		t.Fatalf("expected 4 sub-projects, got %d: %+v", len(repo.SubProjects), repo.SubProjects)
	}

	// Verify sub-project paths
	spPaths := make(map[string]string)
	for _, sp := range repo.SubProjects {
		spPaths[sp.Path] = sp.Type
	}

	for _, expected := range []string{"apps/web", "apps/api", "packages/ui", "packages/config"} {
		if _, ok := spPaths[expected]; !ok {
			t.Errorf("expected sub-project %q not found", expected)
		}
	}

	// apps/web has tsconfig.json → typescript
	if spPaths["apps/web"] != "typescript" {
		t.Errorf("apps/web type = %q, want %q", spPaths["apps/web"], "typescript")
	}
}

func TestIntegration_TurborepoMonorepoFixture(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	fixtureDir := filepath.Join(filepath.Dir(filename), "..", "..", "..", "test", "turborepo-monorepo")

	t.Run("load config from .claude-workspace.json", func(t *testing.T) {
		configPath := filepath.Join(fixtureDir, ".claude-workspace.json")
		cfg, err := workspace.LoadConfig(configPath)
		if err != nil {
			t.Fatalf("failed to load config: %v", err)
		}

		if cfg.Name != "turborepo-monorepo" {
			t.Errorf("name = %q, want %q", cfg.Name, "turborepo-monorepo")
		}

		if len(cfg.Repos) != 1 {
			t.Fatalf("expected 1 repo, got %d", len(cfg.Repos))
		}

		repo := cfg.Repos[0]
		if repo.MonorepoType != "turborepo" {
			t.Errorf("MonorepoType = %q, want %q", repo.MonorepoType, "turborepo")
		}

		if len(repo.SubProjects) < 4 {
			t.Fatalf("expected at least 4 sub-projects, got %d", len(repo.SubProjects))
		}

		for _, sp := range repo.SubProjects {
			if sp.Type != "typescript" {
				t.Errorf("sub-project %q type = %q, want %q", sp.Path, sp.Type, "typescript")
			}
		}
	})

	t.Run("auto-detect monorepo from fixture directory", func(t *testing.T) {
		monorepoType, subProjects := workspace.DetectMonorepo(fixtureDir)

		if monorepoType != "turborepo" {
			t.Errorf("monorepo type = %q, want %q", monorepoType, "turborepo")
		}

		if len(subProjects) < 4 {
			t.Fatalf("expected at least 4 sub-projects, got %d: %+v", len(subProjects), subProjects)
		}

		spTypes := make(map[string]string)
		for _, sp := range subProjects {
			spTypes[sp.Path] = sp.Type
		}

		expectedPaths := []string{"apps/web", "apps/api", "packages/ui", "packages/contracts"}
		for _, p := range expectedPaths {
			typ, ok := spTypes[p]
			if !ok {
				t.Errorf("expected sub-project %q not found", p)
				continue
			}
			if typ != "typescript" {
				t.Errorf("sub-project %q type = %q, want %q", p, typ, "typescript")
			}
		}
	})
}

func TestIntegration_DetectMixedLanguageMonorepo(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	testDir := filepath.Join(filepath.Dir(filename), "..", "..", "..", "test", "workspace-mixed")

	cfg, err := workspace.Detect(testDir)
	if err != nil {
		t.Fatalf("failed to detect workspace: %v", err)
	}

	if len(cfg.Repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(cfg.Repos))
	}

	repo := cfg.Repos[0]
	if repo.Path != "fullstack" {
		t.Errorf("repo path = %q, want %q", repo.Path, "fullstack")
	}
	if repo.MonorepoType != "mixed" {
		t.Errorf("MonorepoType = %q, want %q", repo.MonorepoType, "mixed")
	}

	// backend (go) + frontend (typescript), proto/ should be excluded (no tech markers)
	if len(repo.SubProjects) != 2 {
		t.Fatalf("expected 2 sub-projects, got %d: %+v", len(repo.SubProjects), repo.SubProjects)
	}

	spTypes := make(map[string]string)
	for _, sp := range repo.SubProjects {
		spTypes[sp.Path] = sp.Type
	}

	if spTypes["backend"] != "go" {
		t.Errorf("backend type = %q, want %q", spTypes["backend"], "go")
	}
	if spTypes["frontend"] != "typescript" {
		t.Errorf("frontend type = %q, want %q", spTypes["frontend"], "typescript")
	}

	// proto/ should NOT be in sub-projects
	if _, ok := spTypes["proto"]; ok {
		t.Error("proto/ should not be detected as a sub-project (no tech markers)")
	}
}

func TestIntegration_MixedMonorepoFixture(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	fixtureDir := filepath.Join(filepath.Dir(filename), "..", "..", "..", "test", "mixed-monorepo")

	t.Run("load config from .claude-workspace.json", func(t *testing.T) {
		configPath := filepath.Join(fixtureDir, ".claude-workspace.json")
		cfg, err := workspace.LoadConfig(configPath)
		if err != nil {
			t.Fatalf("failed to load config: %v", err)
		}

		if cfg.Name != "mixed-monorepo" {
			t.Errorf("name = %q, want %q", cfg.Name, "mixed-monorepo")
		}

		if len(cfg.Repos) != 1 {
			t.Fatalf("expected 1 repo, got %d", len(cfg.Repos))
		}

		repo := cfg.Repos[0]
		if repo.MonorepoType != "mixed" {
			t.Errorf("MonorepoType = %q, want %q", repo.MonorepoType, "mixed")
		}

		if len(repo.SubProjects) != 2 {
			t.Fatalf("expected 2 sub-projects in config, got %d", len(repo.SubProjects))
		}

		spTypes := make(map[string]string)
		for _, sp := range repo.SubProjects {
			spTypes[sp.Path] = sp.Type
		}

		if spTypes["backend"] != "go" {
			t.Errorf("backend type = %q, want %q", spTypes["backend"], "go")
		}
		if spTypes["frontend"] != "typescript" {
			t.Errorf("frontend type = %q, want %q", spTypes["frontend"], "typescript")
		}
	})

	t.Run("auto-detect monorepo from fixture directory", func(t *testing.T) {
		monorepoType, subProjects := workspace.DetectMonorepo(fixtureDir)

		if monorepoType != "mixed" {
			t.Errorf("monorepo type = %q, want %q", monorepoType, "mixed")
		}

		if len(subProjects) != 2 {
			t.Fatalf("expected 2 sub-projects, got %d: %+v", len(subProjects), subProjects)
		}

		spTypes := make(map[string]string)
		for _, sp := range subProjects {
			spTypes[sp.Path] = sp.Type
		}

		if spTypes["backend"] != "go" {
			t.Errorf("backend type = %q, want %q", spTypes["backend"], "go")
		}
		if spTypes["frontend"] != "typescript" {
			t.Errorf("frontend type = %q, want %q", spTypes["frontend"], "typescript")
		}

		// proto/ should NOT be in sub-projects (no tech markers like go.mod or package.json)
		if _, ok := spTypes["proto"]; ok {
			t.Error("proto/ should not be detected as a sub-project (no tech markers)")
		}
	})
}
