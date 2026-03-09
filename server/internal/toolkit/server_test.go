package toolkit_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/bis-code/claude-toolkit/server/internal/db"
	"github.com/bis-code/claude-toolkit/server/internal/toolkit"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

func setupClient(t *testing.T) (*client.Client, *db.Store) {
	t.Helper()
	store, err := db.NewMemoryStore()
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	s := toolkit.NewServer(toolkit.WithStore(store))
	c, err := client.NewInProcessClient(s)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	t.Cleanup(func() { c.Close() })

	_, err = c.Initialize(context.Background(), mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: "2024-11-05",
			Capabilities:    mcp.ClientCapabilities{},
			ClientInfo:      mcp.Implementation{Name: "test", Version: "1.0.0"},
		},
	})
	if err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}
	return c, store
}

func callTool(t *testing.T, c *client.Client, name string, args map[string]interface{}) string {
	t.Helper()
	result, err := c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: args,
		},
	})
	if err != nil {
		t.Fatalf("calling %s failed: %v", name, err)
	}
	if len(result.Content) == 0 {
		t.Fatalf("%s returned empty content", name)
	}
	text, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatalf("%s result is not text", name)
	}
	return text.Text
}

func TestNewServer_ReturnsValidServer(t *testing.T) {
	s := toolkit.NewServer()
	if s == nil {
		t.Fatal("NewServer returned nil")
	}
}

func TestServer_ListTools(t *testing.T) {
	c, _ := setupClient(t)

	result, err := c.ListTools(context.Background(), mcp.ListToolsRequest{})
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
			t.Errorf("expected tool %q not found", expected)
		}
	}
}

func TestServer_HealthCheck(t *testing.T) {
	c, _ := setupClient(t)
	text := callTool(t, c, "toolkit__health_check", nil)

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["status"] != "healthy" {
		t.Errorf("status = %v, want 'healthy'", result["status"])
	}
	if result["version"] != toolkit.ServerVersion {
		t.Errorf("version = %v, want %s", result["version"], toolkit.ServerVersion)
	}
}

// Integration tests: full MCP tool call → rule engine → response

func TestIntegration_CreateAndGetRule(t *testing.T) {
	c, _ := setupClient(t)

	// Create a rule via MCP
	createResult := callTool(t, c, "toolkit__create_rule", map[string]interface{}{
		"content":         "Always use context.Context as first param",
		"scope":           "global",
		"source_evidence": "Go best practice",
	})

	var created map[string]interface{}
	json.Unmarshal([]byte(createResult), &created)

	if created["message"] != "rule created" {
		t.Errorf("unexpected create result: %s", createResult)
	}
	if created["sensitive"] != false {
		t.Error("clean rule should not be sensitive")
	}

	// Get active rules — should include the created rule
	getResult := callTool(t, c, "toolkit__get_active_rules", map[string]interface{}{
		"project": "my-go-project",
	})

	var got map[string]interface{}
	json.Unmarshal([]byte(getResult), &got)

	count := got["count"].(float64)
	if count < 1 {
		t.Errorf("expected at least 1 rule, got %v", count)
	}
}

func TestIntegration_CreateSensitiveRule(t *testing.T) {
	c, _ := setupClient(t)

	createResult := callTool(t, c, "toolkit__create_rule", map[string]interface{}{
		"content": "Use api_key: sk-abc123def456789012345678 for testing",
		"scope":   "project",
		"project": "my-project",
	})

	var created map[string]interface{}
	json.Unmarshal([]byte(createResult), &created)

	if created["sensitive"] != true {
		t.Error("rule with API key should be flagged as sensitive")
	}
	if created["local_only"] != true {
		t.Error("sensitive rule should be local_only")
	}
}

func TestIntegration_CreateUpdateDeleteRule(t *testing.T) {
	c, _ := setupClient(t)

	// Create
	createResult := callTool(t, c, "toolkit__create_rule", map[string]interface{}{
		"content": "Original content",
		"scope":   "global",
	})
	var created map[string]interface{}
	json.Unmarshal([]byte(createResult), &created)
	ruleID := created["id"].(string)

	// Update
	callTool(t, c, "toolkit__update_rule", map[string]interface{}{
		"id":      ruleID,
		"content": "Updated content",
	})

	// List and verify update
	listResult := callTool(t, c, "toolkit__list_rules", map[string]interface{}{})
	var listed map[string]interface{}
	json.Unmarshal([]byte(listResult), &listed)

	rulesList := listed["rules"].([]interface{})
	found := false
	for _, r := range rulesList {
		rule := r.(map[string]interface{})
		if rule["id"] == ruleID && rule["content"] == "Updated content" {
			found = true
		}
	}
	if !found {
		t.Error("updated rule not found in list")
	}

	// Delete
	callTool(t, c, "toolkit__delete_rule", map[string]interface{}{
		"id": ruleID,
	})

	// Verify deleted
	listResult2 := callTool(t, c, "toolkit__list_rules", map[string]interface{}{})
	var listed2 map[string]interface{}
	json.Unmarshal([]byte(listResult2), &listed2)
	count := listed2["count"].(float64)
	if count != 0 {
		t.Errorf("expected 0 rules after delete, got %v", count)
	}
}

func TestIntegration_ScoreRule(t *testing.T) {
	c, store := setupClient(t)

	// Create rule directly in store (known ID)
	store.CreateRule(&db.Rule{
		ID:            "score-test",
		Content:       "Test rule",
		Scope:         "global",
		Effectiveness: 0.5,
	})

	// Score as helpful via MCP
	scoreResult := callTool(t, c, "toolkit__score_rule", map[string]interface{}{
		"id":      "score-test",
		"helpful": true,
		"context": "helped with testing",
	})

	var scored map[string]interface{}
	json.Unmarshal([]byte(scoreResult), &scored)

	effectiveness := scored["effectiveness"].(float64)
	if effectiveness != 1.0 {
		t.Errorf("effectiveness = %f, want 1.0 (one helpful score)", effectiveness)
	}
}

func TestIntegration_MergeScopesViaMCP(t *testing.T) {
	c, store := setupClient(t)

	// Create rules in different scopes
	store.CreateRule(&db.Rule{ID: "global-1", Content: "Global rule", Scope: "global", Effectiveness: 0.9})
	store.CreateRule(&db.Rule{ID: "proj-1", Content: "Project rule", Scope: "project", Project: "my-proj", Effectiveness: 0.8})
	store.CreateRule(&db.Rule{ID: "proj-other", Content: "Other project", Scope: "project", Project: "other", Effectiveness: 0.7})

	// Get active rules for my-proj — should get global + my-proj, NOT other
	getResult := callTool(t, c, "toolkit__get_active_rules", map[string]interface{}{
		"project": "my-proj",
	})

	var got map[string]interface{}
	json.Unmarshal([]byte(getResult), &got)

	count := int(got["count"].(float64))
	if count != 2 {
		t.Errorf("expected 2 rules (global + project), got %d", count)
	}
}
