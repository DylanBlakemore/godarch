package scene

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dylanblakemore/godarch/internal/model"
)

// Extract runs the scene/resource/config extractor over every scene, resource,
// and GDExtension file already discovered in p, plus each asset's .import
// sidecar and any addons plugin.cfg. It emits scene_node nodes, the
// editor-configured edges of DESIGN §3.2, and editor_connection ingress boundary
// points, appending them to p. Unparseable input is recorded in the returned
// diagnostics, never dropped silently (M1 exit criterion).
//
// root is the filesystem project root; node paths are the res:// IDs discovery
// produced. Cross-file symbol resolution is deferred to M2 — edges whose target
// cannot be determined from the file alone are left unresolved with a match key.
func Extract(root string, p *model.Project) ([]Diagnostic, error) {
	var diags []Diagnostic

	ids := make([]string, 0, len(p.Nodes))
	for id := range p.Nodes {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		n := p.Nodes[id]
		switch n.Kind {
		case model.KindScene:
			d, err := extractScene(root, p, n)
			if err != nil {
				return nil, err
			}
			diags = append(diags, d...)
		case model.KindResource:
			d, err := extractResource(root, p, n)
			if err != nil {
				return nil, err
			}
			diags = append(diags, d...)
		case model.KindExtension:
			d, err := extractExtension(root, n)
			if err != nil {
				return nil, err
			}
			diags = append(diags, d...)
		case model.KindAsset:
			d, err := extractImport(root, p, n)
			if err != nil {
				return nil, err
			}
			diags = append(diags, d...)
		}
	}

	d, err := scanPlugins(root, p)
	if err != nil {
		return nil, err
	}
	diags = append(diags, d...)

	return diags, nil
}

// extResource is a resolved [ext_resource] table entry.
type extResource struct {
	typ  string
	path string
	uid  string
}

// sceneCtx carries the per-scene node index needed to wire connections: the
// script attached to each in-scene node path, and each node path's Godot type.
type sceneCtx struct {
	scriptByPath map[string]string
	typeByPath   map[string]string
}

func extractScene(root string, p *model.Project, scene *model.Node) ([]Diagnostic, error) {
	doc, err := parseFile(root, scene.Path)
	if err != nil {
		return nil, err
	}
	ext := extResources(doc)
	editable := editablePaths(doc)
	ctx := &sceneCtx{scriptByPath: map[string]string{}, typeByPath: map[string]string{}}

	for _, sec := range doc.Sections {
		if sec.Type == "node" {
			extractNode(p, scene, sec, ext, editable, ctx)
		}
	}
	for _, sec := range doc.Sections {
		if sec.Type == "sub_resource" {
			extractRefs(p, scene.ID, scene.ID, sec, ext)
		}
	}
	for _, sec := range doc.Sections {
		if sec.Type == "connection" {
			extractConnection(p, scene, sec, ctx)
		}
	}
	return doc.Diags, nil
}

// extractNode turns one [node] into a scene_node and emits its editor edges:
// attaches_script, instances, in_group, and any property references.
func extractNode(p *model.Project, scene *model.Node, sec *Section, ext map[string]extResource, editable map[string]bool, ctx *sceneCtx) {
	name := attrStr(sec, "name")
	typ := attrStr(sec, "type")
	parent := attrStr(sec, "parent")
	path := nodePathFor(parent, name)
	sid := model.SceneNodeID(scene.ID, path)

	ident := map[string]any{"name": name, "scene": scene.ID}
	if typ != "" {
		ident["node_type"] = typ
		ctx.typeByPath[path] = typ
	}
	if parent == "" {
		ident["root"] = true
	} else {
		ident["parent"] = parent
	}

	if v, ok := sec.Attrs["instance"]; ok {
		if id, ok := v.RefID(); ok {
			if target := resolveRef(p, ext, id); target != "" {
				ident["instance"] = target
				props := map[string]any{"node": path}
				if editable[path] {
					props["editable"] = true
				}
				addEdge(p, model.EdgeInstances, scene.ID, target, model.OriginEditor, true,
					model.Evidence{File: scene.ID, Line: sec.Line}, props)
			}
		}
	}

	if v, ok := sec.Props["script"]; ok {
		if id, ok := v.RefID(); ok {
			if target := resolveRef(p, ext, id); target != "" {
				ident["script"] = target
				ctx.scriptByPath[path] = target
				addEdge(p, model.EdgeAttachesScript, sid, target, model.OriginEditor, true,
					model.Evidence{File: scene.ID, Line: sec.PropLines["script"]}, nil)
			}
		}
	}

	if v, ok := sec.Attrs["groups"]; ok {
		for _, g := range v.StringItems() {
			addEdge(p, model.EdgeInGroup, sid, model.GroupID(g), model.OriginEditor, true,
				model.Evidence{File: scene.ID, Line: sec.Line}, nil)
		}
	}

	for key, val := range sec.Props {
		if key == "script" {
			continue
		}
		extractPropRef(p, sid, scene.ID, key, val, sec.PropLines[key], ext)
	}

	p.Nodes[sid] = &model.Node{
		ID: sid, Kind: model.KindSceneNode, Path: scene.ID, Line: sec.Line, Identity: ident,
	}
}

// extractPropRef emits the edges implied by a single property value. A NodePath
// attaches to the owning node (nodeID) as an unresolved references_node keyed by
// the raw expression (for fragile-reach analysis); an ExtResource asset/resource
// attaches to the owning scene/resource file (ownerID) as uses_asset /
// loads_resource / uses_shader / … chosen by its declared type (DESIGN §3.2). It
// recurses into arrays and dicts so nested references are not missed.
func extractPropRef(p *model.Project, nodeID, ownerID, key string, val Value, line int, ext map[string]extResource) {
	switch {
	case val.IsCtor("NodePath"):
		expr := ""
		if len(val.Args) > 0 {
			expr = val.Args[0].scalarString()
		}
		addEdge(p, model.EdgeReferencesNode, nodeID, string(model.NodePathKey(expr)), model.OriginEditor, false,
			model.Evidence{File: ownerID, Line: line}, map[string]any{"property": key, "path": expr})
	case val.IsCtor("ExtResource"):
		if id, ok := val.RefID(); ok {
			if target := resolveRef(p, ext, id); target != "" {
				e := ext[id]
				addEdge(p, assetEdgeType(e.typ), ownerID, target, model.OriginEditor, true,
					model.Evidence{File: ownerID, Line: line}, map[string]any{"property": key})
			}
		}
	case val.Type == ValArray:
		for _, a := range val.Args {
			extractPropRef(p, nodeID, ownerID, key, a, line, ext)
		}
	case val.Type == ValDict:
		for _, pr := range val.Pairs {
			extractPropRef(p, nodeID, ownerID, key, pr.Val, line, ext)
		}
	}
}

// extractRefs walks a [sub_resource] / [resource] section's property values for
// nested ExtResource references (asset/resource dependencies of the file).
func extractRefs(p *model.Project, src, sceneID string, sec *Section, ext map[string]extResource) {
	for key, val := range sec.Props {
		if key == "script" {
			continue
		}
		extractPropRef(p, src, sceneID, key, val, sec.PropLines[key], ext)
	}
}

// extractConnection turns a [connection] into a connects_signal edge (emitter
// node → handler method) and an editor_connection ingress on the handler. The
// handler method resolves locally when the target node's script is attached in
// this scene; otherwise the edge target falls back to the signal match key.
func extractConnection(p *model.Project, scene *model.Node, sec *Section, ctx *sceneCtx) {
	signal := attrStr(sec, "signal")
	from := attrStr(sec, "from")
	to := attrStr(sec, "to")
	method := attrStr(sec, "method")

	emitterID := model.SceneNodeID(scene.ID, from)
	emitterType := ctx.typeByPath[from]
	key := model.SignalKey(emitterType, signal)

	props := map[string]any{"signal": signal, "from": from, "to": to, "method": method}
	if v, ok := sec.Attrs["flags"]; ok && v.Type == ValInt {
		props["flags"] = v.Int
	}
	ev := model.Evidence{File: scene.ID, Line: sec.Line}

	handlerScript := ctx.scriptByPath[to]
	target := string(key)
	resolved := false
	handlerID := ""
	if handlerScript != "" {
		handlerID = model.SymbolID(handlerScript, method)
		target = handlerID
		resolved = true
	}

	addEdge(p, model.EdgeConnectsSignal, emitterID, target, model.OriginEditor, resolved, ev, props)

	p.Boundaries = append(p.Boundaries, &model.BoundaryPoint{
		Direction: model.DirectionIngress,
		Type:      model.BoundaryEditorConnection,
		NodeID:    handlerID,
		MatchKey:  key,
		Evidence:  ev,
		Meta:      props,
	})
}

func extractResource(root string, p *model.Project, res *model.Node) ([]Diagnostic, error) {
	doc, err := parseFile(root, res.Path)
	if err != nil {
		return nil, err
	}
	ext := extResources(doc)
	for _, sec := range doc.Sections {
		if sec.Type != "resource" && sec.Type != "sub_resource" {
			continue
		}
		if v, ok := sec.Props["script"]; ok {
			if id, ok := v.RefID(); ok {
				if target := resolveRef(p, ext, id); target != "" {
					addEdge(p, model.EdgeAttachesScript, res.ID, target, model.OriginEditor, true,
						model.Evidence{File: res.ID, Line: sec.PropLines["script"]}, nil)
				}
			}
		}
		extractRefs(p, res.ID, res.ID, sec, ext)
	}
	return doc.Diags, nil
}

// extractImport parses an asset's .import sidecar into an imports edge (asset →
// .import config) carrying the importer and produced type. The sidecar path was
// paired onto the asset's identity during discovery.
func extractImport(root string, p *model.Project, asset *model.Node) ([]Diagnostic, error) {
	imp, ok := asset.Identity["import"].(string)
	if !ok || imp == "" {
		return nil, nil
	}
	doc, err := parseFile(root, imp)
	if err != nil {
		return nil, err
	}
	props := map[string]any{}
	for _, sec := range doc.Sections {
		if sec.Type == "remap" {
			if s := propStr(sec, "importer"); s != "" {
				props["importer"] = s
			}
			if s := propStr(sec, "type"); s != "" {
				props["type"] = s
			}
		}
	}
	addEdge(p, model.EdgeImports, asset.ID, imp, model.OriginConfig, true,
		model.Evidence{File: imp, Line: 0}, props)
	return doc.Diags, nil
}

// extractExtension records a .gdextension's registration metadata onto its node
// (entry symbol, minimum compatibility). The classes it registers are parsed in
// milestone 99.
func extractExtension(root string, ext *model.Node) ([]Diagnostic, error) {
	doc, err := parseFile(root, ext.Path)
	if err != nil {
		return nil, err
	}
	if ext.Identity == nil {
		ext.Identity = map[string]any{}
	}
	for _, sec := range doc.Sections {
		if sec.Type != "configuration" {
			continue
		}
		if s := propStr(sec, "entry_symbol"); s != "" {
			ext.Identity["entry_symbol"] = s
		}
		if s := propStr(sec, "compatibility_minimum"); s != "" {
			ext.Identity["compatibility_minimum"] = s
		}
	}
	return doc.Diags, nil
}

// scanPlugins finds addons/*/plugin.cfg files and flags the editor-plugin script
// they register (identity editor_plugin=true) when that script is a known node.
func scanPlugins(root string, p *model.Project) ([]Diagnostic, error) {
	var diags []Diagnostic
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".godot" || d.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if d.Name() != "plugin.cfg" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		pluginRes := model.NormalizePath("res://" + filepath.ToSlash(rel))
		doc := Parse(pluginRes, data)
		diags = append(diags, doc.Diags...)

		for _, sec := range doc.Sections {
			if sec.Type != "plugin" {
				continue
			}
			script := propStr(sec, "script")
			if script == "" {
				continue
			}
			// plugin.cfg script paths are relative to the plugin.cfg's own dir.
			dir := filepath.ToSlash(filepath.Dir(rel))
			scriptID := model.NormalizePath("res://" + dir + "/" + script)
			if n, ok := p.Nodes[scriptID]; ok {
				if n.Identity == nil {
					n.Identity = map[string]any{}
				}
				n.Identity["editor_plugin"] = true
			}
		}
		return nil
	})
	return diags, err
}

// extResources builds the id → resource table from a document's [ext_resource]
// headers so ExtResource("id") references resolve to a path.
func extResources(doc *Document) map[string]extResource {
	m := map[string]extResource{}
	for _, sec := range doc.Sections {
		if sec.Type != "ext_resource" {
			continue
		}
		var e extResource
		e.typ = attrStr(sec, "type")
		e.path = attrStr(sec, "path")
		e.uid = attrStr(sec, "uid")
		id := ""
		if v, ok := sec.Attrs["id"]; ok {
			id = v.scalarString()
		}
		m[id] = e
	}
	return m
}

// editablePaths collects the node paths marked [editable path="…"] — property
// overrides into instanced sub-scenes.
func editablePaths(doc *Document) map[string]bool {
	m := map[string]bool{}
	for _, sec := range doc.Sections {
		if sec.Type == "editable" {
			if s := attrStr(sec, "path"); s != "" {
				m[s] = true
			}
		}
	}
	return m
}

// resolveRef maps an ext_resource id to a res:// node ID, preferring the
// declared path and falling back to the uid via the project's UID map.
func resolveRef(p *model.Project, ext map[string]extResource, id string) string {
	e, ok := ext[id]
	if !ok {
		return ""
	}
	if e.path != "" {
		return model.NormalizePath(e.path)
	}
	if e.uid != "" {
		if rp, ok := p.UIDMap[e.uid]; ok {
			return rp
		}
	}
	return ""
}

// assetEdgeType chooses the edge type for an ExtResource property reference by
// its declared Godot type.
func assetEdgeType(typ string) model.EdgeType {
	switch typ {
	case "PackedScene", "Resource":
		return model.EdgeLoadsResource
	case "Shader":
		return model.EdgeUsesShader
	case "Theme":
		return model.EdgeUsesTheme
	}
	if strings.Contains(typ, "Material") {
		return model.EdgeUsesMaterial
	}
	return model.EdgeUsesAsset
}

// nodePathFor computes a node's in-scene path from its parent attribute and
// name, matching the paths Godot uses in [connection] from=/to=: the root node
// is ".", a direct child is its name, deeper nodes are "Parent/Child".
func nodePathFor(parent, name string) string {
	switch parent {
	case "":
		return "."
	case ".":
		return name
	default:
		return parent + "/" + name
	}
}

func addEdge(p *model.Project, typ model.EdgeType, src, tgt string, origin model.Origin, resolved bool, ev model.Evidence, props map[string]any) {
	e := &model.Edge{
		Type:       typ,
		SourceID:   src,
		TargetID:   tgt,
		Origin:     origin,
		Resolved:   resolved,
		Confidence: 1.0,
		Evidence:   ev,
	}
	if len(props) > 0 {
		e.Properties = props
	}
	p.Edges = append(p.Edges, e)
}

func attrStr(sec *Section, key string) string {
	if v, ok := sec.Attrs[key]; ok {
		if s, ok := v.AsString(); ok {
			return s
		}
	}
	return ""
}

func propStr(sec *Section, key string) string {
	if v, ok := sec.Props[key]; ok {
		if s, ok := v.AsString(); ok {
			return s
		}
	}
	return ""
}

// parseFile reads and parses a res://-identified INI file under root. Godot's
// binary resource formats (.scn/.res, and any .tscn/.tres re-saved as binary)
// are not INI text: rather than let the parser silently yield an empty document
// — dropping every instance/script/connection edge with no trace (M1 exit
// criterion: nothing unparseable is lost silently) — parseFile detects the
// binary magic and returns a document carrying a single explanatory diagnostic.
func parseFile(root, resPath string) (*Document, error) {
	data, err := os.ReadFile(fsPath(root, resPath))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", resPath, err)
	}
	if isBinaryGodot(data) {
		return &Document{Diags: []Diagnostic{{
			File: resPath,
			Msg: "binary Godot resource format (RSRC/RSCC); no edges extracted — " +
				"re-save as text (.tscn/.tres) or add Godot-assisted conversion",
		}}}, nil
	}
	return Parse(resPath, data), nil
}

// godotBinaryMagics are the leading bytes of Godot's binary resource formats:
// "RSRC" for an uncompressed binary scene/resource, "RSCC" for a compressed one.
// Text scenes/resources begin with a "[gd_scene…]"/"[gd_resource…]" header, so
// the magic cleanly distinguishes the two regardless of file extension.
var godotBinaryMagics = [][]byte{[]byte("RSRC"), []byte("RSCC")}

// isBinaryGodot reports whether data begins with a Godot binary-resource magic.
func isBinaryGodot(data []byte) bool {
	for _, magic := range godotBinaryMagics {
		if bytes.HasPrefix(data, magic) {
			return true
		}
	}
	return false
}

// fsPath converts a res:// node ID back to its filesystem path under root.
func fsPath(root, resID string) string {
	rel := strings.TrimPrefix(resID, "res://")
	return filepath.Join(root, filepath.FromSlash(rel))
}
