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

// registerEvolutionTools registers all evolution and auto-deprecation MCP tools.
func (h *handlers) registerEvolutionTools(s *server.MCPServer) {
	s.AddTool(
		mcp.NewTool("toolkit__run_evolution_cycle",
			mcp.WithDescription("Run a full evolution cycle: detect patterns in session history, propose new rules as improvements, and auto-deprecate low-scoring rules."),
			mcp.WithString("project",
				mcp.Description("Optional project filter for pattern detection"),
			),
			mcp.WithNumber("session_limit",
				mcp.Description("Number of recent sessions to analyse (default: 20)"),
			),
			mcp.WithNumber("deprecation_threshold",
				mcp.Description("Effectiveness threshold below which rules are deprecated (default: 0.3)"),
			),
			mcp.WithNumber("min_scores",
				mcp.Description("Minimum number of scores before a rule can be deprecated (default: 5)"),
			),
		),
		h.handleRunEvolutionCycle,
	)

	s.AddTool(
		mcp.NewTool("toolkit__get_pending_improvements",
			mcp.WithDescription("List all pending improvements proposed by the evolution engine."),
		),
		h.handleGetPendingImprovements,
	)

	s.AddTool(
		mcp.NewTool("toolkit__apply_improvement",
			mcp.WithDescription("Promote a pending improvement to a live rule."),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("Improvement ID to apply"),
			),
		),
		h.handleApplyImprovement,
	)

	s.AddTool(
		mcp.NewTool("toolkit__reject_improvement",
			mcp.WithDescription("Reject a pending improvement with an optional reason."),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("Improvement ID to reject"),
			),
			mcp.WithString("reason",
				mcp.Description("Why the improvement is being rejected"),
			),
		),
		h.handleRejectImprovement,
	)

	s.AddTool(
		mcp.NewTool("toolkit__audit_rules",
			mcp.WithDescription("List rules enriched with their score count and effectiveness. Useful for identifying candidates for deprecation."),
			mcp.WithString("scope",
				mcp.Description("Filter by scope"),
				mcp.Enum("global", "workspace", "project", "task"),
			),
		),
		h.handleAuditRules,
	)

	s.AddTool(
		mcp.NewTool("toolkit__get_evolution_stats",
			mcp.WithDescription("Get aggregate statistics about the evolution engine: improvements by status, deprecated rule count, and patterns detected."),
		),
		h.handleGetEvolutionStats,
	)
}

// handleRunEvolutionCycle detects patterns, proposes improvements, and deprecates weak rules.
func (h *handlers) handleRunEvolutionCycle(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	project := req.GetString("project", "")
	sessionLimit := int(req.GetFloat("session_limit", 20))
	depThreshold := req.GetFloat("deprecation_threshold", 0.3)
	minScores := int(req.GetFloat("min_scores", 5))

	// Step 1: detect patterns.
	patterns, err := h.evEngine.DetectPatterns(project, sessionLimit)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("pattern detection failed: %v", err)), nil
	}

	// Step 2: propose improvements from each pattern and persist them.
	improvementsProposed := 0
	for _, pattern := range patterns {
		imp := h.evEngine.ProposeRule(pattern)
		if imp == nil {
			continue
		}
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
		if err := h.store.CreateImprovement(dbImp); err != nil {
			// Non-fatal: log and continue.
			continue
		}
		improvementsProposed++
	}

	// Step 3: auto-deprecate low-scoring rules.
	deprecated, err := h.store.DeprecateLowScoreRules(depThreshold, minScores)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("deprecation failed: %v", err)), nil
	}

	result, _ := json.Marshal(map[string]interface{}{
		"patterns_found":        len(patterns),
		"improvements_proposed": improvementsProposed,
		"rules_deprecated":      len(deprecated),
		"deprecated_ids":        deprecated,
	})
	return mcp.NewToolResultText(string(result)), nil
}

// handleGetPendingImprovements returns all pending improvements.
func (h *handlers) handleGetPendingImprovements(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	imps, err := h.store.ListImprovements("pending")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list improvements: %v", err)), nil
	}
	if imps == nil {
		imps = []*db.Improvement{}
	}
	result, _ := json.Marshal(map[string]interface{}{
		"improvements": imps,
		"count":        len(imps),
	})
	return mcp.NewToolResultText(string(result)), nil
}

// handleApplyImprovement promotes an improvement to a live rule.
func (h *handlers) handleApplyImprovement(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetString("id", "")
	if id == "" {
		return mcp.NewToolResultError("id is required"), nil
	}

	// Fetch the improvement. ListImprovements doesn't support get-by-id, so we
	// list all and find the one we want — the table is expected to be small.
	all, err := h.store.ListImprovements("")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list improvements: %v", err)), nil
	}
	var target *db.Improvement
	for _, imp := range all {
		if imp.ID == id {
			target = imp
			break
		}
	}
	if target == nil {
		return mcp.NewToolResultError(fmt.Sprintf("improvement %q not found", id)), nil
	}
	if target.Status != "pending" {
		return mcp.NewToolResultError(fmt.Sprintf("improvement %q is not pending (status: %s)", id, target.Status)), nil
	}

	// Create a rule from the improvement.
	ruleID := fmt.Sprintf("r-ev-%d", time.Now().UnixNano())
	rule := &db.Rule{
		ID:             ruleID,
		Content:        target.Content,
		Scope:          target.Scope,
		Project:        target.Project,
		Tags:           target.Tags,
		Effectiveness:  0.5,
		CreatedFrom:    "evolution",
		SourceEvidence: target.Evidence,
	}
	if err := h.store.CreateRule(rule); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create rule: %v", err)), nil
	}

	// Mark improvement as applied.
	if err := h.store.UpdateImprovementStatus(id, "applied", ""); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to update improvement status: %v", err)), nil
	}

	result, _ := json.Marshal(map[string]interface{}{
		"rule_id": ruleID,
		"message": "improvement applied",
	})
	return mcp.NewToolResultText(string(result)), nil
}

// handleRejectImprovement marks an improvement as rejected.
func (h *handlers) handleRejectImprovement(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetString("id", "")
	if id == "" {
		return mcp.NewToolResultError("id is required"), nil
	}
	reason := req.GetString("reason", "")

	if err := h.store.UpdateImprovementStatus(id, "rejected", reason); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to reject improvement: %v", err)), nil
	}

	result, _ := json.Marshal(map[string]interface{}{
		"message": "improvement rejected",
	})
	return mcp.NewToolResultText(string(result)), nil
}

// auditedRule is a Rule enriched with score count for audit output.
type auditedRule struct {
	db.Rule
	ScoreCount int `json:"score_count"`
}

// handleAuditRules lists rules with their score counts.
func (h *handlers) handleAuditRules(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	scope := req.GetString("scope", "")

	rules, err := h.store.ListRules(scope, "", "")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list rules: %v", err)), nil
	}

	scoreCounts, err := h.store.CountScoresPerRule()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to count scores: %v", err)), nil
	}

	audited := make([]auditedRule, 0, len(rules))
	for _, r := range rules {
		audited = append(audited, auditedRule{
			Rule:       r,
			ScoreCount: scoreCounts[r.ID],
		})
	}

	result, _ := json.Marshal(map[string]interface{}{
		"rules": audited,
		"count": len(audited),
	})
	return mcp.NewToolResultText(string(result)), nil
}

// handleGetEvolutionStats returns aggregate evolution statistics.
func (h *handlers) handleGetEvolutionStats(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	statuses := []string{"pending", "applied", "rejected"}
	improvementCounts := make(map[string]int)
	for _, status := range statuses {
		imps, err := h.store.ListImprovements(status)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to list improvements (%s): %v", status, err)), nil
		}
		improvementCounts[status] = len(imps)
	}

	// Count deprecated rules (these have deprecated=1, so we need a helper or
	// can infer from the difference — use a dedicated query via a new store method).
	deprecatedCount, err := h.store.CountDeprecatedRules()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to count deprecated rules: %v", err)), nil
	}

	result, _ := json.Marshal(map[string]interface{}{
		"improvements_by_status": improvementCounts,
		"deprecated_rules":       deprecatedCount,
		"total_improvements":     improvementCounts["pending"] + improvementCounts["applied"] + improvementCounts["rejected"],
	})
	return mcp.NewToolResultText(string(result)), nil
}

