// Package store persists a model.Project to a local SQLite file (one
// .godarch.db per analysed project) and loads it back. It owns the schema and
// the embedded numbered migrations applied on Open.
//
// Depends on internal/model (and internal/report for findings) only.
package store
