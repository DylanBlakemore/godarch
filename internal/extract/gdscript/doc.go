// Package gdscript extracts nodes and edges from GDScript source via the
// tree-sitter-gdscript grammar. It emits schema-conformant model types; it
// does not resolve cross-file references or persist anything.
//
// Implemented in milestone 01. Depends on internal/model only.
package gdscript
