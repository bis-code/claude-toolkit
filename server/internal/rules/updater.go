package rules

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bis-code/claude-toolkit/server/internal/db"
)

// UpdateResult contains the result of a seed rules update check.
type UpdateResult struct {
	NewRulesLoaded int    `json:"new_rules_loaded"`
	RulesSkipped   int    `json:"rules_skipped"`
	FilesChecked   int    `json:"files_checked"`
	CurrentHash    string `json:"current_hash"`
}

// CheckAndUpdate loads any new seed rules from the seed directory.
// It computes a hash of all seed files to detect changes.
func CheckAndUpdate(store *db.Store, seedDir string) (*UpdateResult, error) {
	hash, fileCount, err := hashSeedDir(seedDir)
	if err != nil {
		return nil, fmt.Errorf("failed to hash seed directory: %w", err)
	}

	loaded, skipped, err := LoadSeeds(store, seedDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load seeds: %w", err)
	}

	return &UpdateResult{
		NewRulesLoaded: loaded,
		RulesSkipped:   skipped,
		FilesChecked:   fileCount,
		CurrentHash:    hash,
	}, nil
}

// hashSeedDir computes a SHA-256 hash of all JSON files in the seed directory.
// Includes filenames in the hash to prevent collisions when file boundaries shift.
func hashSeedDir(seedDir string) (string, int, error) {
	files, err := filepath.Glob(filepath.Join(seedDir, "*.json"))
	if err != nil {
		return "", 0, err
	}

	h := sha256.New()
	for _, file := range files {
		// Include the base filename to differentiate files with the same content.
		h.Write([]byte(filepath.Base(file)))

		data, err := os.ReadFile(file)
		if err != nil {
			return "", 0, fmt.Errorf("cannot read %s: %w", file, err)
		}
		h.Write(data)
	}

	return hex.EncodeToString(h.Sum(nil)), len(files), nil
}
