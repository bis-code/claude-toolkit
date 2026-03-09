package rules_test

import (
	"testing"

	"github.com/bis-code/claude-toolkit/server/internal/rules"
)

func TestIsSensitive_DetectsPrivateKeys(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    bool
	}{
		{"ethereum private key", "Use private key 0x1234567890abcdef1234567890abcdef1234567890abcdef", true},
		{"key=value format", "private_key: sk_test_abc123def456ghi789", true},
		{"prefixed key", "sk-abcdefghijklmnopqrstuvwxyz", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := rules.IsSensitive(tc.content); got != tc.want {
				t.Errorf("IsSensitive(%q) = %v, want %v", tc.content, got, tc.want)
			}
		})
	}
}

func TestIsSensitive_DetectsAPIKeys(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    bool
	}{
		{"api key assignment", "api_key: abcdef123456789012345", true},
		{"github token", "Use ghp_abcdefghijklmnopqrstuvwxyz1234567890", true},
		{"bearer token", "bearer: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9", true},
		{"slack token", "xoxb-123456789012-abcdefghij", true},
		{"aws key", "AKIAIOSFODNN7EXAMPLE1", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := rules.IsSensitive(tc.content); got != tc.want {
				t.Errorf("IsSensitive(%q) = %v, want %v", tc.content, got, tc.want)
			}
		})
	}
}

func TestIsSensitive_DetectsPasswords(t *testing.T) {
	if !rules.IsSensitive("password: mysecretpassword123") {
		t.Error("should detect password assignment")
	}
	if !rules.IsSensitive("DB_PASSWORD=secretpass") {
		t.Error("should detect password in env var format")
	}
}

func TestIsSensitive_DetectsEmails(t *testing.T) {
	if !rules.IsSensitive("Contact user@example.com for access") {
		t.Error("should detect email address")
	}
}

func TestIsSensitive_DetectsURLsWithCredentials(t *testing.T) {
	if !rules.IsSensitive("Connect to https://admin:password123@internal.example.com") {
		t.Error("should detect URL with credentials")
	}
	if !rules.IsSensitive("postgres://user:pass@localhost:5432/db") {
		t.Error("should detect connection string with password")
	}
}

func TestIsSensitive_AllowsCleanRules(t *testing.T) {
	cleanRules := []string{
		"Always write tests first",
		"Use conventional commits format",
		"Run {{test_cmd}} before committing",
		"Check Application.isPlaying before accessing EditorPrefs",
		"Use context.Context as the first parameter in Go functions",
		"Always estimate gas before blockchain transactions",
		"Never use SELECT * in production queries",
		"Paginate all list endpoints with a max page size of 100",
	}

	for _, content := range cleanRules {
		if rules.IsSensitive(content) {
			t.Errorf("false positive: %q flagged as sensitive", content)
		}
	}
}

func TestSensitivityReport_ReturnsMatches(t *testing.T) {
	content := "api_key: abc123def456789012345 and password: secret123"
	matches := rules.SensitivityReport(content)

	if len(matches) < 2 {
		t.Errorf("expected at least 2 matches, got %d: %v", len(matches), matches)
	}
}

func TestSensitivityReport_EmptyForClean(t *testing.T) {
	matches := rules.SensitivityReport("Always write tests first")
	if len(matches) != 0 {
		t.Errorf("expected 0 matches for clean content, got %d: %v", len(matches), matches)
	}
}
