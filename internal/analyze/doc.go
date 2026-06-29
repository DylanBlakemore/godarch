// Package analyze runs findings engines over the graph: integrity checks
// (milestone 02) and coupling/domain analyses (milestone 04+). It also hosts
// the Pipeline seam that both the CLI and the Wails UI call to run the full
// discover → extract → resolve → build-graph → analyze flow.
//
// Depends on internal/model, internal/graph, and internal/resolve.
package analyze
