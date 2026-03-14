package workflow

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bis-code/claude-toolkit/server/internal/db"
)

// confidenceFromOccurrences mirrors the formula used in toolkit__record_workflow_pattern:
// starts at 0.5 on first observation, grows linearly capped at 1.0 after 10.
func confidenceFromOccurrences(n int) float64 {
	return math.Min(1.0, float64(n)/10.0)
}

// DetectedPattern is the result of analysing a slice of events.
// It is intentionally kept separate from db.WorkflowEvent so that
// LearnFromEvents remains a pure function with no DB dependency.
type DetectedPattern struct {
	Category string
	Pattern  string
	Details  string
}

// Learner analyses session events and persists workflow patterns.
type Learner struct {
	store *db.Store
}

// NewLearner creates a Learner backed by the given store.
func NewLearner(store *db.Store) *Learner {
	return &Learner{store: store}
}

// LearnFromEvents is a pure function: it analyses a slice of events and returns
// the detected patterns without touching the database.
func LearnFromEvents(events []*db.Event) []DetectedPattern {
	if len(events) == 0 {
		return nil
	}

	var patterns []DetectedPattern

	// --- coding_pattern: files_edited_together ---
	// Two or more distinct files edited inside one session.
	editedFiles := collectEditedFiles(events)
	if len(editedFiles) >= 2 {
		patterns = append(patterns, DetectedPattern{
			Category: "coding_pattern",
			Pattern:  "files_edited_together",
			Details:  fmt.Sprintf("%d files edited in session", len(editedFiles)),
		})
	}

	// --- coding_pattern: uses_table_driven_tests ---
	// Any test-file edit whose details mention table-driven keywords.
	if detectTableDrivenTests(events) {
		patterns = append(patterns, DetectedPattern{
			Category: "coding_pattern",
			Pattern:  "uses_table_driven_tests",
			Details:  "table-driven test pattern observed in test file edits",
		})
	}

	// --- coding_pattern: groups_by_feature ---
	// Files are from feature-organised paths (same directory, multiple concerns).
	if detectGroupsByFeature(editedFiles) {
		patterns = append(patterns, DetectedPattern{
			Category: "coding_pattern",
			Pattern:  "groups_by_feature",
			Details:  "files organised by feature directory rather than by type",
		})
	}

	// --- task_execution: writes_tests_first ---
	// A test file edit appears before any production file edit.
	if detectTestsFirst(events) {
		patterns = append(patterns, DetectedPattern{
			Category: "task_execution",
			Pattern:  "writes_tests_first",
			Details:  "test file edited before production file in session",
		})
	}

	// --- task_execution: commits_frequently ---
	// Multiple distinct git commit bash commands in one session.
	if countCommits(events) >= 2 {
		patterns = append(patterns, DetectedPattern{
			Category: "task_execution",
			Pattern:  "commits_frequently",
			Details:  fmt.Sprintf("%d commits in session", countCommits(events)),
		})
	}

	// --- task_execution: uses_plan_skill ---
	// A /plan skill was used (event type == "skill" with details containing "plan").
	if detectPlanSkill(events) {
		patterns = append(patterns, DetectedPattern{
			Category: "task_execution",
			Pattern:  "uses_plan_skill",
			Details:  "/plan skill invoked during session",
		})
	}

	// --- task_execution: explores_before_coding ---
	// Read/Grep events dominate the first third of the session.
	if detectExploresBeforeCoding(events) {
		patterns = append(patterns, DetectedPattern{
			Category: "task_execution",
			Pattern:  "explores_before_coding",
			Details:  "Read/Grep tool calls dominate the first third of session",
		})
	}

	// --- problem_solving: explores_with_grep_first ---
	// The first non-trivial tool call is a Grep.
	if detectGrepFirst(events) {
		patterns = append(patterns, DetectedPattern{
			Category: "problem_solving",
			Pattern:  "explores_with_grep_first",
			Details:  "Grep used as first tool in session",
		})
	}

	// --- problem_solving: reads_tests_to_understand ---
	// Test files are read before production files.
	if detectReadsTestsFirst(events) {
		patterns = append(patterns, DetectedPattern{
			Category: "problem_solving",
			Pattern:  "reads_tests_to_understand",
			Details:  "test files read before production files",
		})
	}

	// --- problem_solving: systematic_debugger ---
	// Read → (bash/tool) → Read sequence appears, suggesting hypothesis-driven debugging.
	if detectSystematicDebugger(events) {
		patterns = append(patterns, DetectedPattern{
			Category: "problem_solving",
			Pattern:  "systematic_debugger",
			Details:  "Read→action→Read pattern detected",
		})
	}

	// --- preference: prefers_conventional_commits ---
	// Commit messages match the conventional commit format.
	if detectConventionalCommits(events) {
		patterns = append(patterns, DetectedPattern{
			Category: "preference",
			Pattern:  "prefers_conventional_commits",
			Details:  "commit messages follow conventional format (type(scope): description)",
		})
	}

	return patterns
}

// AnalyzeSession loads events for the session, detects patterns, and upserts
// them into the workflow_events table. Returns the detected patterns.
func (l *Learner) AnalyzeSession(sessionID string, project string) ([]DetectedPattern, error) {
	if _, err := l.store.GetSession(sessionID); err != nil {
		return nil, fmt.Errorf("session %q not found: %w", sessionID, err)
	}

	events, err := l.store.ListEvents(sessionID)
	if err != nil {
		return nil, fmt.Errorf("list events for session %q: %w", sessionID, err)
	}

	patterns := LearnFromEvents(events)
	now := time.Now().UTC()

	for _, p := range patterns {
		existing, err := l.store.GetWorkflowEventByPatternAndProject(p.Pattern, project)
		if err != nil {
			// Best-effort: skip this pattern rather than aborting the whole analysis.
			continue
		}

		if existing != nil {
			newOccurrences := existing.Occurrences + 1
			newConfidence := confidenceFromOccurrences(newOccurrences)
			_ = l.store.UpdateWorkflowEvent(existing.ID, newOccurrences, newConfidence)
		} else {
			we := &db.WorkflowEvent{
				ID:          fmt.Sprintf("we-%d", now.UnixNano()),
				SessionID:   sessionID,
				Category:    p.Category,
				Pattern:     p.Pattern,
				Details:     p.Details,
				Confidence:  0.5,
				Occurrences: 1,
				Project:     project,
				FirstSeen:   now,
				LastSeen:    now,
			}
			_ = l.store.CreateWorkflowEvent(we)
			// Small nanosecond drift prevents ID collisions in tight loops.
			now = now.Add(time.Nanosecond)
		}
	}

	return patterns, nil
}

// SurfaceToMemory writes high-confidence patterns (> 0.7) to a workflow_profile.md
// memory file inside memoryDir. The file is created or overwritten on each call.
func (l *Learner) SurfaceToMemory(project, memoryDir string) error {
	events, err := l.store.ListWorkflowEvents(project)
	if err != nil {
		return fmt.Errorf("list workflow events: %w", err)
	}

	// Group by category, keep only high-confidence entries.
	const confidenceThreshold = 0.7
	byCategory := map[string][]db.WorkflowEvent{}
	for _, we := range events {
		if we.Confidence > confidenceThreshold {
			byCategory[we.Category] = append(byCategory[we.Category], we)
		}
	}

	if len(byCategory) == 0 {
		// Nothing to surface yet — skip file creation silently.
		return nil
	}

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString("name: workflow-profile\n")
	sb.WriteString("description: Learned developer workflow patterns from session telemetry\n")
	sb.WriteString("type: user\n")
	sb.WriteString("---\n\n")

	categoryOrder := []struct {
		key     string
		heading string
	}{
		{"coding_pattern", "Coding Style"},
		{"task_execution", "Task Execution"},
		{"problem_solving", "Problem Solving"},
		{"preference", "Preferences"},
	}

	for _, cat := range categoryOrder {
		entries := byCategory[cat.key]
		if len(entries) == 0 {
			continue
		}
		sb.WriteString("## " + cat.heading + "\n")
		for _, we := range entries {
			label := humanisePattern(we.Pattern)
			obs := we.Occurrences
			suffix := "observations"
			if obs == 1 {
				suffix = "observation"
			}
			consistency := ""
			if we.Confidence >= 0.9 {
				consistency = ", very consistent"
			} else if we.Confidence >= 0.8 {
				consistency = ", consistent"
			}
			sb.WriteString(fmt.Sprintf("- %s (%d %s%s)\n", label, obs, suffix, consistency))
		}
		sb.WriteString("\n")
	}

	if err := os.MkdirAll(memoryDir, 0o755); err != nil {
		return fmt.Errorf("create memory directory %q: %w", memoryDir, err)
	}

	dest := filepath.Join(memoryDir, "workflow_profile.md")
	if err := os.WriteFile(dest, []byte(sb.String()), 0o644); err != nil {
		return fmt.Errorf("write workflow profile: %w", err)
	}

	return nil
}

// --- detection helpers (pure, no DB) ---

func collectEditedFiles(events []*db.Event) []string {
	seen := map[string]struct{}{}
	var files []string
	for _, e := range events {
		if isEditEvent(e) && e.Details != "" {
			if _, ok := seen[e.Details]; !ok {
				seen[e.Details] = struct{}{}
				files = append(files, e.Details)
			}
		}
	}
	return files
}

func isEditEvent(e *db.Event) bool {
	t := strings.ToLower(e.Type)
	return t == "edit" || t == "write" || t == "file_edit" || t == "tool_call" && strings.Contains(strings.ToLower(e.Details), ".go")
}

func detectTableDrivenTests(events []*db.Event) bool {
	for _, e := range events {
		if isEditEvent(e) && isTestFile(e.Details) {
			d := strings.ToLower(e.Details + e.Context)
			if strings.Contains(d, "testcases") || strings.Contains(d, "test_cases") ||
				strings.Contains(d, "tt.name") || strings.Contains(d, "for _, tt") ||
				strings.Contains(d, "table") {
				return true
			}
		}
	}
	return false
}

func isTestFile(path string) bool {
	return strings.HasSuffix(path, "_test.go") || strings.Contains(path, "_test.")
}

func detectGroupsByFeature(files []string) bool {
	if len(files) < 2 {
		return false
	}
	// Heuristic: files from the same directory that serve different concerns
	// (e.g., handler.go + service.go + model.go in the same feature dir).
	dirCount := map[string]int{}
	for _, f := range files {
		dirCount[filepath.Dir(f)]++
	}
	for _, count := range dirCount {
		if count >= 2 {
			return true
		}
	}
	return false
}

func detectTestsFirst(events []*db.Event) bool {
	for _, e := range events {
		if !isEditEvent(e) {
			continue
		}
		if isTestFile(e.Details) {
			return true // test file edited; check if any prod file came before
		}
		// A production file appeared before we saw a test file.
		if !isTestFile(e.Details) && e.Details != "" {
			return false
		}
	}
	return false
}

func countCommits(events []*db.Event) int {
	count := 0
	for _, e := range events {
		if isBashEvent(e) {
			d := strings.ToLower(e.Details)
			if strings.Contains(d, "git commit") {
				count++
			}
		}
	}
	return count
}

func isBashEvent(e *db.Event) bool {
	t := strings.ToLower(e.Type)
	return t == "bash" || t == "shell" || t == "tool_call"
}

func detectPlanSkill(events []*db.Event) bool {
	for _, e := range events {
		t := strings.ToLower(e.Type)
		if (t == "skill" || t == "tool_call") &&
			strings.Contains(strings.ToLower(e.Details+e.Context), "/plan") {
			return true
		}
	}
	return false
}

func detectExploresBeforeCoding(events []*db.Event) bool {
	if len(events) < 3 {
		return false
	}
	firstThird := events[:len(events)/3]
	if len(firstThird) == 0 {
		return false
	}
	exploration := 0
	for _, e := range firstThird {
		t := strings.ToLower(e.Type)
		if t == "read" || t == "grep" || t == "search" || t == "glob" {
			exploration++
		}
	}
	return float64(exploration)/float64(len(firstThird)) > 0.5
}

func detectGrepFirst(events []*db.Event) bool {
	for _, e := range events {
		t := strings.ToLower(e.Type)
		// Skip trivial session-management events.
		if t == "session_start" || t == "start" {
			continue
		}
		return t == "grep" || t == "search"
	}
	return false
}

func detectReadsTestsFirst(events []*db.Event) bool {
	seenTestRead := false
	for _, e := range events {
		t := strings.ToLower(e.Type)
		if t != "read" {
			continue
		}
		if isTestFile(e.Details) {
			seenTestRead = true
		} else if seenTestRead {
			// Saw a prod file read after a test file read.
			return true
		}
	}
	return false
}

func detectSystematicDebugger(events []*db.Event) bool {
	// Look for Read → non-Read → Read triplet.
	state := 0 // 0=init, 1=sawRead, 2=sawAction
	for _, e := range events {
		t := strings.ToLower(e.Type)
		switch state {
		case 0:
			if t == "read" {
				state = 1
			}
		case 1:
			if t != "read" {
				state = 2
			}
		case 2:
			if t == "read" {
				return true
			}
		}
	}
	return false
}

func detectConventionalCommits(events []*db.Event) bool {
	// conventional commit: type(scope): description OR type: description
	conventionalPrefixes := []string{
		"feat:", "fix:", "refactor:", "test:", "docs:", "chore:", "build:", "ci:", "perf:", "style:", "revert:",
		"feat(", "fix(", "refactor(", "test(", "docs(", "chore(", "build(", "ci(", "perf(", "style(", "revert(",
	}
	found := 0
	total := 0
	for _, e := range events {
		if !isBashEvent(e) {
			continue
		}
		d := strings.ToLower(e.Details)
		if !strings.Contains(d, "git commit") {
			continue
		}
		total++
		for _, prefix := range conventionalPrefixes {
			if strings.Contains(d, prefix) {
				found++
				break
			}
		}
	}
	return total > 0 && found == total
}

// humanisePattern converts snake_case pattern names to readable prose.
func humanisePattern(pattern string) string {
	labels := map[string]string{
		"files_edited_together":     "Edits multiple files together in a session",
		"prefers_early_returns":     "Prefers early returns over deep nesting",
		"uses_table_driven_tests":   "Uses table-driven tests",
		"groups_by_feature":         "Groups files by feature rather than by type",
		"writes_tests_first":        "Writes tests before production code",
		"commits_frequently":        "Commits frequently within a session",
		"uses_plan_skill":           "Uses /plan skill before implementing",
		"explores_before_coding":    "Explores codebase before writing code",
		"explores_with_grep_first":  "Uses Grep as first exploration tool",
		"reads_tests_to_understand": "Reads tests to understand production code",
		"systematic_debugger":       "Follows systematic Read-hypothesize-Read debugging",
		"prefers_conventional_commits": "Uses conventional commit format",
		"prefers_monospace_formatting": "Prefers monospace formatting",
	}
	if label, ok := labels[pattern]; ok {
		return label
	}
	return strings.ReplaceAll(pattern, "_", " ")
}
