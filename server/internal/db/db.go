package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"time"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS rules (
	id TEXT PRIMARY KEY,
	content TEXT NOT NULL,
	scope TEXT NOT NULL CHECK(scope IN ('global', 'workspace', 'project', 'task')),
	project TEXT DEFAULT '',
	workspace TEXT DEFAULT '',
	tags_json TEXT DEFAULT '{}',
	effectiveness REAL DEFAULT 0.5,
	local_only INTEGER DEFAULT 0,
	sensitive INTEGER DEFAULT 0,
	created_from TEXT DEFAULT '',
	source_evidence TEXT DEFAULT '',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	deprecated INTEGER DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_rules_scope ON rules(scope);
CREATE INDEX IF NOT EXISTS idx_rules_project ON rules(project);
CREATE INDEX IF NOT EXISTS idx_rules_workspace ON rules(workspace);
CREATE INDEX IF NOT EXISTS idx_rules_deprecated ON rules(deprecated);

CREATE TABLE IF NOT EXISTS rule_scores (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	rule_id TEXT NOT NULL REFERENCES rules(id) ON DELETE CASCADE,
	helpful INTEGER NOT NULL,
	context TEXT DEFAULT '',
	session_id TEXT DEFAULT '',
	scored_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_rule_scores_rule_id ON rule_scores(rule_id);

CREATE TABLE IF NOT EXISTS sessions (
	id TEXT PRIMARY KEY,
	project TEXT NOT NULL,
	started_at TEXT NOT NULL,
	ended_at TEXT,
	summary TEXT DEFAULT '',
	confidence REAL DEFAULT 0,
	tasks_completed INTEGER DEFAULT 0,
	tasks_failed INTEGER DEFAULT 0,
	tasks_verified INTEGER DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_sessions_project ON sessions(project);

CREATE TABLE IF NOT EXISTS events (
	id TEXT PRIMARY KEY,
	session_id TEXT NOT NULL,
	type TEXT NOT NULL,
	result TEXT NOT NULL DEFAULT '',
	details TEXT DEFAULT '',
	context TEXT DEFAULT '',
	timestamp TEXT NOT NULL,
	FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_events_session ON events(session_id);
CREATE INDEX IF NOT EXISTS idx_events_type ON events(type);
CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp);

CREATE TABLE IF NOT EXISTS improvements (
	id TEXT PRIMARY KEY,
	content TEXT NOT NULL,
	scope TEXT NOT NULL,
	project TEXT DEFAULT '',
	tags_json TEXT DEFAULT '{}',
	evidence TEXT DEFAULT '',
	confidence REAL DEFAULT 0,
	status TEXT DEFAULT 'pending',
	reason TEXT DEFAULT '',
	created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_improvements_status ON improvements(status);

CREATE TABLE IF NOT EXISTS schema_version (
	version INTEGER PRIMARY KEY
);
`

const currentSchemaVersion = 4

// Store provides database operations for the toolkit.
type Store struct {
	db *sql.DB
}

// Rule represents a rule in the database.
type Rule struct {
	ID             string              `json:"id"`
	Content        string              `json:"content"`
	Scope          string              `json:"scope"`
	Project        string              `json:"project,omitempty"`
	Workspace      string              `json:"workspace,omitempty"`
	Tags           map[string][]string `json:"tags,omitempty"`
	Effectiveness  float64             `json:"effectiveness"`
	LocalOnly      bool                `json:"local_only"`
	Sensitive      bool                `json:"sensitive"`
	CreatedFrom    string              `json:"created_from,omitempty"`
	SourceEvidence string              `json:"source_evidence,omitempty"`
	CreatedAt      time.Time           `json:"created_at"`
	UpdatedAt      time.Time           `json:"updated_at"`
	Deprecated     bool                `json:"deprecated"`
}

// Improvement represents a proposed rule improvement from pattern detection.
type Improvement struct {
	ID         string              `json:"id"`
	Content    string              `json:"content"`
	Scope      string              `json:"scope"`
	Project    string              `json:"project,omitempty"`
	Tags       map[string][]string `json:"tags,omitempty"`
	Evidence   string              `json:"evidence"`
	Confidence float64             `json:"confidence"`
	Status     string              `json:"status"` // pending, applied, rejected
	Reason     string              `json:"reason,omitempty"`
	CreatedAt  time.Time           `json:"created_at"`
}

// Session represents a Claude session in the database.
type Session struct {
	ID             string     `json:"id"`
	Project        string     `json:"project"`
	StartedAt      time.Time  `json:"started_at"`
	EndedAt        *time.Time `json:"ended_at,omitempty"`
	Summary        string     `json:"summary,omitempty"`
	Confidence     float64    `json:"confidence,omitempty"`
	TasksCompleted int        `json:"tasks_completed"`
	TasksFailed    int        `json:"tasks_failed"`
	TasksVerified  int        `json:"tasks_verified"`
}

// Event represents a telemetry event in the database.
type Event struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Type      string    `json:"type"`
	Result    string    `json:"result"`
	Details   string    `json:"details,omitempty"`
	Context   string    `json:"context,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// NewStore creates a new Store with the given database path.
// If dbPath is empty, uses ~/.claude-toolkit/store.db.
func NewStore(dbPath string) (*Store, error) {
	if dbPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("cannot determine home directory: %w", err)
		}
		dir := filepath.Join(home, ".claude-toolkit")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("cannot create data directory: %w", err)
		}
		dbPath = filepath.Join(dir, "store.db")
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("cannot open database: %w", err)
	}

	// Enable WAL mode and foreign keys
	if _, err := db.Exec("PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;"); err != nil {
		db.Close()
		return nil, fmt.Errorf("cannot set pragmas: %w", err)
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return &Store{db: db}, nil
}

// NewMemoryStore creates an in-memory store for testing.
func NewMemoryStore() (*Store, error) {
	return NewStore(":memory:")
}

func migrate(db *sql.DB) error {
	var version int
	err := db.QueryRow("SELECT version FROM schema_version LIMIT 1").Scan(&version)
	if err != nil {
		// Table doesn't exist or empty — run full schema
		if _, err := db.Exec(schema); err != nil {
			return fmt.Errorf("cannot create schema: %w", err)
		}
		_, err = db.Exec("INSERT INTO schema_version (version) VALUES (?)", currentSchemaVersion)
		return err
	}

	if version >= currentSchemaVersion {
		return nil
	}

	// Migration v2: Add sessions and events tables
	if version < 2 {
		migration := `
CREATE TABLE IF NOT EXISTS sessions (
	id TEXT PRIMARY KEY,
	project TEXT NOT NULL,
	started_at TEXT NOT NULL,
	ended_at TEXT,
	summary TEXT DEFAULT '',
	confidence REAL DEFAULT 0,
	tasks_completed INTEGER DEFAULT 0,
	tasks_failed INTEGER DEFAULT 0,
	tasks_verified INTEGER DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_sessions_project ON sessions(project);
CREATE TABLE IF NOT EXISTS events (
	id TEXT PRIMARY KEY,
	session_id TEXT NOT NULL,
	type TEXT NOT NULL,
	result TEXT NOT NULL DEFAULT '',
	details TEXT DEFAULT '',
	context TEXT DEFAULT '',
	timestamp TEXT NOT NULL,
	FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_events_session ON events(session_id);
CREATE INDEX IF NOT EXISTS idx_events_type ON events(type);
CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp);
UPDATE schema_version SET version = 2;`
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration v2 failed: %w", err)
		}
	}

	// Migration v3: Add improvements table
	if version < 3 {
		migration := `
CREATE TABLE IF NOT EXISTS improvements (
	id TEXT PRIMARY KEY,
	content TEXT NOT NULL,
	scope TEXT NOT NULL,
	project TEXT DEFAULT '',
	tags_json TEXT DEFAULT '{}',
	evidence TEXT DEFAULT '',
	confidence REAL DEFAULT 0,
	status TEXT DEFAULT 'pending',
	reason TEXT DEFAULT '',
	created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_improvements_status ON improvements(status);
UPDATE schema_version SET version = 3;`
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration v3 failed: %w", err)
		}
	}

	// Migration v4: Add tasks_verified to sessions
	if version < 4 {
		migration := `
ALTER TABLE sessions ADD COLUMN tasks_verified INTEGER DEFAULT 0;
UPDATE schema_version SET version = 4;`
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration v4 failed: %w", err)
		}
	}

	return nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// CreateRule inserts a new rule into the database.
func (s *Store) CreateRule(r *Rule) error {
	tagsJSON, err := json.Marshal(r.Tags)
	if err != nil {
		return fmt.Errorf("cannot marshal tags: %w", err)
	}

	now := time.Now().UTC()
	r.CreatedAt = now
	r.UpdatedAt = now

	_, err = s.db.Exec(`
		INSERT INTO rules (id, content, scope, project, workspace, tags_json, effectiveness, local_only, sensitive, created_from, source_evidence, created_at, updated_at, deprecated)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.Content, r.Scope, r.Project, r.Workspace,
		string(tagsJSON), r.Effectiveness,
		boolToInt(r.LocalOnly), boolToInt(r.Sensitive),
		r.CreatedFrom, r.SourceEvidence,
		now.Format(time.RFC3339), now.Format(time.RFC3339),
		boolToInt(r.Deprecated),
	)
	return err
}

// GetRule retrieves a rule by ID.
func (s *Store) GetRule(id string) (*Rule, error) {
	r := &Rule{}
	var tagsJSON string
	var localOnly, sensitive, deprecated int
	var createdAt, updatedAt string

	err := s.db.QueryRow(`
		SELECT id, content, scope, project, workspace, tags_json, effectiveness, local_only, sensitive, created_from, source_evidence, created_at, updated_at, deprecated
		FROM rules WHERE id = ?`, id).Scan(
		&r.ID, &r.Content, &r.Scope, &r.Project, &r.Workspace,
		&tagsJSON, &r.Effectiveness,
		&localOnly, &sensitive,
		&r.CreatedFrom, &r.SourceEvidence,
		&createdAt, &updatedAt,
		&deprecated,
	)
	if err != nil {
		return nil, err
	}

	r.LocalOnly = localOnly == 1
	r.Sensitive = sensitive == 1
	r.Deprecated = deprecated == 1
	r.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	r.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	if err := json.Unmarshal([]byte(tagsJSON), &r.Tags); err != nil {
		r.Tags = make(map[string][]string)
	}

	return r, nil
}

// UpdateRule updates a rule's fields.
func (s *Store) UpdateRule(r *Rule) error {
	tagsJSON, err := json.Marshal(r.Tags)
	if err != nil {
		return fmt.Errorf("cannot marshal tags: %w", err)
	}

	r.UpdatedAt = time.Now().UTC()

	result, err := s.db.Exec(`
		UPDATE rules SET content=?, scope=?, project=?, workspace=?, tags_json=?, effectiveness=?, local_only=?, sensitive=?, source_evidence=?, updated_at=?, deprecated=?
		WHERE id=?`,
		r.Content, r.Scope, r.Project, r.Workspace,
		string(tagsJSON), r.Effectiveness,
		boolToInt(r.LocalOnly), boolToInt(r.Sensitive),
		r.SourceEvidence, r.UpdatedAt.Format(time.RFC3339),
		boolToInt(r.Deprecated),
		r.ID,
	)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("rule %q not found", r.ID)
	}
	return nil
}

// DeleteRule removes a rule by ID.
func (s *Store) DeleteRule(id string) error {
	result, err := s.db.Exec("DELETE FROM rules WHERE id = ?", id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("rule %q not found", id)
	}
	return nil
}

// ListRules returns rules matching the given filters.
func (s *Store) ListRules(scope, project, techStack string) ([]Rule, error) {
	query := "SELECT id, content, scope, project, workspace, tags_json, effectiveness, local_only, sensitive, created_from, source_evidence, created_at, updated_at, deprecated FROM rules WHERE deprecated = 0"
	args := []interface{}{}

	if scope != "" {
		query += " AND scope = ?"
		args = append(args, scope)
	}
	if project != "" {
		query += " AND project = ?"
		args = append(args, project)
	}

	query += " ORDER BY effectiveness DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []Rule
	for rows.Next() {
		var r Rule
		var tagsJSON string
		var localOnly, sensitive, deprecated int
		var createdAt, updatedAt string

		err := rows.Scan(
			&r.ID, &r.Content, &r.Scope, &r.Project, &r.Workspace,
			&tagsJSON, &r.Effectiveness,
			&localOnly, &sensitive,
			&r.CreatedFrom, &r.SourceEvidence,
			&createdAt, &updatedAt,
			&deprecated,
		)
		if err != nil {
			return nil, err
		}

		r.LocalOnly = localOnly == 1
		r.Sensitive = sensitive == 1
		r.Deprecated = deprecated == 1
		r.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		r.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

		if err := json.Unmarshal([]byte(tagsJSON), &r.Tags); err != nil {
			r.Tags = make(map[string][]string)
		}

		// Filter by tech_stack if specified
		if techStack != "" {
			if stacks, ok := r.Tags["tech_stack"]; ok {
				found := false
				for _, s := range stacks {
					if s == techStack {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			} else {
				// Rules without tech_stack tags are always included (universal rules)
			}
		}

		rules = append(rules, r)
	}

	return rules, rows.Err()
}

// RecordScore records a score event for a rule.
func (s *Store) RecordScore(ruleID string, helpful bool, context, sessionID string) error {
	helpfulInt := 0
	if helpful {
		helpfulInt = 1
	}

	_, err := s.db.Exec(`
		INSERT INTO rule_scores (rule_id, helpful, context, session_id, scored_at)
		VALUES (?, ?, ?, ?, ?)`,
		ruleID, helpfulInt, context, sessionID, time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return err
	}

	// Recalculate effectiveness
	var totalHelpful, totalScores int
	err = s.db.QueryRow(`
		SELECT COALESCE(SUM(helpful), 0), COUNT(*) FROM rule_scores WHERE rule_id = ?`,
		ruleID).Scan(&totalHelpful, &totalScores)
	if err != nil {
		return err
	}

	if totalScores > 0 {
		newEffectiveness := float64(totalHelpful) / float64(totalScores)
		_, err = s.db.Exec("UPDATE rules SET effectiveness = ?, updated_at = ? WHERE id = ?",
			newEffectiveness, time.Now().UTC().Format(time.RFC3339), ruleID)
	}

	return err
}

// DeprecateLowScoreRules marks rules as deprecated if their effectiveness
// is below the threshold after a minimum number of scores.
// Returns the list of deprecated rule IDs.
func (s *Store) DeprecateLowScoreRules(threshold float64, minScores int) ([]string, error) {
	rows, err := s.db.Query(`
		SELECT r.id FROM rules r
		WHERE r.effectiveness < ? AND r.deprecated = 0
		AND (SELECT COUNT(*) FROM rule_scores rs WHERE rs.rule_id = r.id) >= ?`,
		threshold, minScores,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query low-score rules: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan rule id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(ids) == 0 {
		return ids, nil
	}

	now := time.Now().UTC().Format(time.RFC3339)
	for _, id := range ids {
		if _, err := s.db.Exec(
			"UPDATE rules SET deprecated = 1, updated_at = ? WHERE id = ?", now, id,
		); err != nil {
			return nil, fmt.Errorf("failed to deprecate rule %q: %w", id, err)
		}
	}

	return ids, nil
}

// CreateImprovement inserts a new improvement into the database.
func (s *Store) CreateImprovement(imp *Improvement) error {
	tagsJSON, err := json.Marshal(imp.Tags)
	if err != nil {
		return fmt.Errorf("cannot marshal tags: %w", err)
	}

	now := time.Now().UTC()
	imp.CreatedAt = now

	_, err = s.db.Exec(`
		INSERT INTO improvements (id, content, scope, project, tags_json, evidence, confidence, status, reason, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		imp.ID, imp.Content, imp.Scope, imp.Project,
		string(tagsJSON), imp.Evidence, imp.Confidence,
		imp.Status, imp.Reason,
		now.Format(time.RFC3339),
	)
	return err
}

// ListImprovements returns improvements ordered by created_at DESC.
// If status is empty, all improvements are returned.
func (s *Store) ListImprovements(status string) ([]*Improvement, error) {
	query := `SELECT id, content, scope, project, tags_json, evidence, confidence, status, reason, created_at
	          FROM improvements`
	args := []interface{}{}

	if status != "" {
		query += " WHERE status = ?"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var imps []*Improvement
	for rows.Next() {
		imp := &Improvement{}
		var tagsJSON, createdAt string
		err := rows.Scan(
			&imp.ID, &imp.Content, &imp.Scope, &imp.Project,
			&tagsJSON, &imp.Evidence, &imp.Confidence,
			&imp.Status, &imp.Reason, &createdAt,
		)
		if err != nil {
			return nil, err
		}
		imp.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		if err := json.Unmarshal([]byte(tagsJSON), &imp.Tags); err != nil {
			imp.Tags = make(map[string][]string)
		}
		imps = append(imps, imp)
	}
	return imps, rows.Err()
}

// UpdateImprovementStatus updates the status and optional reason of an improvement.
func (s *Store) UpdateImprovementStatus(id, status, reason string) error {
	result, err := s.db.Exec(
		"UPDATE improvements SET status = ?, reason = ? WHERE id = ?",
		status, reason, id,
	)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("improvement %q not found", id)
	}
	return nil
}

// CreateSession inserts a new session into the database.
func (s *Store) CreateSession(session *Session) error {
	_, err := s.db.Exec(`
		INSERT INTO sessions (id, project, started_at, summary, confidence, tasks_completed, tasks_failed, tasks_verified)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		session.ID, session.Project,
		session.StartedAt.Format(time.RFC3339),
		session.Summary, session.Confidence,
		session.TasksCompleted, session.TasksFailed, session.TasksVerified,
	)
	return err
}

// GetSession retrieves a session by ID.
func (s *Store) GetSession(id string) (*Session, error) {
	sess := &Session{}
	var startedAt string
	var endedAt sql.NullString

	err := s.db.QueryRow(`
		SELECT id, project, started_at, ended_at, summary, confidence, tasks_completed, tasks_failed, tasks_verified
		FROM sessions WHERE id = ?`, id).Scan(
		&sess.ID, &sess.Project, &startedAt, &endedAt,
		&sess.Summary, &sess.Confidence,
		&sess.TasksCompleted, &sess.TasksFailed, &sess.TasksVerified,
	)
	if err != nil {
		return nil, err
	}

	sess.StartedAt, _ = time.Parse(time.RFC3339, startedAt)
	if endedAt.Valid {
		t, _ := time.Parse(time.RFC3339, endedAt.String)
		sess.EndedAt = &t
	}

	return sess, nil
}

// EndSession marks a session as ended with summary data.
func (s *Store) EndSession(id string, summary string, confidence float64, tasksCompleted, tasksFailed, tasksVerified int) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.db.Exec(`
		UPDATE sessions SET ended_at=?, summary=?, confidence=?, tasks_completed=?, tasks_failed=?, tasks_verified=?
		WHERE id=?`,
		now, summary, confidence, tasksCompleted, tasksFailed, tasksVerified, id,
	)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("session %q not found", id)
	}
	return nil
}

// CreateEvent inserts a new event into the database.
func (s *Store) CreateEvent(event *Event) error {
	_, err := s.db.Exec(`
		INSERT INTO events (id, session_id, type, result, details, context, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		event.ID, event.SessionID, event.Type, event.Result,
		event.Details, event.Context,
		event.Timestamp.Format(time.RFC3339),
	)
	return err
}

// ListEvents returns all events for a given session, ordered by timestamp.
func (s *Store) ListEvents(sessionID string) ([]*Event, error) {
	rows, err := s.db.Query(`
		SELECT id, session_id, type, result, details, context, timestamp
		FROM events WHERE session_id = ? ORDER BY timestamp ASC`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*Event
	for rows.Next() {
		e := &Event{}
		var ts string
		err := rows.Scan(&e.ID, &e.SessionID, &e.Type, &e.Result, &e.Details, &e.Context, &ts)
		if err != nil {
			return nil, err
		}
		e.Timestamp, _ = time.Parse(time.RFC3339, ts)
		events = append(events, e)
	}
	return events, rows.Err()
}

// ListSessions returns sessions, optionally filtered by project, ordered by started_at DESC.
func (s *Store) ListSessions(project string, limit int) ([]*Session, error) {
	query := "SELECT id, project, started_at, ended_at, summary, confidence, tasks_completed, tasks_failed, tasks_verified FROM sessions"
	args := []interface{}{}

	if project != "" {
		query += " WHERE project = ?"
		args = append(args, project)
	}

	query += " ORDER BY started_at DESC"

	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		sess := &Session{}
		var startedAt string
		var endedAt sql.NullString

		err := rows.Scan(
			&sess.ID, &sess.Project, &startedAt, &endedAt,
			&sess.Summary, &sess.Confidence,
			&sess.TasksCompleted, &sess.TasksFailed, &sess.TasksVerified,
		)
		if err != nil {
			return nil, err
		}

		sess.StartedAt, _ = time.Parse(time.RFC3339, startedAt)
		if endedAt.Valid {
			t, _ := time.Parse(time.RFC3339, endedAt.String)
			sess.EndedAt = &t
		}

		sessions = append(sessions, sess)
	}
	return sessions, rows.Err()
}

// PurgeOldEvents deletes events older than the specified retention period.
func (s *Store) PurgeOldEvents(retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		return 0, fmt.Errorf("retentionDays must be positive, got %d", retentionDays)
	}
	cutoff := time.Now().UTC().Add(-time.Duration(retentionDays) * 24 * time.Hour).Format(time.RFC3339)
	result, err := s.db.Exec("DELETE FROM events WHERE timestamp < ?", cutoff)
	if err != nil {
		return 0, fmt.Errorf("purge failed: %w", err)
	}
	return result.RowsAffected()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// CountScoresPerRule returns a map of rule_id → score count for all rules.
func (s *Store) CountScoresPerRule() (map[string]int, error) {
	rows, err := s.db.Query(`SELECT rule_id, COUNT(*) FROM rule_scores GROUP BY rule_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var id string
		var count int
		if err := rows.Scan(&id, &count); err != nil {
			return nil, err
		}
		counts[id] = count
	}
	return counts, rows.Err()
}

// CountDeprecatedRules returns the total number of deprecated rules.
func (s *Store) CountDeprecatedRules() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM rules WHERE deprecated = 1").Scan(&count)
	return count, err
}

// IncrementTasksVerified atomically increments tasks_verified by 1 for the given session.
func (s *Store) IncrementTasksVerified(sessionID string) error {
	result, err := s.db.Exec(
		"UPDATE sessions SET tasks_verified = tasks_verified + 1 WHERE id = ?",
		sessionID,
	)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("session %q not found", sessionID)
	}
	return nil
}
