package rules_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// seedDir returns the absolute path to templates/rules/seed relative to this
// test file. Using runtime.Caller(0) makes the path stable regardless of the
// working directory when 'go test' is invoked.
func seedDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// thisFile: .../server/internal/rules/seed_community_test.go
	// repo root: three levels up from the rules/ directory
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	dir := filepath.Join(repoRoot, "templates", "rules", "seed")
	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatalf("cannot resolve seed dir: %v", err)
	}
	return abs
}

// expectedFiles lists every seed JSON file that must exist.
var expectedFiles = []struct {
	filename  string
	minRules  int
}{
	{"global.json", 10},
	{"go.json", 8},
	{"typescript.json", 8},
	{"python.json", 8},
	{"rust.json", 6},
	{"csharp.json", 6},
	{"blockchain.json", 6},
	{"frontend.json", 6},
}

// expectedTotalRules is the sum of all minRules above.
const expectedTotalRules = 10 + 8 + 8 + 8 + 6 + 6 + 6 + 6 // 58

func TestCommunitySeedFiles_AllFilesPresent(t *testing.T) {
	dir := seedDir(t)

	for _, tc := range expectedFiles {
		path := filepath.Join(dir, tc.filename)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("missing seed file: %s", path)
		}
	}
}

func TestCommunitySeedFiles_AllFilesParseCorrectly(t *testing.T) {
	dir := seedDir(t)

	for _, tc := range expectedFiles {
		tc := tc
		t.Run(tc.filename, func(t *testing.T) {
			path := filepath.Join(dir, tc.filename)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("cannot read %s: %v", tc.filename, err)
			}

			var sf SeedFile
			if err := json.Unmarshal(data, &sf); err != nil {
				t.Fatalf("cannot parse %s: %v", tc.filename, err)
			}

			if len(sf.Rules) < tc.minRules {
				t.Errorf("%s: expected at least %d rules, got %d", tc.filename, tc.minRules, len(sf.Rules))
			}

			for i, rule := range sf.Rules {
				if rule.ID == "" {
					t.Errorf("%s rule[%d]: id must not be empty", tc.filename, i)
				}
				if rule.Content == "" {
					t.Errorf("%s rule[%d] (id=%q): content must not be empty", tc.filename, i, rule.ID)
				}
				if rule.Scope == "" {
					t.Errorf("%s rule[%d] (id=%q): scope must not be empty", tc.filename, i, rule.ID)
				}
			}
		})
	}
}

func TestCommunitySeedFiles_TotalRuleCount(t *testing.T) {
	dir := seedDir(t)

	total := 0
	for _, tc := range expectedFiles {
		path := filepath.Join(dir, tc.filename)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("cannot read %s: %v", tc.filename, err)
		}
		var sf SeedFile
		if err := json.Unmarshal(data, &sf); err != nil {
			t.Fatalf("cannot parse %s: %v", tc.filename, err)
		}
		total += len(sf.Rules)
	}

	if total != expectedTotalRules {
		t.Errorf("total rule count = %d, want %d", total, expectedTotalRules)
	}
}

func TestCommunitySeedFiles_NoDuplicateIDs(t *testing.T) {
	dir := seedDir(t)

	seen := make(map[string]string) // id -> filename
	var duplicates []string

	for _, tc := range expectedFiles {
		path := filepath.Join(dir, tc.filename)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("cannot read %s: %v", tc.filename, err)
		}
		var sf SeedFile
		if err := json.Unmarshal(data, &sf); err != nil {
			t.Fatalf("cannot parse %s: %v", tc.filename, err)
		}
		for _, rule := range sf.Rules {
			if first, exists := seen[rule.ID]; exists {
				duplicates = append(duplicates, fmt.Sprintf("id %q appears in both %s and %s", rule.ID, first, tc.filename))
			} else {
				seen[rule.ID] = tc.filename
			}
		}
	}

	for _, dup := range duplicates {
		t.Error(dup)
	}
}

// SeedFile mirrors the rules.SeedFile type to keep this test independent of
// the production package's exported type — avoiding circular coupling with
// the package-level type used in seed_test.go.
//
// The production LoadSeeds function uses this same JSON shape.
type SeedFile struct {
	Rules []struct {
		ID             string              `json:"id"`
		Content        string              `json:"content"`
		Scope          string              `json:"scope"`
		Tags           map[string][]string `json:"tags,omitempty"`
		SourceEvidence string              `json:"source_evidence,omitempty"`
	} `json:"rules"`
}
