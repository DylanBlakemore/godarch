// Package graph builds an in-memory gonum directed graph from a model.Project
// and answers the structural queries analyses need: forward/reverse
// reachability (blast radius), strongly-connected components, fan-in/fan-out,
// and unit projections.
//
// Depends on internal/model only.
package graph
