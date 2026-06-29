package graph_test

import (
	"reflect"
	"testing"

	"github.com/dylanblakemore/godarch/internal/graph"
	"github.com/dylanblakemore/godarch/internal/model"
)

// toyProject builds a small graph with a cycle and a tail:
//
//	A → B → C → A   (a 3-node strongly-connected cycle)
//	        C → D → E   (a tail hanging off the cycle)
//
// plus one unresolved edge (target is a match key, not a node) that must be
// ignored by the graph builder.
func toyProject() *model.Project {
	p := model.NewProject("res://")
	for _, id := range []string{"A", "B", "C", "D", "E"} {
		p.Nodes[id] = &model.Node{ID: id, Kind: model.KindScript}
	}
	add := func(s, t string) {
		p.Edges = append(p.Edges, &model.Edge{
			Type: model.EdgeCalls, SourceID: s, TargetID: t, Resolved: true,
		})
	}
	add("A", "B")
	add("B", "C")
	add("C", "A")
	add("C", "D")
	add("D", "E")

	// An unresolved edge whose target is a match key, not a node: must be skipped.
	p.Edges = append(p.Edges, &model.Edge{
		Type: model.EdgeLoadsResource, SourceID: "A",
		TargetID: string(model.ResourceKey("missing.tres")), Resolved: false,
	})
	return p
}

func TestForwardReach(t *testing.T) {
	g := graph.Build(toyProject())

	got := g.ForwardReach("A", 0) // unbounded
	if !reflect.DeepEqual(got.IDs, []string{"B", "C", "D", "E"}) {
		t.Errorf("ForwardReach(A) IDs = %v, want [B C D E]", got.IDs)
	}
	want := map[string]int{"B": 1, "C": 2, "D": 3, "E": 4}
	if !reflect.DeepEqual(got.Depths, want) {
		t.Errorf("ForwardReach(A) depths = %v, want %v", got.Depths, want)
	}
}

func TestForwardReachMaxDepth(t *testing.T) {
	g := graph.Build(toyProject())

	got := g.ForwardReach("A", 2)
	if !reflect.DeepEqual(got.IDs, []string{"B", "C"}) {
		t.Errorf("ForwardReach(A, 2) IDs = %v, want [B C]", got.IDs)
	}
}

func TestReverseReach(t *testing.T) {
	g := graph.Build(toyProject())

	got := g.ReverseReach("E", 0)
	if !reflect.DeepEqual(got.IDs, []string{"A", "B", "C", "D"}) {
		t.Errorf("ReverseReach(E) IDs = %v, want [A B C D]", got.IDs)
	}
	if got.Depths["D"] != 1 || got.Depths["A"] != 4 {
		t.Errorf("ReverseReach(E) depths = %v, want D=1, A=4", got.Depths)
	}
}

func TestReachUnknownNode(t *testing.T) {
	g := graph.Build(toyProject())
	got := g.ForwardReach("nope", 0)
	if len(got.IDs) != 0 {
		t.Errorf("ForwardReach(unknown) = %v, want empty", got.IDs)
	}
}

func TestSCC(t *testing.T) {
	g := graph.Build(toyProject())
	got := g.SCC()
	want := [][]string{{"A", "B", "C"}, {"D"}, {"E"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("SCC() = %v, want %v", got, want)
	}
}

func TestFanInOut(t *testing.T) {
	g := graph.Build(toyProject())

	cases := []struct {
		id            string
		fanIn, fanOut int
	}{
		{"A", 1, 1}, // in: C→A; out: A→B
		{"C", 1, 2}, // in: B→C; out: C→A, C→D
		{"E", 1, 0}, // in: D→E; out: none
	}
	for _, c := range cases {
		if in := g.FanIn(c.id); in != c.fanIn {
			t.Errorf("FanIn(%s) = %d, want %d", c.id, in, c.fanIn)
		}
		if out := g.FanOut(c.id); out != c.fanOut {
			t.Errorf("FanOut(%s) = %d, want %d", c.id, out, c.fanOut)
		}
	}
}

func TestUnresolvedEdgeIgnored(t *testing.T) {
	g := graph.Build(toyProject())
	if g.Order() != 5 {
		t.Errorf("Order() = %d, want 5 (match-key target must not add a node)", g.Order())
	}
}

func TestProject(t *testing.T) {
	g := graph.Build(toyProject())

	// Group the cycle into "core" and the tail into "leaf".
	core := map[string]bool{"A": true, "B": true, "C": true}
	pg := g.Project(func(n *model.Node) (string, bool) {
		if core[n.ID] {
			return "core", true
		}
		return "leaf", true
	})

	if pg.Order() != 2 {
		t.Fatalf("projected Order() = %d, want 2", pg.Order())
	}
	// Only C→D crosses core→leaf, so the projection has a single core→leaf edge.
	if out := pg.FanOut("core"); out != 1 {
		t.Errorf("projected FanOut(core) = %d, want 1", out)
	}
	if in := pg.FanIn("leaf"); in != 1 {
		t.Errorf("projected FanIn(leaf) = %d, want 1", in)
	}
}
