// Package version holds the single source of truth for the godarch build
// version. It is a leaf package (no godarch imports) so any layer — the store's
// meta table, the CLI banner — can record it without an import cycle.
package version

// Version is the godarch release identifier written into a database's meta table
// (godarch_version) so an analysed graph records the tool that produced it.
const Version = "0.1.0-m1"
