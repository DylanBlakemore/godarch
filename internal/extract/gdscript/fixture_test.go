package gdscript_test

import (
	"sort"
	"testing"

	"github.com/dylanblakemore/godarch/internal/discovery"
	"github.com/dylanblakemore/godarch/internal/extract/gdscript"
	"github.com/dylanblakemore/godarch/internal/golden"
	"github.com/dylanblakemore/godarch/internal/model"
)

// codeOriginKinds are the node kinds the GDScript extractor creates (as opposed
// to the file nodes discovery produces); the node golden pins only these.
var codeOriginKinds = map[model.Kind]bool{
	model.KindMethod: true,
	model.KindSignal: true,
	model.KindClass:  true,
}

func assertFixtureGolden(t *testing.T, fixture string) {
	t.Helper()

	p, err := discovery.Discover(fixture)
	if err != nil {
		t.Fatalf("Discover(%s): %v", fixture, err)
	}
	diags, err := gdscript.Extract(fixture, p)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	for _, d := range diags {
		t.Logf("diag: %s:%d %s", d.File, d.Line, d.Msg)
	}

	nodes := make([]*model.Node, 0)
	for _, n := range p.Nodes {
		if codeOriginKinds[n.Kind] {
			nodes = append(nodes, n)
		}
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	golden.AssertJSON(t, fixture+"/golden/script_nodes.json", nodes)

	edges := append([]*model.Edge(nil), p.Edges...)
	sort.Slice(edges, func(i, j int) bool {
		a, b := edges[i], edges[j]
		if a.SourceID != b.SourceID {
			return a.SourceID < b.SourceID
		}
		if a.Type != b.Type {
			return a.Type < b.Type
		}
		return a.TargetID < b.TargetID
	})
	golden.AssertJSON(t, fixture+"/golden/script_edges.json", edges)

	bounds := append([]*model.BoundaryPoint(nil), p.Boundaries...)
	sort.Slice(bounds, func(i, j int) bool {
		a, b := bounds[i], bounds[j]
		if a.NodeID != b.NodeID {
			return a.NodeID < b.NodeID
		}
		if a.Type != b.Type {
			return a.Type < b.Type
		}
		return a.MatchKey < b.MatchKey
	})
	golden.AssertJSON(t, fixture+"/golden/script_boundaries.json", bounds)
}

func TestMinimalFixtureGolden(t *testing.T) {
	assertFixtureGolden(t, "../../../testdata/fixtures/minimal")
}

func TestCoupledFixtureGolden(t *testing.T) {
	assertFixtureGolden(t, "../../../testdata/fixtures/coupled")
}
