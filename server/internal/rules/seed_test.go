package rules_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bis-code/claude-toolkit/server/internal/db"
	"github.com/bis-code/claude-toolkit/server/internal/rules"
)

func writeSeedFile(t *testing.T, dir, filename, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0o644); err != nil {
		t.Fatalf("cannot write seed file: %v", err)
	}
}

func TestLoadSeeds_LoadsFromJSON(t *testing.T) {
	store, _ := db.NewMemoryStore()
	defer store.Close()

	dir := t.TempDir()
	writeSeedFile(t, dir, "global.json", `{
		"rules": [
			{
				"id": "seed-global-001",
				"content": "Always write tests first",
				"scope": "global",
				"source_evidence": "TDD best practice"
			},
			{
				"id": "seed-global-002",
				"content": "Use conventional commits",
				"scope": "global"
			}
		]
	}`)

	loaded, skipped, err := rules.LoadSeeds(store, dir)
	if err != nil {
		t.Fatalf("LoadSeeds failed: %v", err)
	}
	if loaded != 2 {
		t.Errorf("expected 2 loaded, got %d", loaded)
	}
	if skipped != 0 {
		t.Errorf("expected 0 skipped, got %d", skipped)
	}

	// Verify rules exist
	rule, err := store.GetRule("seed-global-001")
	if err != nil {
		t.Fatalf("GetRule failed: %v", err)
	}
	if rule.Content != "Always write tests first" {
		t.Errorf("unexpected content: %q", rule.Content)
	}
	if rule.CreatedFrom != "seed" {
		t.Errorf("created_from = %q, want 'seed'", rule.CreatedFrom)
	}
	if rule.Effectiveness != 0.5 {
		t.Errorf("effectiveness = %f, want 0.5", rule.Effectiveness)
	}
}

func TestLoadSeeds_Idempotent(t *testing.T) {
	store, _ := db.NewMemoryStore()
	defer store.Close()

	dir := t.TempDir()
	writeSeedFile(t, dir, "test.json", `{
		"rules": [{"id": "seed-001", "content": "Test rule", "scope": "global"}]
	}`)

	// First load
	loaded1, _, _ := rules.LoadSeeds(store, dir)
	if loaded1 != 1 {
		t.Errorf("first load: expected 1, got %d", loaded1)
	}

	// Second load — should skip
	loaded2, skipped2, _ := rules.LoadSeeds(store, dir)
	if loaded2 != 0 {
		t.Errorf("second load: expected 0 loaded, got %d", loaded2)
	}
	if skipped2 != 1 {
		t.Errorf("second load: expected 1 skipped, got %d", skipped2)
	}
}

func TestLoadSeeds_WithTags(t *testing.T) {
	store, _ := db.NewMemoryStore()
	defer store.Close()

	dir := t.TempDir()
	writeSeedFile(t, dir, "go.json", `{
		"rules": [{
			"id": "seed-go-001",
			"content": "Use context.Context as first param",
			"scope": "global",
			"tags": {"tech_stack": ["go"], "domain": ["api"]}
		}]
	}`)

	rules.LoadSeeds(store, dir)

	rule, _ := store.GetRule("seed-go-001")
	if len(rule.Tags["tech_stack"]) != 1 || rule.Tags["tech_stack"][0] != "go" {
		t.Errorf("expected tech_stack [go], got %v", rule.Tags["tech_stack"])
	}
}

func TestLoadSeeds_MultipleFiles(t *testing.T) {
	store, _ := db.NewMemoryStore()
	defer store.Close()

	dir := t.TempDir()
	writeSeedFile(t, dir, "global.json", `{"rules": [{"id": "g-001", "content": "Global", "scope": "global"}]}`)
	writeSeedFile(t, dir, "go.json", `{"rules": [{"id": "go-001", "content": "Go", "scope": "global"}]}`)

	loaded, _, err := rules.LoadSeeds(store, dir)
	if err != nil {
		t.Fatalf("LoadSeeds failed: %v", err)
	}
	if loaded != 2 {
		t.Errorf("expected 2 loaded from 2 files, got %d", loaded)
	}
}

func TestLoadSeeds_EmptyDir(t *testing.T) {
	store, _ := db.NewMemoryStore()
	defer store.Close()

	dir := t.TempDir()
	loaded, _, err := rules.LoadSeeds(store, dir)
	if err != nil {
		t.Fatalf("LoadSeeds failed: %v", err)
	}
	if loaded != 0 {
		t.Errorf("expected 0 loaded from empty dir, got %d", loaded)
	}
}
