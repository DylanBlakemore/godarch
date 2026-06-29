package model

// Project is the in-memory container for a fully extracted (and, after resolve,
// resolved) Godot project. It is what flows through the pipeline and what
// internal/store persists and reloads.
type Project struct {
	Root         string            `json:"root"`                 // project root (res:// or filesystem)
	Nodes        map[string]*Node  `json:"nodes,omitempty"`      // keyed by Node.ID
	Edges        []*Edge           `json:"edges,omitempty"`      //
	Boundaries   []*BoundaryPoint  `json:"boundaries,omitempty"` //
	Unresolved   []*Edge           `json:"unresolved,omitempty"` // edges whose TargetID never resolved
	UIDMap       map[string]string `json:"uid_map,omitempty"`    // uid:// → res://
	GodotVersion string            `json:"godot_version,omitempty"`
}

// NewProject returns a Project with its maps initialized, ready for extractors
// to populate.
func NewProject(root string) *Project {
	return &Project{
		Root:   root,
		Nodes:  map[string]*Node{},
		UIDMap: map[string]string{},
	}
}
