package model_test

import (
	"testing"

	"github.com/dylanblakemore/godarch/internal/model"
)

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"scheme-less becomes res", "player/player.gd", "res://player/player.gd"},
		{"already res is preserved", "res://player/player.gd", "res://player/player.gd"},
		{"collapses double slashes", "res://ui//hud.gd", "res://ui/hud.gd"},
		{"resolves dot segments", "./player/../player/player.gd", "res://player/player.gd"},
		{"trims surrounding space", "  res://a.gd  ", "res://a.gd"},
		{"empty stays empty", "", ""},
		{"user scheme preserved", "user://save.cfg", "user://save.cfg"},
		{"uid left untouched", "uid://abc123", "uid://abc123"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := model.NormalizePath(tt.in); got != tt.want {
				t.Errorf("NormalizePath(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestFileIDConstructors(t *testing.T) {
	// Script/scene/resource/asset IDs are all the normalized res:// path.
	const raw = "player/player.gd"
	const want = "res://player/player.gd"
	for name, got := range map[string]string{
		"ScriptID":   model.ScriptID(raw),
		"SceneID":    model.SceneID(raw),
		"ResourceID": model.ResourceID(raw),
		"AssetID":    model.AssetID(raw),
		"FileID":     model.FileID(raw),
	} {
		if got != want {
			t.Errorf("%s(%q) = %q, want %q", name, raw, got, want)
		}
	}
}

func TestConceptIDConstructors(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"SymbolID", model.SymbolID("player/player.gd", "take_damage"), "res://player/player.gd::take_damage"},
		{"SignalDeclID", model.SignalDeclID("player/player.gd", "died"), "res://player/player.gd::signal:died"},
		{"SceneNodeID", model.SceneNodeID("ui/hud.tscn", "HBox/HealthBar"), "res://ui/hud.tscn::HBox/HealthBar"},
		{"AutoloadID", model.AutoloadID("GameState"), "autoload:GameState"},
		{"ActionID", model.ActionID("jump"), "action:jump"},
		{"GroupID", model.GroupID("enemies"), "group:enemies"},
		{"LayerID", model.LayerID(3), "layer:3"},
		{"ClassID", model.ClassID("Player"), "class:Player"},
		{"ExtensionID", model.ExtensionID("bin/my_ext.gdextension"), "ext:res://bin/my_ext.gdextension"},
		{"DocID", model.DocID("docs/systems/combat.md"), "docs/systems/combat.md"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.want)
			}
		})
	}
}

func TestParseSymbolID(t *testing.T) {
	file, sym, ok := model.ParseSymbolID("res://player/player.gd::take_damage")
	if !ok {
		t.Fatal("ParseSymbolID returned ok=false for a valid symbol ID")
	}
	if file != "res://player/player.gd" || sym != "take_damage" {
		t.Errorf("ParseSymbolID = (%q, %q), want (res://player/player.gd, take_damage)", file, sym)
	}

	// A signal-decl ID is file-scoped but not a plain symbol.
	if _, _, ok := model.ParseSymbolID("res://player/player.gd::signal:died"); ok {
		t.Error("ParseSymbolID accepted a signal-decl ID; want ok=false")
	}
	// A scene-node ID is file-scoped on a scene, not a script symbol.
	if _, _, ok := model.ParseSymbolID("res://ui/hud.tscn::HBox/HealthBar"); ok {
		t.Error("ParseSymbolID accepted a scene-node ID; want ok=false")
	}
	if _, _, ok := model.ParseSymbolID("autoload:GameState"); ok {
		t.Error("ParseSymbolID accepted a concept ID; want ok=false")
	}
}

func TestParseSignalDeclID(t *testing.T) {
	file, name, ok := model.ParseSignalDeclID("res://player/player.gd::signal:died")
	if !ok {
		t.Fatal("ParseSignalDeclID returned ok=false for a valid signal ID")
	}
	if file != "res://player/player.gd" || name != "died" {
		t.Errorf("ParseSignalDeclID = (%q, %q), want (res://player/player.gd, died)", file, name)
	}
	if _, _, ok := model.ParseSignalDeclID("res://player/player.gd::take_damage"); ok {
		t.Error("ParseSignalDeclID accepted a plain symbol ID; want ok=false")
	}
}

func TestParseSceneNodeID(t *testing.T) {
	scene, np, ok := model.ParseSceneNodeID("res://ui/hud.tscn::HBox/HealthBar")
	if !ok {
		t.Fatal("ParseSceneNodeID returned ok=false for a valid scene-node ID")
	}
	if scene != "res://ui/hud.tscn" || np != "HBox/HealthBar" {
		t.Errorf("ParseSceneNodeID = (%q, %q), want (res://ui/hud.tscn, HBox/HealthBar)", scene, np)
	}
	if _, _, ok := model.ParseSceneNodeID("res://player/player.gd::take_damage"); ok {
		t.Error("ParseSceneNodeID accepted a script symbol ID; want ok=false")
	}
}

func TestIDRoundTrips(t *testing.T) {
	t.Run("symbol", func(t *testing.T) {
		id := model.SymbolID("player/player.gd", "take_damage")
		file, sym, ok := model.ParseSymbolID(id)
		if !ok || file != "res://player/player.gd" || sym != "take_damage" {
			t.Errorf("round trip failed: %q -> (%q, %q, %v)", id, file, sym, ok)
		}
	})
	t.Run("signal", func(t *testing.T) {
		id := model.SignalDeclID("player/player.gd", "died")
		file, name, ok := model.ParseSignalDeclID(id)
		if !ok || file != "res://player/player.gd" || name != "died" {
			t.Errorf("round trip failed: %q -> (%q, %q, %v)", id, file, name, ok)
		}
	})
	t.Run("scene_node", func(t *testing.T) {
		id := model.SceneNodeID("ui/hud.tscn", "HBox/HealthBar")
		scene, np, ok := model.ParseSceneNodeID(id)
		if !ok || scene != "res://ui/hud.tscn" || np != "HBox/HealthBar" {
			t.Errorf("round trip failed: %q -> (%q, %q, %v)", id, scene, np, ok)
		}
	})
}
