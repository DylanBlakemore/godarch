package gdscript

import (
	"testing"

	"github.com/dylanblakemore/godarch/internal/model"
)

// newProject builds a Project seeded with a single script node and the given
// autoload names, mirroring what discovery produces before the GDScript
// extractor runs.
func newProject(file string, autoloads ...string) *model.Project {
	p := model.NewProject("res://")
	p.Nodes[file] = &model.Node{ID: file, Kind: model.KindScript, Path: file}
	for _, a := range autoloads {
		p.Nodes[model.AutoloadID(a)] = &model.Node{ID: model.AutoloadID(a), Kind: model.KindAutoload}
	}
	return p
}

// run parses src as file into a fresh project and returns it.
func run(t *testing.T, file, src string, autoloads ...string) *model.Project {
	t.Helper()
	p := newProject(file, autoloads...)
	diags := extract(p, file, []byte(src))
	for _, d := range diags {
		t.Logf("diag: %s:%d %s", d.File, d.Line, d.Msg)
	}
	return p
}

// findEdge returns the first edge matching type+source (and target if non-empty).
func findEdge(p *model.Project, typ model.EdgeType, src, tgt string) *model.Edge {
	for _, e := range p.Edges {
		if e.Type == typ && e.SourceID == src && (tgt == "" || e.TargetID == tgt) {
			return e
		}
	}
	return nil
}

func findBoundary(p *model.Project, dir model.Direction, typ model.BoundaryType, nodeID string) *model.BoundaryPoint {
	for _, b := range p.Boundaries {
		if b.Direction == dir && b.Type == typ && (nodeID == "" || b.NodeID == nodeID) {
			return b
		}
	}
	return nil
}

func mustEdge(t *testing.T, p *model.Project, typ model.EdgeType, src, tgt string) *model.Edge {
	t.Helper()
	e := findEdge(p, typ, src, tgt)
	if e == nil {
		t.Fatalf("missing edge %s %s -> %s\nedges: %s", typ, src, tgt, dumpEdges(p))
	}
	return e
}

func mustBoundary(t *testing.T, p *model.Project, dir model.Direction, typ model.BoundaryType, nodeID string) *model.BoundaryPoint {
	t.Helper()
	b := findBoundary(p, dir, typ, nodeID)
	if b == nil {
		t.Fatalf("missing %s boundary %s on %s", dir, typ, nodeID)
	}
	return b
}

func dumpEdges(p *model.Project) string {
	s := ""
	for _, e := range p.Edges {
		s += "\n  " + string(e.Type) + " " + e.SourceID + " -> " + e.TargetID
	}
	return s
}

const file = "res://player.gd"

func TestExtends_BuiltinClass(t *testing.T) {
	p := run(t, file, "extends Node2D\n")
	e := mustEdge(t, p, model.EdgeExtends, file, model.ClassID("Node2D"))
	if e.Origin != model.OriginCode || e.Resolved {
		t.Errorf("extends edge: origin=%s resolved=%v, want code/false", e.Origin, e.Resolved)
	}
	if p.Nodes[model.ClassID("Node2D")] == nil {
		t.Error("expected class:Node2D node")
	}
}

func TestExtends_PathString(t *testing.T) {
	p := newProject(file)
	p.Nodes["res://base.gd"] = &model.Node{ID: "res://base.gd", Kind: model.KindScript}
	extract(p, file, []byte("extends \"res://base.gd\"\n"))
	e := mustEdge(t, p, model.EdgeExtends, file, "res://base.gd")
	if !e.Resolved {
		t.Errorf("extends-to-existing-path should be resolved")
	}
}

func TestClassNameAndSignal(t *testing.T) {
	p := run(t, file, "class_name Player\nsignal died(who)\n")
	if p.Nodes[model.ClassID("Player")] == nil {
		t.Fatal("expected class:Player node")
	}
	if got := p.Nodes[file].Identity["class_name"]; got != "Player" {
		t.Errorf("script identity class_name = %v, want Player", got)
	}
	sigID := model.SignalDeclID(file, "died")
	if p.Nodes[sigID] == nil {
		t.Fatalf("expected signal node %s", sigID)
	}
	mustEdge(t, p, model.EdgeDeclaresSignal, file, sigID)
}

func TestFuncNodeAndComplexity(t *testing.T) {
	src := "func act(a, b) -> int:\n" +
		"\tif a:\n\t\treturn 1\n" +
		"\telif b:\n\t\treturn 2\n" +
		"\tfor i in range(3):\n\t\tpass\n" +
		"\treturn 0\n"
	p := run(t, file, src)
	id := model.SymbolID(file, "act")
	n := p.Nodes[id]
	if n == nil || n.Kind != model.KindMethod {
		t.Fatalf("expected method node %s, got %+v", id, n)
	}
	if n.Identity["arity"] != 2 {
		t.Errorf("arity = %v, want 2", n.Identity["arity"])
	}
	// base 1 + if + elif + for = 4
	if n.Properties["complexity"] != 4 {
		t.Errorf("complexity = %v, want 4", n.Properties["complexity"])
	}
}

func TestLifecycleAndHandlerBoundaries(t *testing.T) {
	src := "func _ready():\n\tpass\n" +
		"func _input(ev):\n\tpass\n" +
		"func _notification(w):\n\tpass\n" +
		"func _on_timer_timeout():\n\tpass\n"
	p := run(t, file, src)
	mustBoundary(t, p, model.DirectionIngress, model.BoundaryLifecycle, model.SymbolID(file, "_ready"))
	mustBoundary(t, p, model.DirectionIngress, model.BoundaryInputHandler, model.SymbolID(file, "_input"))
	mustBoundary(t, p, model.DirectionIngress, model.BoundaryNotification, model.SymbolID(file, "_notification"))
	mustBoundary(t, p, model.DirectionIngress, model.BoundarySignalHandler, model.SymbolID(file, "_on_timer_timeout"))
}

func TestEmitSignalForms(t *testing.T) {
	src := "func a():\n\temit_signal(\"died\", 1)\n" +
		"func b():\n\tdied.emit()\n"
	p := run(t, file, src)
	mustEdge(t, p, model.EdgeEmitsSignal, model.SymbolID(file, "a"), string(model.SignalKey("*", "died")))
	mustEdge(t, p, model.EdgeEmitsSignal, model.SymbolID(file, "b"), string(model.SignalKey("*", "died")))
	mustBoundary(t, p, model.DirectionEgress, model.BoundarySignalEmit, model.SymbolID(file, "a"))
}

func TestConnectForms(t *testing.T) {
	src := "func a():\n\tbutton.pressed.connect(_on_press)\n" +
		"func b():\n\tconnect(\"timeout\", _on_to)\n"
	p := run(t, file, src)
	mustEdge(t, p, model.EdgeConnectsSignal, model.SymbolID(file, "a"), string(model.SignalKey("*", "pressed")))
	mustEdge(t, p, model.EdgeConnectsSignal, model.SymbolID(file, "b"), string(model.SignalKey("*", "timeout")))
	mustBoundary(t, p, model.DirectionEgress, model.BoundarySignalConnect, model.SymbolID(file, "a"))
}

func TestResourceLoads(t *testing.T) {
	p := newProject(file)
	p.Nodes["res://x.tscn"] = &model.Node{ID: "res://x.tscn", Kind: model.KindScene}
	src := "func a():\n\tvar r = preload(\"res://x.tscn\")\n" +
		"func b():\n\tvar d = load(some_path)\n"
	extract(p, file, []byte(src))

	pre := mustEdge(t, p, model.EdgeLoadsResource, model.SymbolID(file, "a"), string(model.ResourceKey("res://x.tscn")))
	if !pre.Resolved || pre.Confidence != 1.0 {
		t.Errorf("preload of existing resource: resolved=%v conf=%v, want true/1.0", pre.Resolved, pre.Confidence)
	}
	dyn := findEdge(p, model.EdgeLoadsResource, model.SymbolID(file, "b"), "")
	if dyn == nil {
		t.Fatal("expected dynamic load edge")
	}
	if dyn.Resolved || dyn.Confidence >= 1.0 {
		t.Errorf("dynamic load: resolved=%v conf=%v, want false/<1", dyn.Resolved, dyn.Confidence)
	}
	mustBoundary(t, p, model.DirectionEgress, model.BoundaryResourceLoad, model.SymbolID(file, "a"))
}

func TestNodeReach(t *testing.T) {
	src := "func a():\n\tget_node(\"Enemy/Sprite\").hide()\n" +
		"func b():\n\t$Player/Gun.shoot()\n"
	p := run(t, file, src)
	mustEdge(t, p, model.EdgeReferencesNode, model.SymbolID(file, "a"), string(model.NodePathKey("Enemy/Sprite")))
	mustBoundary(t, p, model.DirectionEgress, model.BoundaryNodeReach, model.SymbolID(file, "a"))
}

func TestAutoloadAccess(t *testing.T) {
	src := "func a():\n\tGameState.score += 1\n"
	p := run(t, file, src, "GameState")
	e := mustEdge(t, p, model.EdgeAccessesAutoload, model.SymbolID(file, "a"), model.AutoloadID("GameState"))
	if !e.Resolved {
		t.Error("autoload access to known autoload should be resolved")
	}
	mustBoundary(t, p, model.DirectionEgress, model.BoundaryAutoloadAccess, model.SymbolID(file, "a"))
}

func TestSceneChangeGroupsActions(t *testing.T) {
	src := "func a():\n\tget_tree().change_scene_to_file(\"res://m.tscn\")\n" +
		"func b():\n\tadd_to_group(\"enemies\")\n" +
		"func c():\n\tget_tree().call_group(\"enemies\", \"hit\")\n" +
		"func d():\n\tif Input.is_action_pressed(\"jump\"):\n\t\tpass\n"
	p := run(t, file, src)
	mustEdge(t, p, model.EdgeChangesSceneTo, model.SymbolID(file, "a"), string(model.ResourceKey("res://m.tscn")))
	mustEdge(t, p, model.EdgeInGroup, file, model.GroupID("enemies"))
	mustEdge(t, p, model.EdgeCallsGroup, model.SymbolID(file, "c"), model.GroupID("enemies"))
	mustEdge(t, p, model.EdgeUsesAction, model.SymbolID(file, "d"), model.ActionID("jump"))
	mustBoundary(t, p, model.DirectionEgress, model.BoundarySceneChange, model.SymbolID(file, "a"))
	mustBoundary(t, p, model.DirectionEgress, model.BoundaryGroupCall, model.SymbolID(file, "c"))
}

func TestRPC(t *testing.T) {
	src := "@rpc(\"any_peer\", \"reliable\")\nfunc take_dmg(x):\n\tpass\n" +
		"func a():\n\trpc(\"take_dmg\", 5)\n" +
		"func b():\n\trpc_id(1, \"take_dmg\", 5)\n"
	p := run(t, file, src)
	ep := model.SymbolID(file, "take_dmg")
	mustEdge(t, p, model.EdgeRPCEndpoint, ep, string(model.RPCKey("*", "take_dmg")))
	mustBoundary(t, p, model.DirectionIngress, model.BoundaryRPCEndpoint, ep)
	mustEdge(t, p, model.EdgeRPCCall, model.SymbolID(file, "a"), string(model.RPCKey("*", "take_dmg")))
	mustEdge(t, p, model.EdgeRPCCall, model.SymbolID(file, "b"), string(model.RPCKey("*", "take_dmg")))
	mustBoundary(t, p, model.DirectionEgress, model.BoundaryRPCCall, model.SymbolID(file, "a"))
}

func TestExportAndFileIO(t *testing.T) {
	src := "@export var speed: int = 100\n" +
		"func a():\n\tvar f = FileAccess.open(\"user://save.dat\", 2)\n"
	p := run(t, file, src)
	mustEdge(t, p, model.EdgeExportsVar, file, model.SymbolID(file, "speed"))
	mustBoundary(t, p, model.DirectionEgress, model.BoundaryFileIO, model.SymbolID(file, "a"))
}

func TestSelfCall(t *testing.T) {
	src := "func a():\n\thelper()\n" +
		"func helper():\n\tpass\n"
	p := run(t, file, src)
	e := mustEdge(t, p, model.EdgeCalls, model.SymbolID(file, "a"), model.SymbolID(file, "helper"))
	if !e.Resolved {
		t.Error("self-call to local func should be resolved")
	}
}

func TestParseErrorProducesDiagnostic(t *testing.T) {
	p := newProject(file)
	diags := extract(p, file, []byte("func broken(:\n\t@@@\n"))
	if len(diags) == 0 {
		t.Error("expected a diagnostic for unparseable source")
	}
}
