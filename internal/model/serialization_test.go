package model_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/dylanblakemore/godarch/internal/model"
)

// TestProjectJSONRoundTrip builds a Project that exercises every Kind, every
// EdgeType, and every BoundaryType, marshals it to JSON, unmarshals it back,
// and asserts the result is identical. This is the DESIGN §3 coverage guarantee
// from the milestone exit criteria: the model can represent every node/edge/
// boundary kind, and they all survive serialization.
func TestProjectJSONRoundTrip(t *testing.T) {
	p := &model.Project{
		Root:         "res://",
		Nodes:        map[string]*model.Node{},
		UIDMap:       map[string]string{"uid://abc": "res://player/player.gd"},
		GodotVersion: "4.3",
	}

	for i, k := range model.AllKinds {
		id := model.AutoloadID(string(k))
		p.Nodes[id] = &model.Node{
			ID:         id,
			Kind:       k,
			Path:       "res://x.gd",
			Line:       i + 1,
			Identity:   map[string]any{"name": string(k)},
			Properties: map[string]any{"fan_in": float64(i)},
		}
	}

	for i, et := range model.AllEdgeTypes {
		p.Edges = append(p.Edges, &model.Edge{
			Type:       et,
			SourceID:   "res://a.gd",
			TargetID:   "res://b.gd",
			Origin:     model.AllOrigins[i%len(model.AllOrigins)],
			Resolved:   i%2 == 0,
			Confidence: 1.0,
			Evidence:   model.Evidence{File: "res://a.gd", Line: i + 1, Snippet: "x"},
			Properties: map[string]any{"method": "foo"},
		})
	}

	for i, bt := range model.AllBoundaryTypes {
		p.Boundaries = append(p.Boundaries, &model.BoundaryPoint{
			Direction: model.AllDirections[i%len(model.AllDirections)],
			Type:      bt,
			NodeID:    "res://a.gd::foo",
			MatchKey:  model.SignalKey("Player", "died"),
			Evidence:  model.Evidence{File: "res://a.gd", Line: i + 1},
			Meta:      map[string]any{"k": "v"},
		})
	}

	// One unresolved edge to populate the diagnostic list.
	p.Unresolved = append(p.Unresolved, &model.Edge{
		Type:     model.EdgeLoadsResource,
		SourceID: "res://a.gd",
		TargetID: string(model.ResourceKey("missing.tres")),
		Origin:   model.OriginCode,
	})

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got model.Project
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !reflect.DeepEqual(p, &got) {
		t.Errorf("round trip mismatch:\n got: %+v\nwant: %+v", &got, p)
	}
}

func TestEnumerationsAreComplete(t *testing.T) {
	if len(model.AllKinds) != 13 {
		t.Errorf("AllKinds has %d entries, want 13", len(model.AllKinds))
	}
	if len(model.AllEdgeTypes) != 25 {
		t.Errorf("AllEdgeTypes has %d entries, want 25", len(model.AllEdgeTypes))
	}
	if len(model.AllBoundaryTypes) != 16 {
		t.Errorf("AllBoundaryTypes has %d entries, want 16", len(model.AllBoundaryTypes))
	}
	assertNoDuplicates(t, "AllKinds", stringify(model.AllKinds))
	assertNoDuplicates(t, "AllEdgeTypes", stringify(model.AllEdgeTypes))
	assertNoDuplicates(t, "AllBoundaryTypes", stringify(model.AllBoundaryTypes))
}

func stringify[T ~string](xs []T) []string {
	out := make([]string, len(xs))
	for i, x := range xs {
		out[i] = string(x)
	}
	return out
}

func assertNoDuplicates(t *testing.T, name string, xs []string) {
	t.Helper()
	seen := map[string]bool{}
	for _, x := range xs {
		if seen[x] {
			t.Errorf("%s has duplicate entry %q", name, x)
		}
		seen[x] = true
	}
}
