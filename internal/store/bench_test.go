package store_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/dylanblakemore/godarch/internal/graph"
	"github.com/dylanblakemore/godarch/internal/model"
	"github.com/dylanblakemore/godarch/internal/store"
)

// largeProject synthesises a project of roughly n file-nodes with a few edges
// each, standing in for a ~1k-file Godot project (plan 00.03 perf target: load +
// graph build of a 1k-file project in under a second).
func largeProject(n int) *model.Project {
	p := model.NewProject("res://")
	for i := range n {
		id := fmt.Sprintf("res://f%d.gd", i)
		p.Nodes[id] = &model.Node{
			ID:       id,
			Kind:     model.KindScript,
			Path:     id,
			Identity: map[string]any{"name": fmt.Sprintf("F%d", i)},
		}
	}
	// Each node calls the next three (wrapping), giving ~3n resolved edges.
	for i := range n {
		src := fmt.Sprintf("res://f%d.gd", i)
		for d := 1; d <= 3; d++ {
			dst := fmt.Sprintf("res://f%d.gd", (i+d)%n)
			p.Edges = append(p.Edges, &model.Edge{
				Type: model.EdgeCalls, SourceID: src, TargetID: dst,
				Origin: model.OriginCode, Resolved: true, Confidence: 1.0,
			})
		}
	}
	return p
}

// BenchmarkLoadAndBuild measures the LoadProject + graph.Build cost on a
// 1k-node project — the analysis hot path on app open.
func BenchmarkLoadAndBuild(b *testing.B) {
	dbPath := filepath.Join(b.TempDir(), "bench.godarch.db")
	s, err := store.Open(dbPath)
	if err != nil {
		b.Fatalf("open: %v", err)
	}
	defer func() { _ = s.Close() }()
	if err := s.SaveProject(largeProject(1000)); err != nil {
		b.Fatalf("save: %v", err)
	}

	for b.Loop() {
		p, err := s.LoadProject()
		if err != nil {
			b.Fatalf("load: %v", err)
		}
		g := graph.Build(p)
		if g.Order() != 1000 {
			b.Fatalf("expected 1000 nodes, got %d", g.Order())
		}
	}
}
