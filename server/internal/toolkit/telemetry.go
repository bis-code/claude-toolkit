package toolkit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bis-code/claude-toolkit/server/internal/db"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerTelemetryTools registers all telemetry-related MCP tools.
func (h *handlers) registerTelemetryTools(s *server.MCPServer) {
	s.AddTool(
		mcp.NewTool("toolkit__start_session",
			mcp.WithDescription("Start a new telemetry session"),
			mcp.WithString("session_id",
				mcp.Required(),
				mcp.Description("Unique session identifier"),
			),
			mcp.WithString("project",
				mcp.Required(),
				mcp.Description("Project name"),
			),
		),
		h.handleStartSession,
	)

	s.AddTool(
		mcp.NewTool("toolkit__log_event",
			mcp.WithDescription("Log a telemetry event to the current session"),
			mcp.WithString("session_id",
				mcp.Required(),
				mcp.Description("Session ID"),
			),
			mcp.WithString("type",
				mcp.Required(),
				mcp.Description("Event type (e.g., tool_call, test_run, compile, edit)"),
			),
			mcp.WithString("result",
				mcp.Description("Event result (success, failure, error)"),
			),
			mcp.WithString("details",
				mcp.Description("Event details"),
			),
			mcp.WithString("context",
				mcp.Description("Additional context"),
			),
			mcp.WithString("project",
				mcp.Description("Project name (used when auto-creating the session)"),
			),
		),
		h.handleLogEvent,
	)

	s.AddTool(
		mcp.NewTool("toolkit__log_stuck",
			mcp.WithDescription("Log a stuck event with problem details and hypothesis"),
			mcp.WithString("session_id",
				mcp.Required(),
				mcp.Description("Session ID"),
			),
			mcp.WithString("problem",
				mcp.Required(),
				mcp.Description("What is stuck"),
			),
			mcp.WithNumber("attempts",
				mcp.Description("Number of attempts made"),
			),
			mcp.WithString("hypothesis",
				mcp.Description("Current hypothesis for the issue"),
			),
			mcp.WithString("project",
				mcp.Description("Project name (used when auto-creating the session)"),
			),
		),
		h.handleLogStuck,
	)

	s.AddTool(
		mcp.NewTool("toolkit__log_blocked",
			mcp.WithDescription("Log a blocked event with what was tried and why it failed"),
			mcp.WithString("session_id",
				mcp.Required(),
				mcp.Description("Session ID"),
			),
			mcp.WithString("problem",
				mcp.Required(),
				mcp.Description("What is blocked"),
			),
			mcp.WithString("tried",
				mcp.Description("What was tried"),
			),
			mcp.WithString("failed_because",
				mcp.Description("Why it failed"),
			),
			mcp.WithString("hypothesis",
				mcp.Description("Hypothesis for resolution"),
			),
			mcp.WithString("project",
				mcp.Description("Project name (used when auto-creating the session)"),
			),
		),
		h.handleLogBlocked,
	)

	s.AddTool(
		mcp.NewTool("toolkit__end_session",
			mcp.WithDescription("End the current session with summary and outcome metrics"),
			mcp.WithString("session_id",
				mcp.Required(),
				mcp.Description("Session ID to end"),
			),
			mcp.WithString("summary",
				mcp.Description("Session summary"),
			),
			mcp.WithNumber("confidence",
				mcp.Description("Confidence score 0.0-1.0"),
			),
			mcp.WithNumber("tasks_completed",
				mcp.Description("Number of tasks completed"),
			),
			mcp.WithNumber("tasks_failed",
				mcp.Description("Number of tasks failed"),
			),
		),
		h.handleEndSession,
	)

	s.AddTool(
		mcp.NewTool("toolkit__get_session_history",
			mcp.WithDescription("Get session history, optionally filtered by project"),
			mcp.WithString("project",
				mcp.Description("Filter by project name"),
			),
			mcp.WithNumber("limit",
				mcp.Description("Maximum number of sessions to return (default: 20)"),
			),
		),
		h.handleGetSessionHistory,
	)

	s.AddTool(
		mcp.NewTool("toolkit__get_project_stats",
			mcp.WithDescription("Get aggregate statistics for a project"),
			mcp.WithString("project",
				mcp.Required(),
				mcp.Description("Project name"),
			),
		),
		h.handleGetProjectStats,
	)
}

// handleStartSession explicitly creates a new session.
func (h *handlers) handleStartSession(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID := req.GetString("session_id", "")
	project := req.GetString("project", "")

	if sessionID == "" || project == "" {
		return mcp.NewToolResultError("session_id and project are required"), nil
	}

	// Reject duplicate session IDs to avoid silent data loss.
	if _, err := h.store.GetSession(sessionID); err == nil {
		return mcp.NewToolResultError(fmt.Sprintf("session %q already exists", sessionID)), nil
	}

	sess := &db.Session{
		ID:        sessionID,
		Project:   project,
		StartedAt: time.Now().UTC(),
	}
	if err := h.store.CreateSession(sess); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create session: %v", err)), nil
	}

	result, _ := json.Marshal(map[string]interface{}{
		"id":      sessionID,
		"message": "session started",
	})
	return mcp.NewToolResultText(string(result)), nil
}

// handleLogEvent logs a generic telemetry event, auto-creating the session if needed.
func (h *handlers) handleLogEvent(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID := req.GetString("session_id", "")
	eventType := req.GetString("type", "")

	if sessionID == "" || eventType == "" {
		return mcp.NewToolResultError("session_id and type are required"), nil
	}

	if err := h.ensureSession(sessionID, req.GetString("project", "unknown")); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to ensure session: %v", err)), nil
	}

	eventID := fmt.Sprintf("e-%d", time.Now().UnixNano())
	event := &db.Event{
		ID:        eventID,
		SessionID: sessionID,
		Type:      eventType,
		Result:    req.GetString("result", ""),
		Details:   req.GetString("details", ""),
		Context:   req.GetString("context", ""),
		Timestamp: time.Now().UTC(),
	}
	if err := h.store.CreateEvent(event); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create event: %v", err)), nil
	}

	result, _ := json.Marshal(map[string]interface{}{
		"id":      eventID,
		"message": "event logged",
	})
	return mcp.NewToolResultText(string(result)), nil
}

// handleLogStuck logs a stuck event with structured problem details.
func (h *handlers) handleLogStuck(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID := req.GetString("session_id", "")
	problem := req.GetString("problem", "")

	if sessionID == "" || problem == "" {
		return mcp.NewToolResultError("session_id and problem are required"), nil
	}

	if err := h.ensureSession(sessionID, req.GetString("project", "unknown")); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to ensure session: %v", err)), nil
	}

	attempts := int(req.GetFloat("attempts", 0))
	hypothesis := req.GetString("hypothesis", "")
	details := fmt.Sprintf("problem: %s\nattempts: %d\nhypothesis: %s", problem, attempts, hypothesis)

	eventID := fmt.Sprintf("e-%d", time.Now().UnixNano())
	event := &db.Event{
		ID:        eventID,
		SessionID: sessionID,
		Type:      "stuck",
		Result:    "stuck",
		Details:   details,
		Timestamp: time.Now().UTC(),
	}
	if err := h.store.CreateEvent(event); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create stuck event: %v", err)), nil
	}

	result, _ := json.Marshal(map[string]interface{}{
		"id":      eventID,
		"message": "stuck event logged",
	})
	return mcp.NewToolResultText(string(result)), nil
}

// handleLogBlocked logs a blocked event with structured escalation template.
func (h *handlers) handleLogBlocked(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID := req.GetString("session_id", "")
	problem := req.GetString("problem", "")

	if sessionID == "" || problem == "" {
		return mcp.NewToolResultError("session_id and problem are required"), nil
	}

	if err := h.ensureSession(sessionID, req.GetString("project", "unknown")); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to ensure session: %v", err)), nil
	}

	tried := req.GetString("tried", "")
	failedBecause := req.GetString("failed_because", "")
	hypothesis := req.GetString("hypothesis", "")

	details := fmt.Sprintf("problem: %s\ntried: %s\nfailed_because: %s\nhypothesis: %s",
		problem, tried, failedBecause, hypothesis)
	blockedTemplate := fmt.Sprintf("Problem: %s\nTried: %s\nFailed because: %s\nHypothesis: %s",
		problem, tried, failedBecause, hypothesis)

	eventID := fmt.Sprintf("e-%d", time.Now().UnixNano())
	event := &db.Event{
		ID:        eventID,
		SessionID: sessionID,
		Type:      "blocked",
		Result:    "blocked",
		Details:   details,
		Timestamp: time.Now().UTC(),
	}
	if err := h.store.CreateEvent(event); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create blocked event: %v", err)), nil
	}

	result, _ := json.Marshal(map[string]interface{}{
		"id":               eventID,
		"message":          "blocked event logged",
		"blocked_template": blockedTemplate,
	})
	return mcp.NewToolResultText(string(result)), nil
}

// handleEndSession closes a session with summary metrics.
func (h *handlers) handleEndSession(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID := req.GetString("session_id", "")
	if sessionID == "" {
		return mcp.NewToolResultError("session_id is required"), nil
	}

	summary := req.GetString("summary", "")
	confidence := req.GetFloat("confidence", 0)
	tasksCompleted := int(req.GetFloat("tasks_completed", 0))
	tasksFailed := int(req.GetFloat("tasks_failed", 0))

	if err := h.store.EndSession(sessionID, summary, confidence, tasksCompleted, tasksFailed); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to end session: %v", err)), nil
	}

	result, _ := json.Marshal(map[string]interface{}{
		"message": "session ended",
	})
	return mcp.NewToolResultText(string(result)), nil
}

// handleGetSessionHistory returns sessions optionally filtered by project.
func (h *handlers) handleGetSessionHistory(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	project := req.GetString("project", "")
	limit := int(req.GetFloat("limit", 20))
	if limit <= 0 {
		limit = 20
	}

	sessions, err := h.store.ListSessions(project, limit)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list sessions: %v", err)), nil
	}

	// Normalise nil slice to empty slice so JSON encodes as [].
	if sessions == nil {
		sessions = []*db.Session{}
	}

	result, err := json.Marshal(map[string]interface{}{
		"sessions": sessions,
		"count":    len(sessions),
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal sessions: %v", err)), nil
	}
	return mcp.NewToolResultText(string(result)), nil
}

// handleGetProjectStats returns aggregate statistics for a project.
func (h *handlers) handleGetProjectStats(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	project := req.GetString("project", "")
	if project == "" {
		return mcp.NewToolResultError("project is required"), nil
	}

	sessions, err := h.store.ListSessions(project, 0)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list sessions: %v", err)), nil
	}

	totalSessions := len(sessions)
	totalEvents := 0
	eventsByType := make(map[string]int)
	totalConfidence := 0.0
	totalTasksCompleted := 0
	totalTasksFailed := 0

	for _, sess := range sessions {
		totalTasksCompleted += sess.TasksCompleted
		totalTasksFailed += sess.TasksFailed
		totalConfidence += sess.Confidence

		events, err := h.store.ListEvents(sess.ID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to list events for session %s: %v", sess.ID, err)), nil
		}
		totalEvents += len(events)
		for _, e := range events {
			eventsByType[e.Type]++
		}
	}

	avgConfidence := 0.0
	if totalSessions > 0 {
		avgConfidence = totalConfidence / float64(totalSessions)
	}

	result, err := json.Marshal(map[string]interface{}{
		"project":               project,
		"total_sessions":        totalSessions,
		"total_events":          totalEvents,
		"events_by_type":        eventsByType,
		"avg_confidence":        avgConfidence,
		"total_tasks_completed": totalTasksCompleted,
		"total_tasks_failed":    totalTasksFailed,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal stats: %v", err)), nil
	}
	return mcp.NewToolResultText(string(result)), nil
}

// ensureSession checks whether the session exists and creates it if not.
// This allows callers to log events without explicitly starting a session first.
func (h *handlers) ensureSession(sessionID, project string) error {
	if _, err := h.store.GetSession(sessionID); err == nil {
		// Session already exists — nothing to do.
		return nil
	}

	// Auto-create with the provided project (or "unknown" as default).
	if project == "" {
		project = "unknown"
	}
	return h.store.CreateSession(&db.Session{
		ID:        sessionID,
		Project:   project,
		StartedAt: time.Now().UTC(),
	})
}
