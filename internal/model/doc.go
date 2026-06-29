// Package model holds godarch's core data types: Node, Edge, BoundaryPoint,
// MatchKey, the Project container, and the identifier/match-key helpers.
//
// It is the leaf of the dependency graph: model imports nothing from
// internal/*. Every extractor emits these types and every analysis consumes
// them, so keeping model dependency-free is what prevents import cycles.
package model
