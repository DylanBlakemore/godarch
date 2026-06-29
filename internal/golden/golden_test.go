package golden_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dylanblakemore/godarch/internal/golden"
)

type sample struct {
	Name string `json:"name"`
	N    int    `json:"n"`
}

func TestAssertJSONMatchesExistingGolden(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.json")
	// A golden written exactly as Marshal would render it must compare equal.
	if err := os.WriteFile(path, []byte("{\n  \"name\": \"a\",\n  \"n\": 1\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	golden.AssertJSON(t, path, sample{Name: "a", N: 1})
}

func TestAssertJSONRegeneratesWhenEnabled(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "out.json")
	t.Setenv("UPDATE_GOLDEN", "1")

	// File does not exist yet; regeneration must create it (and its parent dir).
	golden.AssertJSON(t, path, sample{Name: "b", N: 2})

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("golden not written: %v", err)
	}
	want := "{\n  \"name\": \"b\",\n  \"n\": 2\n}\n"
	if string(data) != want {
		t.Errorf("golden = %q, want %q", data, want)
	}
}
