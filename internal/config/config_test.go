package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dylanblakemore/godarch/internal/config"
)

func TestLoadMissingFileYieldsZeroConfig(t *testing.T) {
	c, err := config.Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(c.Ignore) != 0 {
		t.Errorf("Ignore = %v, want empty", c.Ignore)
	}
}

func TestLoadReadsIgnoreGlobs(t *testing.T) {
	root := t.TempDir()
	body := "ignore:\n  - addons\n  - build/*\n  - \"*.tmp\"\n"
	if err := os.WriteFile(filepath.Join(root, config.FileName), []byte(body), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	c, err := config.Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := []string{"addons", "build/*", "*.tmp"}
	if len(c.Ignore) != len(want) {
		t.Fatalf("Ignore = %v, want %v", c.Ignore, want)
	}
	for i, g := range want {
		if c.Ignore[i] != g {
			t.Errorf("Ignore[%d] = %q, want %q", i, c.Ignore[i], g)
		}
	}
}

func TestIsIgnored(t *testing.T) {
	c := config.Config{Ignore: []string{"addons", "build/*", "*.tmp"}}

	cases := map[string]bool{
		".godot":            true, // always-ignored engine cache
		".git":              true, // always-ignored VCS dir
		"addons":            true, // configured dir glob (base match)
		"addons/foo/bar.gd": true, // base segment of the path matches "addons"
		"build/out.pck":     true, // build/* matches
		"scratch.tmp":       true, // *.tmp matches base name
		"src/player.gd":     false,
		"main.tscn":         false,
		"build":             false, // "build/*" needs a child; the dir itself is kept
	}
	for relPath, want := range cases {
		if got := c.IsIgnored(relPath); got != want {
			t.Errorf("IsIgnored(%q) = %v, want %v", relPath, got, want)
		}
	}
}
