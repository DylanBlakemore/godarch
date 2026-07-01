package discovery

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/dylanblakemore/godarch/internal/model"
)

// autoload is one singleton registered in project.godot's [autoload] section.
type autoload struct {
	name    string
	path    string // res:// path of the script/scene, with the leading "*" stripped
	enabled bool   // a leading "*" marks the singleton as enabled
}

// layerName is one [layer_names] entry: a category (e.g. "2d_physics",
// "3d_render"), the 1-based layer index, and the user-assigned name.
type layerName struct {
	category string
	index    int
	name     string
}

// projectConfig holds the slice of project.godot that discovery understands: the
// engine version, the main scene, autoload singletons, input-action names,
// physics/render layer names, and predeclared global groups.
type projectConfig struct {
	godotVersion string
	mainScene    string
	autoloads    []autoload
	actions      []string
	layers       []layerName
	groups       []string
}

// keyValRe matches a section key assignment at column 0 ("name=..."). Anchoring
// to the line start excludes the indented/quoted continuation lines of Godot's
// multi-line dictionary values (e.g. the body of an [input] action). The leading
// character allows a digit so layer keys like "2d_physics/layer_1" match;
// continuation lines start with a quote or bracket and so still do not.
var keyValRe = regexp.MustCompile(`^([A-Za-z0-9_][A-Za-z0-9_/]*)\s*=(.*)$`)

// layerKeyRe splits a [layer_names] key into its category and 1-based index,
// e.g. "2d_physics/layer_3" → ("2d_physics", 3).
var layerKeyRe = regexp.MustCompile(`^(\w+)/layer_(\d+)$`)

// versionTokenRe finds a Godot version token like "4.2" inside config/features.
var versionTokenRe = regexp.MustCompile(`"(\d+\.\d+(?:\.\d+)?)"`)

// loadProjectConfig parses <dir>/project.godot. A missing file is not an error —
// discovery can still classify files — so it returns a zero config in that case.
func loadProjectConfig(dir string) (projectConfig, error) {
	f, err := os.Open(filepath.Join(dir, "project.godot"))
	if errors.Is(err, fs.ErrNotExist) {
		return projectConfig{}, nil
	}
	if err != nil {
		return projectConfig{}, fmt.Errorf("open project.godot: %w", err)
	}
	defer func() { _ = f.Close() }()

	cfg := projectConfig{}
	section := ""
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, ";") {
			continue
		}
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			section = strings.TrimSpace(trimmed[1 : len(trimmed)-1])
			continue
		}

		m := keyValRe.FindStringSubmatch(line)
		if m == nil {
			continue // continuation line of a multi-line value
		}
		key, val := m[1], strings.TrimSpace(m[2])

		// config/features carries the engine version and run/main_scene the
		// scene-flow root; both live under [application] but are matched by key
		// so section drift won't hide them.
		switch key {
		case "config/features":
			if t := versionTokenRe.FindStringSubmatch(val); t != nil {
				cfg.godotVersion = t[1]
			}
			continue
		case "run/main_scene":
			cfg.mainScene = model.NormalizePath(strings.Trim(val, `"`))
			continue
		}

		switch section {
		case "autoload":
			cfg.autoloads = append(cfg.autoloads, parseAutoload(key, val))
		case "input":
			cfg.actions = append(cfg.actions, key)
		case "layer_names":
			if m := layerKeyRe.FindStringSubmatch(key); m != nil {
				idx, err := strconv.Atoi(m[2])
				if err != nil {
					continue
				}
				cfg.layers = append(cfg.layers, layerName{
					category: m[1],
					index:    idx,
					name:     strings.Trim(val, `"`),
				})
			}
		case "global_group":
			cfg.groups = append(cfg.groups, key)
		}
	}
	if err := scanner.Err(); err != nil {
		return projectConfig{}, fmt.Errorf("read project.godot: %w", err)
	}
	return cfg, nil
}

// parseAutoload turns an [autoload] entry (Name="*res://path.gd") into an
// autoload: the leading "*" marks the singleton enabled, and the path is
// normalised so it matches the corresponding script/scene node ID.
func parseAutoload(name, val string) autoload {
	val = strings.Trim(val, `"`)
	enabled := strings.HasPrefix(val, "*")
	val = strings.TrimPrefix(val, "*")
	return autoload{
		name:    name,
		path:    model.NormalizePath(val),
		enabled: enabled,
	}
}
