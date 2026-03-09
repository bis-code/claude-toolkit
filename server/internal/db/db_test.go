package db_test

import (
	"fmt"
	"testing"
	"time"

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

// --- Session & Event tests ---

func TestCreateSession(t *testing.T) {
	store := setupStore(t)

	session := &db.Session{
		ID:        "sess-001",
		Project:   "my-project",
		StartedAt: time.Now().UTC(),
	}

	if err := store.CreateSession(session); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	got, err := store.GetSession("sess-001")
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}

	if got.ID != "sess-001" {
		t.Errorf("id = %q, want %q", got.ID, "sess-001")
	}
	if got.Project != "my-project" {
		t.Errorf("project = %q, want %q", got.Project, "my-project")
	}
	if got.EndedAt != nil {
		t.Errorf("ended_at should be nil for new session, got %v", got.EndedAt)
	}
	if got.TasksCompleted != 0 {
		t.Errorf("tasks_completed = %d, want 0", got.TasksCompleted)
	}
}

func TestGetSession_NotFound(t *testing.T) {
	store := setupStore(t)

	_, err := store.GetSession("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent session, got nil")
	}
}

func TestEndSession(t *testing.T) {
	store := setupStore(t)

	session := &db.Session{
		ID:        "sess-end-001",
		Project:   "my-project",
		StartedAt: time.Now().UTC(),
	}
	store.CreateSession(session)

	err := store.EndSession("sess-end-001", "Did great work", 0.95, 5, 1)
	if err != nil {
		t.Fatalf("EndSession failed: %v", err)
	}

	got, err := store.GetSession("sess-end-001")
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}

	if got.EndedAt == nil {
		t.Fatal("ended_at should not be nil after EndSession")
	}
	if got.Summary != "Did great work" {
		t.Errorf("summary = %q, want %q", got.Summary, "Did great work")
	}
	if got.Confidence != 0.95 {
		t.Errorf("confidence = %f, want 0.95", got.Confidence)
	}
	if got.TasksCompleted != 5 {
		t.Errorf("tasks_completed = %d, want 5", got.TasksCompleted)
	}
	if got.TasksFailed != 1 {
		t.Errorf("tasks_failed = %d, want 1", got.TasksFailed)
	}
}

func TestEndSession_NotFound(t *testing.T) {
	store := setupStore(t)

	err := store.EndSession("nonexistent", "summary", 0.5, 1, 0)
	if err == nil {
		t.Error("expected error for nonexistent session, got nil")
	}
}

func TestCreateEvent(t *testing.T) {
	store := setupStore(t)

	session := &db.Session{
		ID:        "sess-evt-001",
		Project:   "my-project",
		StartedAt: time.Now().UTC(),
	}
	store.CreateSession(session)

	event := &db.Event{
		ID:        "evt-001",
		SessionID: "sess-evt-001",
		Type:      "tool_call",
		Result:    "success",
		Details:   "Called CreateRule",
		Context:   "testing",
		Timestamp: time.Now().UTC(),
	}

	if err := store.CreateEvent(event); err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}

	events, err := store.ListEvents("sess-evt-001")
	if err != nil {
		t.Fatalf("ListEvents failed: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].ID != "evt-001" {
		t.Errorf("event id = %q, want %q", events[0].ID, "evt-001")
	}
	if events[0].Type != "tool_call" {
		t.Errorf("event type = %q, want %q", events[0].Type, "tool_call")
	}
	if events[0].Result != "success" {
		t.Errorf("event result = %q, want %q", events[0].Result, "success")
	}
}

func TestListEvents(t *testing.T) {
	store := setupStore(t)

	store.CreateSession(&db.Session{
		ID: "sess-list-001", Project: "proj", StartedAt: time.Now().UTC(),
	})
	store.CreateSession(&db.Session{
		ID: "sess-list-002", Project: "proj", StartedAt: time.Now().UTC(),
	})

	// Create events for session 1
	store.CreateEvent(&db.Event{
		ID: "evt-a", SessionID: "sess-list-001", Type: "tool_call",
		Result: "success", Timestamp: time.Now().UTC(),
	})
	store.CreateEvent(&db.Event{
		ID: "evt-b", SessionID: "sess-list-001", Type: "error",
		Result: "failure", Timestamp: time.Now().UTC(),
	})

	// Create event for session 2
	store.CreateEvent(&db.Event{
		ID: "evt-c", SessionID: "sess-list-002", Type: "tool_call",
		Result: "success", Timestamp: time.Now().UTC(),
	})

	events, err := store.ListEvents("sess-list-001")
	if err != nil {
		t.Fatalf("ListEvents failed: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("expected 2 events for session 1, got %d", len(events))
	}

	events2, err := store.ListEvents("sess-list-002")
	if err != nil {
		t.Fatalf("ListEvents failed: %v", err)
	}
	if len(events2) != 1 {
		t.Errorf("expected 1 event for session 2, got %d", len(events2))
	}
}

func TestListSessions(t *testing.T) {
	store := setupStore(t)

	// Create sessions with different projects and times
	store.CreateSession(&db.Session{
		ID: "sess-ls-001", Project: "proj-a",
		StartedAt: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
	})
	store.CreateSession(&db.Session{
		ID: "sess-ls-002", Project: "proj-a",
		StartedAt: time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC),
	})
	store.CreateSession(&db.Session{
		ID: "sess-ls-003", Project: "proj-b",
		StartedAt: time.Date(2026, 1, 3, 10, 0, 0, 0, time.UTC),
	})

	// Filter by project
	sessions, err := store.ListSessions("proj-a", 10)
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}
	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions for proj-a, got %d", len(sessions))
	}

	// Verify ordering (newest first)
	if len(sessions) == 2 && sessions[0].ID != "sess-ls-002" {
		t.Errorf("expected newest session first (sess-ls-002), got %s", sessions[0].ID)
	}

	// No filter — all sessions
	all, err := store.ListSessions("", 10)
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("expected 3 sessions total, got %d", len(all))
	}

	// Test limit
	limited, err := store.ListSessions("", 2)
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}
	if len(limited) != 2 {
		t.Errorf("expected 2 sessions with limit=2, got %d", len(limited))
	}
}

func TestPurgeOldEvents(t *testing.T) {
	store := setupStore(t)

	store.CreateSession(&db.Session{
		ID: "sess-purge", Project: "proj", StartedAt: time.Now().UTC(),
	})

	// Create an old event (60 days ago)
	oldTime := time.Now().UTC().Add(-60 * 24 * time.Hour)
	store.CreateEvent(&db.Event{
		ID: "evt-old", SessionID: "sess-purge", Type: "tool_call",
		Result: "success", Timestamp: oldTime,
	})

	// Create a recent event
	store.CreateEvent(&db.Event{
		ID: "evt-new", SessionID: "sess-purge", Type: "tool_call",
		Result: "success", Timestamp: time.Now().UTC(),
	})

	// Purge events older than 30 days
	deleted, err := store.PurgeOldEvents(30)
	if err != nil {
		t.Fatalf("PurgeOldEvents failed: %v", err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 deleted event, got %d", deleted)
	}

	// Verify only the new event remains
	events, err := store.ListEvents("sess-purge")
	if err != nil {
		t.Fatalf("ListEvents failed: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 remaining event, got %d", len(events))
	}
	if events[0].ID != "evt-new" {
		t.Errorf("expected remaining event to be evt-new, got %s", events[0].ID)
	}
}

// ---------------------------------------------------------------------------
// Improvement CRUD tests (US-016)
// ---------------------------------------------------------------------------

func TestCreateImprovement(t *testing.T) {
	store := setupStore(t)

	imp := &db.Improvement{
		ID:         "imp-001",
		Content:    "Avoid: missing error handling — seen 3 times.",
		Scope:      "global",
		Evidence:   "Detected 3 occurrences",
		Confidence: 0.7,
		Status:     "pending",
	}

	if err := store.CreateImprovement(imp); err != nil {
		t.Fatalf("CreateImprovement failed: %v", err)
	}

	imps, err := store.ListImprovements("pending")
	if err != nil {
		t.Fatalf("ListImprovements failed: %v", err)
	}
	if len(imps) != 1 {
		t.Fatalf("expected 1 improvement, got %d", len(imps))
	}
	got := imps[0]
	if got.ID != "imp-001" {
		t.Errorf("id = %q, want %q", got.ID, "imp-001")
	}
	if got.Content != "Avoid: missing error handling — seen 3 times." {
		t.Errorf("content = %q", got.Content)
	}
	if got.Confidence != 0.7 {
		t.Errorf("confidence = %f, want 0.7", got.Confidence)
	}
	if got.CreatedAt.IsZero() {
		t.Error("created_at should not be zero")
	}
}

func TestListImprovements_FilterByStatus(t *testing.T) {
	store := setupStore(t)

	store.CreateImprovement(&db.Improvement{
		ID: "imp-pending-1", Content: "P1", Scope: "global",
		Evidence: "e1", Confidence: 0.5, Status: "pending",
	})
	store.CreateImprovement(&db.Improvement{
		ID: "imp-pending-2", Content: "P2", Scope: "project",
		Evidence: "e2", Confidence: 0.6, Status: "pending",
	})
	store.CreateImprovement(&db.Improvement{
		ID: "imp-applied-1", Content: "A1", Scope: "global",
		Evidence: "e3", Confidence: 0.8, Status: "applied",
	})
	store.CreateImprovement(&db.Improvement{
		ID: "imp-rejected-1", Content: "R1", Scope: "global",
		Evidence: "e4", Confidence: 0.3, Status: "rejected",
	})

	pending, err := store.ListImprovements("pending")
	if err != nil {
		t.Fatalf("ListImprovements(pending) failed: %v", err)
	}
	if len(pending) != 2 {
		t.Errorf("expected 2 pending, got %d", len(pending))
	}

	applied, err := store.ListImprovements("applied")
	if err != nil {
		t.Fatalf("ListImprovements(applied) failed: %v", err)
	}
	if len(applied) != 1 {
		t.Errorf("expected 1 applied, got %d", len(applied))
	}

	// Empty status should return all improvements
	all, err := store.ListImprovements("")
	if err != nil {
		t.Fatalf("ListImprovements(\"\") failed: %v", err)
	}
	if len(all) != 4 {
		t.Errorf("expected 4 total improvements, got %d", len(all))
	}
}

func TestUpdateImprovementStatus(t *testing.T) {
	store := setupStore(t)

	store.CreateImprovement(&db.Improvement{
		ID: "imp-upd-001", Content: "Test", Scope: "global",
		Evidence: "ev", Confidence: 0.5, Status: "pending",
	})

	// Promote to applied
	if err := store.UpdateImprovementStatus("imp-upd-001", "applied", ""); err != nil {
		t.Fatalf("UpdateImprovementStatus(applied) failed: %v", err)
	}

	applied, _ := store.ListImprovements("applied")
	if len(applied) != 1 {
		t.Fatalf("expected 1 applied, got %d", len(applied))
	}
	if applied[0].Status != "applied" {
		t.Errorf("status = %q, want applied", applied[0].Status)
	}

	// Reject another one
	store.CreateImprovement(&db.Improvement{
		ID: "imp-upd-002", Content: "Test2", Scope: "global",
		Evidence: "ev2", Confidence: 0.4, Status: "pending",
	})
	if err := store.UpdateImprovementStatus("imp-upd-002", "rejected", "not relevant"); err != nil {
		t.Fatalf("UpdateImprovementStatus(rejected) failed: %v", err)
	}

	rejected, _ := store.ListImprovements("rejected")
	if len(rejected) != 1 {
		t.Fatalf("expected 1 rejected, got %d", len(rejected))
	}
	if rejected[0].Reason != "not relevant" {
		t.Errorf("reason = %q, want 'not relevant'", rejected[0].Reason)
	}
}

func TestUpdateImprovementStatus_NotFound(t *testing.T) {
	store := setupStore(t)

	err := store.UpdateImprovementStatus("nonexistent", "applied", "")
	if err == nil {
		t.Error("expected error for nonexistent improvement, got nil")
	}
}

func TestDeprecateLowScoreRules(t *testing.T) {
	store := setupStore(t)

	// Rule with enough scores and low effectiveness → should be deprecated.
	store.CreateRule(&db.Rule{
		ID: "low-eff-rule", Content: "Bad rule", Scope: "global",
		Effectiveness: 0.2,
	})
	// Record 5 scores: all unhelpful → effectiveness stays at 0.
	for i := 0; i < 5; i++ {
		store.RecordScore("low-eff-rule", false, "not helpful", fmt.Sprintf("s%d", i))
	}

	// Rule with enough scores but high effectiveness → should NOT be deprecated.
	store.CreateRule(&db.Rule{
		ID: "high-eff-rule", Content: "Good rule", Scope: "global",
		Effectiveness: 0.9,
	})
	for i := 0; i < 5; i++ {
		store.RecordScore("high-eff-rule", true, "helpful", fmt.Sprintf("s-h%d", i))
	}

	// Rule with too few scores → should NOT be deprecated even if effectiveness is low.
	store.CreateRule(&db.Rule{
		ID: "few-scores-rule", Content: "New rule, low score", Scope: "global",
		Effectiveness: 0.1,
	})
	store.RecordScore("few-scores-rule", false, "not helpful", "s-f1")
	// Only 1 score — below minScores=5 threshold.

	// Rule already deprecated → should not appear in the returned list.
	store.CreateRule(&db.Rule{
		ID: "already-deprecated", Content: "Old rule", Scope: "global",
		Effectiveness: 0.0, Deprecated: true,
	})

	deprecated, err := store.DeprecateLowScoreRules(0.3, 5)
	if err != nil {
		t.Fatalf("DeprecateLowScoreRules failed: %v", err)
	}

	if len(deprecated) != 1 {
		t.Errorf("expected 1 deprecated rule, got %d: %v", len(deprecated), deprecated)
	}
	if len(deprecated) > 0 && deprecated[0] != "low-eff-rule" {
		t.Errorf("deprecated[0] = %q, want %q", deprecated[0], "low-eff-rule")
	}

	// Verify the rule is actually marked deprecated in the DB.
	rule, err := store.GetRule("low-eff-rule")
	if err != nil {
		t.Fatalf("GetRule failed: %v", err)
	}
	if !rule.Deprecated {
		t.Error("low-eff-rule should be marked deprecated")
	}

	// Verify the good rule is still active.
	goodRule, _ := store.GetRule("high-eff-rule")
	if goodRule.Deprecated {
		t.Error("high-eff-rule should NOT be deprecated")
	}

	// Verify the rule with too few scores is still active.
	fewRule, _ := store.GetRule("few-scores-rule")
	if fewRule.Deprecated {
		t.Error("few-scores-rule should NOT be deprecated (insufficient scores)")
	}
}
