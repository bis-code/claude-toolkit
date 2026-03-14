package toolkit

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/bis-code/claude-toolkit/server/internal/db"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerWorkflowTools registers workflow learning MCP tools.
func (h *handlers) registerWorkflowTools(s *server.MCPServer) {
	s.AddTool(
		mcp.NewTool("toolkit__get_workflow_profile",
			mcp.WithDescription("Get learned workflow patterns grouped by category, sorted by confidence"),
			mcp.WithString("project",
				mcp.Description("Optional project filter"),
			),
		),
		h.handleGetWorkflowProfile,
	)

	s.AddTool(
		mcp.NewTool("toolkit__record_workflow_pattern",
			mcp.WithDescription("Record or update a workflow pattern. If the pattern already exists for the project, increments occurrences and recalculates confidence."),
			mcp.WithString("category",
				mcp.Required(),
				mcp.Description("Pattern category"),
				mcp.Enum("coding_pattern", "task_execution", "problem_solving", "preference"),
			),
			mcp.WithString("pattern",
				mcp.Required(),
				mcp.Description("The workflow pattern description"),
			),
			mcp.WithString("project",
				mcp.Description("Optional project the pattern applies to"),
			),
			mcp.WithString("details",
				mcp.Description("Optional additional context for the pattern"),
			),
		),
		h.handleRecordWorkflowPattern,
	)
}

// handleGetWorkflowProfile returns learned patterns grouped by category, sorted by confidence.
func (h *handlers) handleGetWorkflowProfile(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	project := req.GetString("project", "")

	stats, err := h.store.GetWorkflowStats(project)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get workflow stats: %v", err)), nil
	}

	if stats == nil {
		stats = []db.WorkflowStats{}
	}

	result, err := json.Marshal(map[string]interface{}{
		"project":    project,
		"categories": stats,
		"count":      len(stats),
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal profile: %v", err)), nil
	}

	return mcp.NewToolResultText(string(result)), nil
}

// handleRecordWorkflowPattern creates or updates a workflow pattern.
func (h *handlers) handleRecordWorkflowPattern(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	category := req.GetString("category", "")
	pattern := req.GetString("pattern", "")
	project := req.GetString("project", "")
	details := req.GetString("details", "")

	if category == "" || pattern == "" {
		return mcp.NewToolResultError("category and pattern are required"), nil
	}

	existing, err := h.store.GetWorkflowEventByPatternAndProject(pattern, project)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to look up pattern: %v", err)), nil
	}

	var eventID string
	var occurrences int
	var confidence float64

	if existing != nil {
		// Update: increment occurrences, recalculate confidence (capped at 1.0)
		occurrences = existing.Occurrences + 1
		confidence = math.Min(1.0, float64(occurrences)/10.0)

		if err := h.store.UpdateWorkflowEvent(existing.ID, occurrences, confidence); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to update pattern: %v", err)), nil
		}
		eventID = existing.ID
	} else {
		// Create new with default confidence 0.5
		occurrences = 1
		confidence = 0.5
		eventID = fmt.Sprintf("we-%d", time.Now().UnixNano())

		we := &db.WorkflowEvent{
			ID:          eventID,
			SessionID:   "",
			Category:    category,
			Pattern:     pattern,
			Details:     details,
			Confidence:  confidence,
			Occurrences: occurrences,
			Project:     project,
		}
		if err := h.store.CreateWorkflowEvent(we); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create pattern: %v", err)), nil
		}
	}

	result, _ := json.Marshal(map[string]interface{}{
		"id":          eventID,
		"pattern":     pattern,
		"category":    category,
		"occurrences": occurrences,
		"confidence":  confidence,
		"updated":     existing != nil,
		"message":     "workflow pattern recorded",
	})

	return mcp.NewToolResultText(string(result)), nil
}
