package model

import (
	"path"
	"strings"
)

// MatchKey is the normalization contract — the "secret sauce" of resolution
// (DESIGN §4). It is a deterministic string with one property: two boundary
// points, or an edge and a candidate target, link iff their keys are equal.
//
// Every constructor below normalizes its inputs so that the same logical
// target produces the same key regardless of how it was spelled at the call
// site. The fixtures in testdata/matchkey_fixtures.yml lock the expected output
// of each constructor (archi-style ground truth).
type MatchKey string

// wildcard stands in for a target component that cannot be statically resolved
// (e.g. a signal emitter or RPC class whose type is unknown).
const wildcard = "*"

// orWildcard trims s and returns "*" when it is empty — the canonical
// "unresolved component" marker used by signal and RPC keys.
func orWildcard(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return wildcard
	}
	return s
}

// SignalKey builds signal:<emitter_type_or_*>:<name>. The emitter type is "*"
// when it can't be statically resolved.
func SignalKey(emitterType, name string) MatchKey {
	return MatchKey("signal:" + orWildcard(emitterType) + ":" + strings.TrimSpace(name))
}

// ResourceKey builds res:<normalized_path>. uid:// and relative paths are
// canonicalised first via NormalizePath so identical resources collapse to one
// key.
func ResourceKey(resPath string) MatchKey {
	return MatchKey("res:" + NormalizePath(resPath))
}

// ActionKey builds action:<name> from Input.is_action_* call sites and the
// project.godot [input] table.
func ActionKey(name string) MatchKey {
	return MatchKey("action:" + strings.TrimSpace(name))
}

// GroupKey builds group:<name> from add_to_group / groups=[…] and call_group.
func GroupKey(name string) MatchKey {
	return MatchKey("group:" + strings.TrimSpace(name))
}

// AutoloadKey builds autoload:<Name> from the project.godot [autoload] global
// name. The name is case-sensitive (it is a Godot identifier), so case is
// preserved.
func AutoloadKey(name string) MatchKey {
	return MatchKey("autoload:" + strings.TrimSpace(name))
}

// RPCKey builds rpc:<class_or_*>:<method>, matching an @rpc endpoint to its
// rpc()/rpc_id() call sites. The class is "*" when it can't be resolved.
func RPCKey(class, method string) MatchKey {
	return MatchKey("rpc:" + orWildcard(class) + ":" + strings.TrimSpace(method))
}

// NodePathKey builds nodepath:<expr> — the fallback key used when a node path
// expression can't be resolved to a concrete scene-node ID. Fragile-reach
// analysis keys off the raw expression, so it is preserved verbatim (only
// surrounding whitespace is trimmed).
func NodePathKey(expr string) MatchKey {
	return MatchKey("nodepath:" + strings.TrimSpace(expr))
}

// NormalizePath canonicalises a Godot virtual path so that paths spelled
// differently but pointing at the same file produce one string:
//
//   - scheme-less paths gain the res:// prefix;
//   - res:// and user:// schemes are preserved;
//   - uid:// is left untouched (it can only be canonicalised against the UID↔
//     path map built during discovery);
//   - the path body is cleaned: "." and ".." segments are resolved and repeated
//     slashes collapsed, using forward-slash (virtual-fs) semantics regardless
//     of host OS.
//
// Case is preserved: Godot treats res:// paths as case-sensitive.
func NormalizePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}

	scheme := "res://"
	body := p
	switch {
	case strings.HasPrefix(p, "res://"):
		body = strings.TrimPrefix(p, "res://")
	case strings.HasPrefix(p, "user://"):
		scheme, body = "user://", strings.TrimPrefix(p, "user://")
	case strings.HasPrefix(p, "uid://"):
		return p
	}

	// Anchor with a leading slash so path.Clean resolves "." / ".." without
	// letting a leading ".." escape the root, then drop the anchor.
	body = strings.TrimPrefix(path.Clean("/"+body), "/")
	return scheme + body
}
