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
