package gdscript

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/dylanblakemore/godarch/internal/extract/gdscript/grammar"
	"github.com/dylanblakemore/godarch/internal/model"
)

// Diagnostic records source godarch could not parse. Nothing is dropped
// silently (M1 exit criterion): the problem is surfaced with its location.
type Diagnostic struct {
	File string
	Line int
	Msg  string
}

// Extract runs the GDScript extractor over every .gd script already discovered
// in p, emitting the code-origin nodes (methods, signals, classes), the edges of
// DESIGN §3.2, and the ingress/egress boundary points of DESIGN §4. Unparseable
// input is recorded in the returned diagnostics, never dropped.
//
// root is the filesystem project root; script node IDs are the res:// paths
// discovery produced. Cross-file symbol resolution is deferred to M2 — a call,
// emit, or load whose target cannot be determined from the file alone is left
// unresolved (its raw target held as a match key or expression).
func Extract(root string, p *model.Project) ([]Diagnostic, error) {
	ids := make([]string, 0, len(p.Nodes))
	for id, n := range p.Nodes {
		if n.Kind == model.KindScript && strings.HasSuffix(n.Path, ".gd") {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)

	var diags []Diagnostic
	for _, id := range ids {
		data, err := os.ReadFile(fsPath(root, id))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", id, err)
		}
		diags = append(diags, extract(p, id, data)...)
	}
	return diags, nil
}

// autoloadSet collects the autoload names registered in the project so the
// reference pass can recognise <AutoloadName>.member access.
func autoloadSet(p *model.Project) map[string]bool {
	set := map[string]bool{}
	for _, n := range p.Nodes {
		if n.Kind == model.KindAutoload {
			set[strings.TrimPrefix(n.ID, "autoload:")] = true
		}
	}
	return set
}

// extractor holds the per-file state shared by the declaration and reference
// passes: the parsed source, the owning script's node ID, and the small
// best-effort type lattice used to sharpen signal/RPC match keys.
type extractor struct {
	p         *model.Project
	file      string
	src       []byte
	autoloads map[string]bool

	localFuncs   map[string]bool   // functions declared in this file (self-call resolution)
	varTypes     map[string]string // var/param name → declared type (best effort)
	selfClass    string            // this script's class_name, if any
	seenAutoload map[string]bool   // owner|name pairs already recorded (dedup)

	diags []Diagnostic
}

// extract parses one script's source and appends its code-origin nodes, edges,
// and boundaries to p. file is the script's res:// node ID.
func extract(p *model.Project, file string, src []byte) []Diagnostic {
	e := &extractor{
		p:            p,
		file:         file,
		src:          src,
		autoloads:    autoloadSet(p),
		localFuncs:   map[string]bool{},
		varTypes:     map[string]string{},
		seenAutoload: map[string]bool{},
	}

	parser := sitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(grammar.Language()); err != nil {
		e.diags = append(e.diags, Diagnostic{File: file, Msg: "set language: " + err.Error()})
		return e.diags
	}
	tree := parser.Parse(src, nil)
	defer tree.Close()

	root := tree.RootNode()
	if root.HasError() {
		e.diags = append(e.diags, Diagnostic{
			File: file,
			Line: 1,
			Msg:  "source contains parse errors; extraction is best-effort",
		})
	}

	e.declare(root)
	e.reference(root)
	return e.diags
}

// --- text helpers ---

func (e *extractor) text(n *sitter.Node) string {
	if n == nil {
		return ""
	}
	return n.Utf8Text(e.src)
}

func (e *extractor) line(n *sitter.Node) int {
	if n == nil {
		return 0
	}
	return int(n.StartPosition().Row) + 1
}

// stringText returns the contents of a string / string_name literal with its
// quotes (and any & / ^ prefix) stripped.
func (e *extractor) stringText(n *sitter.Node) string {
	s := e.text(n)
	s = strings.TrimPrefix(s, "&")
	s = strings.TrimPrefix(s, "^")
	switch {
	case len(s) >= 6 && strings.HasPrefix(s, `"""`):
		return s[3 : len(s)-3]
	case len(s) >= 2 && (s[0] == '"' || s[0] == '\''):
		return s[1 : len(s)-1]
	}
	return s
}

// isStringLit reports whether n is a string / string_name literal node.
func isStringLit(n *sitter.Node) bool {
	return n != nil && (n.Kind() == "string" || n.Kind() == "string_name")
}

// stringArg returns the i-th argument of an arguments node when it is a string
// literal, plus whether it was one.
func (e *extractor) stringArg(args *sitter.Node, i int) (string, bool) {
	if args == nil {
		return "", false
	}
	if uint(i) >= args.NamedChildCount() {
		return "", false
	}
	a := args.NamedChild(uint(i))
	if isStringLit(a) {
		return e.stringText(a), true
	}
	return "", false
}

// namedChildren returns n's named children as a slice for convenient iteration.
func namedChildren(n *sitter.Node) []*sitter.Node {
	if n == nil {
		return nil
	}
	out := make([]*sitter.Node, 0, n.NamedChildCount())
	for i := uint(0); i < n.NamedChildCount(); i++ {
		out = append(out, n.NamedChild(i))
	}
	return out
}

// fsPath converts a res:// node ID back to its filesystem path under root.
func fsPath(root, resID string) string {
	rel := strings.TrimPrefix(resID, "res://")
	return filepath.Join(root, filepath.FromSlash(rel))
}
