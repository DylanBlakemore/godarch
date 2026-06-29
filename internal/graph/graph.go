package graph

import (
	"sort"

	gonum "gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"

	"github.com/dylanblakemore/godarch/internal/model"
)

// Graph is the in-memory analysis surface: a gonum directed graph built from a
// model.Project plus the bookkeeping needed to translate between model node IDs
// (strings) and gonum's int64 node IDs, and to recover edge metadata.
//
// Only edges whose source and target are both real nodes are added to the gonum
// graph; unresolved edges (whose target is still a match key) carry no traversable
// structure and are skipped.
type Graph struct {
	g       *simple.DirectedGraph
	nodes   map[string]*model.Node // model nodes by ID
	idToGID map[string]int64       // model ID → gonum node ID
	gidToID map[int64]string       // gonum node ID → model ID
	edges   map[[2]string][]*model.Edge
}

// ReachResult is the outcome of a reachability query: the set of reachable node
// IDs (excluding the root, sorted for determinism) and each one's minimum hop
// distance from the root.
type ReachResult struct {
	Root   string
	IDs    []string
	Depths map[string]int
}

// ProjectionFunc maps a node to a group ID for unit projections (M4). Returning
// keep=false drops the node (and its incident edges) from the projection.
type ProjectionFunc func(*model.Node) (group string, keep bool)

// Build constructs the gonum graph from a resolved Project.
func Build(p *model.Project) *Graph {
	gr := &Graph{
		g:       simple.NewDirectedGraph(),
		nodes:   make(map[string]*model.Node, len(p.Nodes)),
		idToGID: make(map[string]int64, len(p.Nodes)),
		gidToID: make(map[int64]string, len(p.Nodes)),
		edges:   make(map[[2]string][]*model.Edge),
	}

	// Add nodes in sorted ID order so gonum IDs are assigned deterministically.
	ids := make([]string, 0, len(p.Nodes))
	for id := range p.Nodes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		gr.addNode(p.Nodes[id])
	}

	for _, e := range p.Edges {
		gr.addEdge(e)
	}
	return gr
}

func (gr *Graph) addNode(n *model.Node) {
	if _, ok := gr.idToGID[n.ID]; ok {
		return
	}
	gid := int64(len(gr.idToGID))
	gr.idToGID[n.ID] = gid
	gr.gidToID[gid] = n.ID
	gr.nodes[n.ID] = n
	gr.g.AddNode(simple.Node(gid))
}

func (gr *Graph) addEdge(e *model.Edge) {
	from, okF := gr.idToGID[e.SourceID]
	to, okT := gr.idToGID[e.TargetID]
	if !okF || !okT || from == to {
		// Skip edges that don't connect two distinct known nodes; the simple
		// graph has no place for dangling endpoints or self-loops, but the
		// metadata is still recorded so callers can inspect parallel edges.
		if okF && okT {
			key := [2]string{e.SourceID, e.TargetID}
			gr.edges[key] = append(gr.edges[key], e)
		}
		return
	}
	key := [2]string{e.SourceID, e.TargetID}
	if _, seen := gr.edges[key]; !seen {
		gr.g.SetEdge(gr.g.NewEdge(simple.Node(from), simple.Node(to)))
	}
	gr.edges[key] = append(gr.edges[key], e)
}

// Order returns the number of nodes in the graph.
func (gr *Graph) Order() int { return len(gr.idToGID) }

// ForwardReach returns the nodes reachable by following edges out of id ("blast
// radius"). maxDepth limits the number of hops; a value <= 0 means unbounded.
func (gr *Graph) ForwardReach(id string, maxDepth int) ReachResult {
	return gr.reach(id, maxDepth, gr.g.From)
}

// ReverseReach returns the nodes that can reach id by following edges backwards
// ("who depends on me"). maxDepth limits hops; <= 0 means unbounded.
func (gr *Graph) ReverseReach(id string, maxDepth int) ReachResult {
	return gr.reach(id, maxDepth, gr.g.To)
}

// reach runs a breadth-first traversal from id using next to enumerate the
// neighbours to follow (From for forward, To for reverse).
func (gr *Graph) reach(id string, maxDepth int, next func(int64) gonum.Nodes) ReachResult {
	res := ReachResult{Root: id, Depths: map[string]int{}}
	start, ok := gr.idToGID[id]
	if !ok {
		return res
	}

	type item struct {
		node  int64
		depth int
	}
	visited := map[int64]bool{start: true}
	queue := []item{{start, 0}}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if maxDepth > 0 && cur.depth >= maxDepth {
			continue
		}
		it := next(cur.node)
		for it.Next() {
			nb := it.Node().ID()
			if visited[nb] {
				continue
			}
			visited[nb] = true
			depth := cur.depth + 1
			res.Depths[gr.gidToID[nb]] = depth
			res.IDs = append(res.IDs, gr.gidToID[nb])
			queue = append(queue, item{nb, depth})
		}
	}
	sort.Strings(res.IDs)
	return res
}

// SCC returns the strongly-connected components (gonum's Tarjan), each as a
// sorted slice of node IDs; the components themselves are sorted by their first
// member for deterministic output.
func (gr *Graph) SCC() [][]string {
	raw := topo.TarjanSCC(gr.g)
	out := make([][]string, 0, len(raw))
	for _, comp := range raw {
		ids := make([]string, 0, len(comp))
		for _, n := range comp {
			ids = append(ids, gr.gidToID[n.ID()])
		}
		sort.Strings(ids)
		out = append(out, ids)
	}
	sort.Slice(out, func(i, j int) bool { return out[i][0] < out[j][0] })
	return out
}

// FanIn is the in-degree of id: how many distinct nodes point at it.
func (gr *Graph) FanIn(id string) int {
	gid, ok := gr.idToGID[id]
	if !ok {
		return 0
	}
	return countNodes(gr.g.To(gid))
}

// FanOut is the out-degree of id: how many distinct nodes it points at.
func (gr *Graph) FanOut(id string) int {
	gid, ok := gr.idToGID[id]
	if !ok {
		return 0
	}
	return countNodes(gr.g.From(gid))
}

// Project collapses nodes into groups via by and returns a new Graph over those
// groups, with an edge between two groups when any original edge crosses between
// them (intra-group edges and dropped nodes are ignored).
func (gr *Graph) Project(by ProjectionFunc) *Graph {
	p := &model.Project{Nodes: map[string]*model.Node{}}
	group := make(map[string]string, len(gr.nodes))

	for id, n := range gr.nodes {
		g, keep := by(n)
		if !keep {
			continue
		}
		group[id] = g
		if _, ok := p.Nodes[g]; !ok {
			p.Nodes[g] = &model.Node{ID: g, Kind: "projection"}
		}
	}

	seen := map[[2]string]bool{}
	for key := range gr.edges {
		sg, okS := group[key[0]]
		tg, okT := group[key[1]]
		if !okS || !okT || sg == tg {
			continue
		}
		pair := [2]string{sg, tg}
		if seen[pair] {
			continue
		}
		seen[pair] = true
		p.Edges = append(p.Edges, &model.Edge{
			Type: model.EdgeType("projection"), SourceID: sg, TargetID: tg, Resolved: true,
		})
	}
	return Build(p)
}

func countNodes(it gonum.Nodes) int {
	n := 0
	for it.Next() {
		n++
	}
	return n
}
