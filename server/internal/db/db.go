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

CREATE TABLE IF NOT EXISTS schema_version (
	version INTEGER PRIMARY KEY
);
`

const currentSchemaVersion = 1

// Store provides database operations for the toolkit.
type Store struct {
	db *sql.DB
}

// Rule represents a rule in the database.
type Rule struct {
	ID             string            `json:"id"`
	Content        string            `json:"content"`
	Scope          string            `json:"scope"`
	Project        string            `json:"project,omitempty"`
	Workspace      string            `json:"workspace,omitempty"`
	Tags           map[string][]string `json:"tags,omitempty"`
	Effectiveness  float64           `json:"effectiveness"`
	LocalOnly      bool              `json:"local_only"`
	Sensitive      bool              `json:"sensitive"`
	CreatedFrom    string            `json:"created_from,omitempty"`
	SourceEvidence string            `json:"source_evidence,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	Deprecated     bool              `json:"deprecated"`
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

	// Future migrations go here
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

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
