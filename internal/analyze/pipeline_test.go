package analyze_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dylanblakemore/godarch/internal/analyze"
	"github.com/dylanblakemore/godarch/internal/model"
)

const minimalFixture = "../../testdata/fixtures/minimal"

// TestRunMinimalFixture exercises the full assembly: discovery + both extractors
// merged into one project. It asserts the pipeline wires all three stages (file
// nodes from discovery, editor edges from the scene extractor, code-origin nodes
// from the GDScript extractor) and populates the unresolved subset.
func TestRunMinimalFixture(t *testing.T) {
	p, diags, err := analyze.Run(minimalFixture)
	if err != nil {
		t.Fatalf("Run(%s): %v", minimalFixture, err)
	}
	_ = diags // clean fixture may still surface best-effort diagnostics; not asserted here.

	// Discovery contributes file + concept nodes.
	if _, ok := p.Nodes["res://main.tscn"]; !ok {
		t.Errorf("missing scene node res://main.tscn")
	}
	if _, ok := p.Nodes["autoload:GameState"]; !ok {
		t.Errorf("missing autoload node autoload:GameState")
	}

	// The scene extractor contributes scene_node nodes and editor-origin edges.
	var sceneNodes, editorEdges int
	for _, n := range p.Nodes {
		if n.Kind == model.KindSceneNode {
			sceneNodes++
		}
	}
	for _, e := range p.Edges {
		if e.Origin == model.OriginEditor {
			editorEdges++
		}
	}
	if sceneNodes == 0 {
		t.Errorf("expected scene_node nodes from the scene extractor, got 0")
	}
	if editorEdges == 0 {
		t.Errorf("expected editor-origin edges from the scene extractor, got 0")
	}

	// The GDScript extractor contributes method nodes and code-origin edges.
	var methodNodes, codeEdges int
	for _, n := range p.Nodes {
		if n.Kind == model.KindMethod {
			methodNodes++
		}
	}
	for _, e := range p.Edges {
		if e.Origin == model.OriginCode {
			codeEdges++
		}
	}
	if methodNodes == 0 {
		t.Errorf("expected method nodes from the GDScript extractor, got 0")
	}
	if codeEdges == 0 {
		t.Errorf("expected code-origin edges from the GDScript extractor, got 0")
	}

	if len(p.Boundaries) == 0 {
		t.Errorf("expected boundary points, got 0")
	}

	// Unresolved is the subset of edges whose target is still a match key.
	var want int
	for _, e := range p.Edges {
		if !e.Resolved {
			want++
		}
	}
	if len(p.Unresolved) != want {
		t.Errorf("Unresolved has %d edges, want %d (all !Resolved edges)", len(p.Unresolved), want)
	}
	if want == 0 {
		t.Errorf("expected some unresolved edges in the minimal fixture (resolution is M2)")
	}
}

// TestRunDedupesStubClassNodes verifies the stub-enrichment rule the plan calls
// out: a class referenced before its declaring script is parsed produces exactly
// one class node, enriched with the declaring script — regardless of the order in
// which the files are visited (extractor visits IDs sorted, so the two fixtures
// force both orderings by name).
func TestRunDedupesStubClassNodes(t *testing.T) {
	tests := []struct {
		name       string
		declFile   string // file whose name declares class_name Foo
		refFile    string // file that references Foo via extends
		class      string
		wantScript string
	}{
		{
			name:       "declaration sorts before reference",
			declFile:   "a_decl.gd",
			refFile:    "b_ref.gd",
			class:      "Foo",
			wantScript: "res://a_decl.gd",
		},
		{
			name:       "reference sorts before declaration",
			declFile:   "y_decl.gd",
			refFile:    "x_ref.gd",
			class:      "Bar",
			wantScript: "res://y_decl.gd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, dir, "project.godot", "config_version=5\n")
			writeFile(t, dir, tt.declFile, "class_name "+tt.class+"\nextends Node\n")
			writeFile(t, dir, tt.refFile, "extends "+tt.class+"\n")

			p, _, err := analyze.Run(dir)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}

			id := model.ClassID(tt.class)
			n, ok := p.Nodes[id]
			if !ok {
				t.Fatalf("missing class node %s", id)
			}
			if got := n.Identity["script"]; got != tt.wantScript {
				t.Errorf("%s identity[script] = %v, want %s", id, got, tt.wantScript)
			}
		})
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}
