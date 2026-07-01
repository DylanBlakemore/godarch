// Package grammar wraps the vendored tree-sitter-gdscript C grammar
// (PrestonKnopp, pinned — see VERSION) as a tree-sitter Language.
//
// Only the generated grammar sources (parser.c, scanner.c, tree_sitter/*.h) are
// vendored here; the tree-sitter runtime is the pinned
// github.com/tree-sitter/go-tree-sitter module. The grammar's own Go module
// cannot be fetched (a case-insensitive filename collision in its Swift
// bindings makes the module zip invalid on every platform), so its generated C
// is vendored at a reviewed commit while the runtime stays a normal dependency.
//
// cgo compiles parser.c and scanner.c alongside this file; Language() returns
// the *sitter.Language a parser binds to a document.
package grammar

// #include "tree_sitter/parser.h"
// TSLanguage *tree_sitter_gdscript(void);
import "C"

import (
	"unsafe"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// Language returns the tree-sitter Language for GDScript.
func Language() *sitter.Language {
	return sitter.NewLanguage(unsafe.Pointer(C.tree_sitter_gdscript()))
}
