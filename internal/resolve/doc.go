// Package resolve stitches raw extractor output into a connected graph by
// matching edge targets and boundary points on their deterministic match keys.
// It produces resolved edges plus diagnostics for anything that fails to link.
//
// Implemented in milestone 02. Depends on internal/model and internal/graph.
package resolve
