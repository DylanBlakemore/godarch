package store_test

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/dylanblakemore/godarch/internal/model"
	"github.com/dylanblakemore/godarch/internal/store"
)

// hand-built Project that exercises every Kind, EdgeType, and BoundaryType — the
// DESIGN §3 coverage guarantee from the milestone exit criteria, now over the
// persistence layer rather than JSON. Edges holds the full edge set; Unresolved
// is the subset whose Resolved flag is false (the store's documented invariant).
func sampleProject() *model.Project {
	p := model.NewProject("res://")
	p.GodotVersion = "4.3"
	p.UIDMap["uid://abc"] = "res://player/player.gd"
	p.UIDMap["uid://def"] = "res://ui/hud.tscn"

	for i, k := range model.AllKinds {
		id := model.AutoloadID(string(k))
		n := &model.Node{
			ID:   id,
			Kind: k,
			Path: "res://x.gd",
			Line: i + 1,
		}
		// Leave some maps nil to exercise SQL NULL round-tripping.
		if i%2 == 0 {
			n.Identity = map[string]any{"name": string(k), "arity": float64(i)}
		}
		if i%3 == 0 {
			n.Properties = map[string]any{"fan_in": float64(i)}
		}
		p.Nodes[id] = n
	}

	for i, et := range model.AllEdgeTypes {
		resolved := i%2 == 0
		e := &model.Edge{
			Type:       et,
			SourceID:   "res://a.gd",
			TargetID:   "res://b.gd",
			Origin:     model.AllOrigins[i%len(model.AllOrigins)],
			Resolved:   resolved,
			Confidence: 0.5 + float64(i)/100,
			Evidence:   model.Evidence{File: "res://a.gd", Line: i + 1, Snippet: "x"},
			Properties: map[string]any{"method": "foo"},
		}
		p.Edges = append(p.Edges, e)
		if !resolved {
			p.Unresolved = append(p.Unresolved, e)
		}
	}

	for i, bt := range model.AllBoundaryTypes {
		b := &model.BoundaryPoint{
			Direction: model.AllDirections[i%len(model.AllDirections)],
			Type:      bt,
			NodeID:    "res://a.gd::foo",
			MatchKey:  model.SignalKey("Player", "died"),
			Evidence:  model.Evidence{File: "res://a.gd", Line: i + 1},
			Meta:      map[string]any{"k": "v"},
		}
		p.Boundaries = append(p.Boundaries, b)
	}

	return p
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.godarch.db")

	s, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	want := sampleProject()
	if err := s.SaveProject(want); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := s.LoadProject()
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if !reflect.DeepEqual(want, got) {
		t.Errorf("round trip mismatch:\n want: %#v\n got:  %#v", want, got)
	}
}

// TestSaveProjectReplaces verifies a second save fully replaces the first rather
// than accumulating rows.
func TestSaveProjectReplaces(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.godarch.db")
	s, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	if err := s.SaveProject(sampleProject()); err != nil {
		t.Fatalf("first save: %v", err)
	}

	small := model.NewProject("res://small")
	small.Nodes["res://only.gd"] = &model.Node{ID: "res://only.gd", Kind: model.KindScript}
	if err := s.SaveProject(small); err != nil {
		t.Fatalf("second save: %v", err)
	}

	got, err := s.LoadProject()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(got.Nodes) != 1 {
		t.Errorf("expected 1 node after replace, got %d", len(got.Nodes))
	}
	if len(got.Edges) != 0 || len(got.Boundaries) != 0 {
		t.Errorf("expected edges/boundaries cleared, got %d edges, %d boundaries",
			len(got.Edges), len(got.Boundaries))
	}
	if got.Root != "res://small" {
		t.Errorf("root = %q, want res://small", got.Root)
	}
}

// TestReopenPersists confirms the data survives closing and reopening the file,
// and that migrations are idempotent on an already-migrated database.
func TestReopenPersists(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.godarch.db")

	s1, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open 1: %v", err)
	}
	if err := s1.SaveProject(sampleProject()); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := s1.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	s2, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	t.Cleanup(func() { _ = s2.Close() })

	got, err := s2.LoadProject()
	if err != nil {
		t.Fatalf("load after reopen: %v", err)
	}
	if !reflect.DeepEqual(sampleProject(), got) {
		t.Errorf("round trip mismatch after reopen")
	}
}
