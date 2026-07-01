package main

import (
	"fmt"
	"io"
	"sort"

	"github.com/dylanblakemore/godarch/internal/analyze"
	"github.com/dylanblakemore/godarch/internal/graph"
	"github.com/dylanblakemore/godarch/internal/model"
)

// printSummary writes the milestone-1 analyze summary: headline totals, the
// per-kind node breakdown, the per-type edge breakdown, boundary counts split by
// direction, and the unresolved/diagnostic tallies. Everything is sorted so the
// output is stable across runs.
func printSummary(w io.Writer, projectDir, dest string, p *model.Project, diags []analyze.Diagnostic) {
	version := p.GodotVersion
	if version == "" {
		version = "unknown"
	}
	fmt.Fprintf(w, "godarch: analyzed %s (Godot %s)\n", projectDir, version)

	fmt.Fprintf(w, "  %d nodes\n", len(p.Nodes))
	for _, kc := range sortedCounts(nodeCounts(p)) {
		fmt.Fprintf(w, "    %-12s %d\n", kc.key, kc.count)
	}

	fmt.Fprintf(w, "  %d edges\n", len(p.Edges))
	for _, kc := range sortedCounts(edgeCounts(p)) {
		fmt.Fprintf(w, "    %-16s %d\n", kc.key, kc.count)
	}

	ingress, egress := boundaryCounts(p)
	fmt.Fprintf(w, "  %d boundaries (ingress %d, egress %d)\n", len(p.Boundaries), ingress, egress)
	fmt.Fprintf(w, "  %d unresolved edges\n", len(p.Unresolved))
	fmt.Fprintf(w, "  %d diagnostics\n", len(diags))
	fmt.Fprintf(w, "  database: %s\n", dest)
}

// keyCount is one row of a sorted breakdown.
type keyCount struct {
	key   string
	count int
}

// sortedCounts flattens a name→count map into rows sorted by name, dropping
// zero-count entries so the breakdown lists only what is present.
func sortedCounts(m map[string]int) []keyCount {
	out := make([]keyCount, 0, len(m))
	for k, c := range m {
		if c > 0 {
			out = append(out, keyCount{k, c})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].key < out[j].key })
	return out
}

func nodeCounts(p *model.Project) map[string]int {
	m := map[string]int{}
	for _, n := range p.Nodes {
		m[string(n.Kind)]++
	}
	return m
}

func edgeCounts(p *model.Project) map[string]int {
	m := map[string]int{}
	for _, e := range p.Edges {
		m[string(e.Type)]++
	}
	return m
}

func boundaryCounts(p *model.Project) (ingress, egress int) {
	for _, b := range p.Boundaries {
		if b.Direction == model.DirectionIngress {
			ingress++
		} else {
			egress++
		}
	}
	return ingress, egress
}

// edgeView is one edge as shown in a node inspection: the far endpoint plus the
// edge's type, origin, and resolution state.
type edgeView struct {
	Type     model.EdgeType `json:"type"`
	Peer     string         `json:"peer"` // target for outbound, source for inbound
	Origin   model.Origin   `json:"origin"`
	Resolved bool           `json:"resolved"`
}

// nodeView is the graph subcommand's payload: a node with its inbound and
// outbound edges, sorted for deterministic output.
type nodeView struct {
	ID       string         `json:"id"`
	Kind     model.Kind     `json:"kind"`
	Path     string         `json:"path,omitempty"`
	Identity map[string]any `json:"identity,omitempty"`
	Outbound []edgeView     `json:"outbound"`
	Inbound  []edgeView     `json:"inbound"`
}

// buildNodeView collects the edges incident to node from the project. Outbound
// edges are those the node sources; inbound are those that target it (including
// unresolved edges whose match key happens to equal the node ID).
func buildNodeView(p *model.Project, node *model.Node) nodeView {
	v := nodeView{ID: node.ID, Kind: node.Kind, Path: node.Path, Identity: node.Identity}
	for _, e := range p.Edges {
		if e.SourceID == node.ID {
			v.Outbound = append(v.Outbound, edgeView{e.Type, e.TargetID, e.Origin, e.Resolved})
		}
		if e.TargetID == node.ID {
			v.Inbound = append(v.Inbound, edgeView{e.Type, e.SourceID, e.Origin, e.Resolved})
		}
	}
	sortEdgeViews(v.Outbound)
	sortEdgeViews(v.Inbound)
	return v
}

func sortEdgeViews(es []edgeView) {
	sort.Slice(es, func(i, j int) bool {
		if es[i].Type != es[j].Type {
			return es[i].Type < es[j].Type
		}
		return es[i].Peer < es[j].Peer
	})
}

func printNodeView(w io.Writer, v nodeView) {
	fmt.Fprintf(w, "%s  [%s]\n", v.ID, v.Kind)
	if v.Path != "" && v.Path != v.ID {
		fmt.Fprintf(w, "  path: %s\n", v.Path)
	}
	fmt.Fprintf(w, "  outbound (%d):\n", len(v.Outbound))
	for _, e := range v.Outbound {
		fmt.Fprintf(w, "    -%s-> %s  (%s%s)\n", e.Type, e.Peer, e.Origin, resolvedTag(e.Resolved))
	}
	fmt.Fprintf(w, "  inbound (%d):\n", len(v.Inbound))
	for _, e := range v.Inbound {
		fmt.Fprintf(w, "    <-%s- %s  (%s%s)\n", e.Type, e.Peer, e.Origin, resolvedTag(e.Resolved))
	}
}

func resolvedTag(resolved bool) string {
	if resolved {
		return ""
	}
	return ", unresolved"
}

// fanInNode is one entry in the stats fan-in ranking.
type fanInNode struct {
	ID    string     `json:"id"`
	Kind  model.Kind `json:"kind"`
	FanIn int        `json:"fan_in"`
}

// statsView is the stats subcommand's payload: graph totals plus the top fan-in
// nodes by in-degree.
type statsView struct {
	Nodes      int         `json:"nodes"`
	Edges      int         `json:"edges"`
	Boundaries int         `json:"boundaries"`
	Unresolved int         `json:"unresolved"`
	TopFanIn   []fanInNode `json:"top_fan_in"`
}

// buildStats computes the totals and the top-N fan-in nodes. Fan-in is the
// in-degree over resolved edges (the graph package skips unresolved endpoints).
// Ties break by node ID so the ranking is deterministic.
func buildStats(p *model.Project, top int) statsView {
	s := statsView{
		Nodes:      len(p.Nodes),
		Edges:      len(p.Edges),
		Boundaries: len(p.Boundaries),
		Unresolved: len(p.Unresolved),
	}

	gr := graph.Build(p)
	ranked := make([]fanInNode, 0, len(p.Nodes))
	for id, n := range p.Nodes {
		if fi := gr.FanIn(id); fi > 0 {
			ranked = append(ranked, fanInNode{ID: id, Kind: n.Kind, FanIn: fi})
		}
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].FanIn != ranked[j].FanIn {
			return ranked[i].FanIn > ranked[j].FanIn
		}
		return ranked[i].ID < ranked[j].ID
	})
	if top > 0 && len(ranked) > top {
		ranked = ranked[:top]
	}
	s.TopFanIn = ranked
	return s
}

func printStats(w io.Writer, s statsView) {
	fmt.Fprintf(w, "nodes:       %d\n", s.Nodes)
	fmt.Fprintf(w, "edges:       %d\n", s.Edges)
	fmt.Fprintf(w, "boundaries:  %d\n", s.Boundaries)
	fmt.Fprintf(w, "unresolved:  %d\n", s.Unresolved)
	fmt.Fprintf(w, "top fan-in:\n")
	for _, f := range s.TopFanIn {
		fmt.Fprintf(w, "  %4d  %s  [%s]\n", f.FanIn, f.ID, f.Kind)
	}
}
