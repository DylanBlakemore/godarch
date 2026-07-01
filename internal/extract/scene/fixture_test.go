package scene_test

import (
	"sort"
	"testing"

	"github.com/dylanblakemore/godarch/internal/discovery"
	"github.com/dylanblakemore/godarch/internal/extract/scene"
	"github.com/dylanblakemore/godarch/internal/golden"
	"github.com/dylanblakemore/godarch/internal/model"
)

func discoverOrFail(t *testing.T, dir string) *model.Project {
	t.Helper()
	p, err := discovery.Discover(dir)
	if err != nil {
		t.Fatalf("Discover(%s): %v", dir, err)
	}
	return p
}

// TestExtractMinimalFixtureGolden pins the scene extractor's edge and boundary
// output for the minimal fixture. Run with UPDATE_GOLDEN=1 to regenerate.
func TestExtractMinimalFixtureGolden(t *testing.T) {
	const fixture = "../../../testdata/fixtures/minimal"

	p := discoverOrFail(t, fixture)
	if _, err := scene.Extract(fixture, p); err != nil {
		t.Fatalf("Extract: %v", err)
	}

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
	golden.AssertJSON(t, fixture+"/golden/edges.json", edges)

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
	golden.AssertJSON(t, fixture+"/golden/boundaries.json", bounds)
}
