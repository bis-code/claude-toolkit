package dashboard

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net"
	"net/http"
	"strconv"

	"github.com/bis-code/claude-toolkit/server/internal/db"
	"github.com/bis-code/claude-toolkit/server/internal/patrol"
)

//go:embed static/*
var staticFiles embed.FS

// Server is the dashboard HTTP server.
type Server struct {
	store    *db.Store
	detector *patrol.Detector
	mux      *http.ServeMux
}

// NewServer creates a new dashboard server backed by the given store and patrol detector.
func NewServer(store *db.Store, detector *patrol.Detector) *Server {
	s := &Server{store: store, detector: detector}
	s.mux = http.NewServeMux()
	s.registerRoutes()
	return s
}

// Handler returns the HTTP handler for use with httptest or a custom listener.
func (s *Server) Handler() http.Handler {
	return s.mux
}

// ListenAndServe starts the dashboard on the given address.
func (s *Server) ListenAndServe(addr string) error {
	return http.ListenAndServe(addr, s.mux) //nolint:gosec
}

// Serve accepts connections on the given listener.
func (s *Server) Serve(ln net.Listener) error {
	srv := &http.Server{Handler: s.mux} //nolint:gosec
	return srv.Serve(ln)
}

func (s *Server) registerRoutes() {
	// Serve embedded static files at root.
	// fs.Sub strips the "static/" prefix so "/" → index.html.
	staticFS, _ := fs.Sub(staticFiles, "static")
	s.mux.Handle("/", http.FileServer(http.FS(staticFS)))

	// API endpoints — all return JSON.
	s.mux.HandleFunc("/api/health", s.handleHealth)
	s.mux.HandleFunc("/api/rules", s.handleRules)
	s.mux.HandleFunc("/api/sessions", s.handleSessions)
	s.mux.HandleFunc("/api/events", s.handleEvents)
	s.mux.HandleFunc("/api/improvements", s.handleImprovements)
	s.mux.HandleFunc("/api/stats", s.handleStats)
	s.mux.HandleFunc("/api/patrol", s.handlePatrol)
	s.mux.HandleFunc("/api/audit", s.handleAudit)
}

// handleHealth returns a simple liveness probe response.
func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]string{"status": "ok"})
}

// handleRules returns non-deprecated rules.
// Query params: scope, project, tech_stack (all optional).
func (s *Server) handleRules(w http.ResponseWriter, r *http.Request) {
	scope := r.URL.Query().Get("scope")
	project := r.URL.Query().Get("project")
	techStack := r.URL.Query().Get("tech_stack")

	rules, err := s.store.ListRules(scope, project, techStack)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]interface{}{
		"rules": rules,
		"count": len(rules),
	})
}

// handleSessions returns sessions ordered by started_at DESC.
// Query params: project (optional), limit (default 20).
func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	project := r.URL.Query().Get("project")
	limit := 20
	if ls := r.URL.Query().Get("limit"); ls != "" {
		if n, err := strconv.Atoi(ls); err == nil && n > 0 {
			limit = n
		}
	}

	sessions, err := s.store.ListSessions(project, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]interface{}{
		"sessions": sessions,
		"count":    len(sessions),
	})
}

// handleEvents returns events for a given session.
// Query param: session_id (required).
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		http.Error(w, "session_id is required", http.StatusBadRequest)
		return
	}

	events, err := s.store.ListEvents(sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]interface{}{
		"events": events,
		"count":  len(events),
	})
}

// handleImprovements returns improvement proposals.
// Query param: status (optional — pending | applied | rejected).
func (s *Server) handleImprovements(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")

	improvements, err := s.store.ListImprovements(status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]interface{}{
		"improvements": improvements,
		"count":        len(improvements),
	})
}

// handleStats returns aggregate statistics across all resources.
// NOTE: ListSessions with limit=0 returns all sessions; acceptable for a
// developer dashboard that is not exposed to the public internet.
func (s *Server) handleStats(w http.ResponseWriter, _ *http.Request) {
	rules, _ := s.store.ListRules("", "", "")
	sessions, _ := s.store.ListSessions("", 0)
	deprecated, _ := s.store.CountDeprecatedRules()
	improvements, _ := s.store.ListImprovements("")

	pending, applied, rejected := 0, 0, 0
	for _, imp := range improvements {
		switch imp.Status {
		case "pending":
			pending++
		case "applied":
			applied++
		case "rejected":
			rejected++
		}
	}

	// Count verification events across all sessions.
	var verifiedCount, failedCount int
	for _, sess := range sessions {
		events, err := s.store.ListEvents(sess.ID)
		if err != nil {
			continue // best-effort: skip sessions we cannot load events for
		}
		for _, e := range events {
			if e.Type == "verification" {
				if e.Result == "verified" {
					verifiedCount++
				} else if e.Result == "failed" {
					failedCount++
				}
			}
		}
	}

	verificationRate := 0.0
	total := verifiedCount + failedCount
	if total > 0 {
		verificationRate = float64(verifiedCount) / float64(total)
	}

	writeJSON(w, map[string]interface{}{
		"total_rules":      len(rules),
		"total_sessions":   len(sessions),
		"deprecated_rules": deprecated,
		"improvements": map[string]int{
			"pending":  pending,
			"applied":  applied,
			"rejected": rejected,
		},
		"verification": map[string]interface{}{
			"verified": verifiedCount,
			"failed":   failedCount,
			"rate":     verificationRate,
		},
	})
}

// handlePatrol loads events for a session and runs the patrol detector.
// Query param: session_id (required).
func (s *Server) handlePatrol(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		http.Error(w, "session_id is required", http.StatusBadRequest)
		return
	}

	events, err := s.store.ListEvents(sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	alerts := s.detector.Analyze(events)

	// Always return an array, never null.
	if alerts == nil {
		alerts = []patrol.Alert{}
	}

	writeJSON(w, map[string]interface{}{
		"session_id": sessionID,
		"alerts":     alerts,
		"count":      len(alerts),
	})
}

// auditRule is a Rule enriched with the number of scores it has received.
type auditRule struct {
	db.Rule
	ScoreCount int `json:"score_count"`
}

// handleAudit returns all non-deprecated rules enriched with their score counts.
func (s *Server) handleAudit(w http.ResponseWriter, r *http.Request) {
	rules, err := s.store.ListRules("", "", "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	scoreCounts, err := s.store.CountScoresPerRule()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	enriched := make([]auditRule, len(rules))
	for i, rule := range rules {
		enriched[i] = auditRule{
			Rule:       rule,
			ScoreCount: scoreCounts[rule.ID],
		}
	}

	writeJSON(w, map[string]interface{}{
		"rules": enriched,
		"count": len(enriched),
	})
}

// writeJSON sets Content-Type and encodes data as JSON.
func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data) //nolint:errcheck
}
