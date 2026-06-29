package model_test

import (
	"os"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/dylanblakemore/godarch/internal/model"
)

func TestMatchKeyConstructors(t *testing.T) {
	tests := []struct {
		name string
		got  model.MatchKey
		want model.MatchKey
	}{
		{"signal with emitter", model.SignalKey("Player", "died"), "signal:Player:died"},
		{"signal wildcard emitter", model.SignalKey("", "ready"), "signal:*:ready"},
		{"signal trims space", model.SignalKey("  Player  ", "  died  "), "signal:Player:died"},
		{"resource scheme-less", model.ResourceKey("player/player.tscn"), "res:res://player/player.tscn"},
		{"resource canonicalised", model.ResourceKey("res://ui//hud.tscn"), "res:res://ui/hud.tscn"},
		{"resource dot segments", model.ResourceKey("./a/../b.tres"), "res:res://b.tres"},
		{"action", model.ActionKey("jump"), "action:jump"},
		{"group", model.GroupKey("enemies"), "group:enemies"},
		{"autoload preserves case", model.AutoloadKey("GameState"), "autoload:GameState"},
		{"rpc with class", model.RPCKey("Net", "spawn"), "rpc:Net:spawn"},
		{"rpc wildcard class", model.RPCKey("", "ping"), "rpc:*:ping"},
		{"nodepath fallback", model.NodePathKey("../Enemy/Sprite"), "nodepath:../Enemy/Sprite"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.want)
			}
		})
	}
}

// matchKeyFixtures mirrors testdata/matchkey_fixtures.yml: the archi-style
// ground truth that locks expected match-key output. Editing the YAML changes
// the contract; this test enforces it.
type matchKeyFixtures struct {
	Signal []struct {
		Emitter string `yaml:"emitter"`
		Name    string `yaml:"name"`
		Want    string `yaml:"want"`
	} `yaml:"signal"`
	Resource []struct {
		Path string `yaml:"path"`
		Want string `yaml:"want"`
	} `yaml:"resource"`
	Action []struct {
		Name string `yaml:"name"`
		Want string `yaml:"want"`
	} `yaml:"action"`
	Group []struct {
		Name string `yaml:"name"`
		Want string `yaml:"want"`
	} `yaml:"group"`
	Autoload []struct {
		Name string `yaml:"name"`
		Want string `yaml:"want"`
	} `yaml:"autoload"`
	RPC []struct {
		Class  string `yaml:"class"`
		Method string `yaml:"method"`
		Want   string `yaml:"want"`
	} `yaml:"rpc"`
	NodePath []struct {
		Expr string `yaml:"expr"`
		Want string `yaml:"want"`
	} `yaml:"nodepath"`
}

func TestMatchKeyFixtures(t *testing.T) {
	raw, err := os.ReadFile("testdata/matchkey_fixtures.yml")
	if err != nil {
		t.Fatalf("reading fixtures: %v", err)
	}
	var fx matchKeyFixtures
	if err := yaml.Unmarshal(raw, &fx); err != nil {
		t.Fatalf("parsing fixtures: %v", err)
	}

	total := 0
	for _, c := range fx.Signal {
		if got := string(model.SignalKey(c.Emitter, c.Name)); got != c.Want {
			t.Errorf("SignalKey(%q,%q) = %q, want %q", c.Emitter, c.Name, got, c.Want)
		}
		total++
	}
	for _, c := range fx.Resource {
		if got := string(model.ResourceKey(c.Path)); got != c.Want {
			t.Errorf("ResourceKey(%q) = %q, want %q", c.Path, got, c.Want)
		}
		total++
	}
	for _, c := range fx.Action {
		if got := string(model.ActionKey(c.Name)); got != c.Want {
			t.Errorf("ActionKey(%q) = %q, want %q", c.Name, got, c.Want)
		}
		total++
	}
	for _, c := range fx.Group {
		if got := string(model.GroupKey(c.Name)); got != c.Want {
			t.Errorf("GroupKey(%q) = %q, want %q", c.Name, got, c.Want)
		}
		total++
	}
	for _, c := range fx.Autoload {
		if got := string(model.AutoloadKey(c.Name)); got != c.Want {
			t.Errorf("AutoloadKey(%q) = %q, want %q", c.Name, got, c.Want)
		}
		total++
	}
	for _, c := range fx.RPC {
		if got := string(model.RPCKey(c.Class, c.Method)); got != c.Want {
			t.Errorf("RPCKey(%q,%q) = %q, want %q", c.Class, c.Method, got, c.Want)
		}
		total++
	}
	for _, c := range fx.NodePath {
		if got := string(model.NodePathKey(c.Expr)); got != c.Want {
			t.Errorf("NodePathKey(%q) = %q, want %q", c.Expr, got, c.Want)
		}
		total++
	}

	if total == 0 {
		t.Fatal("fixtures file produced no cases; expected archi-style ground truth")
	}
}
