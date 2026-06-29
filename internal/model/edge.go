package model

// EdgeType enumerates the typed relationships between nodes (DESIGN §3.2). Many
// are editor- or config-declared and so invisible to code-only tools — that
// distinction is what godarch exists to capture (see Origin).
type EdgeType string

const (
	EdgeInstances        EdgeType = "instances"         // scene → scene (composition)
	EdgeAttachesScript   EdgeType = "attaches_script"   // node/scene → script
	EdgeExtends          EdgeType = "extends"           // script → script/class
	EdgeConnectsSignal   EdgeType = "connects_signal"   // emitter → handler method
	EdgeEmitsSignal      EdgeType = "emits_signal"      // method → signal
	EdgeDeclaresSignal   EdgeType = "declares_signal"   // script → signal name
	EdgeCalls            EdgeType = "calls"             // method → method
	EdgeReferencesNode   EdgeType = "references_node"   // script → node path
	EdgeExportsVar       EdgeType = "exports_var"       // script var ↔ inspector value
	EdgeBindsExport      EdgeType = "binds_export"      // editor-assigned export value
	EdgeAccessesAutoload EdgeType = "accesses_autoload" // script → autoload
	EdgeLoadsResource    EdgeType = "loads_resource"    // script/scene → resource/scene/asset
	EdgeChangesSceneTo   EdgeType = "changes_scene_to"  // script → scene
	EdgeInGroup          EdgeType = "in_group"          // node → group
	EdgeCallsGroup       EdgeType = "calls_group"       // script → group
	EdgeUsesAction       EdgeType = "uses_action"       // script → input action
	EdgeUsesLayer        EdgeType = "uses_layer"        // node → collision layer
	EdgeRPCCall          EdgeType = "rpc_call"          // method → @rpc method (call site)
	EdgeRPCEndpoint      EdgeType = "rpc_endpoint"      // the @rpc-annotated method
	EdgeAnimates         EdgeType = "animates"          // AnimationPlayer track → node prop/method
	EdgeUsesAsset        EdgeType = "uses_asset"        // resource/scene → asset file
	EdgeImports          EdgeType = "imports"           // asset ↔ .import config
	EdgeUsesShader       EdgeType = "uses_shader"       // resource → shader asset
	EdgeUsesMaterial     EdgeType = "uses_material"     // resource → material asset
	EdgeUsesTheme        EdgeType = "uses_theme"        // resource → theme asset
)

// AllEdgeTypes lists every EdgeType. Used by tests to guarantee DESIGN §3.2
// coverage and available to callers that need to enumerate edge types.
var AllEdgeTypes = []EdgeType{
	EdgeInstances, EdgeAttachesScript, EdgeExtends, EdgeConnectsSignal,
	EdgeEmitsSignal, EdgeDeclaresSignal, EdgeCalls, EdgeReferencesNode,
	EdgeExportsVar, EdgeBindsExport, EdgeAccessesAutoload, EdgeLoadsResource,
	EdgeChangesSceneTo, EdgeInGroup, EdgeCallsGroup, EdgeUsesAction,
	EdgeUsesLayer, EdgeRPCCall, EdgeRPCEndpoint, EdgeAnimates,
	EdgeUsesAsset, EdgeImports, EdgeUsesShader, EdgeUsesMaterial, EdgeUsesTheme,
}

// Origin records where an edge was declared — the C/E column from DESIGN §3.2.
// It is what lets analysis flag editor↔code seams that code-only tools miss.
type Origin string

const (
	OriginCode   Origin = "code"   // declared in GDScript/C#
	OriginEditor Origin = "editor" // declared in a .tscn/.tres (inspector wiring)
	OriginConfig Origin = "config" // declared in project.godot / .import
	OriginDocs   Origin = "docs"   // declared in developer documentation
)

// AllOrigins lists every Origin.
var AllOrigins = []Origin{OriginCode, OriginEditor, OriginConfig, OriginDocs}

// Evidence pins an observation to a source location for explainability and
// for the "go to where this was declared" affordance in the UI.
type Evidence struct {
	File    string `json:"file"`
	Line    int    `json:"line,omitempty"`
	Snippet string `json:"snippet,omitempty"`
}

// Edge is a typed, directed relationship. Until resolve runs, TargetID may be a
// MatchKey rather than a real Node ID; Resolved records whether it was bound.
type Edge struct {
	Type       EdgeType       `json:"type"`
	SourceID   string         `json:"source_id"`
	TargetID   string         `json:"target_id"` // a Node ID, or a MatchKey while unresolved
	Origin     Origin         `json:"origin"`
	Resolved   bool           `json:"resolved"`
	Confidence float64        `json:"confidence,omitempty"` // 1.0 exact; <1 variant/fuzzy/dynamic
	Evidence   Evidence       `json:"evidence"`
	Properties map[string]any `json:"properties,omitempty"` // {signal_name, method, flags, binds, path_expr, ...}
}
