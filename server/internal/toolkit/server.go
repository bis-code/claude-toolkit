package toolkit

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"time"

	"github.com/bis-code/claude-toolkit/server/internal/db"
	"github.com/bis-code/claude-toolkit/server/internal/evolution"
	"github.com/bis-code/claude-toolkit/server/internal/patrol"
	"github.com/bis-code/claude-toolkit/server/internal/rules"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	ServerName    = "claude-toolkit-server"
	ServerVersion = "4.0.0-dev"
)

// handlers holds the dependencies for MCP tool handlers.
type handlers struct {
	store         *db.Store
	engine        *rules.Engine
	detector      *patrol.Detector
	evEngine      *evolution.Engine
	dashboardAddr string
}

// NewServer creates a new MCP server with all toolkit tools registered.
// If store is nil, creates an in-memory store (useful for testing).
func NewServer(opts ...Option) *server.MCPServer {
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}

	var store *db.Store
	if cfg.store != nil {
		store = cfg.store
	} else {
		var err error
		store, err = db.NewMemoryStore()
		if err != nil {
			panic(fmt.Sprintf("cannot create memory store: %v", err))
		}
	}

	h := &handlers{
		store:         store,
		engine:        rules.NewEngine(store),
		detector:      patrol.NewDetector(patrol.DefaultThresholds()),
		evEngine:      evolution.NewEngine(store),
		dashboardAddr: cfg.dashboardAddr,
	}

	s := server.NewMCPServer(
		ServerName,
		ServerVersion,
		server.WithToolCapabilities(true),
		server.WithRecovery(),
	)

	h.registerTools(s)
	return s
}

// Option configures the server.
type Option func(*config)

type config struct {
	store         *db.Store
	dashboardAddr string
}

// WithStore sets the database store for the server.
func WithStore(store *db.Store) Option {
	return func(c *config) {
		c.store = store
	}
}

// WithDashboardAddr records the dashboard address so health_check can report it.
func WithDashboardAddr(addr string) Option {
	return func(c *config) {
		c.dashboardAddr = addr
	}
}

func (h *handlers) registerTools(s *server.MCPServer) {
	s.AddTool(
		mcp.NewTool("toolkit__health_check",
			mcp.WithDescription("Check server health, version, and status"),
		),
		h.handleHealthCheck,
	)

	s.AddTool(
		mcp.NewTool("toolkit__get_active_rules",
			mcp.WithDescription("Get merged active rules for a project/task context. Merges 4 scopes (global → workspace → project → task), substitutes variables, and respects token budget."),
			mcp.WithString("project",
				mcp.Required(),
				mcp.Description("Project name to get rules for"),
			),
			mcp.WithString("task",
				mcp.Description("Optional task/issue context for task-scoped rules"),
			),
			mcp.WithNumber("token_budget",
				mcp.Description("Maximum token budget for merged rules (default: 2000)"),
			),
		),
		h.handleGetActiveRules,
	)

	s.AddTool(
		mcp.NewTool("toolkit__create_rule",
			mcp.WithDescription("Create a new rule. Automatically checks for sensitive content and tags as local-only if detected."),
			mcp.WithString("content",
				mcp.Required(),
				mcp.Description("The rule content (supports {{variable}} substitution)"),
			),
			mcp.WithString("scope",
				mcp.Required(),
				mcp.Description("Rule scope: global, workspace, project, or task"),
				mcp.Enum("global", "workspace", "project", "task"),
			),
			mcp.WithString("project",
				mcp.Description("Project name (required for project/task scope)"),
			),
			mcp.WithString("workspace",
				mcp.Description("Workspace name (required for workspace scope)"),
			),
			mcp.WithObject("tags",
				mcp.Description("Tags for filtering: {\"tech_stack\": [\"go\"], \"domain\": [\"testing\"]}"),
			),
			mcp.WithString("source_evidence",
				mcp.Description("Why this rule exists (e.g., 'Failed 3 times in session X')"),
			),
		),
		h.handleCreateRule,
	)

	s.AddTool(
		mcp.NewTool("toolkit__update_rule",
			mcp.WithDescription("Update an existing rule by ID"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("Rule ID to update"),
			),
			mcp.WithString("content",
				mcp.Description("New rule content"),
			),
			mcp.WithString("scope",
				mcp.Description("New scope"),
				mcp.Enum("global", "workspace", "project", "task"),
			),
			mcp.WithObject("tags",
				mcp.Description("New tags"),
			),
		),
		h.handleUpdateRule,
	)

	s.AddTool(
		mcp.NewTool("toolkit__delete_rule",
			mcp.WithDescription("Delete a rule by ID"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("Rule ID to delete"),
			),
		),
		h.handleDeleteRule,
	)

	s.AddTool(
		mcp.NewTool("toolkit__list_rules",
			mcp.WithDescription("List rules with optional filtering by scope, project, or tech stack"),
			mcp.WithString("scope",
				mcp.Description("Filter by scope"),
				mcp.Enum("global", "workspace", "project", "task"),
			),
			mcp.WithString("project",
				mcp.Description("Filter by project"),
			),
			mcp.WithString("tech_stack",
				mcp.Description("Filter by tech stack (e.g., 'go', 'unity')"),
			),
		),
		h.handleListRules,
	)

	s.AddTool(
		mcp.NewTool("toolkit__score_rule",
			mcp.WithDescription("Score a rule's effectiveness based on session outcome"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("Rule ID to score"),
			),
			mcp.WithBoolean("helpful",
				mcp.Required(),
				mcp.Description("Whether the rule was helpful in this session"),
			),
			mcp.WithString("context",
				mcp.Description("Context for the score (what happened)"),
			),
		),
		h.handleScoreRule,
	)
	s.AddTool(
		mcp.NewTool("toolkit__check_rule_updates",
			mcp.WithDescription("Check for new seed rules and load any that are missing"),
			mcp.WithString("seed_dir",
				mcp.Required(),
				mcp.Description("Path to the seed rules directory"),
			),
		),
		h.handleCheckRuleUpdates,
	)

	h.registerTelemetryTools(s)
	h.registerPatrolTools(s)
	h.registerEvolutionTools(s)
	h.registerWorkspaceTools(s)
}

func (h *handlers) handleHealthCheck(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	result := map[string]string{
		"server":     ServerName,
		"version":    ServerVersion,
		"status":     "healthy",
		"go_version": runtime.Version(),
		"platform":   runtime.GOOS + "/" + runtime.GOARCH,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	}
	if h.dashboardAddr != "" {
		result["dashboard"] = "http://" + h.dashboardAddr
	}
	b, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(b)), nil
}

func (h *handlers) handleGetActiveRules(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	project := req.GetString("project", "")
	task := req.GetString("task", "")
	tokenBudget := int(req.GetFloat("token_budget", 0))

	activeRules, err := h.engine.GetActiveRules(rules.MergeContext{
		Project:     project,
		Task:        task,
		TokenBudget: tokenBudget,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get active rules: %v", err)), nil
	}

	result, err := json.Marshal(map[string]interface{}{
		"rules": activeRules,
		"count": len(activeRules),
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal rules: %v", err)), nil
	}

	return mcp.NewToolResultText(string(result)), nil
}

func (h *handlers) handleCreateRule(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	content := req.GetString("content", "")
	scope := req.GetString("scope", "")
	project := req.GetString("project", "")
	workspace := req.GetString("workspace", "")
	evidence := req.GetString("source_evidence", "")

	if content == "" || scope == "" {
		return mcp.NewToolResultError("content and scope are required"), nil
	}

	// Generate ID
	id := fmt.Sprintf("r-%d", time.Now().UnixNano())

	// Parse tags from the request
	tags := make(map[string][]string)
	if tagsRaw := req.GetString("tags", ""); tagsRaw != "" {
		json.Unmarshal([]byte(tagsRaw), &tags)
	}

	// Check sensitivity
	sensitive := rules.IsSensitive(content)
	localOnly := sensitive

	rule := &db.Rule{
		ID:             id,
		Content:        content,
		Scope:          scope,
		Project:        project,
		Workspace:      workspace,
		Tags:           tags,
		Effectiveness:  0.5,
		LocalOnly:      localOnly,
		Sensitive:       sensitive,
		CreatedFrom:    "session",
		SourceEvidence: evidence,
	}

	if err := h.store.CreateRule(rule); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create rule: %v", err)), nil
	}

	result, _ := json.Marshal(map[string]interface{}{
		"id":        id,
		"sensitive": sensitive,
		"local_only": localOnly,
		"message":   "rule created",
	})

	return mcp.NewToolResultText(string(result)), nil
}

func (h *handlers) handleUpdateRule(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetString("id", "")
	if id == "" {
		return mcp.NewToolResultError("id is required"), nil
	}

	rule, err := h.store.GetRule(id)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("rule not found: %v", err)), nil
	}

	if content := req.GetString("content", ""); content != "" {
		rule.Content = content
		rule.Sensitive = rules.IsSensitive(content)
		rule.LocalOnly = rule.Sensitive
	}
	if scope := req.GetString("scope", ""); scope != "" {
		rule.Scope = scope
	}

	if err := h.store.UpdateRule(rule); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to update rule: %v", err)), nil
	}

	return mcp.NewToolResultText(`{"message": "rule updated"}`), nil
}

func (h *handlers) handleDeleteRule(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetString("id", "")
	if id == "" {
		return mcp.NewToolResultError("id is required"), nil
	}

	if err := h.store.DeleteRule(id); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete rule: %v", err)), nil
	}

	return mcp.NewToolResultText(`{"message": "rule deleted"}`), nil
}

func (h *handlers) handleListRules(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	scope := req.GetString("scope", "")
	project := req.GetString("project", "")
	techStack := req.GetString("tech_stack", "")

	rulesList, err := h.store.ListRules(scope, project, techStack)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list rules: %v", err)), nil
	}

	result, err := json.Marshal(map[string]interface{}{
		"rules": rulesList,
		"count": len(rulesList),
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal rules: %v", err)), nil
	}

	return mcp.NewToolResultText(string(result)), nil
}

func (h *handlers) handleCheckRuleUpdates(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	seedDir := req.GetString("seed_dir", "")
	if seedDir == "" {
		return mcp.NewToolResultError("seed_dir is required"), nil
	}

	updateResult, err := rules.CheckAndUpdate(h.store, seedDir)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to check for updates: %v", err)), nil
	}

	result, err := json.Marshal(updateResult)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(result)), nil
}

func (h *handlers) handleScoreRule(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetString("id", "")
	if id == "" {
		return mcp.NewToolResultError("id is required"), nil
	}

	helpful := req.GetBool("helpful", false)
	ctx := req.GetString("context", "")

	if err := h.store.RecordScore(id, helpful, ctx, ""); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to score rule: %v", err)), nil
	}

	// Get updated effectiveness
	rule, err := h.store.GetRule(id)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get rule: %v", err)), nil
	}

	result, _ := json.Marshal(map[string]interface{}{
		"message":       "score recorded",
		"effectiveness": rule.Effectiveness,
	})

	return mcp.NewToolResultText(string(result)), nil
}
