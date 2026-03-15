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
	tasks_verified INTEGER DEFAULT 0,
	transcript_path TEXT DEFAULT ''
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

CREATE TABLE IF NOT EXISTS workflow_events (
	id TEXT PRIMARY KEY,
	session_id TEXT NOT NULL,
	category TEXT NOT NULL CHECK(category IN ('coding_pattern','task_execution','problem_solving','preference')),
	pattern TEXT NOT NULL,
	details TEXT DEFAULT '',
	confidence REAL DEFAULT 0.5,
	occurrences INTEGER DEFAULT 1,
	first_seen TEXT NOT NULL,
	last_seen TEXT NOT NULL,
	project TEXT DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_workflow_category ON workflow_events(category);
CREATE INDEX IF NOT EXISTS idx_workflow_pattern ON workflow_events(pattern);
CREATE INDEX IF NOT EXISTS idx_workflow_confidence ON workflow_events(confidence);

CREATE TABLE IF NOT EXISTS skill_scores (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	skill TEXT NOT NULL,
	score REAL NOT NULL,
	session_id TEXT DEFAULT '',
	project TEXT DEFAULT '',
	details TEXT DEFAULT '',
	scored_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_skill_scores_skill ON skill_scores(skill);

CREATE TABLE IF NOT EXISTS schema_version (
	version INTEGER PRIMARY KEY
);
`

const currentSchemaVersion = 5

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
	TranscriptPath string     `json:"transcript_path,omitempty"`
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

	// Migration v5: Add workflow_events and skill_scores tables
	if version < 5 {
		migration := `
CREATE TABLE IF NOT EXISTS workflow_events (
	id TEXT PRIMARY KEY,
	session_id TEXT NOT NULL,
	category TEXT NOT NULL CHECK(category IN ('coding_pattern','task_execution','problem_solving','preference')),
	pattern TEXT NOT NULL,
	details TEXT DEFAULT '',
	confidence REAL DEFAULT 0.5,
	occurrences INTEGER DEFAULT 1,
	first_seen TEXT NOT NULL,
	last_seen TEXT NOT NULL,
	project TEXT DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_workflow_category ON workflow_events(category);
CREATE INDEX IF NOT EXISTS idx_workflow_pattern ON workflow_events(pattern);
CREATE INDEX IF NOT EXISTS idx_workflow_confidence ON workflow_events(confidence);
CREATE TABLE IF NOT EXISTS skill_scores (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	skill TEXT NOT NULL,
	score REAL NOT NULL,
	session_id TEXT DEFAULT '',
	project TEXT DEFAULT '',
	details TEXT DEFAULT '',
	scored_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_skill_scores_skill ON skill_scores(skill);
UPDATE schema_version SET version = 5;`
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration v5 failed: %w", err)
		}
	}

	// Migration v6: Add transcript_path to sessions
	if version < 6 {
		migration := `ALTER TABLE sessions ADD COLUMN transcript_path TEXT DEFAULT '';
UPDATE schema_version SET version = 6;`
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration v6 failed: %w", err)
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
	query := "SELECT id, project, started_at, ended_at, summary, confidence, tasks_completed, tasks_failed, tasks_verified, COALESCE(transcript_path, '') FROM sessions"
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
			&sess.TranscriptPath,
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

// CountMatchingEvents counts events with matching type, result, and details
// across the most recent N sessions (by started_at).
func (s *Store) CountMatchingEvents(eventType, eventResult, details string, sessionLimit int) (int, error) {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM events e
		JOIN sessions s ON e.session_id = s.id
		WHERE e.type = ? AND e.result = ? AND e.details = ?
		AND s.id IN (SELECT id FROM sessions ORDER BY started_at DESC LIMIT ?)`,
		eventType, eventResult, details, sessionLimit,
	).Scan(&count)
	return count, err
}

// ImprovementExistsForEvidence checks if a pending or applied improvement
// already exists with the given evidence key.
func (s *Store) ImprovementExistsForEvidence(evidence string) bool {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM improvements
		WHERE evidence = ? AND status IN ('pending', 'applied')`,
		evidence,
	).Scan(&count)
	return err == nil && count > 0
}

// ListRecentEvents returns the most recent events across all sessions,
// ordered by timestamp DESC. Used by the dashboard live-events panel.
func (s *Store) ListRecentEvents(limit int) ([]*Event, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(`
		SELECT id, session_id, type, result, details, context, timestamp
		FROM events
		ORDER BY timestamp DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*Event
	for rows.Next() {
		e := &Event{}
		var ts string
		if err := rows.Scan(&e.ID, &e.SessionID, &e.Type, &e.Result, &e.Details, &e.Context, &ts); err != nil {
			return nil, err
		}
		e.Timestamp, _ = time.Parse(time.RFC3339, ts)
		events = append(events, e)
	}
	return events, rows.Err()
}

// SkillScore holds the aggregated effectiveness metrics for a skill (event type).
type SkillScore struct {
	Name          string  `json:"name"`
	Total         int     `json:"total"`
	Successes     int     `json:"successes"`
	Failures      int     `json:"failures"`
	Effectiveness float64 `json:"effectiveness"`
}

// ListSkillScores returns effectiveness scores aggregated per event type.
func (s *Store) ListSkillScores() ([]SkillScore, error) {
	rows, err := s.db.Query(`
		SELECT
			type,
			COUNT(*) AS total,
			SUM(CASE WHEN result = 'success' THEN 1 ELSE 0 END) AS successes,
			SUM(CASE WHEN result = 'failure' THEN 1 ELSE 0 END) AS failures
		FROM events
		GROUP BY type
		ORDER BY total DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scores []SkillScore
	for rows.Next() {
		var ss SkillScore
		if err := rows.Scan(&ss.Name, &ss.Total, &ss.Successes, &ss.Failures); err != nil {
			return nil, err
		}
		if ss.Total > 0 {
			ss.Effectiveness = float64(ss.Successes) / float64(ss.Total)
		}
		scores = append(scores, ss)
	}
	return scores, rows.Err()
}

// WorkflowPattern holds a learned workflow pattern (improvement with confidence).
type WorkflowPattern struct {
	Content    string  `json:"content"`
	Scope      string  `json:"scope"`
	Confidence float64 `json:"confidence"`
	Evidence   string  `json:"evidence,omitempty"`
}

// ListWorkflowPatterns returns applied improvements as learned workflow patterns.
func (s *Store) ListWorkflowPatterns(limit int) ([]WorkflowPattern, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.Query(`
		SELECT content, scope, confidence, evidence
		FROM improvements
		WHERE status IN ('applied', 'pending')
		ORDER BY confidence DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var patterns []WorkflowPattern
	for rows.Next() {
		var p WorkflowPattern
		if err := rows.Scan(&p.Content, &p.Scope, &p.Confidence, &p.Evidence); err != nil {
			return nil, err
		}
		patterns = append(patterns, p)
	}
	return patterns, rows.Err()
}

// WorkflowEvent represents a learned workflow pattern stored in workflow_events.
type WorkflowEvent struct {
	ID          string    `json:"id"`
	SessionID   string    `json:"session_id"`
	Category    string    `json:"category"`
	Pattern     string    `json:"pattern"`
	Details     string    `json:"details,omitempty"`
	Confidence  float64   `json:"confidence"`
	Occurrences int       `json:"occurrences"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
	Project     string    `json:"project,omitempty"`
}

// SkillScoreRecord represents a skill score entry stored in skill_scores.
type SkillScoreRecord struct {
	ID        int64     `json:"id"`
	Skill     string    `json:"skill"`
	Score     float64   `json:"score"`
	SessionID string    `json:"session_id,omitempty"`
	Project   string    `json:"project,omitempty"`
	Details   string    `json:"details,omitempty"`
	ScoredAt  time.Time `json:"scored_at"`
}

// WorkflowStats holds aggregated stats for workflow events, grouped by category.
type WorkflowStats struct {
	Category      string          `json:"category"`
	Patterns      []WorkflowEvent `json:"patterns"`
	TotalCount    int             `json:"total_count"`
	AvgConfidence float64         `json:"avg_confidence"`
}

// SkillStats holds aggregated stats for a single skill.
type SkillStats struct {
	Skill      string    `json:"skill"`
	AvgScore   float64   `json:"avg_score"`
	UsageCount int       `json:"usage_count"`
	Trend      []float64 `json:"trend"`
}

// CreateWorkflowEvent inserts a new workflow event into the database.
func (s *Store) CreateWorkflowEvent(we *WorkflowEvent) error {
	now := time.Now().UTC()
	if we.FirstSeen.IsZero() {
		we.FirstSeen = now
	}
	if we.LastSeen.IsZero() {
		we.LastSeen = now
	}

	_, err := s.db.Exec(`
		INSERT INTO workflow_events (id, session_id, category, pattern, details, confidence, occurrences, first_seen, last_seen, project)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		we.ID, we.SessionID, we.Category, we.Pattern, we.Details,
		we.Confidence, we.Occurrences,
		we.FirstSeen.Format(time.RFC3339),
		we.LastSeen.Format(time.RFC3339),
		we.Project,
	)
	return err
}

// ListWorkflowEvents returns workflow events, optionally filtered by project, sorted by confidence DESC.
func (s *Store) ListWorkflowEvents(project string) ([]WorkflowEvent, error) {
	query := `SELECT id, session_id, category, pattern, details, confidence, occurrences, first_seen, last_seen, project
	          FROM workflow_events`
	args := []interface{}{}

	if project != "" {
		query += " WHERE project = ?"
		args = append(args, project)
	}
	query += " ORDER BY confidence DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []WorkflowEvent
	for rows.Next() {
		var we WorkflowEvent
		var firstSeen, lastSeen string
		if err := rows.Scan(
			&we.ID, &we.SessionID, &we.Category, &we.Pattern,
			&we.Details, &we.Confidence, &we.Occurrences,
			&firstSeen, &lastSeen, &we.Project,
		); err != nil {
			return nil, err
		}
		we.FirstSeen, _ = time.Parse(time.RFC3339, firstSeen)
		we.LastSeen, _ = time.Parse(time.RFC3339, lastSeen)
		events = append(events, we)
	}
	return events, rows.Err()
}

// UpdateWorkflowEvent updates occurrences, last_seen, and confidence for an existing workflow event.
func (s *Store) UpdateWorkflowEvent(id string, occurrences int, confidence float64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.db.Exec(`
		UPDATE workflow_events SET occurrences = ?, confidence = ?, last_seen = ? WHERE id = ?`,
		occurrences, confidence, now, id,
	)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("workflow event %q not found", id)
	}
	return nil
}

// GetWorkflowEventByPatternAndProject finds an existing workflow event by pattern+project.
// Returns nil, nil if not found.
func (s *Store) GetWorkflowEventByPatternAndProject(pattern, project string) (*WorkflowEvent, error) {
	we := &WorkflowEvent{}
	var firstSeen, lastSeen string

	err := s.db.QueryRow(`
		SELECT id, session_id, category, pattern, details, confidence, occurrences, first_seen, last_seen, project
		FROM workflow_events WHERE pattern = ? AND project = ? LIMIT 1`,
		pattern, project,
	).Scan(
		&we.ID, &we.SessionID, &we.Category, &we.Pattern,
		&we.Details, &we.Confidence, &we.Occurrences,
		&firstSeen, &lastSeen, &we.Project,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	we.FirstSeen, _ = time.Parse(time.RFC3339, firstSeen)
	we.LastSeen, _ = time.Parse(time.RFC3339, lastSeen)
	return we, nil
}

// GetWorkflowStats returns workflow events grouped by category with aggregate stats.
func (s *Store) GetWorkflowStats(project string) ([]WorkflowStats, error) {
	events, err := s.ListWorkflowEvents(project)
	if err != nil {
		return nil, err
	}

	grouped := make(map[string][]WorkflowEvent)
	for _, we := range events {
		grouped[we.Category] = append(grouped[we.Category], we)
	}

	var stats []WorkflowStats
	for category, patterns := range grouped {
		totalConf := 0.0
		for _, p := range patterns {
			totalConf += p.Confidence
		}
		avg := 0.0
		if len(patterns) > 0 {
			avg = totalConf / float64(len(patterns))
		}
		stats = append(stats, WorkflowStats{
			Category:      category,
			Patterns:      patterns,
			TotalCount:    len(patterns),
			AvgConfidence: avg,
		})
	}
	return stats, nil
}

// CreateSkillScore inserts a new skill score record into the database.
func (s *Store) CreateSkillScore(rec *SkillScoreRecord) error {
	now := time.Now().UTC()
	if rec.ScoredAt.IsZero() {
		rec.ScoredAt = now
	}

	result, err := s.db.Exec(`
		INSERT INTO skill_scores (skill, score, session_id, project, details, scored_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		rec.Skill, rec.Score, rec.SessionID, rec.Project,
		rec.Details, rec.ScoredAt.Format(time.RFC3339),
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	rec.ID = id
	return nil
}

// ListSkillScoreRecords returns skill score records, optionally filtered by skill and project.
func (s *Store) ListSkillScoreRecords(skill, project string) ([]SkillScoreRecord, error) {
	query := `SELECT id, skill, score, session_id, project, details, scored_at FROM skill_scores WHERE 1=1`
	args := []interface{}{}

	if skill != "" {
		query += " AND skill = ?"
		args = append(args, skill)
	}
	if project != "" {
		query += " AND project = ?"
		args = append(args, project)
	}
	query += " ORDER BY scored_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []SkillScoreRecord
	for rows.Next() {
		var rec SkillScoreRecord
		var scoredAt string
		if err := rows.Scan(
			&rec.ID, &rec.Skill, &rec.Score,
			&rec.SessionID, &rec.Project, &rec.Details, &scoredAt,
		); err != nil {
			return nil, err
		}
		rec.ScoredAt, _ = time.Parse(time.RFC3339, scoredAt)
		records = append(records, rec)
	}
	return records, rows.Err()
}

// GetSkillStats returns aggregated stats for skills. If skill is non-empty, returns stats for that skill only.
func (s *Store) GetSkillStats(skill, project string) ([]SkillStats, error) {
	records, err := s.ListSkillScoreRecords(skill, project)
	if err != nil {
		return nil, err
	}

	grouped := make(map[string][]float64)
	for _, rec := range records {
		grouped[rec.Skill] = append(grouped[rec.Skill], rec.Score)
	}

	var stats []SkillStats
	for sk, scores := range grouped {
		total := 0.0
		for _, sc := range scores {
			total += sc
		}
		avg := total / float64(len(scores))

		// Trend: last 5 scores (scores are DESC ordered, so these are the most recent)
		trend := scores
		if len(trend) > 5 {
			trend = trend[:5]
		}

		stats = append(stats, SkillStats{
			Skill:      sk,
			AvgScore:   avg,
			UsageCount: len(scores),
			Trend:      trend,
		})
	}
	return stats, nil
}
