package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bis-code/claude-toolkit/server/internal/db"
)

// --- helpers ---

func newStore(t *testing.T) *db.Store {
	t.Helper()
	store, err := db.NewMemoryStore()
	if err != nil {
		t.Fatalf("failed to create memory store: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func newLearner(t *testing.T) (*Learner, *db.Store) {
	t.Helper()
	store := newStore(t)
	return NewLearner(store), store
}

func makeEvent(typ, result, details, context string, ts time.Time) *db.Event {
	return &db.Event{
		ID:        "e-" + typ + "-" + details,
		SessionID: "sess-1",
		Type:      typ,
		Result:    result,
		Details:   details,
		Context:   context,
		Timestamp: ts,
	}
}

func mustCreateSession(t *testing.T, store *db.Store, id, project string) {
	t.Helper()
	if err := store.CreateSession(&db.Session{
		ID:        id,
		Project:   project,
		StartedAt: time.Now(),
	}); err != nil {
		t.Fatalf("CreateSession(%q): %v", id, err)
	}
}

func mustCreateEvent(t *testing.T, store *db.Store, e *db.Event) {
	t.Helper()
	if err := store.CreateEvent(e); err != nil {
		t.Fatalf("CreateEvent(%q): %v", e.ID, err)
	}
}

// hasPattern returns true when any DetectedPattern has the given pattern name.
func hasPattern(patterns []DetectedPattern, name string) bool {
	for _, p := range patterns {
		if p.Pattern == name {
			return true
		}
	}
	return false
}

// --- LearnFromEvents tests ---

func TestLearnFromEvents_EmptyEvents(t *testing.T) {
	patterns := LearnFromEvents(nil)
	if len(patterns) != 0 {
		t.Errorf("expected no patterns for nil input, got %d", len(patterns))
	}

	patterns = LearnFromEvents([]*db.Event{})
	if len(patterns) != 0 {
		t.Errorf("expected no patterns for empty input, got %d", len(patterns))
	}
}

func TestLearnFromEvents_DetectsTestsFirst(t *testing.T) {
	now := time.Now()
	events := []*db.Event{
		makeEvent("edit", "success", "internal/auth/handler_test.go", "", now),
		makeEvent("edit", "success", "internal/auth/handler.go", "", now.Add(time.Second)),
	}

	patterns := LearnFromEvents(events)
	if !hasPattern(patterns, "writes_tests_first") {
		t.Errorf("expected writes_tests_first pattern; got %+v", patterns)
	}
}

func TestLearnFromEvents_NoTestsFirst_WhenProdComesFirst(t *testing.T) {
	now := time.Now()
	events := []*db.Event{
		makeEvent("edit", "success", "internal/auth/handler.go", "", now),
		makeEvent("edit", "success", "internal/auth/handler_test.go", "", now.Add(time.Second)),
	}

	patterns := LearnFromEvents(events)
	if hasPattern(patterns, "writes_tests_first") {
		t.Error("writes_tests_first should not be detected when production file comes first")
	}
}

func TestLearnFromEvents_DetectsGrepFirst(t *testing.T) {
	now := time.Now()
	events := []*db.Event{
		makeEvent("grep", "success", "pattern", "", now),
		makeEvent("read", "success", "some/file.go", "", now.Add(time.Second)),
	}

	patterns := LearnFromEvents(events)
	if !hasPattern(patterns, "explores_with_grep_first") {
		t.Errorf("expected explores_with_grep_first pattern; got %+v", patterns)
	}
}

func TestLearnFromEvents_GrepFirst_SkipsSessionStartEvents(t *testing.T) {
	now := time.Now()
	events := []*db.Event{
		makeEvent("session_start", "", "", "", now),
		makeEvent("grep", "success", "pattern", "", now.Add(time.Second)),
	}

	patterns := LearnFromEvents(events)
	if !hasPattern(patterns, "explores_with_grep_first") {
		t.Errorf("expected explores_with_grep_first after skipping session_start; got %+v", patterns)
	}
}

func TestLearnFromEvents_DetectsFilesEditedTogether(t *testing.T) {
	now := time.Now()
	events := []*db.Event{
		makeEvent("edit", "success", "pkg/user/handler.go", "", now),
		makeEvent("edit", "success", "pkg/user/service.go", "", now.Add(time.Second)),
	}

	patterns := LearnFromEvents(events)
	if !hasPattern(patterns, "files_edited_together") {
		t.Errorf("expected files_edited_together; got %+v", patterns)
	}
}

func TestLearnFromEvents_DetectsCommitsFrequently(t *testing.T) {
	now := time.Now()
	events := []*db.Event{
		makeEvent("bash", "success", "git commit -m 'feat: add handler'", "", now),
		makeEvent("bash", "success", "git commit -m 'fix: edge case'", "", now.Add(time.Minute)),
	}

	patterns := LearnFromEvents(events)
	if !hasPattern(patterns, "commits_frequently") {
		t.Errorf("expected commits_frequently; got %+v", patterns)
	}
}

func TestLearnFromEvents_NoCommitsFrequently_WhenSingleCommit(t *testing.T) {
	now := time.Now()
	events := []*db.Event{
		makeEvent("bash", "success", "git commit -m 'feat: initial'", "", now),
	}

	patterns := LearnFromEvents(events)
	if hasPattern(patterns, "commits_frequently") {
		t.Error("commits_frequently should not be detected for a single commit")
	}
}

func TestLearnFromEvents_DetectsConventionalCommits(t *testing.T) {
	now := time.Now()
	events := []*db.Event{
		makeEvent("bash", "success", "git commit -m 'feat(auth): add login endpoint'", "", now),
		makeEvent("bash", "success", "git commit -m 'fix(db): handle null session'", "", now.Add(time.Minute)),
	}

	patterns := LearnFromEvents(events)
	if !hasPattern(patterns, "prefers_conventional_commits") {
		t.Errorf("expected prefers_conventional_commits; got %+v", patterns)
	}
}

func TestLearnFromEvents_DetectsSystematicDebugger(t *testing.T) {
	now := time.Now()
	events := []*db.Event{
		makeEvent("read", "success", "internal/auth/handler.go", "", now),
		makeEvent("bash", "success", "go test ./...", "", now.Add(time.Second)),
		makeEvent("read", "success", "internal/auth/service.go", "", now.Add(2*time.Second)),
	}

	patterns := LearnFromEvents(events)
	if !hasPattern(patterns, "systematic_debugger") {
		t.Errorf("expected systematic_debugger; got %+v", patterns)
	}
}

func TestLearnFromEvents_DetectsReadsTestsToUnderstand(t *testing.T) {
	now := time.Now()
	events := []*db.Event{
		makeEvent("read", "success", "internal/auth/handler_test.go", "", now),
		makeEvent("read", "success", "internal/auth/handler.go", "", now.Add(time.Second)),
	}

	patterns := LearnFromEvents(events)
	if !hasPattern(patterns, "reads_tests_to_understand") {
		t.Errorf("expected reads_tests_to_understand; got %+v", patterns)
	}
}

func TestLearnFromEvents_DetectsGroupsByFeature(t *testing.T) {
	now := time.Now()
	events := []*db.Event{
		makeEvent("edit", "success", "internal/user/handler.go", "", now),
		makeEvent("edit", "success", "internal/user/service.go", "", now.Add(time.Second)),
	}

	patterns := LearnFromEvents(events)
	if !hasPattern(patterns, "groups_by_feature") {
		t.Errorf("expected groups_by_feature; got %+v", patterns)
	}
}

func TestLearnFromEvents_DetectsExploresBeforeCoding(t *testing.T) {
	now := time.Now()
	// 6 events: first 2 are exploration (> 50% of first third = first 2)
	events := []*db.Event{
		makeEvent("read", "success", "some/file.go", "", now),
		makeEvent("grep", "success", "pattern", "", now.Add(time.Second)),
		makeEvent("read", "success", "another/file.go", "", now.Add(2*time.Second)),
		makeEvent("edit", "success", "src/handler.go", "", now.Add(3*time.Second)),
		makeEvent("edit", "success", "src/service.go", "", now.Add(4*time.Second)),
		makeEvent("bash", "success", "go test ./...", "", now.Add(5*time.Second)),
	}

	patterns := LearnFromEvents(events)
	if !hasPattern(patterns, "explores_before_coding") {
		t.Errorf("expected explores_before_coding; got %+v", patterns)
	}
}

func TestLearnFromEvents_DetectsPlanSkill(t *testing.T) {
	now := time.Now()
	events := []*db.Event{
		makeEvent("skill", "success", "/plan", "", now),
		makeEvent("edit", "success", "src/handler.go", "", now.Add(time.Second)),
	}

	patterns := LearnFromEvents(events)
	if !hasPattern(patterns, "uses_plan_skill") {
		t.Errorf("expected uses_plan_skill; got %+v", patterns)
	}
}

// --- AnalyzeSession tests ---

func TestAnalyzeSession_PersistsPatterns(t *testing.T) {
	learner, store := newLearner(t)

	mustCreateSession(t, store, "sess-1", "proj-a")
	now := time.Now()
	mustCreateEvent(t, store, &db.Event{
		ID:        "e-1",
		SessionID: "sess-1",
		Type:      "grep",
		Result:    "success",
		Details:   "some pattern",
		Timestamp: now,
	})
	mustCreateEvent(t, store, &db.Event{
		ID:        "e-2",
		SessionID: "sess-1",
		Type:      "read",
		Result:    "success",
		Details:   "some/file.go",
		Timestamp: now.Add(time.Second),
	})

	patterns, err := learner.AnalyzeSession("sess-1", "proj-a")
	if err != nil {
		t.Fatalf("AnalyzeSession: %v", err)
	}

	if !hasPattern(patterns, "explores_with_grep_first") {
		t.Errorf("expected explores_with_grep_first in returned patterns; got %+v", patterns)
	}

	// Verify persisted to DB.
	stored, err := store.ListWorkflowEvents("proj-a")
	if err != nil {
		t.Fatalf("ListWorkflowEvents: %v", err)
	}
	found := false
	for _, we := range stored {
		if we.Pattern == "explores_with_grep_first" {
			found = true
			if we.Occurrences != 1 {
				t.Errorf("expected occurrences=1, got %d", we.Occurrences)
			}
			if we.Confidence != 0.5 {
				t.Errorf("expected initial confidence=0.5, got %f", we.Confidence)
			}
		}
	}
	if !found {
		t.Error("explores_with_grep_first not persisted to workflow_events")
	}
}

func TestAnalyzeSession_UpdatesConfidenceOnRepeat(t *testing.T) {
	learner, store := newLearner(t)

	buildSession := func(sessionID string, ts time.Time) {
		mustCreateSession(t, store, sessionID, "proj-a")
		mustCreateEvent(t, store, &db.Event{
			ID:        "e-grep-" + sessionID,
			SessionID: sessionID,
			Type:      "grep",
			Result:    "success",
			Details:   "pattern",
			Timestamp: ts,
		})
		mustCreateEvent(t, store, &db.Event{
			ID:        "e-read-" + sessionID,
			SessionID: sessionID,
			Type:      "read",
			Result:    "success",
			Details:   "some/file.go",
			Timestamp: ts.Add(time.Second),
		})
	}

	now := time.Now()
	buildSession("sess-a", now)
	_, err := learner.AnalyzeSession("sess-a", "proj-a")
	if err != nil {
		t.Fatalf("first AnalyzeSession: %v", err)
	}

	buildSession("sess-b", now.Add(time.Minute))
	_, err = learner.AnalyzeSession("sess-b", "proj-a")
	if err != nil {
		t.Fatalf("second AnalyzeSession: %v", err)
	}

	stored, err := store.ListWorkflowEvents("proj-a")
	if err != nil {
		t.Fatalf("ListWorkflowEvents: %v", err)
	}

	for _, we := range stored {
		if we.Pattern == "explores_with_grep_first" {
			if we.Occurrences != 2 {
				t.Errorf("expected occurrences=2, got %d", we.Occurrences)
			}
			expected := confidenceFromOccurrences(2) // 0.2
			if we.Confidence != expected {
				t.Errorf("expected confidence=%f, got %f", expected, we.Confidence)
			}
			return
		}
	}
	t.Error("explores_with_grep_first not found after two sessions")
}

func TestAnalyzeSession_UnknownSession_ReturnsError(t *testing.T) {
	learner, _ := newLearner(t)

	_, err := learner.AnalyzeSession("nonexistent-session", "proj-a")
	if err == nil {
		t.Error("expected error for nonexistent session, got nil")
	}
}

// --- SurfaceToMemory tests ---

func TestSurfaceToMemory_WritesFile(t *testing.T) {
	learner, store := newLearner(t)

	// Insert a high-confidence workflow event directly.
	now := time.Now().UTC()
	err := store.CreateWorkflowEvent(&db.WorkflowEvent{
		ID:          "we-test-1",
		SessionID:   "sess-1",
		Category:    "task_execution",
		Pattern:     "writes_tests_first",
		Details:     "test file edited before production file",
		Confidence:  0.9,
		Occurrences: 9,
		Project:     "proj-a",
		FirstSeen:   now,
		LastSeen:    now,
	})
	if err != nil {
		t.Fatalf("CreateWorkflowEvent: %v", err)
	}

	dir := t.TempDir()
	if err := learner.SurfaceToMemory("proj-a", dir); err != nil {
		t.Fatalf("SurfaceToMemory: %v", err)
	}

	dest := filepath.Join(dir, "workflow_profile.md")
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("expected workflow_profile.md to exist: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "name: workflow-profile") {
		t.Error("missing frontmatter name field")
	}
	if !strings.Contains(content, "Task Execution") {
		t.Error("missing Task Execution section")
	}
	if !strings.Contains(content, "Writes tests before production code") {
		t.Error("missing writes_tests_first human label in profile")
	}
	if !strings.Contains(content, "9 observations") {
		t.Error("missing occurrence count")
	}
	if !strings.Contains(content, "very consistent") {
		t.Error("expected 'very consistent' for confidence >= 0.9")
	}
}

func TestSurfaceToMemory_SkipsLowConfidencePatterns(t *testing.T) {
	learner, store := newLearner(t)

	now := time.Now().UTC()
	// Low confidence — should not appear in profile.
	err := store.CreateWorkflowEvent(&db.WorkflowEvent{
		ID:          "we-low-1",
		SessionID:   "sess-1",
		Category:    "preference",
		Pattern:     "prefers_conventional_commits",
		Details:     "test",
		Confidence:  0.5,
		Occurrences: 5,
		Project:     "proj-a",
		FirstSeen:   now,
		LastSeen:    now,
	})
	if err != nil {
		t.Fatalf("CreateWorkflowEvent: %v", err)
	}

	dir := t.TempDir()
	// Should succeed but not create a file (nothing above threshold).
	if err := learner.SurfaceToMemory("proj-a", dir); err != nil {
		t.Fatalf("SurfaceToMemory: %v", err)
	}

	dest := filepath.Join(dir, "workflow_profile.md")
	if _, statErr := os.Stat(dest); statErr == nil {
		t.Error("expected workflow_profile.md NOT to be created when no patterns exceed threshold")
	}
}

func TestSurfaceToMemory_CreatesDirectoryIfMissing(t *testing.T) {
	learner, store := newLearner(t)

	now := time.Now().UTC()
	err := store.CreateWorkflowEvent(&db.WorkflowEvent{
		ID:          "we-dir-1",
		SessionID:   "sess-1",
		Category:    "coding_pattern",
		Pattern:     "uses_table_driven_tests",
		Details:     "test",
		Confidence:  0.8,
		Occurrences: 8,
		Project:     "proj-b",
		FirstSeen:   now,
		LastSeen:    now,
	})
	if err != nil {
		t.Fatalf("CreateWorkflowEvent: %v", err)
	}

	dir := filepath.Join(t.TempDir(), "nested", "memory")
	if err := learner.SurfaceToMemory("proj-b", dir); err != nil {
		t.Fatalf("SurfaceToMemory: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "workflow_profile.md")); err != nil {
		t.Errorf("expected workflow_profile.md in created directory: %v", err)
	}
}

func TestSurfaceToMemory_EmptyProject_NoFile(t *testing.T) {
	learner, _ := newLearner(t)

	dir := t.TempDir()
	if err := learner.SurfaceToMemory("unknown-project", dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dest := filepath.Join(dir, "workflow_profile.md")
	if _, err := os.Stat(dest); err == nil {
		t.Error("expected no file for project with no patterns")
	}
}
