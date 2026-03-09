package rules

import (
	"regexp"
	"strings"
)

// sensitivityPatterns detect secrets, PII, and sensitive content in rule text.
var sensitivityPatterns = []*regexp.Regexp{
	// Private keys (hex, base64-ish)
	regexp.MustCompile(`(?i)(private[_\s-]?key|secret[_\s-]?key)\s*[:=]\s*\S+`),
	regexp.MustCompile(`0x[0-9a-fA-F]{40,}`),                        // Ethereum-style private key
	regexp.MustCompile(`(?i)(sk|pk|priv)[_-][a-zA-Z0-9]{20,}`),      // Prefixed secret keys

	// API keys and tokens
	regexp.MustCompile(`(?i)(api[_\s-]?key|api[_\s-]?token|auth[_\s-]?token|bearer)\s*[:=]\s*\S{10,}`),
	regexp.MustCompile(`(?i)(access[_\s-]?token|refresh[_\s-]?token)\s*[:=]\s*\S{10,}`),
	regexp.MustCompile(`ghp_[a-zA-Z0-9]{36,}`),                      // GitHub personal access token
	regexp.MustCompile(`sk-[a-zA-Z0-9]{20,}`),                       // OpenAI/Stripe secret key
	regexp.MustCompile(`xox[bpras]-[a-zA-Z0-9\-]{10,}`),             // Slack tokens

	// Passwords
	regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[:=]\s*\S{4,}`),

	// Email addresses (bulk/specific, not generic mentions)
	regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`),

	// URLs with credentials
	regexp.MustCompile(`https?://[^:]+:[^@]+@`),

	// AWS keys
	regexp.MustCompile(`AKIA[0-9A-Z]{16}`),

	// Connection strings with passwords
	regexp.MustCompile(`(?i)(mongodb|postgres|mysql|redis)://[^:]+:[^@]+@`),
}

// IsSensitive checks if rule content contains potential secrets or PII.
func IsSensitive(content string) bool {
	for _, pattern := range sensitivityPatterns {
		if pattern.MatchString(content) {
			return true
		}
	}
	return false
}

// SensitivityReport returns which patterns matched in the content.
func SensitivityReport(content string) []string {
	var matches []string
	for _, pattern := range sensitivityPatterns {
		if loc := pattern.FindString(content); loc != "" {
			// Truncate the match to avoid exposing the actual secret
			if len(loc) > 30 {
				loc = loc[:30] + "..."
			}
			matches = append(matches, strings.TrimSpace(loc))
		}
	}
	return matches
}
