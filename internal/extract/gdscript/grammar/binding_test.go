package grammar_test

import (
	"testing"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/dylanblakemore/godarch/internal/extract/gdscript/grammar"
)

// TestParseSmoke confirms the vendored grammar is wired to the runtime: a small
// GDScript snippet parses into a non-empty, error-free source_file tree.
func TestParseSmoke(t *testing.T) {
	src := []byte("extends Node\nclass_name Foo\nsignal died\nfunc _ready() -> void:\n\tprint(\"hi\")\n")

	p := sitter.NewParser()
	defer p.Close()
	if err := p.SetLanguage(grammar.Language()); err != nil {
		t.Fatalf("SetLanguage: %v", err)
	}

	tree := p.Parse(src, nil)
	defer tree.Close()

	root := tree.RootNode()
	if got := root.Kind(); got != "source" {
		t.Fatalf("root kind = %q, want source", got)
	}
	if root.HasError() {
		t.Fatalf("parse produced errors in tree:\n%s", root.ToSexp())
	}
	if root.ChildCount() == 0 {
		t.Fatal("root has no children")
	}
	t.Logf("parsed tree: %s", root.ToSexp())
}
