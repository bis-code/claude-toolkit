package toolkit_test

import (
	"fmt"
	"time"
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

// ---------------------------------------------------------------------------
// Telemetry tool tests (US-011)
// ---------------------------------------------------------------------------

func TestServer_ListTools_IncludesTelemetryTools(t *testing.T) {
	c, _ := setupClient(t)

	result, err := c.ListTools(context.Background(), mcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	telemetryTools := []string{
		"toolkit__start_session",
		"toolkit__log_event",
		"toolkit__log_stuck",
		"toolkit__log_blocked",
		"toolkit__end_session",
		"toolkit__get_session_history",
		"toolkit__get_project_stats",
	}

	toolNames := make(map[string]bool)
	for _, tool := range result.Tools {
		toolNames[tool.Name] = true
	}

	for _, expected := range telemetryTools {
		if !toolNames[expected] {
			t.Errorf("expected telemetry tool %q not found in registered tools", expected)
		}
	}
}

func TestIntegration_StartSession(t *testing.T) {
	c, store := setupClient(t)

	result := callTool(t, c, "toolkit__start_session", map[string]interface{}{
		"session_id": "sess-start-001",
		"project":    "my-project",
	})

	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(result), &resp); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if resp["message"] != "session started" {
		t.Errorf("unexpected message: %v", resp["message"])
	}

	// Verify session exists in DB
	sess, err := store.GetSession("sess-start-001")
	if err != nil {
		t.Fatalf("session not found in store: %v", err)
	}
	if sess.Project != "my-project" {
		t.Errorf("project = %q, want %q", sess.Project, "my-project")
	}
	if sess.StartedAt.IsZero() {
		t.Error("started_at should not be zero")
	}
}

func TestIntegration_StartSession_DuplicateReturnsError(t *testing.T) {
	c, _ := setupClient(t)

	// First call succeeds
	callTool(t, c, "toolkit__start_session", map[string]interface{}{
		"session_id": "sess-dup-001",
		"project":    "proj",
	})

	// Second call with same ID should return an error result (not a fatal transport error)
	result, err := c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "toolkit__start_session",
			Arguments: map[string]interface{}{
				"session_id": "sess-dup-001",
				"project":    "proj",
			},
		},
	})
	if err != nil {
		t.Fatalf("CallTool transport error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true for duplicate session_id")
	}
}

func TestIntegration_LogEventAutoCreatesSession(t *testing.T) {
	c, store := setupClient(t)

	// Log event for a session_id that doesn't exist yet
	result := callTool(t, c, "toolkit__log_event", map[string]interface{}{
		"session_id": "sess-auto-001",
		"type":       "tool_call",
		"result":     "success",
		"details":    "called read_file",
		"project":    "auto-project",
	})

	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(result), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp["message"] != "event logged" {
		t.Errorf("unexpected message: %v", resp["message"])
	}
	eventID, ok := resp["id"].(string)
	if !ok || eventID == "" {
		t.Error("expected non-empty event id")
	}

	// Session should have been auto-created
	sess, err := store.GetSession("sess-auto-001")
	if err != nil {
		t.Fatalf("session was not auto-created: %v", err)
	}
	if sess.Project != "auto-project" {
		t.Errorf("auto-created session project = %q, want %q", sess.Project, "auto-project")
	}

	// Event should be stored
	events, err := store.ListEvents("sess-auto-001")
	if err != nil {
		t.Fatalf("failed to list events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != "tool_call" {
		t.Errorf("event type = %q, want %q", events[0].Type, "tool_call")
	}
	if events[0].Result != "success" {
		t.Errorf("event result = %q, want %q", events[0].Result, "success")
	}
}

func TestIntegration_LogEventOnExistingSession(t *testing.T) {
	c, store := setupClient(t)

	// Pre-create session
	sess := &db.Session{
		ID:        "sess-existing-001",
		Project:   "existing-project",
		StartedAt: nowTime(),
	}
	if err := store.CreateSession(sess); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	callTool(t, c, "toolkit__log_event", map[string]interface{}{
		"session_id": "sess-existing-001",
		"type":       "compile",
		"result":     "failure",
	})

	events, err := store.ListEvents("sess-existing-001")
	if err != nil {
		t.Fatalf("failed to list events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != "compile" {
		t.Errorf("event type = %q, want %q", events[0].Type, "compile")
	}
}

func TestIntegration_LogStuckEvent(t *testing.T) {
	c, store := setupClient(t)

	// Pre-create session
	store.CreateSession(&db.Session{
		ID: "sess-stuck-001", Project: "p", StartedAt: nowTime(),
	})

	result := callTool(t, c, "toolkit__log_stuck", map[string]interface{}{
		"session_id": "sess-stuck-001",
		"problem":    "cannot resolve import",
		"attempts":   float64(3),
		"hypothesis": "wrong module path",
	})

	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(result), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp["message"] != "stuck event logged" {
		t.Errorf("unexpected message: %v", resp["message"])
	}

	events, err := store.ListEvents("sess-stuck-001")
	if err != nil {
		t.Fatalf("failed to list events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != "stuck" {
		t.Errorf("event type = %q, want %q", events[0].Type, "stuck")
	}
	if events[0].Result != "stuck" {
		t.Errorf("event result = %q, want %q", events[0].Result, "stuck")
	}
	// Details should contain the problem info
	if events[0].Details == "" {
		t.Error("details should not be empty for stuck event")
	}
}

func TestIntegration_LogBlockedEvent(t *testing.T) {
	c, store := setupClient(t)

	store.CreateSession(&db.Session{
		ID: "sess-blocked-001", Project: "p", StartedAt: nowTime(),
	})

	result := callTool(t, c, "toolkit__log_blocked", map[string]interface{}{
		"session_id":    "sess-blocked-001",
		"problem":       "build fails",
		"tried":         "clean build cache",
		"failed_because": "missing dependency",
		"hypothesis":    "need to run go mod tidy",
	})

	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(result), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp["message"] != "blocked event logged" {
		t.Errorf("unexpected message: %v", resp["message"])
	}

	// blocked_template must be present and non-empty
	template, ok := resp["blocked_template"].(string)
	if !ok || template == "" {
		t.Error("expected non-empty blocked_template in response")
	}

	events, err := store.ListEvents("sess-blocked-001")
	if err != nil {
		t.Fatalf("failed to list events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != "blocked" {
		t.Errorf("event type = %q, want %q", events[0].Type, "blocked")
	}
}

func TestIntegration_EndSession(t *testing.T) {
	c, store := setupClient(t)

	store.CreateSession(&db.Session{
		ID: "sess-end-001", Project: "p", StartedAt: nowTime(),
	})

	result := callTool(t, c, "toolkit__end_session", map[string]interface{}{
		"session_id":      "sess-end-001",
		"summary":         "implemented feature X",
		"confidence":      0.85,
		"tasks_completed": float64(3),
		"tasks_failed":    float64(1),
	})

	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(result), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp["message"] != "session ended" {
		t.Errorf("unexpected message: %v", resp["message"])
	}

	// Verify session updated in DB
	sess, err := store.GetSession("sess-end-001")
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}
	if sess.Summary != "implemented feature X" {
		t.Errorf("summary = %q, want %q", sess.Summary, "implemented feature X")
	}
	if sess.Confidence != 0.85 {
		t.Errorf("confidence = %f, want 0.85", sess.Confidence)
	}
	if sess.TasksCompleted != 3 {
		t.Errorf("tasks_completed = %d, want 3", sess.TasksCompleted)
	}
	if sess.TasksFailed != 1 {
		t.Errorf("tasks_failed = %d, want 1", sess.TasksFailed)
	}
	if sess.EndedAt == nil {
		t.Error("ended_at should be set")
	}
}

func TestIntegration_EndSession_NotFound(t *testing.T) {
	c, _ := setupClient(t)

	result, err := c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "toolkit__end_session",
			Arguments: map[string]interface{}{
				"session_id": "sess-nonexistent",
				"summary":    "done",
			},
		},
	})
	if err != nil {
		t.Fatalf("CallTool transport error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true for non-existent session")
	}
}

func TestIntegration_GetSessionHistory(t *testing.T) {
	c, store := setupClient(t)

	// Create sessions for two projects
	for i := 0; i < 3; i++ {
		store.CreateSession(&db.Session{
			ID:        fmt.Sprintf("sess-hist-proj-a-%d", i),
			Project:   "project-a",
			StartedAt: nowTime(),
		})
	}
	for i := 0; i < 2; i++ {
		store.CreateSession(&db.Session{
			ID:        fmt.Sprintf("sess-hist-proj-b-%d", i),
			Project:   "project-b",
			StartedAt: nowTime(),
		})
	}

	// Get all sessions (no filter)
	result := callTool(t, c, "toolkit__get_session_history", map[string]interface{}{})

	var allResp map[string]interface{}
	if err := json.Unmarshal([]byte(result), &allResp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	allCount := int(allResp["count"].(float64))
	if allCount != 5 {
		t.Errorf("expected 5 total sessions, got %d", allCount)
	}

	// Get sessions filtered by project-a
	resultA := callTool(t, c, "toolkit__get_session_history", map[string]interface{}{
		"project": "project-a",
	})

	var projAResp map[string]interface{}
	if err := json.Unmarshal([]byte(resultA), &projAResp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	projACount := int(projAResp["count"].(float64))
	if projACount != 3 {
		t.Errorf("expected 3 sessions for project-a, got %d", projACount)
	}

	// Get sessions with limit
	resultLimited := callTool(t, c, "toolkit__get_session_history", map[string]interface{}{
		"limit": float64(2),
	})

	var limitedResp map[string]interface{}
	if err := json.Unmarshal([]byte(resultLimited), &limitedResp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	limitedCount := int(limitedResp["count"].(float64))
	if limitedCount != 2 {
		t.Errorf("expected 2 sessions with limit=2, got %d", limitedCount)
	}
}

func TestIntegration_GetProjectStats(t *testing.T) {
	c, store := setupClient(t)

	// Create 2 sessions for "stats-project"
	sess1 := &db.Session{ID: "sess-stats-001", Project: "stats-project", StartedAt: nowTime()}
	sess2 := &db.Session{ID: "sess-stats-002", Project: "stats-project", StartedAt: nowTime()}
	store.CreateSession(sess1)
	store.CreateSession(sess2)

	// End sess1 with stats
	store.EndSession("sess-stats-001", "done", 0.9, 5, 1)

	// Add events to sess1
	store.CreateEvent(&db.Event{
		ID: "e-stats-1", SessionID: "sess-stats-001",
		Type: "tool_call", Result: "success", Timestamp: nowTime(),
	})
	store.CreateEvent(&db.Event{
		ID: "e-stats-2", SessionID: "sess-stats-001",
		Type: "compile", Result: "success", Timestamp: nowTime(),
	})
	store.CreateEvent(&db.Event{
		ID: "e-stats-3", SessionID: "sess-stats-001",
		Type: "tool_call", Result: "failure", Timestamp: nowTime(),
	})

	// Add events to sess2
	store.CreateEvent(&db.Event{
		ID: "e-stats-4", SessionID: "sess-stats-002",
		Type: "tool_call", Result: "success", Timestamp: nowTime(),
	})

	result := callTool(t, c, "toolkit__get_project_stats", map[string]interface{}{
		"project": "stats-project",
	})

	var stats map[string]interface{}
	if err := json.Unmarshal([]byte(result), &stats); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if int(stats["total_sessions"].(float64)) != 2 {
		t.Errorf("total_sessions = %v, want 2", stats["total_sessions"])
	}
	if int(stats["total_events"].(float64)) != 4 {
		t.Errorf("total_events = %v, want 4", stats["total_events"])
	}

	eventsByType, ok := stats["events_by_type"].(map[string]interface{})
	if !ok {
		t.Fatal("events_by_type should be a map")
	}
	if int(eventsByType["tool_call"].(float64)) != 3 {
		t.Errorf("tool_call count = %v, want 3", eventsByType["tool_call"])
	}
	if int(eventsByType["compile"].(float64)) != 1 {
		t.Errorf("compile count = %v, want 1", eventsByType["compile"])
	}

	// avg_confidence: only sess1 has ended_at (confidence 0.9); sess2 is open (0.0)
	// We include both — avg = (0.9 + 0.0) / 2 = 0.45, or only ended sessions
	// The spec says "avg_confidence" from sessions — implementation can choose either.
	// We'll just assert it's present and is a number.
	if _, ok := stats["avg_confidence"].(float64); !ok {
		t.Error("avg_confidence should be a float64")
	}

	totalCompleted, ok := stats["total_tasks_completed"].(float64)
	if !ok {
		t.Error("total_tasks_completed should be present")
	}
	// sess1 has 5 completed; sess2 has 0
	if int(totalCompleted) != 5 {
		t.Errorf("total_tasks_completed = %v, want 5", totalCompleted)
	}
}

// nowTime is a test helper that returns the current time with second precision
// to avoid sub-second precision issues with SQLite RFC3339 storage.
func nowTime() time.Time {
	return time.Now().UTC().Truncate(time.Second)
}
