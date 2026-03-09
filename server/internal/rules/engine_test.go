package rules_test

import (
	"testing"

	"github.com/bis-code/claude-toolkit/server/internal/db"
	"github.com/bis-code/claude-toolkit/server/internal/rules"
)

func setupEngine(t *testing.T) (*rules.Engine, *db.Store) {
	t.Helper()
	store, err := db.NewMemoryStore()
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return rules.NewEngine(store), store
}

func TestGetActiveRules_MergesScopes(t *testing.T) {
	engine, store := setupEngine(t)

	store.CreateRule(&db.Rule{ID: "g-001", Content: "Global rule", Scope: "global", Effectiveness: 0.9})
	store.CreateRule(&db.Rule{ID: "w-001", Content: "Workspace rule", Scope: "workspace", Workspace: "my-ws", Effectiveness: 0.8})
	store.CreateRule(&db.Rule{ID: "p-001", Content: "Project rule", Scope: "project", Project: "my-proj", Effectiveness: 0.7})

	result, err := engine.GetActiveRules(rules.MergeContext{
		Project:   "my-proj",
		Workspace: "my-ws",
	})
	if err != nil {
		t.Fatalf("GetActiveRules failed: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("expected 3 merged rules, got %d", len(result))
		for _, r := range result {
			t.Logf("  rule: %s (scope=%s)", r.ID, r.Scope)
		}
	}
}

func TestGetActiveRules_FiltersbyTechStack(t *testing.T) {
	engine, store := setupEngine(t)

	store.CreateRule(&db.Rule{
		ID: "go-001", Content: "Go rule", Scope: "global", Effectiveness: 0.9,
		Tags: map[string][]string{"tech_stack": {"go"}},
	})
	store.CreateRule(&db.Rule{
		ID: "ts-001", Content: "TS rule", Scope: "global", Effectiveness: 0.8,
		Tags: map[string][]string{"tech_stack": {"typescript"}},
	})
	store.CreateRule(&db.Rule{
		ID: "uni-001", Content: "Universal rule", Scope: "global", Effectiveness: 0.7,
	})

	result, err := engine.GetActiveRules(rules.MergeContext{
		Project:   "my-go-project",
		TechStack: []string{"go"},
	})
	if err != nil {
		t.Fatalf("GetActiveRules failed: %v", err)
	}

	// Should get go-001 + uni-001, NOT ts-001
	if len(result) != 2 {
		t.Errorf("expected 2 rules (go + universal), got %d", len(result))
		for _, r := range result {
			t.Logf("  rule: %s", r.ID)
		}
	}

	for _, r := range result {
		if r.ID == "ts-001" {
			t.Error("typescript rule should have been filtered out")
		}
	}
}

func TestGetActiveRules_VariableSubstitution(t *testing.T) {
	engine, store := setupEngine(t)

	store.CreateRule(&db.Rule{
		ID:      "var-001",
		Content: "Run tests with: {{test_cmd}} in {{project_name}}",
		Scope:   "global",
	})

	result, err := engine.GetActiveRules(rules.MergeContext{
		Project: "my-proj",
		Variables: map[string]string{
			"test_cmd":     "go test ./...",
			"project_name": "my-proj",
		},
	})
	if err != nil {
		t.Fatalf("GetActiveRules failed: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(result))
	}

	expected := "Run tests with: go test ./... in my-proj"
	if result[0].Content != expected {
		t.Errorf("content = %q, want %q", result[0].Content, expected)
	}
}

func TestGetActiveRules_TokenBudget(t *testing.T) {
	engine, store := setupEngine(t)

	// Each rule ~25 chars ≈ 6 tokens. Budget of 10 tokens should fit ~1-2 rules.
	store.CreateRule(&db.Rule{ID: "r-001", Content: "High effectiveness rule!!", Scope: "global", Effectiveness: 0.9})
	store.CreateRule(&db.Rule{ID: "r-002", Content: "Medium effectiveness rule", Scope: "global", Effectiveness: 0.7})
	store.CreateRule(&db.Rule{ID: "r-003", Content: "Low effectiveness rule!!!", Scope: "global", Effectiveness: 0.3})

	result, err := engine.GetActiveRules(rules.MergeContext{
		Project:     "my-proj",
		TokenBudget: 10, // Very tight budget
	})
	if err != nil {
		t.Fatalf("GetActiveRules failed: %v", err)
	}

	// With budget of 10 tokens and ~6 tokens per rule, we should get 1 rule
	if len(result) > 2 {
		t.Errorf("expected at most 2 rules with tight budget, got %d", len(result))
	}

	// The included rule(s) should be the highest effectiveness
	if len(result) > 0 && result[0].ID != "r-001" {
		t.Errorf("expected highest effectiveness rule first, got %s", result[0].ID)
	}
}

func TestGetActiveRules_DefaultTokenBudget(t *testing.T) {
	engine, store := setupEngine(t)

	// With default budget (2000), should fit many rules
	for i := 0; i < 10; i++ {
		store.CreateRule(&db.Rule{
			ID:            "r-" + string(rune('a'+i)),
			Content:       "Short rule",
			Scope:         "global",
			Effectiveness: 0.5,
		})
	}

	result, err := engine.GetActiveRules(rules.MergeContext{
		Project: "my-proj",
	})
	if err != nil {
		t.Fatalf("GetActiveRules failed: %v", err)
	}

	if len(result) != 10 {
		t.Errorf("expected all 10 rules within default budget, got %d", len(result))
	}
}

func TestGetActiveRules_ExcludesDeprecated(t *testing.T) {
	engine, store := setupEngine(t)

	store.CreateRule(&db.Rule{ID: "active", Content: "Active rule", Scope: "global"})
	store.CreateRule(&db.Rule{ID: "deprecated", Content: "Deprecated", Scope: "global", Deprecated: true})

	result, err := engine.GetActiveRules(rules.MergeContext{Project: "proj"})
	if err != nil {
		t.Fatalf("GetActiveRules failed: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("expected 1 active rule, got %d", len(result))
	}
}

func TestGetActiveRules_WorkspaceOnly(t *testing.T) {
	engine, store := setupEngine(t)

	store.CreateRule(&db.Rule{ID: "w-001", Content: "WS1 rule", Scope: "workspace", Workspace: "ws1"})
	store.CreateRule(&db.Rule{ID: "w-002", Content: "WS2 rule", Scope: "workspace", Workspace: "ws2"})

	result, err := engine.GetActiveRules(rules.MergeContext{
		Project:   "proj",
		Workspace: "ws1",
	})
	if err != nil {
		t.Fatalf("GetActiveRules failed: %v", err)
	}

	// Should only get ws1 rule, not ws2
	for _, r := range result {
		if r.ID == "w-002" {
			t.Error("should not include workspace ws2 rules when context is ws1")
		}
	}
}
