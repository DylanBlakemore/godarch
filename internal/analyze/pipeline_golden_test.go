package analyze_test

import (
	"sort"
	"testing"

	"github.com/dylanblakemore/godarch/internal/analyze"
	"github.com/dylanblakemore/godarch/internal/golden"
	"github.com/dylanblakemore/godarch/internal/model"
)

// assertPipelineGolden pins the fully assembled graph (discovery + both
// extractors, merged) for a fixture: every node, every edge, and every boundary.
// These are the end-to-end M1 goldens — distinct from the per-extractor goldens
// each package pins in isolation. Run with UPDATE_GOLDEN=1 to regenerate.
func assertPipelineGolden(t *testing.T, fixture string) {
	t.Helper()

	p, diags, err := analyze.Run(fixture)
	if err != nil {
		t.Fatalf("Run(%s): %v", fixture, err)
	}
	for _, d := range diags {
		t.Logf("diag: [%s] %s:%d %s", d.Source, d.File, d.Line, d.Msg)
	}

	nodes := make([]*model.Node, 0, len(p.Nodes))
	for _, n := range p.Nodes {
		nodes = append(nodes, n)
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	golden.AssertJSON(t, fixture+"/golden/pipeline_nodes.json", nodes)

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
	golden.AssertJSON(t, fixture+"/golden/pipeline_edges.json", edges)

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
	golden.AssertJSON(t, fixture+"/golden/pipeline_boundaries.json", bounds)
}

func TestPipelineMinimalGolden(t *testing.T) {
	assertPipelineGolden(t, "../../testdata/fixtures/minimal")
}

func TestPipelineCoupledGolden(t *testing.T) {
	assertPipelineGolden(t, "../../testdata/fixtures/coupled")
}
