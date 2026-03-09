package evolution

import (
	"fmt"
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

func mustCreateEvent(t *testing.T, store *db.Store, id, sessionID, typ, result, details string) {
	t.Helper()
	if err := store.CreateEvent(&db.Event{
		ID:        id,
		SessionID: sessionID,
		Type:      typ,
		Result:    result,
		Details:   details,
		Timestamp: time.Now(),
	}); err != nil {
		t.Fatalf("CreateEvent(%q): %v", id, err)
	}
}

// --- DetectPatterns tests ---

func TestDetectPatterns_RepeatedFailures(t *testing.T) {
	store := newStore(t)
	engine := NewEngine(store)

	// Create 3 sessions in same project with same failure detail across events.
	for i := 1; i <= 3; i++ {
		sid := fmt.Sprintf("sess-%d", i)
		mustCreateSession(t, store, sid, "proj-a")
		mustCreateEvent(t, store, fmt.Sprintf("evt-%d", i), sid, "tool_call", "failure", "npm install hangs on postinstall")
	}

	patterns, err := engine.DetectPatterns("", 20)
	if err != nil {
		t.Fatalf("DetectPatterns: %v", err)
	}

	found := false
	for _, p := range patterns {
		if p.Type == "repeated_failure" && p.Details == "npm install hangs on postinstall" {
			found = true
			if p.Frequency != 3 {
				t.Errorf("expected frequency 3, got %d", p.Frequency)
			}
		}
	}
	if !found {
		t.Error("expected repeated_failure pattern not found")
	}
}

func TestDetectPatterns_BelowThreshold(t *testing.T) {
	store := newStore(t)
	engine := NewEngine(store)

	// Only 2 failure events — below the threshold of 3.
	for i := 1; i <= 2; i++ {
		sid := fmt.Sprintf("sess-%d", i)
		mustCreateSession(t, store, sid, "proj-a")
		mustCreateEvent(t, store, fmt.Sprintf("evt-%d", i), sid, "tool_call", "failure", "docker build fails on arm64")
	}

	patterns, err := engine.DetectPatterns("", 20)
	if err != nil {
		t.Fatalf("DetectPatterns: %v", err)
	}

	for _, p := range patterns {
		if p.Type == "repeated_failure" && p.Details == "docker build fails on arm64" {
			t.Errorf("pattern should not be reported below threshold (frequency=%d)", p.Frequency)
		}
	}
}

func TestDetectPatterns_RecurringStuck(t *testing.T) {
	store := newStore(t)
	engine := NewEngine(store)

	// 2 stuck events across different sessions — meets the stuck threshold of 2.
	for i := 1; i <= 2; i++ {
		sid := fmt.Sprintf("sess-stuck-%d", i)
		mustCreateSession(t, store, sid, "proj-b")
		mustCreateEvent(t, store, fmt.Sprintf("evt-stuck-%d", i), sid, "stuck", "", "waiting for DB migration to complete")
	}

	patterns, err := engine.DetectPatterns("", 20)
	if err != nil {
		t.Fatalf("DetectPatterns: %v", err)
	}

	found := false
	for _, p := range patterns {
		if p.Type == "recurring_stuck" && p.Details == "waiting for DB migration to complete" {
			found = true
			if p.Frequency != 2 {
				t.Errorf("expected frequency 2, got %d", p.Frequency)
			}
		}
	}
	if !found {
		t.Error("expected recurring_stuck pattern not found")
	}
}

func TestDetectPatterns_EmptySessions(t *testing.T) {
	store := newStore(t)
	engine := NewEngine(store)

	patterns, err := engine.DetectPatterns("", 20)
	if err != nil {
		t.Fatalf("DetectPatterns: %v", err)
	}

	if len(patterns) != 0 {
		t.Errorf("expected no patterns, got %d", len(patterns))
	}
}

// --- ProposeRule tests ---

func TestProposeRule_SingleProject(t *testing.T) {
	store := newStore(t)
	engine := NewEngine(store)

	p := Pattern{
		Type:      "repeated_failure",
		Details:   "go build fails due to missing CGO",
		Frequency: 4,
		Projects:  []string{"proj-a"},
	}

	imp := engine.ProposeRule(p)
	if imp == nil {
		t.Fatal("expected non-nil improvement")
	}
	if imp.Scope != "project" {
		t.Errorf("expected scope=project, got %q", imp.Scope)
	}
	if imp.Project != "proj-a" {
		t.Errorf("expected project=proj-a, got %q", imp.Project)
	}
	if imp.Status != "pending" {
		t.Errorf("expected status=pending, got %q", imp.Status)
	}
}

func TestProposeRule_MultiProject(t *testing.T) {
	store := newStore(t)
	engine := NewEngine(store)

	p := Pattern{
		Type:      "repeated_failure",
		Details:   "redis connection timeout",
		Frequency: 6,
		Projects:  []string{"proj-a", "proj-b"},
	}

	imp := engine.ProposeRule(p)
	if imp == nil {
		t.Fatal("expected non-nil improvement")
	}
	if imp.Scope != "global" {
		t.Errorf("expected scope=global, got %q", imp.Scope)
	}
	if imp.Project != "" {
		t.Errorf("expected empty project for global rule, got %q", imp.Project)
	}
}

func TestProposeRule_SensitiveContent(t *testing.T) {
	store := newStore(t)
	engine := NewEngine(store)

	// Use a detail string that will trigger sensitivity — an email address embedded in the details.
	p := Pattern{
		Type:      "repeated_failure",
		Details:   "auth failure for user admin@example.com — token invalid",
		Frequency: 5,
		Projects:  []string{"proj-a"},
	}

	imp := engine.ProposeRule(p)
	if imp != nil {
		t.Errorf("expected nil for sensitive content, got improvement with content=%q", imp.Content)
	}
}

func TestProposeRule_ConfidenceScaling(t *testing.T) {
	store := newStore(t)
	engine := NewEngine(store)

	low := engine.ProposeRule(Pattern{
		Type: "repeated_failure", Details: "low-freq issue", Frequency: 2, Projects: []string{"p1"},
	})
	mid := engine.ProposeRule(Pattern{
		Type: "repeated_failure", Details: "mid-freq issue", Frequency: 5, Projects: []string{"p1"},
	})
	high := engine.ProposeRule(Pattern{
		Type: "repeated_failure", Details: "high-freq issue", Frequency: 10, Projects: []string{"p1"},
	})
	crossProject := engine.ProposeRule(Pattern{
		Type: "repeated_failure", Details: "cross-project issue", Frequency: 10, Projects: []string{"p1", "p2", "p3"},
	})

	if low == nil || mid == nil || high == nil || crossProject == nil {
		t.Fatal("all proposals should be non-nil")
	}

	if low.Confidence >= mid.Confidence {
		t.Errorf("low confidence (%v) should be < mid confidence (%v)", low.Confidence, mid.Confidence)
	}
	if mid.Confidence >= high.Confidence {
		t.Errorf("mid confidence (%v) should be < high confidence (%v)", mid.Confidence, high.Confidence)
	}
	if crossProject.Confidence <= high.Confidence {
		t.Errorf("crossProject confidence (%v) should be > high single-project confidence (%v)", crossProject.Confidence, high.Confidence)
	}
	if crossProject.Confidence > 1.0 {
		t.Errorf("confidence must not exceed 1.0, got %v", crossProject.Confidence)
	}
}

// --- patternTracker tests (white-box) ---

func TestPatternTracker_DeduplicatesProjects(t *testing.T) {
	tracker := &patternTracker{}

	tracker.addProject("proj-a")
	tracker.addProject("proj-a") // duplicate
	tracker.addProject("proj-b")

	if len(tracker.projects) != 2 {
		t.Errorf("expected 2 unique projects, got %d: %v", len(tracker.projects), tracker.projects)
	}
}

// ---------------------------------------------------------------------------
// Verification failure pattern tests (US-026)
// ---------------------------------------------------------------------------

func TestDetectPatterns_VerificationFailure(t *testing.T) {
	store := newStore(t)
	engine := NewEngine(store)

	// 2 sessions each with a verification failure for the same reason → pattern detected.
	for i := 1; i <= 2; i++ {
		sid := fmt.Sprintf("sess-vf-%d", i)
		mustCreateSession(t, store, sid, fmt.Sprintf("proj-%d", i))
		mustCreateEvent(t, store, fmt.Sprintf("evt-vf-%d", i), sid, "verification", "failed", "missing test coverage for auth module")
	}

	patterns, err := engine.DetectPatterns("", 20)
	if err != nil {
		t.Fatalf("DetectPatterns: %v", err)
	}

	found := false
	for _, p := range patterns {
		if p.Type == "verification_failure" && p.Details == "missing test coverage for auth module" {
			found = true
			if p.Frequency != 2 {
				t.Errorf("expected frequency 2, got %d", p.Frequency)
			}
		}
	}
	if !found {
		t.Errorf("expected verification_failure pattern not found; patterns: %+v", patterns)
	}
}

func TestDetectPatterns_VerificationFailureBelowThreshold(t *testing.T) {
	store := newStore(t)
	engine := NewEngine(store)

	// Only 1 verification failure — below the threshold of 2.
	mustCreateSession(t, store, "sess-vf-single", "proj-a")
	mustCreateEvent(t, store, "evt-vf-single", "sess-vf-single", "verification", "failed", "lint errors in handler")

	patterns, err := engine.DetectPatterns("", 20)
	if err != nil {
		t.Fatalf("DetectPatterns: %v", err)
	}

	for _, p := range patterns {
		if p.Type == "verification_failure" && p.Details == "lint errors in handler" {
			t.Errorf("pattern should not be reported below threshold (frequency=%d)", p.Frequency)
		}
	}
}

func TestProposeRule_VerificationFailure(t *testing.T) {
	store := newStore(t)
	engine := NewEngine(store)

	p := Pattern{
		Type:      "verification_failure",
		Details:   "missing test coverage for auth module",
		Frequency: 3,
		Projects:  []string{"proj-a", "proj-b"},
	}

	imp := engine.ProposeRule(p)
	if imp == nil {
		t.Fatal("expected non-nil improvement for verification_failure pattern")
	}
	if imp.Scope != "global" {
		t.Errorf("expected scope=global (2 projects), got %q", imp.Scope)
	}
	// Content should mention the reason and the verification failure context.
	if imp.Content == "" {
		t.Error("expected non-empty content")
	}
	// The content should contain "verification" keyword.
	if !strings.Contains(imp.Content, "verification") && !strings.Contains(imp.Content, "Verification") {
		t.Errorf("content should mention 'verification', got: %q", imp.Content)
	}
	// The reason should appear in the content.
	if !strings.Contains(imp.Content, "missing test coverage for auth module") {
		t.Errorf("content should include the failure reason, got: %q", imp.Content)
	}
}

