package toolkit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bis-code/claude-toolkit/server/internal/db"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerEvaluatorTools registers skill scoring MCP tools.
func (h *handlers) registerEvaluatorTools(s *server.MCPServer) {
	s.AddTool(
		mcp.NewTool("toolkit__score_skill",
			mcp.WithDescription("Record a skill effectiveness score (0.0-1.0)"),
			mcp.WithString("skill",
				mcp.Required(),
				mcp.Description("Skill name to score"),
			),
			mcp.WithNumber("score",
				mcp.Required(),
				mcp.Description("Effectiveness score between 0.0 and 1.0"),
			),
			mcp.WithString("session_id",
				mcp.Description("Optional session identifier"),
			),
			mcp.WithString("project",
				mcp.Description("Optional project context"),
			),
			mcp.WithString("details",
				mcp.Description("Optional context or notes for this score"),
			),
		),
		h.handleScoreSkill,
	)

	s.AddTool(
		mcp.NewTool("toolkit__get_skill_stats",
			mcp.WithDescription("Get aggregated skill effectiveness stats. If skill is specified, returns trend data for that skill. If not, returns all skills with average scores."),
			mcp.WithString("skill",
				mcp.Description("Optional skill name filter"),
			),
			mcp.WithString("project",
				mcp.Description("Optional project filter"),
			),
		),
		h.handleGetSkillStats,
	)
}

// handleScoreSkill records a skill effectiveness score.
func (h *handlers) handleScoreSkill(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	skill := req.GetString("skill", "")
	if skill == "" {
		return mcp.NewToolResultError("skill is required"), nil
	}

	score := req.GetFloat("score", -1)
	if score < 0 || score > 1 {
		return mcp.NewToolResultError("score must be between 0.0 and 1.0"), nil
	}

	rec := &db.SkillScoreRecord{
		Skill:     skill,
		Score:     score,
		SessionID: req.GetString("session_id", ""),
		Project:   req.GetString("project", ""),
		Details:   req.GetString("details", ""),
	}

	if err := h.store.CreateSkillScore(rec); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to record skill score: %v", err)), nil
	}

	result, _ := json.Marshal(map[string]interface{}{
		"id":      rec.ID,
		"skill":   skill,
		"score":   score,
		"message": "skill score recorded",
	})

	return mcp.NewToolResultText(string(result)), nil
}

// handleGetSkillStats returns aggregated skill effectiveness stats.
func (h *handlers) handleGetSkillStats(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	skill := req.GetString("skill", "")
	project := req.GetString("project", "")

	stats, err := h.store.GetSkillStats(skill, project)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get skill stats: %v", err)), nil
	}

	if stats == nil {
		stats = []db.SkillStats{}
	}

	result, err := json.Marshal(map[string]interface{}{
		"stats": stats,
		"count": len(stats),
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal stats: %v", err)), nil
	}

	return mcp.NewToolResultText(string(result)), nil
}
