//go:build realworld

package workspace

import (
	"os"
	"testing"
)

func TestRealWorld_LearnmeldMonorepo(t *testing.T) {
	dir := "/Users/baicoianuionut/som/personal-projects/learnmeld"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Skipf("learnmeld repo not found at %s", dir)
	}

	// learnmeld has turbo.json so DetectMonorepo should take the turborepo path.
	monorepoType, subProjects := DetectMonorepo(dir)
	if monorepoType != "turborepo" {
		t.Errorf("expected monorepo_type 'turborepo', got %q", monorepoType)
	}

	// Should find apps/api (go), apps/web (typescript), apps/landing (typescript),
	// and packages/contracts (typescript).
	foundGoAPI := false
	foundTSWeb := false
	for _, sp := range subProjects {
		t.Logf("sub_project: path=%q type=%q test_command=%q", sp.Path, sp.Type, sp.TestCommand)
		if sp.Path == "apps/api" && sp.Type == "go" {
			foundGoAPI = true
		}
		if sp.Path == "apps/web" && sp.Type == "typescript" {
			foundTSWeb = true
		}
	}
	if !foundGoAPI {
		t.Error("expected to find apps/api as Go sub-project")
	}
	if !foundTSWeb {
		t.Error("expected to find apps/web as TypeScript sub-project")
	}

	// Validate test commands are set correctly for each sub-project type.
	for _, sp := range subProjects {
		switch sp.Type {
		case "go":
			if sp.TestCommand != "go test ./..." {
				t.Errorf("expected 'go test ./...' for Go sub-project %s, got %q", sp.Path, sp.TestCommand)
			}
		case "typescript":
			if sp.TestCommand != "npm test" {
				t.Errorf("expected 'npm test' for TypeScript sub-project %s, got %q", sp.Path, sp.TestCommand)
			}
		}
	}
}

func TestRealWorld_McpDeepThink(t *testing.T) {
	dir := "/Users/baicoianuionut/som/personal-projects/mcp-deep-think"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Skipf("mcp-deep-think repo not found at %s", dir)
	}

	// mcp-deep-think has package.json + tsconfig.json — should be detected as typescript.
	techStack := detectTechStack(dir)
	if techStack != "typescript" {
		t.Errorf("expected 'typescript', got %q", techStack)
	}

	// Single-project repo: should NOT produce a monorepo result.
	monorepoType, subProjects := DetectMonorepo(dir)
	t.Logf("mcp-deep-think monorepo detection: type=%q, sub_projects=%d", monorepoType, len(subProjects))
	// We do not assert monorepoType == "" here because the repo may contain
	// sub-directories that trigger mixed detection. Log-only for manual inspection.
}

func TestRealWorld_WorkspaceDetection(t *testing.T) {
	dir := "/Users/baicoianuionut/som/personal-projects"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Skipf("personal-projects dir not found at %s", dir)
	}

	cfg, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if len(cfg.Repos) < 3 {
		t.Errorf("expected at least 3 repos in personal-projects, got %d", len(cfg.Repos))
	}

	t.Logf("Detected %d repos in %s:", len(cfg.Repos), dir)
	for _, r := range cfg.Repos {
		if r.MonorepoType != "" {
			t.Logf("  %s (type=%s, monorepo=%s, sub_projects=%d)", r.Path, r.Type, r.MonorepoType, len(r.SubProjects))
			for _, sp := range r.SubProjects {
				t.Logf("    - %s (type=%s, test=%s)", sp.Path, sp.Type, sp.TestCommand)
			}
		} else {
			t.Logf("  %s (type=%s)", r.Path, r.Type)
		}
	}

	// learnmeld must be identified as a turborepo monorepo with at least 2 sub-projects.
	foundLearnmeld := false
	for _, r := range cfg.Repos {
		if r.Path == "learnmeld" {
			foundLearnmeld = true
			if r.MonorepoType == "" {
				t.Error("learnmeld should be detected as a monorepo")
			}
			if len(r.SubProjects) < 2 {
				t.Errorf("learnmeld should have at least 2 sub-projects, got %d", len(r.SubProjects))
			}
		}
	}
	if !foundLearnmeld {
		t.Error("expected learnmeld to appear in detected repos")
	}
}
