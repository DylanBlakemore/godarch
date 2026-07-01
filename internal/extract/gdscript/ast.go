package gdscript

import (
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// firstChildOfKind returns the first (named or unnamed) child of n with the
// given kind, or nil.
func firstChildOfKind(n *sitter.Node, kind string) *sitter.Node {
	if n == nil {
		return nil
	}
	for i := uint(0); i < n.ChildCount(); i++ {
		c := n.Child(i)
		if c.Kind() == kind {
			return c
		}
	}
	return nil
}

// precedingAnnotations collects annotation nodes that appear as siblings
// immediately before n (how the grammar attaches @tool / @rpc / @export_* to a
// following function). Keyed by annotation name (without the @).
func (e *extractor) precedingAnnotations(n *sitter.Node) map[string]*sitter.Node {
	out := map[string]*sitter.Node{}
	for sib := n.PrevNamedSibling(); sib != nil && sib.Kind() == "annotation"; sib = sib.PrevNamedSibling() {
		if id := firstChildOfKind(sib, "identifier"); id != nil {
			out[e.text(id)] = sib
		}
	}
	return out
}

// nestedAnnotations collects annotation nodes nested inside a variable_statement
// (how the grammar attaches @export / @onready to a var). Keyed by name.
func (e *extractor) nestedAnnotations(n *sitter.Node) map[string]*sitter.Node {
	out := map[string]*sitter.Node{}
	anns := firstChildOfKind(n, "annotations")
	if anns == nil {
		return out
	}
	for _, a := range namedChildren(anns) {
		if a.Kind() != "annotation" {
			continue
		}
		if id := firstChildOfKind(a, "identifier"); id != nil {
			out[e.text(id)] = a
		}
	}
	return out
}

// hasExport reports whether any annotation name begins with "export".
func hasExport(annotations map[string]*sitter.Node) bool {
	for name := range annotations {
		if strings.HasPrefix(name, "export") {
			return true
		}
	}
	return false
}

// hasOnready reports whether the @onready annotation is present.
func hasOnready(annotations map[string]*sitter.Node) bool {
	_, ok := annotations["onready"]
	return ok
}

// virtualMethods are the engine lifecycle callbacks Godot invokes directly.
var virtualMethods = map[string]bool{
	"_init": true, "_ready": true, "_enter_tree": true, "_exit_tree": true,
	"_process": true, "_physics_process": true, "_draw": true,
	"_get_configuration_warnings": true, "_to_string": true,
	"_integrate_forces": true, "_get": true, "_set": true, "_get_property_list": true,
}

// inputHandlers are the virtual callbacks that receive input events.
var inputHandlers = map[string]bool{
	"_input": true, "_unhandled_input": true, "_unhandled_key_input": true,
	"_gui_input": true, "_shortcut_input": true,
}

func isVirtual(name string) bool {
	return virtualMethods[name] || isInputHandler(name) || name == "_notification"
}

func isInputHandler(name string) bool { return inputHandlers[name] }

// decisionKinds are the CST node kinds that add a branch to cyclomatic
// complexity: each is one more independent path through the function.
var decisionKinds = map[string]bool{
	"if_statement":           true,
	"elif_clause":            true,
	"for_statement":          true,
	"while_statement":        true,
	"and":                    true,
	"or":                     true,
	"conditional_expression": true,
	"pattern_section":        true,
}

// complexity computes McCabe cyclomatic complexity for a function body: one base
// path plus one per decision point.
func (e *extractor) complexity(body *sitter.Node) int {
	if body == nil {
		return 1
	}
	count := 1
	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		if decisionKinds[n.Kind()] {
			count++
		}
		for i := uint(0); i < n.NamedChildCount(); i++ {
			walk(n.NamedChild(i))
		}
	}
	walk(body)
	return count
}
