package rules

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bis-code/claude-toolkit/server/internal/db"
)

// SeedFile represents the JSON format for seed rule files.
type SeedFile struct {
	Rules []SeedRule `json:"rules"`
}

// SeedRule is the JSON format for a single seed rule.
type SeedRule struct {
	ID             string              `json:"id"`
	Content        string              `json:"content"`
	Scope          string              `json:"scope"`
	Tags           map[string][]string `json:"tags,omitempty"`
	SourceEvidence string              `json:"source_evidence,omitempty"`
}

// LoadSeeds loads seed rules from JSON files in the given directory.
// Only loads rules that don't already exist in the store (idempotent).
func LoadSeeds(store *db.Store, seedDir string) (loaded int, skipped int, err error) {
	files, err := filepath.Glob(filepath.Join(seedDir, "*.json"))
	if err != nil {
		return 0, 0, fmt.Errorf("cannot glob seed directory: %w", err)
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return loaded, skipped, fmt.Errorf("cannot read %s: %w", file, err)
		}

		var sf SeedFile
		if err := json.Unmarshal(data, &sf); err != nil {
			return loaded, skipped, fmt.Errorf("cannot parse %s: %w", file, err)
		}

		for _, seed := range sf.Rules {
			// Check if rule already exists
			if _, err := store.GetRule(seed.ID); err == nil {
				skipped++
				continue
			}

			rule := &db.Rule{
				ID:             seed.ID,
				Content:        seed.Content,
				Scope:          seed.Scope,
				Tags:           seed.Tags,
				Effectiveness:  0.5, // Start at neutral
				CreatedFrom:    "seed",
				SourceEvidence: seed.SourceEvidence,
			}

			if err := store.CreateRule(rule); err != nil {
				return loaded, skipped, fmt.Errorf("cannot create seed rule %s: %w", seed.ID, err)
			}
			loaded++
		}
	}

	return loaded, skipped, nil
}
