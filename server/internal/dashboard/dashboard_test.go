package dashboard_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bis-code/claude-toolkit/server/internal/dashboard"
	"github.com/bis-code/claude-toolkit/server/internal/db"
	"github.com/bis-code/claude-toolkit/server/internal/patrol"
)

// newTestServer creates a dashboard server backed by an in-memory store.
// It seeds the store with the provided seed function before returning.
func newTestServer(t *testing.T, seed func(*db.Store)) *httptest.Server {
	t.Helper()
	store, err := db.NewMemoryStore()
	if err != nil {
		t.Fatalf("NewMemoryStore: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	if seed != nil {
		seed(store)
	}

	detector := patrol.NewDetector(patrol.DefaultThresholds())
	srv := dashboard.NewServer(store, detector)
	return httptest.NewServer(srv.Handler())
}

func get(t *testing.T, ts *httptest.Server, path string) *http.Response {
	t.Helper()
	resp, err := ts.Client().Get(ts.URL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	return resp
}

func decodeJSON(t *testing.T, resp *http.Response, v interface{}) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
}

// TestDashboard_IndexPage verifies the root path returns an HTML response.
func TestDashboard_IndexPage(t *testing.T) {
	ts := newTestServer(t, nil)
	defer ts.Close()

	resp := get(t, ts, "/")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /: want 200, got %d", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		t.Error("GET /: Content-Type header is empty")
	}
}

// TestDashboard_APIRules verifies /api/rules returns a JSON object with a rules array.
func TestDashboard_APIRules(t *testing.T) {
	ts := newTestServer(t, func(s *db.Store) {
		_ = s.CreateRule(&db.Rule{
			ID:      "rule-1",
			Content: "Always write tests first",
			Scope:   "global",
		})
		_ = s.CreateRule(&db.Rule{
			ID:      "rule-2",
			Content: "Use descriptive names",
			Scope:   "project",
			Project: "my-project",
		})
	})
	defer ts.Close()

	resp := get(t, ts, "/api/rules")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /api/rules: want 200, got %d", resp.StatusCode)
	}

	var payload map[string]interface{}
	decodeJSON(t, resp, &payload)

	rules, ok := payload["rules"]
	if !ok {
		t.Fatal("/api/rules: response missing 'rules' key")
	}

	rulesSlice, ok := rules.([]interface{})
	if !ok {
		t.Fatalf("/api/rules: 'rules' is not an array, got %T", rules)
	}

	if len(rulesSlice) != 2 {
		t.Errorf("/api/rules: want 2 rules, got %d", len(rulesSlice))
	}

	count, ok := payload["count"]
	if !ok {
		t.Fatal("/api/rules: response missing 'count' key")
	}
	if int(count.(float64)) != 2 {
		t.Errorf("/api/rules: want count=2, got %v", count)
	}
}

// TestDashboard_APIRules_ScopeFilter verifies ?scope= filter is applied.
func TestDashboard_APIRules_ScopeFilter(t *testing.T) {
	ts := newTestServer(t, func(s *db.Store) {
		_ = s.CreateRule(&db.Rule{ID: "r1", Content: "global rule", Scope: "global"})
		_ = s.CreateRule(&db.Rule{ID: "r2", Content: "project rule", Scope: "project", Project: "x"})
	})
	defer ts.Close()

	resp := get(t, ts, "/api/rules?scope=global")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /api/rules?scope=global: want 200, got %d", resp.StatusCode)
	}

	var payload map[string]interface{}
	decodeJSON(t, resp, &payload)

	count := int(payload["count"].(float64))
	if count != 1 {
		t.Errorf("/api/rules?scope=global: want 1 rule, got %d", count)
	}
}

// TestDashboard_APISessions verifies /api/sessions returns a JSON object with a sessions array.
func TestDashboard_APISessions(t *testing.T) {
	ts := newTestServer(t, func(s *db.Store) {
		_ = s.CreateSession(&db.Session{
			ID:        "sess-1",
			Project:   "proj-a",
			StartedAt: time.Now(),
		})
		_ = s.CreateSession(&db.Session{
			ID:        "sess-2",
			Project:   "proj-b",
			StartedAt: time.Now(),
		})
	})
	defer ts.Close()

	resp := get(t, ts, "/api/sessions")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /api/sessions: want 200, got %d", resp.StatusCode)
	}

	var payload map[string]interface{}
	decodeJSON(t, resp, &payload)

	if _, ok := payload["sessions"]; !ok {
		t.Fatal("/api/sessions: response missing 'sessions' key")
	}

	count := int(payload["count"].(float64))
	if count != 2 {
		t.Errorf("/api/sessions: want 2, got %d", count)
	}
}

// TestDashboard_APISessions_LimitParam verifies ?limit= is respected.
func TestDashboard_APISessions_LimitParam(t *testing.T) {
	ts := newTestServer(t, func(s *db.Store) {
		for i := 0; i < 5; i++ {
			_ = s.CreateSession(&db.Session{
				ID:        "s" + string(rune('0'+i)),
				Project:   "proj",
				StartedAt: time.Now(),
			})
		}
	})
	defer ts.Close()

	resp := get(t, ts, "/api/sessions?limit=2")
	var payload map[string]interface{}
	decodeJSON(t, resp, &payload)

	count := int(payload["count"].(float64))
	if count != 2 {
		t.Errorf("/api/sessions?limit=2: want 2, got %d", count)
	}
}

// TestDashboard_APIEvents_RequiresSessionID verifies that GET /api/events
// without a session_id query param returns HTTP 400.
func TestDashboard_APIEvents_RequiresSessionID(t *testing.T) {
	ts := newTestServer(t, nil)
	defer ts.Close()

	resp := get(t, ts, "/api/events")
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("GET /api/events (no session_id): want 400, got %d", resp.StatusCode)
	}
}

// TestDashboard_APIEvents_WithSessionID verifies that GET /api/events?session_id=x
// returns the events for that session.
func TestDashboard_APIEvents_WithSessionID(t *testing.T) {
	ts := newTestServer(t, func(s *db.Store) {
		_ = s.CreateSession(&db.Session{
			ID:        "sess-abc",
			Project:   "proj",
			StartedAt: time.Now(),
		})
		_ = s.CreateEvent(&db.Event{
			ID:        "ev-1",
			SessionID: "sess-abc",
			Type:      "tool_call",
			Result:    "success",
			Timestamp: time.Now(),
		})
		_ = s.CreateEvent(&db.Event{
			ID:        "ev-2",
			SessionID: "sess-abc",
			Type:      "tool_call",
			Result:    "error",
			Timestamp: time.Now(),
		})
		// event for a different session — must not appear
		_ = s.CreateSession(&db.Session{ID: "sess-other", Project: "proj", StartedAt: time.Now()})
		_ = s.CreateEvent(&db.Event{
			ID:        "ev-3",
			SessionID: "sess-other",
			Type:      "tool_call",
			Result:    "success",
			Timestamp: time.Now(),
		})
	})
	defer ts.Close()

	resp := get(t, ts, "/api/events?session_id=sess-abc")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /api/events?session_id=sess-abc: want 200, got %d", resp.StatusCode)
	}

	var payload map[string]interface{}
	decodeJSON(t, resp, &payload)

	if _, ok := payload["events"]; !ok {
		t.Fatal("/api/events: response missing 'events' key")
	}

	count := int(payload["count"].(float64))
	if count != 2 {
		t.Errorf("/api/events?session_id=sess-abc: want 2 events, got %d", count)
	}
}

// TestDashboard_APIImprovements verifies /api/improvements returns a JSON object.
func TestDashboard_APIImprovements(t *testing.T) {
	ts := newTestServer(t, func(s *db.Store) {
		_ = s.CreateImprovement(&db.Improvement{
			ID:      "imp-1",
			Content: "Add more error handling",
			Scope:   "global",
			Status:  "pending",
		})
		_ = s.CreateImprovement(&db.Improvement{
			ID:      "imp-2",
			Content: "Improve test coverage",
			Scope:   "project",
			Project: "my-project",
			Status:  "applied",
		})
	})
	defer ts.Close()

	resp := get(t, ts, "/api/improvements")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /api/improvements: want 200, got %d", resp.StatusCode)
	}

	var payload map[string]interface{}
	decodeJSON(t, resp, &payload)

	if _, ok := payload["improvements"]; !ok {
		t.Fatal("/api/improvements: response missing 'improvements' key")
	}

	count := int(payload["count"].(float64))
	if count != 2 {
		t.Errorf("/api/improvements: want 2, got %d", count)
	}
}

// TestDashboard_APIImprovements_StatusFilter verifies ?status= filter.
func TestDashboard_APIImprovements_StatusFilter(t *testing.T) {
	ts := newTestServer(t, func(s *db.Store) {
		_ = s.CreateImprovement(&db.Improvement{ID: "i1", Content: "x", Scope: "global", Status: "pending"})
		_ = s.CreateImprovement(&db.Improvement{ID: "i2", Content: "y", Scope: "global", Status: "applied"})
		_ = s.CreateImprovement(&db.Improvement{ID: "i3", Content: "z", Scope: "global", Status: "pending"})
	})
	defer ts.Close()

	resp := get(t, ts, "/api/improvements?status=pending")
	var payload map[string]interface{}
	decodeJSON(t, resp, &payload)

	count := int(payload["count"].(float64))
	if count != 2 {
		t.Errorf("/api/improvements?status=pending: want 2, got %d", count)
	}
}

// TestDashboard_APIStats verifies /api/stats returns aggregate statistics.
func TestDashboard_APIStats(t *testing.T) {
	ts := newTestServer(t, func(s *db.Store) {
		_ = s.CreateRule(&db.Rule{ID: "r1", Content: "rule 1", Scope: "global"})
		_ = s.CreateRule(&db.Rule{ID: "r2", Content: "rule 2", Scope: "global"})
		// deprecated rule (created, then deprecated)
		_ = s.CreateRule(&db.Rule{ID: "r3", Content: "old rule", Scope: "global"})
		r3, _ := s.GetRule("r3")
		r3.Deprecated = true
		_ = s.UpdateRule(r3)

		_ = s.CreateSession(&db.Session{ID: "s1", Project: "p", StartedAt: time.Now()})
		_ = s.CreateSession(&db.Session{ID: "s2", Project: "p", StartedAt: time.Now()})

		_ = s.CreateImprovement(&db.Improvement{ID: "i1", Content: "a", Scope: "global", Status: "pending"})
		_ = s.CreateImprovement(&db.Improvement{ID: "i2", Content: "b", Scope: "global", Status: "applied"})
		_ = s.CreateImprovement(&db.Improvement{ID: "i3", Content: "c", Scope: "global", Status: "rejected"})
	})
	defer ts.Close()

	resp := get(t, ts, "/api/stats")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /api/stats: want 200, got %d", resp.StatusCode)
	}

	var payload map[string]interface{}
	decodeJSON(t, resp, &payload)

	// total_rules excludes deprecated (ListRules filters WHERE deprecated = 0)
	if v := int(payload["total_rules"].(float64)); v != 2 {
		t.Errorf("/api/stats total_rules: want 2, got %d", v)
	}

	if v := int(payload["total_sessions"].(float64)); v != 2 {
		t.Errorf("/api/stats total_sessions: want 2, got %d", v)
	}

	if v := int(payload["deprecated_rules"].(float64)); v != 1 {
		t.Errorf("/api/stats deprecated_rules: want 1, got %d", v)
	}

	imps, ok := payload["improvements"].(map[string]interface{})
	if !ok {
		t.Fatalf("/api/stats improvements: expected object, got %T", payload["improvements"])
	}

	if v := int(imps["pending"].(float64)); v != 1 {
		t.Errorf("/api/stats improvements.pending: want 1, got %d", v)
	}
	if v := int(imps["applied"].(float64)); v != 1 {
		t.Errorf("/api/stats improvements.applied: want 1, got %d", v)
	}
	if v := int(imps["rejected"].(float64)); v != 1 {
		t.Errorf("/api/stats improvements.rejected: want 1, got %d", v)
	}
}

// TestDashboard_APIHealth verifies the health endpoint returns 200.
func TestDashboard_APIHealth(t *testing.T) {
	ts := newTestServer(t, nil)
	defer ts.Close()

	resp := get(t, ts, "/api/health")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /api/health: want 200, got %d", resp.StatusCode)
	}

	var payload map[string]interface{}
	decodeJSON(t, resp, &payload)

	if payload["status"] != "ok" {
		t.Errorf("/api/health: want status=ok, got %v", payload["status"])
	}
}

// TestDashboard_ContentTypeJSON verifies all API endpoints return JSON content-type.
func TestDashboard_ContentTypeJSON(t *testing.T) {
	ts := newTestServer(t, func(s *db.Store) {
		_ = s.CreateSession(&db.Session{ID: "s1", Project: "p", StartedAt: time.Now()})
	})
	defer ts.Close()

	paths := []string{
		"/api/rules",
		"/api/sessions",
		"/api/improvements",
		"/api/stats",
		"/api/health",
		"/api/events?session_id=s1",
		"/api/patrol?session_id=s1",
		"/api/audit",
	}

	for _, path := range paths {
		resp := get(t, ts, path)
		ct := resp.Header.Get("Content-Type")
		if ct != "application/json" {
			t.Errorf("GET %s: want Content-Type application/json, got %q", path, ct)
		}
		resp.Body.Close()
	}
}

// TestDashboard_APIPatrol_RequiresSessionID verifies that GET /api/patrol
// without a session_id query param returns HTTP 400.
func TestDashboard_APIPatrol_RequiresSessionID(t *testing.T) {
	ts := newTestServer(t, nil)
	defer ts.Close()

	resp := get(t, ts, "/api/patrol")
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("GET /api/patrol (no session_id): want 400, got %d", resp.StatusCode)
	}
}

// TestDashboard_APIPatrol_ReturnsAlerts verifies that a session with repeated
// failure events triggers patrol alerts.
func TestDashboard_APIPatrol_ReturnsAlerts(t *testing.T) {
	ts := newTestServer(t, func(s *db.Store) {
		_ = s.CreateSession(&db.Session{
			ID:        "sess-patrol",
			Project:   "proj",
			StartedAt: time.Now(),
		})
		// Create enough consecutive failures to trigger the retry_loop detector
		// (default threshold is 3 consecutive same-detail failures).
		for i := 0; i < 4; i++ {
			_ = s.CreateEvent(&db.Event{
				ID:        "ev-patrol-" + string(rune('a'+i)),
				SessionID: "sess-patrol",
				Type:      "tool_call",
				Result:    "failure",
				Details:   "go build ./...",
				Timestamp: time.Now(),
			})
		}
	})
	defer ts.Close()

	resp := get(t, ts, "/api/patrol?session_id=sess-patrol")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /api/patrol?session_id=sess-patrol: want 200, got %d", resp.StatusCode)
	}

	var payload map[string]interface{}
	decodeJSON(t, resp, &payload)

	alerts, ok := payload["alerts"].([]interface{})
	if !ok {
		t.Fatalf("/api/patrol: 'alerts' missing or not an array, got %T", payload["alerts"])
	}

	if len(alerts) == 0 {
		t.Error("/api/patrol: expected at least one alert for 4 consecutive failures, got 0")
	}
}

// TestDashboard_APIAudit_ReturnsRulesWithScores verifies that /api/audit returns
// rules enriched with their score_count from the rule_scores table.
func TestDashboard_APIAudit_ReturnsRulesWithScores(t *testing.T) {
	ts := newTestServer(t, func(s *db.Store) {
		_ = s.CreateRule(&db.Rule{ID: "r-audit-1", Content: "TDD always", Scope: "global"})
		_ = s.CreateRule(&db.Rule{ID: "r-audit-2", Content: "Small commits", Scope: "global"})

		// Record two scores for r-audit-1, none for r-audit-2.
		_ = s.RecordScore("r-audit-1", true, "ctx", "sess-x")
		_ = s.RecordScore("r-audit-1", false, "ctx", "sess-y")
	})
	defer ts.Close()

	resp := get(t, ts, "/api/audit")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /api/audit: want 200, got %d", resp.StatusCode)
	}

	var payload map[string]interface{}
	decodeJSON(t, resp, &payload)

	rules, ok := payload["rules"].([]interface{})
	if !ok {
		t.Fatalf("/api/audit: 'rules' missing or not an array, got %T", payload["rules"])
	}

	if len(rules) != 2 {
		t.Errorf("/api/audit: want 2 rules, got %d", len(rules))
	}

	// Find r-audit-1 and verify score_count == 2.
	var found bool
	for _, raw := range rules {
		r, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		if r["id"] == "r-audit-1" {
			found = true
			scoreCount, ok := r["score_count"].(float64)
			if !ok {
				t.Fatalf("/api/audit rule r-audit-1: 'score_count' missing or wrong type, got %T", r["score_count"])
			}
			if int(scoreCount) != 2 {
				t.Errorf("/api/audit rule r-audit-1: want score_count=2, got %d", int(scoreCount))
			}
		}
	}
	if !found {
		t.Error("/api/audit: rule r-audit-1 not found in response")
	}
}
