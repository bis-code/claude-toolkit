package toolkit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bis-code/claude-toolkit/server/internal/workspace"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerWorkspaceTools registers all workspace-related MCP tools.
func (h *handlers) registerWorkspaceTools(s *server.MCPServer) {
	s.AddTool(
		mcp.NewTool("toolkit__get_workspace",
			mcp.WithDescription("Get workspace configuration (auto-detected or from .claude-workspace.json)"),
			mcp.WithString("directory",
				mcp.Required(),
				mcp.Description("Workspace root directory to scan"),
			),
		),
		h.handleGetWorkspace,
	)

	s.AddTool(
		mcp.NewTool("toolkit__get_cross_repo_issues",
			mcp.WithDescription("Get cross-repo issues from the planning repository. Requires gh CLI configured."),
			mcp.WithString("directory",
				mcp.Required(),
				mcp.Description("Workspace root directory"),
			),
			mcp.WithString("label",
				mcp.Description("Filter issues by label (default: claude-ready)"),
			),
			mcp.WithString("state",
				mcp.Description("Issue state: open, closed, all (default: open)"),
			),
		),
		h.handleGetCrossRepoIssues,
	)

	s.AddTool(
		mcp.NewTool("toolkit__register_project",
			mcp.WithDescription("Register a project in the workspace by adding it to .claude-workspace.json"),
			mcp.WithString("directory",
				mcp.Required(),
				mcp.Description("Workspace root directory"),
			),
			mcp.WithString("path",
				mcp.Required(),
				mcp.Description("Relative path to the project"),
			),
			mcp.WithString("type",
				mcp.Description("Tech stack type (go, python, typescript, rust)"),
			),
			mcp.WithString("branch",
				mcp.Description("Default branch"),
			),
		),
		h.handleRegisterProject,
	)
}

func (h *handlers) handleGetCrossRepoIssues(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	directory := req.GetString("directory", "")
	if directory == "" {
		return mcp.NewToolResultError("directory is required"), nil
	}

	cfg, err := workspace.LoadOrDetect(directory)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to load workspace: %v", err)), nil
	}

	if cfg.PlanningRepo == "" {
		return mcp.NewToolResultError("no planning_repo configured in workspace"), nil
	}

	label := req.GetString("label", "claude-ready")
	state := req.GetString("state", "open")

	result, _ := json.Marshal(map[string]interface{}{
		"planning_repo": cfg.PlanningRepo,
		"label":         label,
		"state":         state,
		"hint":          fmt.Sprintf("Run: gh issue list --repo %s --label %s --state %s --json number,title,body,labels", cfg.PlanningRepo, label, state),
	})

	return mcp.NewToolResultText(string(result)), nil
}

func (h *handlers) handleGetWorkspace(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	directory := req.GetString("directory", "")
	if directory == "" {
		return mcp.NewToolResultError("directory is required"), nil
	}

	cfg, err := workspace.LoadOrDetect(directory)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to load workspace: %v", err)), nil
	}

	result, err := json.Marshal(cfg)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal workspace config: %v", err)), nil
	}

	return mcp.NewToolResultText(string(result)), nil
}

func (h *handlers) handleRegisterProject(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	directory := req.GetString("directory", "")
	path := req.GetString("path", "")

	if directory == "" || path == "" {
		return mcp.NewToolResultError("directory and path are required"), nil
	}

	repoType := req.GetString("type", "")
	branch := req.GetString("branch", "")

	cfg, err := workspace.LoadOrDetect(directory)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to load workspace: %v", err)), nil
	}

	// Check for duplicate paths.
	for _, r := range cfg.Repos {
		if r.Path == path {
			return mcp.NewToolResultError(fmt.Sprintf("project %q is already registered", path)), nil
		}
	}

	newRepo := workspace.Repo{
		Path:   path,
		Type:   repoType,
		Branch: branch,
	}
	cfg.Repos = append(cfg.Repos, newRepo)

	configPath := directory + "/.claude-workspace.json"
	if err := workspace.SaveConfig(configPath, cfg); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to save workspace config: %v", err)), nil
	}

	result, _ := json.Marshal(map[string]interface{}{
		"message": "project registered",
		"path":    path,
	})

	return mcp.NewToolResultText(string(result)), nil
}
