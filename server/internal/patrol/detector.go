package patrol

import (
	"fmt"
	"strings"

	"github.com/bis-code/claude-toolkit/server/internal/db"
)

// Alert represents a detected anti-pattern.
type Alert struct {
	Pattern    string `json:"pattern"`
	Severity   string `json:"severity"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion"`
	EventCount int    `json:"event_count"`
}

// Thresholds configures detection sensitivity.
type Thresholds struct {
	RetryLoopCount  int `json:"retry_loop_count"`
	TestSpiralCount int `json:"test_spiral_count"`
	ThrashingCount  int `json:"thrashing_count"`
	StaleToolCalls  int `json:"stale_tool_calls"`
	ReworkCount     int `json:"rework_count"`
}

// DefaultThresholds returns the default detection thresholds.
func DefaultThresholds() Thresholds {
	return Thresholds{
		RetryLoopCount:  3,
		TestSpiralCount: 5,
		ThrashingCount:  5,
		StaleToolCalls:  10,
		ReworkCount:     3,
	}
}

// Detector analyzes telemetry events for anti-patterns.
type Detector struct {
	thresholds Thresholds
}

// NewDetector creates a new patrol detector with the given thresholds.
func NewDetector(thresholds Thresholds) *Detector {
	return &Detector{thresholds: thresholds}
}

// SetThresholds updates detection thresholds.
func (d *Detector) SetThresholds(t Thresholds) {
	d.thresholds = t
}

// Analyze checks a list of events for anti-patterns and returns any detected alerts.
func (d *Detector) Analyze(events []*db.Event) []Alert {
	if len(events) == 0 {
		return nil
	}

	var alerts []Alert

	if a := d.detectRetryLoop(events); a != nil {
		alerts = append(alerts, *a)
	}
	if a := d.detectTestSpiral(events); a != nil {
		alerts = append(alerts, *a)
	}
	if a := d.detectThrashing(events); a != nil {
		alerts = append(alerts, *a)
	}
	if a := d.detectStale(events); a != nil {
		alerts = append(alerts, *a)
	}
	if a := d.detectRework(events); a != nil {
		alerts = append(alerts, *a)
	}

	return alerts
}

// detectRetryLoop finds consecutive failures with the same Details (command).
func (d *Detector) detectRetryLoop(events []*db.Event) *Alert {
	maxCount := 0
	maxDetails := ""
	currentCount := 0
	currentDetails := ""

	for _, e := range events {
		if e.Result == "failure" && e.Details == currentDetails && currentDetails != "" {
			currentCount++
		} else if e.Result == "failure" {
			currentDetails = e.Details
			currentCount = 1
		} else {
			currentDetails = ""
			currentCount = 0
		}

		if currentCount > maxCount {
			maxCount = currentCount
			maxDetails = currentDetails
		}
	}

	if maxCount >= d.thresholds.RetryLoopCount {
		return &Alert{
			Pattern:    "retry_loop",
			Severity:   "warning",
			Message:    fmt.Sprintf("Command %q failed %d consecutive times", maxDetails, maxCount),
			Suggestion: "Step back and rethink the approach instead of retrying the same command.",
			EventCount: maxCount,
		}
	}

	return nil
}

// testCommandPrefixes are prefixes that identify test runner commands.
var testCommandPrefixes = []string{
	"go test",
	"npm test",
	"npx jest",
	"npx vitest",
	"pytest",
	"cargo test",
}

// isTestCommand checks if the details string starts with a known test command prefix.
func isTestCommand(details string) bool {
	for _, prefix := range testCommandPrefixes {
		if strings.HasPrefix(details, prefix) {
			return true
		}
	}
	return false
}

// detectTestSpiral finds repeated test command failures.
func (d *Detector) detectTestSpiral(events []*db.Event) *Alert {
	// Count failures per test command
	failCounts := make(map[string]int)

	for _, e := range events {
		if e.Type == "tool_call" && e.Result == "failure" && isTestCommand(e.Details) {
			failCounts[e.Details]++
		}
	}

	for cmd, count := range failCounts {
		if count >= d.thresholds.TestSpiralCount {
			return &Alert{
				Pattern:    "test_spiral",
				Severity:   "warning",
				Message:    fmt.Sprintf("Test command %q failed %d times", cmd, count),
				Suggestion: "Stop running the same failing test. Read the error output and fix the root cause.",
				EventCount: count,
			}
		}
	}

	return nil
}

// fileEditPrefixes identify tool calls that modify files.
var fileEditPrefixes = []string{"Write ", "Edit "}

// extractFilePath returns the file path from a Write/Edit tool call detail string.
func extractFilePath(details string) (string, bool) {
	for _, prefix := range fileEditPrefixes {
		if strings.HasPrefix(details, prefix) {
			return strings.TrimPrefix(details, prefix), true
		}
	}
	return "", false
}

// detectThrashing finds the same file being edited repeatedly.
func (d *Detector) detectThrashing(events []*db.Event) *Alert {
	fileCounts := make(map[string]int)

	for _, e := range events {
		if e.Type == "tool_call" {
			if path, ok := extractFilePath(e.Details); ok {
				fileCounts[path]++
			}
		}
	}

	for file, count := range fileCounts {
		if count >= d.thresholds.ThrashingCount {
			return &Alert{
				Pattern:    "thrashing",
				Severity:   "warning",
				Message:    fmt.Sprintf("File %q was edited %d times", file, count),
				Suggestion: "Plan the changes before editing. Multiple edits to the same file suggest lack of a clear plan.",
				EventCount: count,
			}
		}
	}

	return nil
}

// detectStale checks if the last N events show no progress (all failures).
func (d *Detector) detectStale(events []*db.Event) *Alert {
	n := d.thresholds.StaleToolCalls
	if len(events) < n {
		return nil
	}

	tail := events[len(events)-n:]
	for _, e := range tail {
		if e.Result == "success" {
			return nil
		}
	}

	return &Alert{
		Pattern:    "stale",
		Severity:   "warning",
		Message:    fmt.Sprintf("No successful operations in the last %d events", n),
		Suggestion: "Take a step back. Consider a different approach or ask for help.",
		EventCount: n,
	}
}

// detectRework counts stuck/blocked events.
func (d *Detector) detectRework(events []*db.Event) *Alert {
	count := 0
	for _, e := range events {
		if e.Type == "stuck" || e.Type == "blocked" {
			count++
		}
	}

	if count >= d.thresholds.ReworkCount {
		return &Alert{
			Pattern:    "rework",
			Severity:   "critical",
			Message:    fmt.Sprintf("Session has %d stuck/blocked events", count),
			Suggestion: "The task may need to be re-scoped or broken into smaller pieces.",
			EventCount: count,
		}
	}

	return nil
}
