package toolkit

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	ServerName    = "claude-toolkit-server"
	ServerVersion = "4.0.0-dev"
)

// NewServer creates a new MCP server with all toolkit tools registered.
func NewServer() *server.MCPServer {
	s := server.NewMCPServer(
		ServerName,
		ServerVersion,
		server.WithToolCapabilities(true),
		server.WithRecovery(),
	)

	registerTools(s)
	return s
}

func registerTools(s *server.MCPServer) {
	// Health check
	s.AddTool(
		mcp.NewTool("toolkit__health_check",
			mcp.WithDescription("Check server health, version, and status"),
		),
		handleHealthCheck,
	)

	// Rule CRUD tools
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
		handleGetActiveRules,
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
		handleCreateRule,
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
		handleUpdateRule,
	)

	s.AddTool(
		mcp.NewTool("toolkit__delete_rule",
			mcp.WithDescription("Delete a rule by ID"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("Rule ID to delete"),
			),
		),
		handleDeleteRule,
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
		handleListRules,
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
		handleScoreRule,
	)
}

func handleHealthCheck(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	status := fmt.Sprintf(
		`{"server": "%s", "version": "%s", "status": "healthy", "go_version": "%s", "platform": "%s/%s", "timestamp": "%s"}`,
		ServerName,
		ServerVersion,
		runtime.Version(),
		runtime.GOOS,
		runtime.GOARCH,
		time.Now().UTC().Format(time.RFC3339),
	)
	return mcp.NewToolResultText(status), nil
}

// Stub handlers — will be wired to the rule engine in US-006/007/008/009

func handleGetActiveRules(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText(`{"status": "not_implemented", "message": "Rule engine not yet connected. This tool will merge 4-scope rules with variable substitution."}`), nil
}

func handleCreateRule(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText(`{"status": "not_implemented", "message": "Rule storage not yet connected."}`), nil
}

func handleUpdateRule(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText(`{"status": "not_implemented", "message": "Rule storage not yet connected."}`), nil
}

func handleDeleteRule(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText(`{"status": "not_implemented", "message": "Rule storage not yet connected."}`), nil
}

func handleListRules(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText(`{"status": "not_implemented", "message": "Rule storage not yet connected."}`), nil
}

func handleScoreRule(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText(`{"status": "not_implemented", "message": "Rule scoring not yet connected."}`), nil
}
