// Package golden implements the project's golden-file test harness: a value is
// serialised to deterministic, indented JSON and compared against a committed
// golden file. Setting UPDATE_GOLDEN=1 in the environment regenerates the golden
// instead of asserting, so an intended change is reviewed as a diff to the
// committed file (archi's snapshot-review workflow).
//
// Callers are responsible for determinism of the value itself (sort slices, emit
// res:// paths not absolute ones, no timestamps) — see plan/00-foundation/05.
package golden

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// UpdateEnv is the environment variable that switches AssertJSON from asserting
// to regenerating the golden file.
const UpdateEnv = "UPDATE_GOLDEN"

// AssertJSON marshals got to indented JSON and compares it byte-for-byte with
// the golden file at path. When UPDATE_GOLDEN is set it (re)writes the golden,
// creating parent directories as needed, and does not assert. The encoded form
// always ends in a trailing newline so golden files are POSIX-clean.
func AssertJSON(t testing.TB, path string, got any) {
	t.Helper()

	data, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		t.Fatalf("golden: marshal value for %s: %v", path, err)
	}
	data = append(data, '\n')

	if os.Getenv(UpdateEnv) != "" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("golden: mkdir for %s: %v", path, err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatalf("golden: write %s: %v", path, err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("golden: read %s: %v (run with %s=1 to create it)", path, err, UpdateEnv)
	}
	if string(want) != string(data) {
		t.Errorf("golden mismatch for %s\n--- want\n%s\n--- got\n%s\nrun with %s=1 to update",
			path, want, data, UpdateEnv)
	}
}
