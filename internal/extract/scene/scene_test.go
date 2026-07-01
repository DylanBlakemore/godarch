package scene_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dylanblakemore/godarch/internal/extract/scene"
	"github.com/dylanblakemore/godarch/internal/model"
)

// runExtract writes files into a temp Godot project, runs discovery, then the
// scene extractor, and returns the populated project. Every case supplies its
// own project.godot so discovery can locate the root.
func runExtract(t *testing.T, files map[string]string) *model.Project {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		full := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	p := discoverOrFail(t, dir)
	if _, err := scene.Extract(dir, p); err != nil {
		t.Fatalf("Extract: %v", err)
	}
	return p
}

func hasEdge(p *model.Project, typ model.EdgeType, src, tgt string) *model.Edge {
	for _, e := range p.Edges {
		if e.Type == typ && e.SourceID == src && e.TargetID == tgt {
			return e
		}
	}
	return nil
}

func TestExtractSceneEdges(t *testing.T) {
	files := map[string]string{
		"project.godot": "config_version=5\n",
		"player.gd":     "extends Node\nfunc _on_body_entered(): pass\n",
		"enemy.tscn":    "[gd_scene format=3]\n[node name=\"Enemy\" type=\"Node2D\"]\n",
		"icon.png":      "\x89PNG\r\n",
		"main.tscn": `[gd_scene load_steps=4 format=3 uid="uid://main"]

[ext_resource type="Script" path="res://player.gd" id="1_p"]
[ext_resource type="PackedScene" path="res://enemy.tscn" id="2_e"]
[ext_resource type="Texture2D" path="res://icon.png" id="3_t"]

[node name="Main" type="Node2D"]
script = ExtResource("1_p")

[node name="Sprite" type="Sprite2D" parent="." groups=["enemies", "hostile"]]
texture = ExtResource("3_t")
target = NodePath("../Main")

[node name="Enemy" parent="." instance=ExtResource("2_e")]

[connection signal="body_entered" from="Sprite" to="." method="_on_body_entered"]
`,
	}
	p := runExtract(t, files)

	main := "res://main.tscn"
	// scene_node nodes for the tree.
	if n := p.Nodes[model.SceneNodeID(main, ".")]; n == nil || n.Kind != model.KindSceneNode {
		t.Fatalf("root scene_node missing")
	} else if n.Identity["node_type"] != "Node2D" || n.Identity["root"] != true {
		t.Errorf("root identity = %+v", n.Identity)
	}

	if hasEdge(p, model.EdgeAttachesScript, model.SceneNodeID(main, "."), "res://player.gd") == nil {
		t.Error("missing attaches_script for root")
	}
	if hasEdge(p, model.EdgeInstances, main, "res://enemy.tscn") == nil {
		t.Error("missing instances edge")
	}
	if hasEdge(p, model.EdgeUsesAsset, main, "res://icon.png") == nil {
		t.Error("missing uses_asset edge")
	}
	if hasEdge(p, model.EdgeInGroup, model.SceneNodeID(main, "Sprite"), model.GroupID("enemies")) == nil {
		t.Error("missing in_group edge")
	}
	ref := hasEdge(p, model.EdgeReferencesNode, model.SceneNodeID(main, "Sprite"), string(model.NodePathKey("../Main")))
	if ref == nil {
		t.Fatal("missing references_node edge")
	}
	if ref.Resolved {
		t.Error("references_node should be unresolved in M1")
	}

	// connects_signal: emitter Sprite -> handler player.gd::_on_body_entered
	cs := hasEdge(p, model.EdgeConnectsSignal, model.SceneNodeID(main, "Sprite"), "res://player.gd::_on_body_entered")
	if cs == nil {
		t.Fatal("missing connects_signal edge")
	}
	if cs.Origin != model.OriginEditor {
		t.Errorf("connects_signal origin = %v", cs.Origin)
	}
	if cs.Properties["signal"] != "body_entered" {
		t.Errorf("connects_signal props = %+v", cs.Properties)
	}

	// editor_connection ingress boundary on the handler, keyed by the signal.
	var found bool
	for _, b := range p.Boundaries {
		if b.Type == model.BoundaryEditorConnection && b.NodeID == "res://player.gd::_on_body_entered" {
			found = true
			if b.MatchKey != model.SignalKey("Sprite2D", "body_entered") {
				t.Errorf("boundary match key = %q", b.MatchKey)
			}
		}
	}
	if !found {
		t.Error("missing editor_connection boundary")
	}
}

func TestExtractResourceBackingScript(t *testing.T) {
	files := map[string]string{
		"project.godot": "config_version=5\n",
		"item.gd":       "extends Resource\n",
		"sword.tres": `[gd_resource type="Resource" script_class="Item" load_steps=2 format=3]

[ext_resource type="Script" path="res://item.gd" id="1_i"]

[resource]
script = ExtResource("1_i")
`,
	}
	p := runExtract(t, files)
	if hasEdge(p, model.EdgeAttachesScript, "res://sword.tres", "res://item.gd") == nil {
		t.Error("missing resource backing-script edge")
	}
}

func TestExtractImportEdge(t *testing.T) {
	files := map[string]string{
		"project.godot": "config_version=5\n",
		"art/icon.png":  "\x89PNG\r\n",
		"art/icon.png.import": `[remap]

importer="texture"
type="CompressedTexture2D"

[deps]

source_file="res://art/icon.png"
`,
	}
	p := runExtract(t, files)
	e := hasEdge(p, model.EdgeImports, "res://art/icon.png", "res://art/icon.png.import")
	if e == nil {
		t.Fatal("missing imports edge")
	}
	if e.Origin != model.OriginConfig || e.Properties["importer"] != "texture" {
		t.Errorf("imports edge = %+v props=%+v", e, e.Properties)
	}
}

func TestExtractGDExtension(t *testing.T) {
	files := map[string]string{
		"project.godot": "config_version=5\n",
		"native.gdextension": `[configuration]

entry_symbol="my_init"
compatibility_minimum="4.2"
`,
	}
	p := runExtract(t, files)
	n := p.Nodes["res://native.gdextension"]
	if n == nil {
		t.Fatal("gdextension node missing")
	}
	if n.Identity["entry_symbol"] != "my_init" {
		t.Errorf("gdextension identity = %+v", n.Identity)
	}
}

func TestExtractPluginCfg(t *testing.T) {
	files := map[string]string{
		"project.godot":         "config_version=5\n",
		"addons/foo/plugin.cfg": "[plugin]\n\nname=\"Foo\"\nscript=\"plugin.gd\"\n",
		"addons/foo/plugin.gd":  "@tool\nextends EditorPlugin\n",
	}
	// addons is not ignored by default config, so the plugin script is a node.
	p := runExtract(t, files)
	n := p.Nodes["res://addons/foo/plugin.gd"]
	if n == nil {
		t.Fatal("plugin script node missing")
	}
	if n.Identity["editor_plugin"] != true {
		t.Errorf("plugin script not flagged as editor_plugin: %+v", n.Identity)
	}
}
