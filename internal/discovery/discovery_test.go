package discovery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dylanblakemore/godarch/internal/model"
)

// writeProject lays out a synthetic project tree under a temp dir and returns
// the root. Each entry maps a slash-separated relative path to file contents.
func writeProject(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", rel, err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	return root
}

const sampleProjectGodot = `config_version=5

[application]

config/name="Sample"
run/main_scene="res://main.tscn"
config/features=PackedStringArray("4.2", "Forward Plus")

[autoload]

GameState="*res://game_state.gd"
Disabled="res://other.gd"

[input]

jump={
"deadzone": 0.5,
"events": []
}
`

func TestDiscoverClassifiesAndParses(t *testing.T) {
	root := writeProject(t, map[string]string{
		"project.godot":        sampleProjectGodot,
		"main.tscn":            "[gd_scene format=3]\n",
		"player/player.gd":     "extends Node\n",
		"game_state.gd":        "extends Node\n",
		"other.gd":             "extends Node\n",
		"ui/theme.tres":        "[gd_resource type=\"Theme\"]\n",
		"art/icon.svg":         "<svg></svg>\n",
		"art/icon.svg.import":  "[remap]\n",         // sidecar: must be ignored
		".godot/uid_cache.bin": "binary",            // engine cache dir: must be ignored
		"README.md":            "# docs",            // unknown: must be ignored
		"bin/ext.gdextension":  "[configuration]\n", // gdextension
	})

	p, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	if got, want := p.GodotVersion, "4.2"; got != want {
		t.Errorf("GodotVersion = %q, want %q", got, want)
	}

	want := map[string]model.Kind{
		"res://main.tscn":           model.KindScene,
		"res://player/player.gd":    model.KindScript,
		"res://game_state.gd":       model.KindScript,
		"res://other.gd":            model.KindScript,
		"res://ui/theme.tres":       model.KindResource,
		"res://art/icon.svg":        model.KindAsset,
		"res://bin/ext.gdextension": model.KindExtension,
		"autoload:GameState":        model.KindAutoload,
		"autoload:Disabled":         model.KindAutoload,
		"action:jump":               model.KindAction,
	}

	for id, kind := range want {
		n, ok := p.Nodes[id]
		if !ok {
			t.Errorf("missing node %q", id)
			continue
		}
		if n.Kind != kind {
			t.Errorf("node %q kind = %q, want %q", id, n.Kind, kind)
		}
	}

	// Ignored entries must not appear as nodes.
	for _, id := range []string{
		"res://art/icon.svg.import",
		"res://README.md",
		"res://project.godot",
		"res://.godot/uid_cache.bin",
	} {
		if _, ok := p.Nodes[id]; ok {
			t.Errorf("node %q should have been ignored", id)
		}
	}

	if len(p.Nodes) != len(want) {
		t.Errorf("node count = %d, want %d; nodes: %v", len(p.Nodes), len(want), nodeIDs(p))
	}

	// Autoload identity carries the singleton name and target path.
	gs := p.Nodes["autoload:GameState"]
	if gs.Identity["name"] != "GameState" {
		t.Errorf("autoload name = %v, want GameState", gs.Identity["name"])
	}
	if gs.Identity["path"] != "res://game_state.gd" {
		t.Errorf("autoload path = %v, want res://game_state.gd", gs.Identity["path"])
	}
	if gs.Identity["enabled"] != true {
		t.Errorf("GameState enabled = %v, want true", gs.Identity["enabled"])
	}
	if p.Nodes["autoload:Disabled"].Identity["enabled"] != false {
		t.Errorf("Disabled enabled = %v, want false", p.Nodes["autoload:Disabled"].Identity["enabled"])
	}

	// The main scene from application/run/main_scene is identified and the
	// scene node carries the marker.
	if p.MainScene != "res://main.tscn" {
		t.Errorf("MainScene = %q, want res://main.tscn", p.MainScene)
	}
	if p.Nodes["res://main.tscn"].Identity["main_scene"] != true {
		t.Errorf("main.tscn main_scene marker = %v, want true",
			p.Nodes["res://main.tscn"].Identity["main_scene"])
	}
}

// TestDiscoverFindsRootFromSubdir checks that Discover ascends to the directory
// holding project.godot when handed a subdirectory of the project.
func TestDiscoverFindsRootFromSubdir(t *testing.T) {
	root := writeProject(t, map[string]string{
		"project.godot":    sampleProjectGodot,
		"main.tscn":        "[gd_scene format=3]\n",
		"world/level.gd":   "extends Node\n",
		"world/sub/x.tres": "[gd_resource]\n",
	})

	p, err := Discover(filepath.Join(root, "world", "sub"))
	if err != nil {
		t.Fatalf("Discover(subdir): %v", err)
	}

	for _, id := range []string{"res://main.tscn", "res://world/level.gd", "res://world/sub/x.tres"} {
		if _, ok := p.Nodes[id]; !ok {
			t.Errorf("missing node %q (root not resolved from subdir?)", id)
		}
	}
}

// TestDiscoverIgnoreGlobs checks that godarch.yml ignore globs prune files, on
// top of the always-ignored .godot/.git dirs.
func TestDiscoverIgnoreGlobs(t *testing.T) {
	root := writeProject(t, map[string]string{
		"project.godot":            sampleProjectGodot,
		"godarch.yml":              "ignore:\n  - addons\n",
		"main.tscn":                "[gd_scene format=3]\n",
		"addons/plugin/tool.gd":    "extends Node\n",
		".git/hooks/pre-commit.gd": "extends Node\n",
	})

	p, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	if _, ok := p.Nodes["res://addons/plugin/tool.gd"]; ok {
		t.Errorf("addons file should have been ignored by godarch.yml glob")
	}
	if _, ok := p.Nodes["res://.git/hooks/pre-commit.gd"]; ok {
		t.Errorf(".git file should always be ignored")
	}
	if _, ok := p.Nodes["res://main.tscn"]; !ok {
		t.Errorf("main.tscn should still be discovered")
	}
}

// TestDiscoverParsesLayersGroupsMainScene checks the project.godot sections that
// milestone-01 discovery added: [layer_names], [global_group], and the main
// scene marker.
func TestDiscoverParsesLayersGroupsMainScene(t *testing.T) {
	const godot = `config_version=5

[application]

run/main_scene="res://main.tscn"

[layer_names]

2d_physics/layer_1="Player"
2d_physics/layer_2="Enemy"
3d_render/layer_1="World"

[global_group]

enemies="The bad guys"
`
	root := writeProject(t, map[string]string{
		"project.godot": godot,
		"main.tscn":     "[gd_scene format=3]\n",
	})

	p, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	group := p.Nodes["group:enemies"]
	if group == nil || group.Kind != model.KindGroup {
		t.Fatalf("group:enemies node = %+v, want a group node", group)
	}

	l1 := p.Nodes["layer:1"]
	if l1 == nil || l1.Kind != model.KindLayer {
		t.Fatalf("layer:1 node = %+v, want a layer node", l1)
	}
	names, ok := l1.Identity["names"].(map[string]any)
	if !ok {
		t.Fatalf("layer:1 names = %v, want a category→name map", l1.Identity["names"])
	}
	if names["2d_physics"] != "Player" || names["3d_render"] != "World" {
		t.Errorf("layer:1 names = %v, want {2d_physics:Player, 3d_render:World}", names)
	}
	if p.Nodes["layer:2"] == nil {
		t.Errorf("missing layer:2 node")
	}
}

// TestDiscoverBuildsUIDMap checks that uids declared in scene headers and in
// .import sidecars are collected into Project.UIDMap.
func TestDiscoverBuildsUIDMap(t *testing.T) {
	root := writeProject(t, map[string]string{
		"project.godot": "config_version=5\n",
		"main.tscn":     "[gd_scene format=3 uid=\"uid://scene123\"]\n",
		"art/icon.png":  "PNG",
		"art/icon.png.import": "[remap]\n\nuid=\"uid://asset456\"\n\n" +
			"[deps]\n\nsource_file=\"res://art/icon.png\"\n",
	})

	p, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	if got := p.UIDMap["uid://scene123"]; got != "res://main.tscn" {
		t.Errorf("UIDMap[uid://scene123] = %q, want res://main.tscn", got)
	}
	if got := p.UIDMap["uid://asset456"]; got != "res://art/icon.png" {
		t.Errorf("UIDMap[uid://asset456] = %q, want res://art/icon.png", got)
	}
}

// TestDiscoverPairsImports checks that an asset with a .import sidecar records
// the sidecar path on its node identity (the basis for the M1.02 imports edge).
func TestDiscoverPairsImports(t *testing.T) {
	root := writeProject(t, map[string]string{
		"project.godot":       "config_version=5\n",
		"art/icon.png":        "PNG",
		"art/icon.png.import": "[remap]\n",
	})

	p, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	asset := p.Nodes["res://art/icon.png"]
	if asset == nil {
		t.Fatalf("missing asset node res://art/icon.png")
	}
	if asset.Identity["import"] != "res://art/icon.png.import" {
		t.Errorf("asset import pairing = %v, want res://art/icon.png.import",
			asset.Identity["import"])
	}
	if _, ok := p.Nodes["res://art/icon.png.import"]; ok {
		t.Errorf(".import sidecar should not be a node")
	}
}

func TestCounts(t *testing.T) {
	p := model.NewProject("res://")
	add := func(id string, k model.Kind) { p.Nodes[id] = &model.Node{ID: id, Kind: k} }
	add("res://a.gd", model.KindScript)
	add("res://b.gd", model.KindScript)
	add("res://m.tscn", model.KindScene)
	add("res://t.tres", model.KindResource)
	add("res://i.png", model.KindAsset)
	add("autoload:G", model.KindAutoload)

	c := Counts(p)
	for kind, want := range map[model.Kind]int{
		model.KindScript:   2,
		model.KindScene:    1,
		model.KindResource: 1,
		model.KindAsset:    1,
		model.KindAutoload: 1,
	} {
		if c[kind] != want {
			t.Errorf("Counts[%s] = %d, want %d", kind, c[kind], want)
		}
	}
}

func nodeIDs(p *model.Project) []string {
	ids := make([]string, 0, len(p.Nodes))
	for id := range p.Nodes {
		ids = append(ids, id)
	}
	return ids
}
