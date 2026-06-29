package discovery_test

import (
	"sort"
	"testing"

	"github.com/dylanblakemore/godarch/internal/discovery"
	"github.com/dylanblakemore/godarch/internal/golden"
	"github.com/dylanblakemore/godarch/internal/model"
)

// TestDiscoverMinimalFixture pins discovery's node output for the minimal
// fixture against testdata/fixtures/minimal/golden/nodes.json. Run with
// UPDATE_GOLDEN=1 to regenerate after an intentional change, then review the
// diff.
func TestDiscoverMinimalFixture(t *testing.T) {
	const fixture = "../../testdata/fixtures/minimal"

	p, err := discovery.Discover(fixture)
	if err != nil {
		t.Fatalf("Discover(%s): %v", fixture, err)
	}

	nodes := make([]*model.Node, 0, len(p.Nodes))
	for _, n := range p.Nodes {
		nodes = append(nodes, n)
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })

	golden.AssertJSON(t, fixture+"/golden/nodes.json", nodes)
}
