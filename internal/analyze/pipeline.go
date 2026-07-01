package analyze

import (
	"fmt"

	"github.com/dylanblakemore/godarch/internal/discovery"
	"github.com/dylanblakemore/godarch/internal/extract/gdscript"
	"github.com/dylanblakemore/godarch/internal/extract/scene"
	"github.com/dylanblakemore/godarch/internal/model"
)

// Diagnostic is a unified extraction diagnostic: something an extractor could not
// parse, surfaced with its source and location rather than dropped (M1 exit
// criterion #3). Source names the extractor that raised it ("scene"/"gdscript").
type Diagnostic struct {
	Source string
	File   string
	Line   int
	Msg    string
}

// Run is the shared pipeline seam both the CLI and (later) the Wails UI call. It
// discovers the Godot project at dir, runs the scene/config and GDScript
// extractors, and returns the assembled Project plus every extraction
// diagnostic.
//
// Assembly order follows plan 01.04: discovery emits the skeleton (file + concept
// nodes, UID map, main scene); each extractor then merges its nodes, edges, and
// boundaries into that shared project. Merging is single-threaded so the node map
// stays race-free, and de-duplication is inherent — nodes are keyed by ID, so a
// stub created by an early reference (e.g. class:Foo from an extends) is enriched
// in place when its declaration is later parsed rather than duplicated.
//
// Edges whose target is still a match key are left unresolved (resolution is M2);
// Run collects them into Project.Unresolved for the summary and downstream passes.
func Run(dir string) (*model.Project, []Diagnostic, error) {
	root, err := discovery.Root(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("locate project root: %w", err)
	}

	p, err := discovery.Discover(root)
	if err != nil {
		return nil, nil, fmt.Errorf("discover: %w", err)
	}

	var diags []Diagnostic

	sceneDiags, err := scene.Extract(root, p)
	if err != nil {
		return nil, nil, fmt.Errorf("scene extract: %w", err)
	}
	for _, d := range sceneDiags {
		diags = append(diags, Diagnostic{Source: "scene", File: d.File, Line: d.Line, Msg: d.Msg})
	}

	gdDiags, err := gdscript.Extract(root, p)
	if err != nil {
		return nil, nil, fmt.Errorf("gdscript extract: %w", err)
	}
	for _, d := range gdDiags {
		diags = append(diags, Diagnostic{Source: "gdscript", File: d.File, Line: d.Line, Msg: d.Msg})
	}

	collectUnresolved(p)
	return p, diags, nil
}

// collectUnresolved (re)builds Project.Unresolved as the subset of edges whose
// target never resolved to a real node ID, preserving Edges' order.
func collectUnresolved(p *model.Project) {
	p.Unresolved = nil
	for _, e := range p.Edges {
		if !e.Resolved {
			p.Unresolved = append(p.Unresolved, e)
		}
	}
}
