package toolkit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerPatrolTools registers all patrol-related MCP tools.
func (h *handlers) registerPatrolTools(s *server.MCPServer) {
	s.AddTool(
		mcp.NewTool("toolkit__patrol_check",
			mcp.WithDescription("Check a session for anti-patterns (retry loops, test spirals, thrashing, staleness, rework)"),
			mcp.WithString("session_id",
				mcp.Required(),
				mcp.Description("Session ID to check"),
			),
		),
		h.handlePatrolCheck,
	)

	s.AddTool(
		mcp.NewTool("toolkit__configure_patrol",
			mcp.WithDescription("Configure patrol detection thresholds"),
			mcp.WithNumber("retry_loop_count",
				mcp.Description("Consecutive same-command failures to trigger (default: 3)"),
			),
			mcp.WithNumber("test_spiral_count",
				mcp.Description("Same test failures to trigger (default: 5)"),
			),
			mcp.WithNumber("thrashing_count",
				mcp.Description("Same file edits to trigger (default: 5)"),
			),
			mcp.WithNumber("stale_tool_calls",
				mcp.Description("Failed tool calls for staleness (default: 10)"),
			),
			mcp.WithNumber("rework_count",
				mcp.Description("Stuck/blocked events for rework (default: 3)"),
			),
		),
		h.handleConfigurePatrol,
	)
}

// handlePatrolCheck analyzes a session's events for anti-patterns and returns
// a structured report. Status is "healthy" when no alerts are found, "critical"
// if any alert has severity "critical", and "warning" otherwise.
func (h *handlers) handlePatrolCheck(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID := req.GetString("session_id", "")
	if sessionID == "" {
		return mcp.NewToolResultError("session_id is required"), nil
	}

	events, err := h.store.ListEvents(sessionID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list events: %v", err)), nil
	}

	alerts := h.detector.Analyze(events)

	if len(alerts) == 0 {
		result, _ := json.Marshal(map[string]interface{}{
			"status":  "healthy",
			"message": "No anti-patterns detected",
		})
		return mcp.NewToolResultText(string(result)), nil
	}

	status := "warning"
	for _, a := range alerts {
		if a.Severity == "critical" {
			status = "critical"
			break
		}
	}

	// Collect suggestions; if there is a rework alert surface its blocked template hint.
	var suggestion string
	for _, a := range alerts {
		if a.Pattern == "rework" {
			suggestion = a.Suggestion
			break
		}
	}
	if suggestion == "" && len(alerts) > 0 {
		suggestion = alerts[0].Suggestion
	}

	result, err := json.Marshal(map[string]interface{}{
		"status":     status,
		"alerts":     alerts,
		"suggestion": suggestion,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(result)), nil
}

// handleConfigurePatrol updates patrol detection thresholds. Only non-zero values
// in the request override the existing threshold, preserving defaults for fields
// that are not supplied.
func (h *handlers) handleConfigurePatrol(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	current := h.detector.Thresholds()

	if v := int(req.GetFloat("retry_loop_count", 0)); v > 0 {
		current.RetryLoopCount = v
	}
	if v := int(req.GetFloat("test_spiral_count", 0)); v > 0 {
		current.TestSpiralCount = v
	}
	if v := int(req.GetFloat("thrashing_count", 0)); v > 0 {
		current.ThrashingCount = v
	}
	if v := int(req.GetFloat("stale_tool_calls", 0)); v > 0 {
		current.StaleToolCalls = v
	}
	if v := int(req.GetFloat("rework_count", 0)); v > 0 {
		current.ReworkCount = v
	}

	h.detector.SetThresholds(current)

	result, err := json.Marshal(map[string]interface{}{
		"message":    "patrol thresholds updated",
		"thresholds": current,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(result)), nil
}
