package gdscript

import (
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/dylanblakemore/godarch/internal/model"
)

// declare runs the declaration pass: it walks the tree top-down and materialises
// the code-origin nodes (class_name, signals, methods) plus the declaration
// edges (extends, declares_signal, exports_var, rpc_endpoint) and the per-method
// ingress boundaries. It also seeds the local-function set and type lattice the
// reference pass consumes.
func (e *extractor) declare(root *sitter.Node) {
	e.declareBlock(root, e.file)
}

// declareBlock processes the direct children of a source / class_body block.
// scope is the ID edges/nodes hang off (the script file, or an inner class).
func (e *extractor) declareBlock(block *sitter.Node, scope string) {
	for _, n := range namedChildren(block) {
		switch n.Kind() {
		case "extends_statement":
			e.declareExtends(n, scope)
		case "class_name_statement":
			e.declareClassName(n)
		case "class_definition":
			e.declareInnerClass(n)
		case "signal_statement":
			e.declareSignal(n)
		case "function_definition":
			e.declareFunc(n)
		case "constructor_definition":
			e.declareFunc(n)
		case "variable_statement", "export_variable_statement", "onready_variable_statement":
			e.declareVar(n)
		case "const_statement":
			if name := e.text(n.ChildByFieldName("name")); name != "" {
				e.appendIdentity("consts", name)
			}
		case "enum_definition":
			if name := e.text(n.ChildByFieldName("name")); name != "" {
				e.appendIdentity("enums", name)
			}
		}
	}
}

// declareExtends emits the file-level extends edge. A string operand is a script
// path (resolved when that node exists); an identifier is a class name, left
// unresolved for M2 to bind against the class registry.
func (e *extractor) declareExtends(n *sitter.Node, scope string) {
	// Only the script's own extends is an inheritance edge; inner-class extends
	// is recorded as identity by declareInnerClass.
	if scope != e.file {
		return
	}
	line := e.line(n)
	if str := firstChildOfKind(n, "string"); str != nil {
		path := model.NormalizePath(e.stringText(str))
		_, exists := e.p.Nodes[path]
		e.addEdge(model.EdgeExtends, e.file, path, exists, 1.0, line, nil)
		e.identity()["extends_path"] = path
		return
	}
	if typ := firstChildOfKind(n, "type"); typ != nil {
		name := e.text(typ)
		e.ensureClassNode(name)
		e.addEdge(model.EdgeExtends, e.file, model.ClassID(name), false, 1.0, line, nil)
		e.identity()["extends"] = name
	}
}

func (e *extractor) declareClassName(n *sitter.Node) {
	name := e.text(n.ChildByFieldName("name"))
	if name == "" {
		return
	}
	e.selfClass = name
	e.ensureClassNode(name)
	// Point the class node at its defining script for M2 resolution.
	e.p.Nodes[model.ClassID(name)].Identity["script"] = e.file
	id := e.identity()
	id["class_name"] = name
	id["is_class_name"] = true
}

func (e *extractor) declareInnerClass(n *sitter.Node) {
	name := e.text(n.ChildByFieldName("name"))
	if name != "" {
		e.appendIdentity("inner_classes", name)
	}
	// Recurse so nested methods/signals still become nodes (flat symbol IDs).
	if body := n.ChildByFieldName("body"); body != nil {
		e.declareBlock(body, e.file)
	}
}

func (e *extractor) declareSignal(n *sitter.Node) {
	name := e.text(n.ChildByFieldName("name"))
	if name == "" {
		return
	}
	id := model.SignalDeclID(e.file, name)
	ident := map[string]any{"name": name, "script": e.file}
	if params := e.paramNames(n.ChildByFieldName("parameters")); len(params) > 0 {
		ident["params"] = params
	}
	e.p.Nodes[id] = &model.Node{ID: id, Kind: model.KindSignal, Path: e.file, Line: e.line(n), Identity: ident}
	e.addEdge(model.EdgeDeclaresSignal, e.file, id, true, 1.0, e.line(n), nil)
}

// declareFunc creates the method node (arity, return type, static, virtual,
// complexity) and emits its ingress boundary and any @rpc endpoint.
func (e *extractor) declareFunc(n *sitter.Node) {
	name := e.text(n.ChildByFieldName("name"))
	if name == "" {
		return
	}
	id := model.SymbolID(e.file, name)
	e.localFuncs[name] = true

	params := n.ChildByFieldName("parameters")
	static := firstChildOfKind(n, "static_keyword") != nil
	virtual := isVirtual(name)
	ident := map[string]any{
		"name":    name,
		"arity":   len(namedChildren(params)),
		"static":  static,
		"virtual": virtual,
	}
	if rt := n.ChildByFieldName("return_type"); rt != nil {
		ident["return_type"] = e.text(rt)
	}

	annotations := e.precedingAnnotations(n)
	if _, ok := annotations["rpc"]; ok {
		ident["rpc"] = true
	}

	node := &model.Node{
		ID: id, Kind: model.KindMethod, Path: e.file, Line: e.line(n),
		Identity:   ident,
		Properties: map[string]any{"complexity": e.complexity(n.ChildByFieldName("body"))},
	}
	e.p.Nodes[id] = node

	// Seed the type lattice with typed parameters (best effort).
	for _, param := range namedChildren(params) {
		if pn, pt := e.typedParam(param); pn != "" && pt != "" {
			e.varTypes[pn] = pt
		}
	}

	e.declareFuncBoundaries(id, name, annotations, e.line(n))
}

// declareFuncBoundaries emits the ingress boundary for a method by name and,
// when @rpc-annotated, the rpc_endpoint edge + ingress.
func (e *extractor) declareFuncBoundaries(id, name string, annotations map[string]*sitter.Node, line int) {
	switch {
	case isInputHandler(name):
		e.addBoundary(model.DirectionIngress, model.BoundaryInputHandler, id, "", line, nil)
	case name == "_notification":
		e.addBoundary(model.DirectionIngress, model.BoundaryNotification, id, "", line, nil)
	case strings.HasPrefix(name, "_on_"):
		e.addBoundary(model.DirectionIngress, model.BoundarySignalHandler, id, "", line, map[string]any{"method": name})
	case isVirtual(name):
		e.addBoundary(model.DirectionIngress, model.BoundaryLifecycle, id, "", line, map[string]any{"method": name})
	}

	if _, ok := annotations["rpc"]; ok {
		key := model.RPCKey(e.rpcClass(), name)
		e.addEdge(model.EdgeRPCEndpoint, id, string(key), false, 1.0, line, nil)
		e.addBoundary(model.DirectionIngress, model.BoundaryRPCEndpoint, id, key, line, nil)
	}
}

// declareVar handles var/const-style statements: it records @export vars as
// exports_var edges and tracks declared types in the lattice.
func (e *extractor) declareVar(n *sitter.Node) {
	name := e.text(n.ChildByFieldName("name"))
	if name == "" {
		return
	}
	if typ := n.ChildByFieldName("type"); typ != nil && typ.Kind() == "type" {
		e.varTypes[name] = e.text(typ)
	}

	annotations := e.nestedAnnotations(n)
	if hasOnready(annotations) {
		e.appendIdentity("onready", name)
	}
	if hasExport(annotations) {
		props := map[string]any{"var": name}
		if t, ok := e.varTypes[name]; ok {
			props["type"] = t
		}
		e.addEdge(model.EdgeExportsVar, e.file, model.SymbolID(e.file, name), true, 1.0, e.line(n), props)
		e.appendIdentity("exports", name)
	}
}

// rpcClass returns the class component of an RPC match key for this script: its
// class_name when declared, otherwise the wildcard "*" is applied by RPCKey.
func (e *extractor) rpcClass() string {
	return e.selfClass
}

// paramNames returns the declared names of a parameters node.
func (e *extractor) paramNames(params *sitter.Node) []string {
	var out []string
	for _, p := range namedChildren(params) {
		if name := e.paramName(p); name != "" {
			out = append(out, name)
		}
	}
	return out
}

func (e *extractor) paramName(p *sitter.Node) string {
	switch p.Kind() {
	case "identifier":
		return e.text(p)
	case "typed_parameter", "typed_default_parameter", "default_parameter", "variadic_parameter":
		if id := firstChildOfKind(p, "identifier"); id != nil {
			return e.text(id)
		}
	}
	return ""
}

// typedParam returns a parameter's (name, type) when it carries a concrete type.
func (e *extractor) typedParam(p *sitter.Node) (string, string) {
	if p.Kind() != "typed_parameter" && p.Kind() != "typed_default_parameter" {
		return "", ""
	}
	name := ""
	if id := firstChildOfKind(p, "identifier"); id != nil {
		name = e.text(id)
	}
	t := p.ChildByFieldName("type")
	if t == nil || t.Kind() != "type" {
		return "", ""
	}
	return name, e.text(t)
}
