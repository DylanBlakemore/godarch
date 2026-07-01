package gdscript

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/dylanblakemore/godarch/internal/model"
)

// reference runs the reference pass: it walks the tree tracking the enclosing
// method (the "owner" of each call site) and dispatches call/attribute/get_node
// nodes through the detector registry to emit egress edges and boundaries.
func (e *extractor) reference(root *sitter.Node) {
	e.walkRef(root, e.file)
}

// walkRef descends n, emitting reference edges. owner is the ID that call sites
// hang off: the enclosing method's symbol ID, or the script file at module
// scope (e.g. an @onready initializer).
func (e *extractor) walkRef(n *sitter.Node, owner string) {
	switch n.Kind() {
	case "function_definition", "constructor_definition":
		newOwner := owner
		if name := e.text(n.ChildByFieldName("name")); name != "" {
			newOwner = model.SymbolID(e.file, name)
		}
		for _, c := range namedChildren(n) {
			e.walkRef(c, newOwner)
		}
		return
	case "call":
		e.detectCall(n, owner)
	case "attribute":
		e.detectAttribute(n, owner)
	case "get_node":
		e.detectGetNode(n, owner)
	}
	for _, c := range namedChildren(n) {
		e.walkRef(c, owner)
	}
}

// detectCall classifies a plain function call foo(args) by callee name.
func (e *extractor) detectCall(n *sitter.Node, owner string) {
	callee := n.NamedChild(0)
	if callee == nil || callee.Kind() != "identifier" {
		return
	}
	name := e.text(callee)
	args := n.ChildByFieldName("arguments")
	line := e.line(n)

	switch name {
	case "emit_signal":
		e.emitSignal(owner, "*", e.argOrExpr(args, 0), line)
	case "connect":
		e.connectSignal(owner, "*", e.argOrExpr(args, 0), line)
	case "preload":
		e.loadResource(owner, args, 0, line, true)
	case "load", "ResourceLoader":
		e.loadResource(owner, args, 0, line, false)
	case "get_node", "get_node_or_null":
		e.nodeReachArg(owner, args, line)
	case "add_to_group":
		if g, ok := e.stringArg(args, 0); ok {
			e.inGroup(g, line)
		}
	case "change_scene_to_file", "change_scene_to_packed":
		e.sceneChange(owner, args, line)
	case "rpc":
		e.rpcCall(owner, args, 0, line)
	case "rpc_id":
		e.rpcCall(owner, args, 1, line)
	default:
		if e.localFuncs[name] {
			e.addEdge(model.EdgeCalls, owner, model.SymbolID(e.file, name), true, 1.0, line, nil)
		}
	}
}

// detectAttribute classifies method calls and property access on a receiver:
// obj.method(args) or Autoload.member.
func (e *extractor) detectAttribute(n *sitter.Node, owner string) {
	kids := namedChildren(n)
	if len(kids) == 0 {
		return
	}
	line := e.line(n)

	head := kids[0]
	headText := ""
	if head.Kind() == "identifier" {
		headText = e.text(head)
	}
	if headText != "" && e.autoloads[headText] {
		e.autoloadAccess(owner, headText, line)
	}

	last := kids[len(kids)-1]
	if last.Kind() != "attribute_call" {
		return // property access only; autoload handled above
	}
	method := e.text(firstChildOfKind(last, "identifier"))
	args := last.ChildByFieldName("arguments")
	recvBefore := ""
	if len(kids) >= 2 {
		recvBefore = e.text(kids[len(kids)-2])
	}

	switch method {
	case "emit":
		e.emitSignal(owner, e.emitterType(kids[:len(kids)-1]), recvBefore, line)
	case "connect":
		e.connectSignal(owner, e.emitterType(kids[:len(kids)-1]), recvBefore, line)
	case "change_scene_to_file", "change_scene_to_packed":
		e.sceneChange(owner, args, line)
	case "call_group":
		if g, ok := e.stringArg(args, 0); ok {
			e.callsGroup(owner, g, line)
		}
	case "call_group_flags":
		if g, ok := e.stringArg(args, 1); ok {
			e.callsGroup(owner, g, line)
		}
	case "get_nodes_in_group":
		if g, ok := e.stringArg(args, 0); ok {
			e.callsGroup(owner, g, line)
		}
	case "rpc":
		e.rpcCall(owner, args, 0, line)
	case "rpc_id":
		e.rpcCall(owner, args, 1, line)
	case "load":
		if headText == "ResourceLoader" {
			e.loadResource(owner, args, 0, line, false)
		}
	default:
		switch {
		case headText == "Input" && isActionMethod(method):
			if a, ok := e.stringArg(args, 0); ok {
				e.usesAction(owner, a, line)
			}
		case isFileIOReceiver(headText):
			path, _ := e.stringArg(args, 0)
			e.fileIO(owner, path, line)
		}
	}
}

// detectGetNode emits a node-reach edge for $Path / %Unique syntax.
func (e *extractor) detectGetNode(n *sitter.Node, owner string) {
	e.nodeReach(owner, e.nodePathText(n), e.line(n), 1.0)
}

// argOrExpr returns the i-th argument's string value, or its raw source text
// when it is not a literal (a dynamic target).
func (e *extractor) argOrExpr(args *sitter.Node, i int) string {
	if s, ok := e.stringArg(args, i); ok {
		return s
	}
	if args != nil && uint(i) < args.NamedChildCount() {
		return e.text(args.NamedChild(uint(i)))
	}
	return ""
}

// emitterType infers the class that owns a signal from its receiver chain: the
// script's own class for self, a typed variable's declared type, else "*".
func (e *extractor) emitterType(chain []*sitter.Node) string {
	if len(chain) == 0 {
		return "*"
	}
	head := chain[0]
	if head.Kind() != "identifier" {
		return "*"
	}
	name := e.text(head)
	if name == "self" {
		if e.selfClass != "" {
			return e.selfClass
		}
		return "*"
	}
	if t, ok := e.varTypes[name]; ok {
		return t
	}
	return "*"
}

// nodePathText extracts the path expression from a $Path / %Unique get_node
// node, stripping the leading $ and any surrounding quotes (the % unique-name
// marker is preserved).
func (e *extractor) nodePathText(n *sitter.Node) string {
	s := e.text(n)
	if len(s) > 0 && s[0] == '$' {
		s = s[1:]
	}
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') {
		s = s[1 : len(s)-1]
	}
	return s
}

func isActionMethod(method string) bool {
	switch method {
	case "is_action_pressed", "is_action_just_pressed", "is_action_just_released",
		"is_action_released", "get_action_strength", "get_action_raw_strength":
		return true
	}
	return false
}

func isFileIOReceiver(head string) bool {
	switch head {
	case "FileAccess", "ConfigFile", "DirAccess":
		return true
	}
	return false
}
