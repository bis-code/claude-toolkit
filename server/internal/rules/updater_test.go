package rules_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bis-code/claude-toolkit/server/internal/db"
	"github.com/bis-code/claude-toolkit/server/internal/rules"
)

func TestCheckAndUpdate_LoadsNewRules(t *testing.T) {
	store, _ := db.NewMemoryStore()
	defer store.Close()

	dir := t.TempDir()
	writeSeedFile(t, dir, "global.json", `{
		"rules": [
			{"id": "seed-upd-001", "content": "Rule one", "scope": "global"},
			{"id": "seed-upd-002", "content": "Rule two", "scope": "global"}
		]
	}`)

	result, err := rules.CheckAndUpdate(store, dir)
	if err != nil {
		t.Fatalf("CheckAndUpdate failed: %v", err)
	}
	if result.NewRulesLoaded != 2 {
		t.Errorf("NewRulesLoaded = %d, want 2", result.NewRulesLoaded)
	}
	if result.RulesSkipped != 0 {
		t.Errorf("RulesSkipped = %d, want 0", result.RulesSkipped)
	}
	if result.FilesChecked != 1 {
		t.Errorf("FilesChecked = %d, want 1", result.FilesChecked)
	}
	if result.CurrentHash == "" {
		t.Error("CurrentHash should not be empty")
	}
}

func TestCheckAndUpdate_SkipsExisting(t *testing.T) {
	store, _ := db.NewMemoryStore()
	defer store.Close()

	dir := t.TempDir()
	writeSeedFile(t, dir, "test.json", `{
		"rules": [
			{"id": "seed-skip-001", "content": "Existing rule", "scope": "global"},
			{"id": "seed-skip-002", "content": "New rule", "scope": "global"}
		]
	}`)

	// Pre-load one rule
	store.CreateRule(&db.Rule{
		ID:      "seed-skip-001",
		Content: "Existing rule",
		Scope:   "global",
	})

	result, err := rules.CheckAndUpdate(store, dir)
	if err != nil {
		t.Fatalf("CheckAndUpdate failed: %v", err)
	}
	if result.NewRulesLoaded != 1 {
		t.Errorf("NewRulesLoaded = %d, want 1", result.NewRulesLoaded)
	}
	if result.RulesSkipped != 1 {
		t.Errorf("RulesSkipped = %d, want 1", result.RulesSkipped)
	}
}

func TestCheckAndUpdate_HashChangesWithContent(t *testing.T) {
	store1, _ := db.NewMemoryStore()
	defer store1.Close()
	store2, _ := db.NewMemoryStore()
	defer store2.Close()

	dir := t.TempDir()
	writeSeedFile(t, dir, "test.json", `{
		"rules": [{"id": "seed-hash-001", "content": "Version 1", "scope": "global"}]
	}`)

	result1, err := rules.CheckAndUpdate(store1, dir)
	if err != nil {
		t.Fatalf("first CheckAndUpdate failed: %v", err)
	}

	// Modify the file content
	writeSeedFile(t, dir, "test.json", `{
		"rules": [{"id": "seed-hash-001", "content": "Version 2", "scope": "global"}]
	}`)

	result2, err := rules.CheckAndUpdate(store2, dir)
	if err != nil {
		t.Fatalf("second CheckAndUpdate failed: %v", err)
	}

	if result1.CurrentHash == result2.CurrentHash {
		t.Error("hash should change when file content changes")
	}
}

func TestCheckAndUpdate_HashConsistentForSameContent(t *testing.T) {
	store1, _ := db.NewMemoryStore()
	defer store1.Close()
	store2, _ := db.NewMemoryStore()
	defer store2.Close()

	dir := t.TempDir()
	writeSeedFile(t, dir, "test.json", `{
		"rules": [{"id": "seed-cons-001", "content": "Stable content", "scope": "global"}]
	}`)

	result1, err := rules.CheckAndUpdate(store1, dir)
	if err != nil {
		t.Fatalf("first CheckAndUpdate failed: %v", err)
	}

	result2, err := rules.CheckAndUpdate(store2, dir)
	if err != nil {
		t.Fatalf("second CheckAndUpdate failed: %v", err)
	}

	if result1.CurrentHash != result2.CurrentHash {
		t.Errorf("hash should be consistent: %q != %q", result1.CurrentHash, result2.CurrentHash)
	}
}

func TestCheckAndUpdate_EmptyDir(t *testing.T) {
	store, _ := db.NewMemoryStore()
	defer store.Close()

	dir := t.TempDir()

	result, err := rules.CheckAndUpdate(store, dir)
	if err != nil {
		t.Fatalf("CheckAndUpdate failed: %v", err)
	}
	if result.NewRulesLoaded != 0 {
		t.Errorf("NewRulesLoaded = %d, want 0", result.NewRulesLoaded)
	}
	if result.RulesSkipped != 0 {
		t.Errorf("RulesSkipped = %d, want 0", result.RulesSkipped)
	}
	if result.FilesChecked != 0 {
		t.Errorf("FilesChecked = %d, want 0", result.FilesChecked)
	}
}

func TestCheckAndUpdate_HashIncludesFilenames(t *testing.T) {
	// Two directories with the same content but different filenames should
	// produce different hashes. This prevents collision when file boundaries shift.
	store1, _ := db.NewMemoryStore()
	defer store1.Close()
	store2, _ := db.NewMemoryStore()
	defer store2.Close()

	dir1 := t.TempDir()
	dir2 := t.TempDir()

	content := `{"rules": [{"id": "seed-fn-001", "content": "Same", "scope": "global"}]}`
	writeSeedFile(t, dir1, "alpha.json", content)
	writeSeedFile(t, dir2, "beta.json", content)

	result1, err := rules.CheckAndUpdate(store1, dir1)
	if err != nil {
		t.Fatalf("first CheckAndUpdate failed: %v", err)
	}
	result2, err := rules.CheckAndUpdate(store2, dir2)
	if err != nil {
		t.Fatalf("second CheckAndUpdate failed: %v", err)
	}

	if result1.CurrentHash == result2.CurrentHash {
		t.Error("hash should differ when filenames differ (same content)")
	}
}

func TestCheckAndUpdate_NonexistentDir(t *testing.T) {
	store, _ := db.NewMemoryStore()
	defer store.Close()

	_, err := rules.CheckAndUpdate(store, filepath.Join(t.TempDir(), "nonexistent"))
	// Should not error -- filepath.Glob returns nil for no matches, not an error.
	// But LoadSeeds will also get no files. Either behavior (error or empty result) is fine,
	// as long as it doesn't panic.
	if err != nil {
		// Acceptable -- the directory doesn't exist.
		t.Logf("got expected error for nonexistent dir: %v", err)
	}
}

func TestHashSeedDir_MultipleFiles_OrderIndependent(t *testing.T) {
	// filepath.Glob returns lexicographic order, so creating files in any order
	// should produce the same hash as long as the filenames and content are the same.
	dir := t.TempDir()

	// Create files in reverse order
	content1 := `{"rules": [{"id": "a", "content": "A", "scope": "global"}]}`
	content2 := `{"rules": [{"id": "b", "content": "B", "scope": "global"}]}`
	writeSeedFile(t, dir, "zzz.json", content2)
	writeSeedFile(t, dir, "aaa.json", content1)

	store1, _ := db.NewMemoryStore()
	defer store1.Close()
	result1, _ := rules.CheckAndUpdate(store1, dir)

	// Recreate in a fresh dir with same files
	dir2 := t.TempDir()
	writeSeedFile(t, dir2, "aaa.json", content1)
	writeSeedFile(t, dir2, "zzz.json", content2)

	store2, _ := db.NewMemoryStore()
	defer store2.Close()
	result2, _ := rules.CheckAndUpdate(store2, dir2)

	if result1.CurrentHash != result2.CurrentHash {
		t.Errorf("hash should be the same regardless of creation order: %q != %q",
			result1.CurrentHash, result2.CurrentHash)
	}
}

// Cleanup: ensure we don't leave test files behind (TempDir handles this).
func TestCheckAndUpdate_InvalidJSON(t *testing.T) {
	store, _ := db.NewMemoryStore()
	defer store.Close()

	dir := t.TempDir()
	// Hash will succeed but LoadSeeds should fail on invalid JSON
	os.WriteFile(filepath.Join(dir, "bad.json"), []byte("not valid json"), 0o644)

	_, err := rules.CheckAndUpdate(store, dir)
	if err == nil {
		t.Error("expected error for invalid JSON seed file")
	}
}
