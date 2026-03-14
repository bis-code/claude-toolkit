package evolution

import (
	"fmt"
	"time"

	"github.com/bis-code/claude-toolkit/server/internal/db"
	"github.com/bis-code/claude-toolkit/server/internal/rules"
)

// Improvement represents a proposed rule from pattern detection.
type Improvement struct {
	ID         string              `json:"id"`
	Content    string              `json:"content"`
	Scope      string              `json:"scope"`             // project, global
	Project    string              `json:"project,omitempty"`
	Tags       map[string][]string `json:"tags,omitempty"`
	Evidence   string              `json:"evidence"`   // why this was proposed
	Confidence float64             `json:"confidence"` // 0.0-1.0
	Status     string              `json:"status"`     // pending, applied, rejected
	Reason     string              `json:"reason,omitempty"` // rejection reason
}

// Pattern represents a detected recurring pattern across sessions.
type Pattern struct {
	Type       string   `json:"type"`        // repeated_failure, common_fix, recurring_stuck
	Details    string   `json:"details"`     // what the pattern is
	Frequency  int      `json:"frequency"`   // how many times seen
	Projects   []string `json:"projects"`    // which projects it appeared in
	TechStacks []string `json:"tech_stacks"` // which tech stacks
}

// Engine detects patterns across sessions and proposes rules.
type Engine struct {
	store *db.Store
}

// NewEngine creates a new evolution engine.
func NewEngine(store *db.Store) *Engine {
	return &Engine{store: store}
}

// patternTracker accumulates occurrence counts and unique projects for a pattern key.
type patternTracker struct {
	count    int
	projects []string
}

func (t *patternTracker) addProject(project string) {
	for _, p := range t.projects {
		if p == project {
			return
		}
	}
	t.projects = append(t.projects, project)
}

// Insight represents a learning that was auto-detected and persisted.
type Insight struct {
	ImprovementID string `json:"improvement_id,omitempty"`
	Pattern       string `json:"pattern"`
	Message       string `json:"message"`
}

// OnEvent is called after every event is stored. It checks whether the new event
// crosses a pattern threshold and, if so, auto-proposes an improvement.
// This makes learning continuous — no manual /learn needed.
func (e *Engine) OnEvent(event *db.Event, project string) *Insight {
	if event.Details == "" {
		return nil
	}

	// Count how many times we've seen this same (type, result, details) combo
	// across recent sessions.
	count, err := e.store.CountMatchingEvents(event.Type, event.Result, event.Details, 20)
	if err != nil || count == 0 {
		return nil
	}

	// Determine threshold based on event severity.
	threshold := 3
	switch {
	case event.Type == "verification" && event.Result == "failed":
		threshold = 2
	case event.Type == "stuck" || event.Type == "blocked":
		threshold = 2
	case event.Result == "failure":
		threshold = 3
	default:
		return nil // Only learn from negative signals
	}

	if count < threshold {
		return nil
	}

	// Check if we already proposed an improvement for this exact pattern.
	existingKey := fmt.Sprintf("%s:%s:%s", event.Type, event.Result, event.Details)
	if e.store.ImprovementExistsForEvidence(existingKey) {
		return nil
	}

	// Build a pattern and propose a rule.
	patternType := "repeated_failure"
	if event.Type == "stuck" || event.Type == "blocked" {
		patternType = "recurring_stuck"
	} else if event.Type == "verification" {
		patternType = "verification_failure"
	}

	p := Pattern{
		Type:      patternType,
		Details:   event.Details,
		Frequency: count,
		Projects:  []string{project},
	}

	imp := e.ProposeRule(p)
	if imp == nil {
		return nil
	}
	imp.Evidence = existingKey

	dbImp := &db.Improvement{
		ID:         imp.ID,
		Content:    imp.Content,
		Scope:      imp.Scope,
		Project:    imp.Project,
		Tags:       imp.Tags,
		Evidence:   imp.Evidence,
		Confidence: imp.Confidence,
		Status:     "pending",
	}
	if err := e.store.CreateImprovement(dbImp); err != nil {
		return nil
	}

	return &Insight{
		ImprovementID: imp.ID,
		Pattern:       patternType,
		Message:       imp.Content,
	}
}

// OnSessionEnd runs a mini evolution cycle for the completed session.
// It also auto-deprecates any rules below threshold.
func (e *Engine) OnSessionEnd(project string) []Insight {
	patterns, err := e.DetectPatterns(project, 10)
	if err != nil || len(patterns) == 0 {
		return nil
	}

	var insights []Insight
	for _, p := range patterns {
		existingKey := fmt.Sprintf("%s:%s:%s", p.Type, "", p.Details)
		if e.store.ImprovementExistsForEvidence(existingKey) {
			continue
		}

		imp := e.ProposeRule(p)
		if imp == nil {
			continue
		}
		imp.Evidence = existingKey

		dbImp := &db.Improvement{
			ID:         imp.ID,
			Content:    imp.Content,
			Scope:      imp.Scope,
			Project:    imp.Project,
			Tags:       imp.Tags,
			Evidence:   imp.Evidence,
			Confidence: imp.Confidence,
			Status:     "pending",
		}
		if err := e.store.CreateImprovement(dbImp); err != nil {
			continue
		}
		insights = append(insights, Insight{
			ImprovementID: imp.ID,
			Pattern:       p.Type,
			Message:       imp.Content,
		})
	}

	// Auto-deprecate weak rules.
	_, _ = e.store.DeprecateLowScoreRules(0.3, 5)

	return insights
}

// DetectPatterns analyzes recent sessions to find recurring patterns.
// It looks at the last N sessions (default 20) across all projects or a specific project.
func (e *Engine) DetectPatterns(project string, sessionLimit int) ([]Pattern, error) {
	if sessionLimit <= 0 {
		sessionLimit = 20
	}

	sessions, err := e.store.ListSessions(project, sessionLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	failureCounts := map[string]*patternTracker{}
	stuckCounts := map[string]*patternTracker{}
	verificationFailureCounts := map[string]*patternTracker{}

	for _, session := range sessions {
		events, err := e.store.ListEvents(session.ID)
		if err != nil {
			// Skip sessions we can't read events for — best effort.
			continue
		}
		for _, event := range events {
			if event.Result == "failure" {
				key := event.Details
				if _, ok := failureCounts[key]; !ok {
					failureCounts[key] = &patternTracker{}
				}
				failureCounts[key].count++
				failureCounts[key].addProject(session.Project)
			}

			if event.Type == "stuck" || event.Type == "blocked" {
				key := event.Details
				if _, ok := stuckCounts[key]; !ok {
					stuckCounts[key] = &patternTracker{}
				}
				stuckCounts[key].count++
				stuckCounts[key].addProject(session.Project)
			}

			if event.Type == "verification" && event.Result == "failed" && event.Details != "" {
				key := event.Details
				if _, ok := verificationFailureCounts[key]; !ok {
					verificationFailureCounts[key] = &patternTracker{}
				}
				verificationFailureCounts[key].count++
				verificationFailureCounts[key].addProject(session.Project)
			}
		}
	}

	var patterns []Pattern

	for details, tracker := range failureCounts {
		if tracker.count >= 3 {
			patterns = append(patterns, Pattern{
				Type:      "repeated_failure",
				Details:   details,
				Frequency: tracker.count,
				Projects:  tracker.projects,
			})
		}
	}

	for details, tracker := range stuckCounts {
		if tracker.count >= 2 {
			patterns = append(patterns, Pattern{
				Type:      "recurring_stuck",
				Details:   details,
				Frequency: tracker.count,
				Projects:  tracker.projects,
			})
		}
	}

	for details, tracker := range verificationFailureCounts {
		if tracker.count >= 2 {
			patterns = append(patterns, Pattern{
				Type:      "verification_failure",
				Details:   details,
				Frequency: tracker.count,
				Projects:  tracker.projects,
			})
		}
	}

	return patterns, nil
}

// ProposeRule generates a rule proposal from a detected pattern.
// Scope-aware promotion:
//   - Pattern in 1 project  → project-scope rule
//   - Pattern across 2+ projects → global rule
func (e *Engine) ProposeRule(pattern Pattern) *Improvement {
	scope := "project"
	project := ""

	if len(pattern.Projects) >= 2 {
		scope = "global"
	} else if len(pattern.Projects) == 1 {
		project = pattern.Projects[0]
	}

	content := generateRuleContent(pattern)

	if rules.IsSensitive(content) {
		return nil
	}

	id := fmt.Sprintf("imp-%d", time.Now().UnixNano())

	return &Improvement{
		ID:         id,
		Content:    content,
		Scope:      scope,
		Project:    project,
		Tags:       buildTags(pattern),
		Evidence:   fmt.Sprintf("Detected %d occurrences across %d project(s)", pattern.Frequency, len(pattern.Projects)),
		Confidence: calculateConfidence(pattern),
		Status:     "pending",
	}
}

func generateRuleContent(p Pattern) string {
	switch p.Type {
	case "repeated_failure":
		return fmt.Sprintf("Avoid: %s — this has failed %d times. Consider an alternative approach.", p.Details, p.Frequency)
	case "recurring_stuck":
		return fmt.Sprintf("Watch out for: %s — this has caused stuck states %d times. Plan a workaround before starting.", p.Details, p.Frequency)
	case "verification_failure":
		return fmt.Sprintf("Verification check: %s. This verification has failed %d times across %d project(s). Add explicit verification step after changes.", p.Details, p.Frequency, len(p.Projects))
	default:
		return fmt.Sprintf("Pattern detected: %s (seen %d times)", p.Details, p.Frequency)
	}
}

// OnSkillScored checks if a skill is consistently underperforming and proposes improvements.
// Called after each skill score is recorded.
func (e *Engine) OnSkillScored(skill string, score float64, project string) *Insight {
	allScores, err := e.store.ListSkillScores()
	if err != nil {
		return nil
	}

	// Find the specific skill's score
	var found *db.SkillScore
	for i := range allScores {
		if allScores[i].Name == skill {
			found = &allScores[i]
			break
		}
	}
	if found == nil || found.Total < 3 {
		return nil // Not enough data
	}

	avg := found.Effectiveness

	// If skill averages below 0.5, propose improvement
	if avg < 0.5 {
		evidenceKey := fmt.Sprintf("skill_underperforming:%s:avg_%.2f", skill, avg)
		if e.store.ImprovementExistsForEvidence(evidenceKey) {
			return nil
		}

		content := fmt.Sprintf("Skill /%s is underperforming (avg score: %.2f over %d events). Review the SKILL.md definition and agent instructions for improvements.", skill, avg, found.Total)
		imp := &db.Improvement{
			ID:         fmt.Sprintf("skill-imp-%s-%d", skill, time.Now().UnixMilli()),
			Content:    content,
			Scope:      "global",
			Project:    project,
			Tags:       map[string][]string{"skill": {skill}},
			Evidence:   evidenceKey,
			Confidence: 0.6,
			Status:     "pending",
		}
		if err := e.store.CreateImprovement(imp); err != nil {
			return nil
		}
		return &Insight{
			ImprovementID: imp.ID,
			Pattern:       "skill_underperforming",
			Message:       content,
		}
	}

	return nil
}

func buildTags(p Pattern) map[string][]string {
	tags := map[string][]string{}
	if len(p.TechStacks) > 0 {
		tags["tech_stack"] = p.TechStacks
	}
	return tags
}

func calculateConfidence(p Pattern) float64 {
	base := 0.3
	if p.Frequency >= 5 {
		base = 0.5
	}
	if p.Frequency >= 10 {
		base = 0.7
	}
	if len(p.Projects) >= 3 {
		base += 0.2
	} else if len(p.Projects) >= 2 {
		base += 0.1
	}
	if base > 1.0 {
		base = 1.0
	}
	return base
}
