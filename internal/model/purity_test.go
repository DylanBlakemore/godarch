package model_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestModelImportsNoInternalPackages enforces the foundational rule from
// plan/00-foundation/01-repo-layout.md: package model is the leaf of the
// dependency graph and must import nothing from internal/*. Violating this is
// the most likely way to introduce an import cycle, so it is guarded here (and
// in CI) rather than left to review.
func TestModelImportsNoInternalPackages(t *testing.T) {
	const internalPrefix = "github.com/dylanblakemore/godarch/internal/"

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("reading model package dir: %v", err)
	}

	fset := token.NewFileSet()
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") {
			continue
		}

		file, err := parser.ParseFile(fset, name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parsing %s: %v", name, err)
		}

		// The purity rule applies to the model package itself, not to external
		// test packages (model_test) which legitimately import other code to
		// exercise it.
		if strings.HasSuffix(file.Name.Name, "_test") {
			continue
		}

		for _, imp := range file.Imports {
			path, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				t.Fatalf("%s: bad import path %s", name, imp.Path.Value)
			}
			if strings.HasPrefix(path, internalPrefix) {
				t.Errorf("%s imports %q; package model must not import any internal/* package", filepath.Base(name), path)
			}
		}
	}
}
