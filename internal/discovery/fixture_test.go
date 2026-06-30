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

	// Beyond the node set, the fixture pins the two global facts the discovery
	// Definition of Done calls out: the identified main scene and the uid map
	// (from the .tscn header and the .import sidecar).
	if p.MainScene != "res://main.tscn" {
		t.Errorf("MainScene = %q, want res://main.tscn", p.MainScene)
	}
	wantUID := map[string]string{
		"uid://b0minimal0scn0":  "res://main.tscn",
		"uid://b0minimal0icon0": "res://art/icon.svg",
	}
	for uid, want := range wantUID {
		if got := p.UIDMap[uid]; got != want {
			t.Errorf("UIDMap[%s] = %q, want %q", uid, got, want)
		}
	}
	if len(p.UIDMap) != len(wantUID) {
		t.Errorf("UIDMap has %d entries, want %d: %v", len(p.UIDMap), len(wantUID), p.UIDMap)
	}
}
