-- 0001_init: initial godarch schema (DESIGN §6.1, plan 00.03).
-- Migrations are additive and never rewritten once shipped; schema_version in
-- the meta table gates which migrations have been applied.

CREATE TABLE meta (
    key   TEXT PRIMARY KEY,
    value TEXT
); -- godot_version, godarch_version, analyzed_at, project_root, schema_version

CREATE TABLE nodes (
    id         TEXT PRIMARY KEY,
    kind       TEXT NOT NULL,
    path       TEXT,
    line       INTEGER,
    identity   TEXT, -- JSON
    properties TEXT  -- JSON (derived; rewritten by analysis passes)
);
CREATE INDEX idx_nodes_kind ON nodes(kind);
CREATE INDEX idx_nodes_path ON nodes(path);

CREATE TABLE edges (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    type        TEXT NOT NULL,
    source_id   TEXT NOT NULL,
    target_id   TEXT,          -- may hold a match key until resolved
    origin      TEXT NOT NULL, -- code|editor|config|docs
    resolved    INTEGER NOT NULL DEFAULT 0,
    confidence  REAL NOT NULL DEFAULT 1.0,
    ev_file     TEXT,
    ev_line     INTEGER,
    ev_snippet  TEXT,
    properties  TEXT           -- JSON
);
CREATE INDEX idx_edges_source ON edges(source_id);
CREATE INDEX idx_edges_target ON edges(target_id);
CREATE INDEX idx_edges_type   ON edges(type);
CREATE INDEX idx_edges_unres  ON edges(resolved);

CREATE TABLE boundaries (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    direction  TEXT NOT NULL, -- ingress|egress
    type       TEXT NOT NULL,
    node_id    TEXT NOT NULL,
    match_key  TEXT,
    ev_file    TEXT,
    ev_line    INTEGER,
    ev_snippet TEXT,
    meta       TEXT           -- JSON
);
CREATE INDEX idx_boundaries_node ON boundaries(node_id);

CREATE TABLE findings ( -- populated from M2 onward
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    rule      TEXT NOT NULL, -- e.g. dangling_connection, dead_export
    severity  TEXT NOT NULL, -- error|warning|info
    node_id   TEXT,
    edge_id   INTEGER,
    message   TEXT NOT NULL,
    detail    TEXT           -- JSON (what the graph says, suggested fix, …)
);
