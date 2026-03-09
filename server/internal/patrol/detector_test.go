package patrol_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/bis-code/claude-toolkit/server/internal/db"
	"github.com/bis-code/claude-toolkit/server/internal/patrol"
)

func makeEvent(typ, result, details string) *db.Event {
	return &db.Event{
		ID:        fmt.Sprintf("e-%d", time.Now().UnixNano()),
		SessionID: "test-session",
		Type:      typ,
		Result:    result,
		Details:   details,
		Timestamp: time.Now().UTC(),
	}
}

func makeEvents(count int, typ, result, details string) []*db.Event {
	events := make([]*db.Event, count)
	for i := range count {
		events[i] = &db.Event{
			ID:        fmt.Sprintf("e-%d-%d", i, time.Now().UnixNano()),
			SessionID: "test-session",
			Type:      typ,
			Result:    result,
			Details:   details,
			Timestamp: time.Now().UTC(),
		}
	}
	return events
}

func TestDetectRetryLoop_Triggered(t *testing.T) {
	d := patrol.NewDetector(patrol.DefaultThresholds())
	events := makeEvents(3, "tool_call", "failure", "go build ./...")

	alerts := d.Analyze(events)

	assertHasPattern(t, alerts, "retry_loop")
	alert := findPattern(alerts, "retry_loop")
	if alert.Severity != "warning" {
		t.Errorf("expected severity warning, got %s", alert.Severity)
	}
	if alert.EventCount != 3 {
		t.Errorf("expected event_count 3, got %d", alert.EventCount)
	}
}

func TestDetectRetryLoop_BelowThreshold(t *testing.T) {
	d := patrol.NewDetector(patrol.DefaultThresholds())
	events := makeEvents(2, "tool_call", "failure", "go build ./...")

	alerts := d.Analyze(events)

	assertNoPattern(t, alerts, "retry_loop")
}

func TestDetectRetryLoop_DifferentCommands(t *testing.T) {
	d := patrol.NewDetector(patrol.DefaultThresholds())
	events := []*db.Event{
		makeEvent("tool_call", "failure", "go build ./..."),
		makeEvent("tool_call", "failure", "npm install"),
		makeEvent("tool_call", "failure", "cargo build"),
	}

	alerts := d.Analyze(events)

	assertNoPattern(t, alerts, "retry_loop")
}

func TestDetectTestSpiral_Triggered(t *testing.T) {
	d := patrol.NewDetector(patrol.DefaultThresholds())
	events := makeEvents(5, "tool_call", "failure", "go test ./pkg/auth/")

	alerts := d.Analyze(events)

	assertHasPattern(t, alerts, "test_spiral")
	alert := findPattern(alerts, "test_spiral")
	if alert.Severity != "warning" {
		t.Errorf("expected severity warning, got %s", alert.Severity)
	}
	if alert.EventCount != 5 {
		t.Errorf("expected event_count 5, got %d", alert.EventCount)
	}
}

func TestDetectTestSpiral_BelowThreshold(t *testing.T) {
	d := patrol.NewDetector(patrol.DefaultThresholds())
	events := makeEvents(4, "tool_call", "failure", "npm test -- --filter auth")

	alerts := d.Analyze(events)

	assertNoPattern(t, alerts, "test_spiral")
}

func TestDetectTestSpiral_VariousTestCommands(t *testing.T) {
	d := patrol.NewDetector(patrol.DefaultThresholds())
	testCmds := []string{"go test ./...", "npm test", "pytest tests/", "cargo test"}

	for _, cmd := range testCmds {
		t.Run(cmd, func(t *testing.T) {
			events := makeEvents(5, "tool_call", "failure", cmd)
			alerts := d.Analyze(events)
			assertHasPattern(t, alerts, "test_spiral")
		})
	}
}

func TestDetectThrashing_Triggered(t *testing.T) {
	d := patrol.NewDetector(patrol.DefaultThresholds())
	events := makeEvents(5, "tool_call", "success", "Write src/main.go")

	alerts := d.Analyze(events)

	assertHasPattern(t, alerts, "thrashing")
	alert := findPattern(alerts, "thrashing")
	if alert.Severity != "warning" {
		t.Errorf("expected severity warning, got %s", alert.Severity)
	}
	if alert.EventCount != 5 {
		t.Errorf("expected event_count 5, got %d", alert.EventCount)
	}
}

func TestDetectThrashing_BelowThreshold(t *testing.T) {
	d := patrol.NewDetector(patrol.DefaultThresholds())
	events := makeEvents(4, "tool_call", "success", "Edit src/main.go")

	alerts := d.Analyze(events)

	assertNoPattern(t, alerts, "thrashing")
}

func TestDetectStale_Triggered(t *testing.T) {
	d := patrol.NewDetector(patrol.DefaultThresholds())
	events := makeEvents(10, "tool_call", "failure", "various commands")

	alerts := d.Analyze(events)

	assertHasPattern(t, alerts, "stale")
	alert := findPattern(alerts, "stale")
	if alert.Severity != "warning" {
		t.Errorf("expected severity warning, got %s", alert.Severity)
	}
}

func TestDetectStale_BrokenBySuccess(t *testing.T) {
	d := patrol.NewDetector(patrol.DefaultThresholds())
	events := make([]*db.Event, 10)
	for i := range 10 {
		result := "failure"
		if i == 5 {
			result = "success"
		}
		events[i] = &db.Event{
			ID:        fmt.Sprintf("e-%d", i),
			SessionID: "test-session",
			Type:      "tool_call",
			Result:    result,
			Details:   fmt.Sprintf("cmd-%d", i),
			Timestamp: time.Now().UTC(),
		}
	}

	alerts := d.Analyze(events)

	assertNoPattern(t, alerts, "stale")
}

func TestDetectRework_Triggered(t *testing.T) {
	d := patrol.NewDetector(patrol.DefaultThresholds())
	events := []*db.Event{
		makeEvent("stuck", "", "blocked on auth"),
		makeEvent("blocked", "", "waiting for API"),
		makeEvent("stuck", "", "still blocked"),
	}

	alerts := d.Analyze(events)

	assertHasPattern(t, alerts, "rework")
	alert := findPattern(alerts, "rework")
	if alert.Severity != "critical" {
		t.Errorf("expected severity critical, got %s", alert.Severity)
	}
	if alert.EventCount != 3 {
		t.Errorf("expected event_count 3, got %d", alert.EventCount)
	}
}

func TestDetectRework_BelowThreshold(t *testing.T) {
	d := patrol.NewDetector(patrol.DefaultThresholds())
	events := []*db.Event{
		makeEvent("stuck", "", "blocked on auth"),
		makeEvent("blocked", "", "waiting for API"),
	}

	alerts := d.Analyze(events)

	assertNoPattern(t, alerts, "rework")
}

func TestAnalyze_MultiplePatterns(t *testing.T) {
	d := patrol.NewDetector(patrol.DefaultThresholds())
	var events []*db.Event

	// 3 retry loop failures
	events = append(events, makeEvents(3, "tool_call", "failure", "go build ./...")...)
	// 3 rework events
	events = append(events, makeEvent("stuck", "", "blocked"))
	events = append(events, makeEvent("blocked", "", "blocked"))
	events = append(events, makeEvent("stuck", "", "blocked"))

	alerts := d.Analyze(events)

	assertHasPattern(t, alerts, "retry_loop")
	assertHasPattern(t, alerts, "rework")
}

func TestAnalyze_EmptyEvents(t *testing.T) {
	d := patrol.NewDetector(patrol.DefaultThresholds())

	alerts := d.Analyze(nil)

	if len(alerts) != 0 {
		t.Errorf("expected no alerts for empty events, got %d", len(alerts))
	}

	alerts = d.Analyze([]*db.Event{})
	if len(alerts) != 0 {
		t.Errorf("expected no alerts for empty slice, got %d", len(alerts))
	}
}

func TestCustomThresholds(t *testing.T) {
	thresholds := patrol.Thresholds{
		RetryLoopCount:  2,
		TestSpiralCount: 2,
		ThrashingCount:  2,
		StaleToolCalls:  3,
		ReworkCount:     1,
	}
	d := patrol.NewDetector(thresholds)

	// 2 retry failures should now trigger (lowered from 3)
	events := makeEvents(2, "tool_call", "failure", "go build ./...")
	alerts := d.Analyze(events)
	assertHasPattern(t, alerts, "retry_loop")

	// 1 stuck event should now trigger rework (lowered from 3)
	events = []*db.Event{makeEvent("stuck", "", "blocked")}
	alerts = d.Analyze(events)
	assertHasPattern(t, alerts, "rework")
}

func TestSetThresholds(t *testing.T) {
	d := patrol.NewDetector(patrol.DefaultThresholds())

	// Default: 3 failures should not trigger retry_loop with count of 2
	events := makeEvents(2, "tool_call", "failure", "go build ./...")
	alerts := d.Analyze(events)
	assertNoPattern(t, alerts, "retry_loop")

	// Lower threshold
	d.SetThresholds(patrol.Thresholds{
		RetryLoopCount:  2,
		TestSpiralCount: 5,
		ThrashingCount:  5,
		StaleToolCalls:  10,
		ReworkCount:     3,
	})

	alerts = d.Analyze(events)
	assertHasPattern(t, alerts, "retry_loop")
}

// --- test helpers ---

func assertHasPattern(t *testing.T, alerts []patrol.Alert, pattern string) {
	t.Helper()
	for _, a := range alerts {
		if a.Pattern == pattern {
			return
		}
	}
	t.Errorf("expected alert with pattern %q, got alerts: %v", pattern, alertPatterns(alerts))
}

func assertNoPattern(t *testing.T, alerts []patrol.Alert, pattern string) {
	t.Helper()
	for _, a := range alerts {
		if a.Pattern == pattern {
			t.Errorf("did not expect alert with pattern %q, but found one", pattern)
			return
		}
	}
}

func findPattern(alerts []patrol.Alert, pattern string) patrol.Alert {
	for _, a := range alerts {
		if a.Pattern == pattern {
			return a
		}
	}
	return patrol.Alert{}
}

func alertPatterns(alerts []patrol.Alert) []string {
	patterns := make([]string, len(alerts))
	for i, a := range alerts {
		patterns[i] = a.Pattern
	}
	return patterns
}
