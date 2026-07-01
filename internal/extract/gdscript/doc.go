// Package gdscript extracts nodes and edges from GDScript source via the
// tree-sitter-gdscript grammar. It parses each .gd file to a CST and walks it
// twice — a declaration pass (class_name, extends, signals, methods, exports,
// @rpc) and a reference pass (the call-site egress detectors) — emitting
// schema-conformant model types: code-origin nodes, the edges of DESIGN §3.2,
// and the boundary points of DESIGN §4. It does not resolve cross-file
// references (M2) or persist anything (04).
//
// The tree-sitter runtime is the pinned github.com/tree-sitter/go-tree-sitter
// module; the grammar's generated C is vendored under ./grammar (see its
// VERSION). Implemented in milestone 01.
package gdscript
