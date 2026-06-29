# 00.03 — Storage

Persist the model to a local SQLite file; load it into an in-memory `gonum/graph` for analysis.
No external DB — it's a desktop app (DESIGN §6.1). Uses archi's generic node/edge + JSONB-style
properties pattern, adapted to SQLite.

## Why SQLite + in-memory graph

- **SQLite** = the durable artifact: one `.godarch.db` file per analysed project. Enables fast
  re-open, incremental re-analysis later, and the snapshot/diff harness (M5/99).
- **In-memory `gonum`** = the analysis surface: traversal, SCC, reachability, projections. Built
  from the SQLite rows on load. Analyses never query SQLite directly for graph walks.

## Driver

- `modernc.org/sqlite` (pure-Go, **no extra cgo** — keeps the cgo surface limited to tree-sitter)
  **or** `mattn/go-sqlite3` (cgo). Recommend `modernc.org/sqlite` to avoid compounding cgo.

## Schema

```sql
CREATE TABLE meta (
    key TEXT PRIMARY KEY, value TEXT
); -- godot_version, godarch_version, analyzed_at, project_root, schema_version

CREATE TABLE nodes (
    id         TEXT PRIMARY KEY,
    kind       TEXT NOT NULL,
    path       TEXT,
    line       INTEGER,
    identity   TEXT,   -- JSON
    properties TEXT    -- JSON (derived; rewritten by analysis passes)
);
CREATE INDEX idx_nodes_kind ON nodes(kind);
CREATE INDEX idx_nodes_path ON nodes(path);

CREATE TABLE edges (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    type        TEXT NOT NULL,
    source_id   TEXT NOT NULL,
    target_id   TEXT,           -- may hold a match key until resolved
    origin      TEXT NOT NULL,  -- code|editor|config|docs
    resolved    INTEGER NOT NULL DEFAULT 0,
    confidence  REAL NOT NULL DEFAULT 1.0,
    ev_file     TEXT, ev_line INTEGER, ev_snippet TEXT,
    properties  TEXT            -- JSON
);
CREATE INDEX idx_edges_source ON edges(source_id);
CREATE INDEX idx_edges_target ON edges(target_id);
CREATE INDEX idx_edges_type   ON edges(type);
CREATE INDEX idx_edges_unres  ON edges(resolved);

CREATE TABLE boundaries (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    direction  TEXT NOT NULL,  -- ingress|egress
    type       TEXT NOT NULL,
    node_id    TEXT NOT NULL,
    match_key  TEXT,
    ev_file    TEXT, ev_line INTEGER,
    meta       TEXT            -- JSON
);

CREATE TABLE findings (   -- populated from M2 onward
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    rule       TEXT NOT NULL,    -- e.g. dangling_connection, dead_export
    severity   TEXT NOT NULL,    -- error|warning|info
    node_id    TEXT, edge_id INTEGER,
    message    TEXT NOT NULL,
    detail     TEXT              -- JSON (what the graph says, suggested fix, …)
);
```

`schema_version` in `meta` gates migrations.

## Store API

```go
package store
func Open(path string) (*Store, error)          // opens/creates, runs migrations
func (s *Store) SaveProject(p *model.Project) error // transactional bulk upsert
func (s *Store) LoadProject() (*model.Project, error)
func (s *Store) ReplaceFindings(fs []report.Finding) error
func (s *Store) Close() error
```

Bulk insert in chunks within one transaction (archi inserts in 500-row chunks — copy that).

## In-memory graph (`internal/graph`)

```go
package graph
type Graph struct { /* wraps gonum simple.DirectedGraph + id↔node maps + edge metadata */ }
func Build(p *model.Project) *Graph
func (g *Graph) ForwardReach(id string, maxDepth int) ReachResult  // blast radius
func (g *Graph) ReverseReach(id string, maxDepth int) ReachResult  // "who depends on me"
func (g *Graph) SCC() [][]string                                   // gonum topo.TarjanSCC
func (g *Graph) FanIn(id string) int
func (g *Graph) FanOut(id string) int
func (g *Graph) Project(by ProjectionFunc) *Graph                  // unit projections (M4)
```

gonum gives Tarjan SCC, topological ordering, traversal out of the box; reachability and degree are
thin wrappers. Edge metadata (type/origin/confidence) is kept in a side map keyed by gonum edge.

## Migrations

- Embed numbered `.sql` files via `embed.FS`; apply in order on `Open`, tracked in `meta`.
- Keep migrations additive; never rewrite history once shipped.

## Tasks

- [ ] Choose SQLite driver (recommend `modernc.org/sqlite`); add migration runner.
- [ ] Implement the schema as migration `0001_init.sql`.
- [ ] Implement `SaveProject`/`LoadProject` with a full round-trip test against a hand-built `Project`.
- [ ] Implement `graph.Build` + `ForwardReach`/`ReverseReach`/`SCC`/`FanIn`/`FanOut` with tests on a toy graph.
- [ ] Benchmark load + graph build on a large fixture (target: a 1k-file project < 1s to graph).

## Definition of done

A `Project` saves to SQLite and loads back identical; the gonum graph builds from it and answers
reachability/SCC/degree correctly on a known toy graph.
