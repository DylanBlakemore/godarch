package discovery

import (
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/dylanblakemore/godarch/internal/model"
)

// Discover walks the Godot project rooted at dir, classifies its files into
// model nodes by extension, and parses project.godot for autoloads, input
// actions, and the engine version.
//
// In milestone 00 discovery emits nodes only: edges and boundary points are the
// job of the M1 extractors, so the returned Project's Edges/Boundaries are
// empty. File paths are normalised to res:// IDs; engine sidecars (.import) and
// the .godot cache directory are skipped.
func Discover(dir string) (*model.Project, error) {
	p := model.NewProject("res://")

	cfg, err := loadProjectConfig(dir)
	if err != nil {
		return nil, err
	}
	p.GodotVersion = cfg.godotVersion

	if err := walkFiles(dir, p); err != nil {
		return nil, err
	}
	addConfigNodes(p, cfg)
	return p, nil
}

// walkFiles classifies every regular file under dir into a node, skipping the
// .godot cache directory, .import sidecars, project.godot itself, and any file
// whose extension does not map to a known Kind.
func walkFiles(dir string, p *model.Project) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != dir && skipDir(d.Name()) {
				return fs.SkipDir
			}
			return nil
		}

		name := d.Name()
		if name == "project.godot" || strings.HasSuffix(name, ".import") {
			return nil
		}

		kind, ok := classify(name)
		if !ok {
			return nil
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		resPath := model.NormalizePath("res://" + filepath.ToSlash(rel))
		p.Nodes[resPath] = &model.Node{ID: resPath, Kind: kind, Path: resPath}
		return nil
	})
}

// addConfigNodes adds the concept nodes that come from project.godot rather than
// from files on disk: autoload singletons and input actions.
func addConfigNodes(p *model.Project, cfg projectConfig) {
	for _, a := range cfg.autoloads {
		id := model.AutoloadID(a.name)
		p.Nodes[id] = &model.Node{
			ID:   id,
			Kind: model.KindAutoload,
			Identity: map[string]any{
				"name":    a.name,
				"path":    a.path,
				"enabled": a.enabled,
			},
		}
	}
	for _, name := range cfg.actions {
		id := model.ActionID(name)
		p.Nodes[id] = &model.Node{
			ID:       id,
			Kind:     model.KindAction,
			Identity: map[string]any{"name": name},
		}
	}
}

// skipDir reports whether a directory should not be descended into.
func skipDir(name string) bool {
	switch name {
	case ".godot", ".git":
		return true
	default:
		return false
	}
}

// assetExts is the set of extensions discovery treats as imported assets
// (textures, audio, models, fonts, shaders, themes, …). Kept explicit so an
// unknown extension is ignored rather than silently miscounted.
var assetExts = map[string]bool{
	// images
	".png": true, ".jpg": true, ".jpeg": true, ".svg": true, ".webp": true,
	".bmp": true, ".tga": true, ".exr": true, ".hdr": true, ".dds": true,
	// audio
	".wav": true, ".ogg": true, ".mp3": true,
	// video
	".ogv": true, ".webm": true,
	// models
	".glb": true, ".gltf": true, ".obj": true, ".fbx": true, ".dae": true, ".blend": true,
	// fonts
	".ttf": true, ".otf": true, ".woff": true, ".woff2": true, ".fnt": true,
	// shaders / themes
	".gdshader": true, ".gdshaderinc": true, ".theme": true,
}

// classify maps a filename to its node Kind. The second return is false for
// files discovery does not model (docs, configs, unknown extensions).
func classify(name string) (model.Kind, bool) {
	switch ext := strings.ToLower(filepath.Ext(name)); {
	case ext == ".gd" || ext == ".cs":
		return model.KindScript, true
	case ext == ".tscn" || ext == ".scn":
		return model.KindScene, true
	case ext == ".tres" || ext == ".res":
		return model.KindResource, true
	case ext == ".gdextension":
		return model.KindExtension, true
	case assetExts[ext]:
		return model.KindAsset, true
	default:
		return "", false
	}
}

// Counts tallies project nodes by Kind. The CLI summary reports the headline
// five (scripts, scenes, resources, assets, autoloads); the full map is
// available for callers that want every kind.
func Counts(p *model.Project) map[model.Kind]int {
	c := make(map[model.Kind]int, len(model.AllKinds))
	for _, n := range p.Nodes {
		c[n.Kind]++
	}
	return c
}
