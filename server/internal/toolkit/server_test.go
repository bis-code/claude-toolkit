package toolkit_test

import (
	"context"
	"testing"

	"github.com/bis-code/claude-toolkit/server/internal/toolkit"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

func setupClient(t *testing.T) *client.Client {
	t.Helper()
	s := toolkit.NewServer()
	c, err := client.NewInProcessClient(s)
	if err != nil {
		t.Fatalf("failed to create in-process client: %v", err)
	}
	t.Cleanup(func() { c.Close() })

	_, err = c.Initialize(context.Background(), mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: "2024-11-05",
			Capabilities:    mcp.ClientCapabilities{},
			ClientInfo: mcp.Implementation{
				Name:    "test-client",
				Version: "1.0.0",
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to initialize client: %v", err)
	}
	return c
}

func TestNewServer_ReturnsValidServer(t *testing.T) {
	s := toolkit.NewServer()
	if s == nil {
		t.Fatal("NewServer returned nil")
	}
}

func TestServer_ListTools(t *testing.T) {
	c := setupClient(t)
	ctx := context.Background()

	result, err := c.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	expectedTools := []string{
		"toolkit__health_check",
		"toolkit__get_active_rules",
		"toolkit__create_rule",
		"toolkit__update_rule",
		"toolkit__delete_rule",
		"toolkit__list_rules",
		"toolkit__score_rule",
	}

	toolNames := make(map[string]bool)
	for _, tool := range result.Tools {
		toolNames[tool.Name] = true
	}

	for _, expected := range expectedTools {
		if !toolNames[expected] {
			t.Errorf("expected tool %q not found in registered tools", expected)
		}
	}
}

func TestServer_HealthCheck(t *testing.T) {
	c := setupClient(t)
	ctx := context.Background()

	result, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "toolkit__health_check",
		},
	})
	if err != nil {
		t.Fatalf("health_check failed: %v", err)
	}

	if len(result.Content) == 0 {
		t.Fatal("health_check returned empty content")
	}

	textContent, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatal("health_check result is not text content")
	}

	if textContent.Text == "" {
		t.Fatal("health_check returned empty text")
	}
}

func TestServer_StubTools_ReturnResponse(t *testing.T) {
	c := setupClient(t)
	ctx := context.Background()

	stubTools := []string{
		"toolkit__get_active_rules",
		"toolkit__create_rule",
		"toolkit__update_rule",
		"toolkit__delete_rule",
		"toolkit__list_rules",
		"toolkit__score_rule",
	}

	for _, toolName := range stubTools {
		t.Run(toolName, func(t *testing.T) {
			result, err := c.CallTool(ctx, mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name: toolName,
				},
			})
			if err != nil {
				t.Fatalf("calling %s failed: %v", toolName, err)
			}

			if len(result.Content) == 0 {
				t.Errorf("%s returned empty content", toolName)
				return
			}

			textContent, ok := mcp.AsTextContent(result.Content[0])
			if !ok {
				t.Errorf("%s result is not text content", toolName)
				return
			}

			if textContent.Text == "" {
				t.Errorf("%s returned empty response", toolName)
			}
		})
	}
}
