package config

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// FileName is the per-project config file discovery reads from the project root.
const FileName = "godarch.yml"

// Config is the parsed godarch.yml. Only the milestone-01 fields are modelled;
// unknown keys are ignored so the schema can grow without breaking older files.
type Config struct {
	// Ignore lists globs, in addition to the always-ignored engine/VCS dirs,
	// that discovery skips. A glob without a "/" is matched against every path
	// segment (so "addons" prunes the whole addons/ tree); a glob with a "/" is
	// matched against the full project-root-relative slash path.
	Ignore []string `yaml:"ignore"`
}

// alwaysIgnore names the directories discovery never descends into regardless of
// configuration: the regenerated engine cache and the VCS metadata dir.
var alwaysIgnore = map[string]bool{".godot": true, ".git": true}

// Load reads <root>/godarch.yml. A missing file is not an error — discovery runs
// with defaults — so Load returns the zero Config in that case.
func Load(root string) (Config, error) {
	data, err := os.ReadFile(filepath.Join(root, FileName))
	if errors.Is(err, fs.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}

	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return Config{}, err
	}
	return c, nil
}

// IsIgnored reports whether the project-root-relative slash path should be
// excluded from discovery.
func (c Config) IsIgnored(relPath string) bool {
	segs := strings.Split(relPath, "/")
	for _, s := range segs {
		if alwaysIgnore[s] {
			return true
		}
	}
	for _, g := range c.Ignore {
		if strings.Contains(g, "/") {
			if match(g, relPath) {
				return true
			}
			continue
		}
		for _, s := range segs {
			if match(g, s) {
				return true
			}
		}
	}
	return false
}

// match is path.Match with the error swallowed: a malformed glob simply never
// matches rather than failing discovery.
func match(glob, name string) bool {
	ok, err := path.Match(glob, name)
	return err == nil && ok
}
