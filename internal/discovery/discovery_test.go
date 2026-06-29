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
