package discovery

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dylanblakemore/godarch/internal/model"
)

// uidRe captures a uid://… token from a scene/resource header or an .import
// [remap] block.
var uidRe = regexp.MustCompile(`uid="(uid://[^"]+)"`)

// importSourceRe captures the asset path an .import sidecar belongs to.
var importSourceRe = regexp.MustCompile(`^\s*source_file\s*=\s*"([^"]+)"`)

// buildUIDMap populates p.UIDMap from the uid declared in each scene/resource
// file header and in each asset's .import sidecar.
//
// Godot caches the authoritative uid↔path mapping in the binary
// .godot/uid_cache.bin, but that file is a regenerated build artefact living
// under the always-ignored .godot dir, and its format is unversioned. The uids
// are also written into the text headers and .import sidecars, which are the
// source of truth, so godarch scans those directly rather than parsing the
// cache.
func buildUIDMap(root string, p *model.Project, imports []string) error {
	for _, n := range p.Nodes {
		if n.Kind != model.KindScene && n.Kind != model.KindResource {
			continue
		}
		uid, err := scanHeaderUID(fsPath(root, n.Path))
		if err != nil {
			return err
		}
		if uid != "" {
			p.UIDMap[uid] = n.ID
		}
	}

	for _, rel := range imports {
		uid, source, err := scanImport(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return err
		}
		if uid != "" && source != "" {
			p.UIDMap[uid] = model.NormalizePath(source)
		}
	}
	return nil
}

// scanHeaderUID returns the uid declared in a text scene/resource's first
// descriptor line ([gd_scene …]/[gd_resource …]), or "" if there is none.
func scanHeaderUID(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}
		// The uid lives only on the leading descriptor line; once a non-blank,
		// non-comment line is seen there is nothing more to look at.
		if m := uidRe.FindStringSubmatch(line); m != nil {
			return m[1], nil
		}
		return "", nil
	}
	return "", sc.Err()
}

// scanImport returns the uid and the source asset path declared in an .import
// sidecar. Either may be "" if the sidecar omits it.
func scanImport(path string) (uid, source string, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = f.Close() }()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if uid == "" {
			if m := uidRe.FindStringSubmatch(line); m != nil {
				uid = m[1]
			}
		}
		if source == "" {
			if m := importSourceRe.FindStringSubmatch(line); m != nil {
				source = m[1]
			}
		}
	}
	return uid, source, sc.Err()
}

// fsPath converts a res:// node ID back to its filesystem path under root.
func fsPath(root, resID string) string {
	rel := strings.TrimPrefix(resID, "res://")
	return filepath.Join(root, filepath.FromSlash(rel))
}
