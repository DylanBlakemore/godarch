package gdscript

import "github.com/dylanblakemore/godarch/internal/model"

// addEdge appends a code-origin edge to the project. All GDScript edges carry
// OriginCode; callers set resolved/confidence/properties per detector.
func (e *extractor) addEdge(typ model.EdgeType, src, tgt string, resolved bool, conf float64, line int, props map[string]any) {
	edge := &model.Edge{
		Type:       typ,
		SourceID:   src,
		TargetID:   tgt,
		Origin:     model.OriginCode,
		Resolved:   resolved,
		Confidence: conf,
		Evidence:   model.Evidence{File: e.file, Line: line},
	}
	if len(props) > 0 {
		edge.Properties = props
	}
	e.p.Edges = append(e.p.Edges, edge)
}

// addBoundary appends a boundary point (ingress or egress) to the project.
func (e *extractor) addBoundary(dir model.Direction, typ model.BoundaryType, nodeID string, key model.MatchKey, line int, meta map[string]any) {
	b := &model.BoundaryPoint{
		Direction: dir,
		Type:      typ,
		NodeID:    nodeID,
		MatchKey:  key,
		Evidence:  model.Evidence{File: e.file, Line: line},
	}
	if len(meta) > 0 {
		b.Meta = meta
	}
	e.p.Boundaries = append(e.p.Boundaries, b)
}

// ensureClassNode creates a class:<Name> node if one does not already exist,
// recording that this script referenced it. Built-in and cross-file classes are
// materialised lazily so the inheritance/class registry is queryable.
func (e *extractor) ensureClassNode(name string) {
	id := model.ClassID(name)
	if _, ok := e.p.Nodes[id]; ok {
		return
	}
	e.p.Nodes[id] = &model.Node{ID: id, Kind: model.KindClass, Identity: map[string]any{"name": name}}
}

// identity returns the script node's identity map, allocating it on first use.
func (e *extractor) identity() map[string]any {
	n := e.p.Nodes[e.file]
	if n.Identity == nil {
		n.Identity = map[string]any{}
	}
	return n.Identity
}

// appendIdentity appends v to the string-slice identity key on the script node.
func (e *extractor) appendIdentity(key, v string) {
	id := e.identity()
	existing, _ := id[key].([]string)
	id[key] = append(existing, v)
}
