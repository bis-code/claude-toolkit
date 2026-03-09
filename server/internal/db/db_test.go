package db_test

import (
	"testing"

	"github.com/bis-code/claude-toolkit/server/internal/db"
)

func setupStore(t *testing.T) *db.Store {
	t.Helper()
	store, err := db.NewMemoryStore()
	if err != nil {
		t.Fatalf("failed to create memory store: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func TestNewMemoryStore(t *testing.T) {
	store := setupStore(t)
	if store == nil {
		t.Fatal("store is nil")
	}
}

func TestCreateRule(t *testing.T) {
	store := setupStore(t)

	rule := &db.Rule{
		ID:            "test-001",
		Content:       "Always write tests first",
		Scope:         "global",
		Tags:          map[string][]string{"domain": {"testing"}},
		Effectiveness: 0.5,
		CreatedFrom:   "test",
	}

	if err := store.CreateRule(rule); err != nil {
		t.Fatalf("CreateRule failed: %v", err)
	}

	// Verify it was created
	got, err := store.GetRule("test-001")
	if err != nil {
		t.Fatalf("GetRule failed: %v", err)
	}

	if got.Content != "Always write tests first" {
		t.Errorf("content = %q, want %q", got.Content, "Always write tests first")
	}
	if got.Scope != "global" {
		t.Errorf("scope = %q, want %q", got.Scope, "global")
	}
	if got.Effectiveness != 0.5 {
		t.Errorf("effectiveness = %f, want 0.5", got.Effectiveness)
	}
}

func TestCreateRule_DuplicateID(t *testing.T) {
	store := setupStore(t)

	rule := &db.Rule{ID: "dup-001", Content: "Rule 1", Scope: "global"}
	if err := store.CreateRule(rule); err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	rule2 := &db.Rule{ID: "dup-001", Content: "Rule 2", Scope: "global"}
	if err := store.CreateRule(rule2); err == nil {
		t.Error("expected error on duplicate ID, got nil")
	}
}

func TestGetRule_NotFound(t *testing.T) {
	store := setupStore(t)

	_, err := store.GetRule("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent rule, got nil")
	}
}

func TestUpdateRule(t *testing.T) {
	store := setupStore(t)

	rule := &db.Rule{ID: "upd-001", Content: "Original", Scope: "global"}
	store.CreateRule(rule)

	rule.Content = "Updated content"
	rule.Scope = "project"
	rule.Project = "my-project"
	if err := store.UpdateRule(rule); err != nil {
		t.Fatalf("UpdateRule failed: %v", err)
	}

	got, _ := store.GetRule("upd-001")
	if got.Content != "Updated content" {
		t.Errorf("content = %q, want %q", got.Content, "Updated content")
	}
	if got.Scope != "project" {
		t.Errorf("scope = %q, want %q", got.Scope, "project")
	}
}

func TestUpdateRule_NotFound(t *testing.T) {
	store := setupStore(t)

	rule := &db.Rule{ID: "nonexistent", Content: "test", Scope: "global"}
	if err := store.UpdateRule(rule); err == nil {
		t.Error("expected error for nonexistent rule, got nil")
	}
}

func TestDeleteRule(t *testing.T) {
	store := setupStore(t)

	rule := &db.Rule{ID: "del-001", Content: "To delete", Scope: "global"}
	store.CreateRule(rule)

	if err := store.DeleteRule("del-001"); err != nil {
		t.Fatalf("DeleteRule failed: %v", err)
	}

	_, err := store.GetRule("del-001")
	if err == nil {
		t.Error("expected error after delete, got nil")
	}
}

func TestDeleteRule_NotFound(t *testing.T) {
	store := setupStore(t)

	if err := store.DeleteRule("nonexistent"); err == nil {
		t.Error("expected error for nonexistent rule, got nil")
	}
}

func TestListRules_ByScope(t *testing.T) {
	store := setupStore(t)

	store.CreateRule(&db.Rule{ID: "g-001", Content: "Global rule", Scope: "global"})
	store.CreateRule(&db.Rule{ID: "p-001", Content: "Project rule", Scope: "project", Project: "my-proj"})
	store.CreateRule(&db.Rule{ID: "p-002", Content: "Another project rule", Scope: "project", Project: "other"})

	rules, err := store.ListRules("global", "", "")
	if err != nil {
		t.Fatalf("ListRules failed: %v", err)
	}
	if len(rules) != 1 {
		t.Errorf("expected 1 global rule, got %d", len(rules))
	}
}

func TestListRules_ByProject(t *testing.T) {
	store := setupStore(t)

	store.CreateRule(&db.Rule{ID: "p-001", Content: "My project rule", Scope: "project", Project: "my-proj"})
	store.CreateRule(&db.Rule{ID: "p-002", Content: "Other project rule", Scope: "project", Project: "other"})

	rules, err := store.ListRules("", "my-proj", "")
	if err != nil {
		t.Fatalf("ListRules failed: %v", err)
	}
	if len(rules) != 1 {
		t.Errorf("expected 1 rule for my-proj, got %d", len(rules))
	}
}

func TestListRules_ByTechStack(t *testing.T) {
	store := setupStore(t)

	store.CreateRule(&db.Rule{
		ID: "go-001", Content: "Go rule", Scope: "global",
		Tags: map[string][]string{"tech_stack": {"go"}},
	})
	store.CreateRule(&db.Rule{
		ID: "ts-001", Content: "TS rule", Scope: "global",
		Tags: map[string][]string{"tech_stack": {"typescript"}},
	})
	store.CreateRule(&db.Rule{
		ID: "uni-001", Content: "Universal rule", Scope: "global",
		Tags: map[string][]string{},
	})

	rules, err := store.ListRules("", "", "go")
	if err != nil {
		t.Fatalf("ListRules failed: %v", err)
	}

	// Should get the go-specific rule + the universal rule (no tech_stack tag)
	if len(rules) != 2 {
		t.Errorf("expected 2 rules (go + universal), got %d", len(rules))
		for _, r := range rules {
			t.Logf("  rule: %s (%s)", r.ID, r.Content)
		}
	}
}

func TestListRules_ExcludesDeprecated(t *testing.T) {
	store := setupStore(t)

	store.CreateRule(&db.Rule{ID: "active-001", Content: "Active", Scope: "global"})
	store.CreateRule(&db.Rule{ID: "dep-001", Content: "Deprecated", Scope: "global", Deprecated: true})

	rules, err := store.ListRules("", "", "")
	if err != nil {
		t.Fatalf("ListRules failed: %v", err)
	}
	if len(rules) != 1 {
		t.Errorf("expected 1 active rule, got %d", len(rules))
	}
}

func TestListRules_OrderedByEffectiveness(t *testing.T) {
	store := setupStore(t)

	store.CreateRule(&db.Rule{ID: "low-001", Content: "Low", Scope: "global", Effectiveness: 0.3})
	store.CreateRule(&db.Rule{ID: "high-001", Content: "High", Scope: "global", Effectiveness: 0.9})
	store.CreateRule(&db.Rule{ID: "mid-001", Content: "Mid", Scope: "global", Effectiveness: 0.6})

	rules, err := store.ListRules("", "", "")
	if err != nil {
		t.Fatalf("ListRules failed: %v", err)
	}
	if len(rules) != 3 {
		t.Fatalf("expected 3 rules, got %d", len(rules))
	}
	if rules[0].ID != "high-001" {
		t.Errorf("expected first rule to be high-001, got %s", rules[0].ID)
	}
	if rules[2].ID != "low-001" {
		t.Errorf("expected last rule to be low-001, got %s", rules[2].ID)
	}
}

func TestRecordScore_UpdatesEffectiveness(t *testing.T) {
	store := setupStore(t)

	store.CreateRule(&db.Rule{ID: "score-001", Content: "Test", Scope: "global", Effectiveness: 0.5})

	// Record 3 helpful, 1 not helpful → effectiveness should be 0.75
	store.RecordScore("score-001", true, "helped", "session-1")
	store.RecordScore("score-001", true, "helped", "session-2")
	store.RecordScore("score-001", true, "helped", "session-3")
	store.RecordScore("score-001", false, "didn't help", "session-4")

	rule, _ := store.GetRule("score-001")
	if rule.Effectiveness != 0.75 {
		t.Errorf("effectiveness = %f, want 0.75", rule.Effectiveness)
	}
}

func TestCreateRule_WithTags(t *testing.T) {
	store := setupStore(t)

	rule := &db.Rule{
		ID:      "tags-001",
		Content: "Unity-specific rule",
		Scope:   "global",
		Tags: map[string][]string{
			"tech_stack": {"unity", "csharp"},
			"domain":     {"editor-scripting"},
		},
	}
	store.CreateRule(rule)

	got, _ := store.GetRule("tags-001")
	if len(got.Tags["tech_stack"]) != 2 {
		t.Errorf("expected 2 tech_stack tags, got %d", len(got.Tags["tech_stack"]))
	}
	if got.Tags["tech_stack"][0] != "unity" {
		t.Errorf("expected first tech_stack tag to be 'unity', got %q", got.Tags["tech_stack"][0])
	}
}
