package discovery

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/dylanblakemore/godarch/internal/config"
	"github.com/dylanblakemore/godarch/internal/model"
)

// Discover locates the Godot project containing dir, classifies its files into
// model nodes by extension, parses project.godot (autoloads, input actions,
// physics/render layers, global groups, engine version, main scene), builds the
// uid↔path map, and pairs each asset with its .import sidecar.
//
// Discovery emits nodes only: edges and boundary points are the job of the M1
// extractors, so the returned Project's Edges/Boundaries are empty. File paths
// are normalised to res:// IDs; .import sidecars are not nodes (they are paired
// onto their asset), and the .godot cache, the VCS dir, and any godarch.yml
// ignore globs are skipped.
func Discover(dir string) (*model.Project, error) {
	root, err := findRoot(dir)
	if err != nil {
		return nil, err
	}

	cfg, err := config.Load(root)
	if err != nil {
		return nil, fmt.Errorf("load %s: %w", config.FileName, err)
	}

	pc, err := loadProjectConfig(root)
	if err != nil {
		return nil, err
	}

	p := model.NewProject("res://")
	p.GodotVersion = pc.godotVersion

	imports, err := walkFiles(root, cfg, p)
	if err != nil {
		return nil, err
	}
	pairImports(p, imports)
	if err := buildUIDMap(root, p, imports); err != nil {
		return nil, err
	}
	addConfigNodes(p, pc)
	setMainScene(p, pc)
	return p, nil
}

// Root returns the Godot project root for dir: the directory containing
// project.godot, ascending from dir if a subdirectory was given. It is exported
// so the pipeline can pass the same filesystem root to Discover and to the
// extractors (which resolve res:// IDs back to files under it).
func Root(dir string) (string, error) { return findRoot(dir) }

// findRoot returns the project root: the directory containing project.godot. It
// accepts dir as-is when project.godot lives there, otherwise it ascends toward
// the filesystem root looking for one. If none is found dir is returned
// unchanged, so discovery stays lenient on a directory that is not (yet) a
// Godot project rather than failing outright.
func findRoot(dir string) (string, error) {
	if hasProjectGodot(dir) {
		return dir, nil
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for d := abs; ; {
		if hasProjectGodot(d) {
			return d, nil
		}
		parent := filepath.Dir(d)
		if parent == d {
			return dir, nil
		}
		d = parent
	}
}

// hasProjectGodot reports whether dir directly contains a project.godot file.
func hasProjectGodot(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, "project.godot"))
	return err == nil && !info.IsDir()
}

// walkFiles classifies every regular file under root into a node, applying the
// ignore filter to both directories (pruned with SkipDir) and files. It returns
// the project-root-relative slash paths of the .import sidecars it saw: those
// are not nodes but are paired onto their assets and mined for uids.
func walkFiles(root string, cfg config.Config, p *model.Project) ([]string, error) {
	var imports []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relSlash := filepath.ToSlash(rel)

		if d.IsDir() {
			if path != root && cfg.IsIgnored(relSlash) {
				return fs.SkipDir
			}
			return nil
		}

		name := d.Name()
		if name == "project.godot" || cfg.IsIgnored(relSlash) {
			return nil
		}
		if strings.HasSuffix(name, ".import") {
			imports = append(imports, relSlash)
			return nil
		}

		kind, ok := classify(name)
		if !ok {
			return nil
		}
		resPath := model.NormalizePath("res://" + relSlash)
		p.Nodes[resPath] = &model.Node{ID: resPath, Kind: kind, Path: resPath}
		return nil
	})
	return imports, err
}

// pairImports records, on each asset node, the res:// path of its .import
// sidecar. The sidecar itself is not a node; the M1.02 scene extractor turns the
// pairing into the imports edge (with importer params parsed from the sidecar).
func pairImports(p *model.Project, imports []string) {
	for _, rel := range imports {
		assetID := model.NormalizePath("res://" + strings.TrimSuffix(rel, ".import"))
		n, ok := p.Nodes[assetID]
		if !ok {
			continue // sidecar whose asset was ignored or is missing on disk
		}
		if n.Identity == nil {
			n.Identity = map[string]any{}
		}
		n.Identity["import"] = model.NormalizePath("res://" + rel)
	}
}

// setMainScene records the project's main scene (application/run/main_scene) and
// marks that scene node as the scene-flow root for later analyses.
func setMainScene(p *model.Project, cfg projectConfig) {
	if cfg.mainScene == "" {
		return
	}
	p.MainScene = cfg.mainScene
	if n, ok := p.Nodes[cfg.mainScene]; ok {
		if n.Identity == nil {
			n.Identity = map[string]any{}
		}
		n.Identity["main_scene"] = true
	}
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
	for _, name := range cfg.groups {
		id := model.GroupID(name)
		p.Nodes[id] = &model.Node{
			ID:       id,
			Kind:     model.KindGroup,
			Identity: map[string]any{"name": name, "predeclared": true},
		}
	}
	addLayerNodes(p, cfg.layers)
}

// addLayerNodes emits one layer:<index> node per distinct layer index, merging
// the per-category names ([layer_names]'s 2d_physics/3d_render/… entries that
// share an index) into the node's identity.
func addLayerNodes(p *model.Project, layers []layerName) {
	names := map[int]map[string]any{}
	for _, l := range layers {
		if names[l.index] == nil {
			names[l.index] = map[string]any{}
		}
		names[l.index][l.category] = l.name
	}
	for idx, byCategory := range names {
		id := model.LayerID(idx)
		p.Nodes[id] = &model.Node{
			ID:       id,
			Kind:     model.KindLayer,
			Identity: map[string]any{"index": idx, "names": byCategory},
		}
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
