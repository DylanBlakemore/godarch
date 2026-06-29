package model

// Direction distinguishes a boundary point's role: ingress is public surface a
// symbol exposes; egress is an outward reach a symbol makes (DESIGN §4).
type Direction string

const (
	DirectionIngress Direction = "ingress"
	DirectionEgress  Direction = "egress"
)

// AllDirections lists every Direction.
var AllDirections = []Direction{DirectionIngress, DirectionEgress}

// BoundaryType is the typed entry/exit semantic attached to a script symbol
// (DESIGN §4). Ingress types describe how a symbol can be reached; egress types
// describe what it reaches out to.
type BoundaryType string

const (
	// Ingress: how a symbol is entered.
	BoundaryLifecycle        BoundaryType = "lifecycle"         // _ready, _process, ...
	BoundarySignalHandler    BoundaryType = "signal_handler"    // a method wired to a signal
	BoundaryEditorConnection BoundaryType = "editor_connection" // connected via the editor
	BoundaryRPCEndpoint      BoundaryType = "rpc_endpoint"      // an @rpc method
	BoundaryGroupTarget      BoundaryType = "group_target"      // invoked via call_group
	BoundaryInputHandler     BoundaryType = "input_handler"     // _input/_unhandled_input
	BoundaryNotification     BoundaryType = "notification"      // _notification

	// Egress: what a symbol reaches out to.
	BoundarySignalEmit     BoundaryType = "signal_emit"     // emit_signal / sig.emit()
	BoundarySignalConnect  BoundaryType = "signal_connect"  // .connect()
	BoundaryResourceLoad   BoundaryType = "resource_load"   // load/preload
	BoundarySceneChange    BoundaryType = "scene_change"    // change_scene_to_*
	BoundaryAutoloadAccess BoundaryType = "autoload_access" // global singleton access
	BoundaryNodeReach      BoundaryType = "node_reach"      // get_node / $ / %
	BoundaryGroupCall      BoundaryType = "group_call"      // call_group / get_nodes_in_group
	BoundaryRPCCall        BoundaryType = "rpc_call"        // rpc()/rpc_id()
	BoundaryFileIO         BoundaryType = "file_io"         // FileAccess and friends
)

// AllBoundaryTypes lists every BoundaryType. Used by tests to guarantee DESIGN
// §4 coverage and available to callers that need to enumerate boundary types.
var AllBoundaryTypes = []BoundaryType{
	BoundaryLifecycle, BoundarySignalHandler, BoundaryEditorConnection,
	BoundaryRPCEndpoint, BoundaryGroupTarget, BoundaryInputHandler,
	BoundaryNotification,
	BoundarySignalEmit, BoundarySignalConnect, BoundaryResourceLoad,
	BoundarySceneChange, BoundaryAutoloadAccess, BoundaryNodeReach,
	BoundaryGroupCall, BoundaryRPCCall, BoundaryFileIO,
}

// BoundaryPoint is a typed entry/exit attached to a script symbol. MatchKey is
// the normalized join target: two boundary points (or an edge and its target)
// link iff their keys are equal.
type BoundaryPoint struct {
	Direction Direction      `json:"direction"`
	Type      BoundaryType   `json:"type"`
	NodeID    string         `json:"node_id"` // the owning symbol (script::method)
	MatchKey  MatchKey       `json:"match_key,omitempty"`
	Evidence  Evidence       `json:"evidence"`
	Meta      map[string]any `json:"meta,omitempty"`
}
