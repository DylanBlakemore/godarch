package store

import (
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite" // pure-Go SQLite driver; keeps the cgo surface limited to tree-sitter

	"github.com/dylanblakemore/godarch/internal/model"
	"github.com/dylanblakemore/godarch/internal/version"
)

// chunkSize bounds how many rows are buffered into a single multi-row INSERT.
// Mirrors archi's 500-row chunking to keep statements and parameter counts sane
// while still amortising round-trips inside one transaction.
const chunkSize = 500

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Store is a handle to a single .godarch.db SQLite file. It owns the schema and
// the migrations applied on Open.
type Store struct {
	db *sql.DB
}

// Open opens (creating if necessary) the SQLite database at path and brings it
// up to the latest schema by applying any pending migrations.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %q: %w", path, err)
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// Close releases the underlying database handle.
func (s *Store) Close() error { return s.db.Close() }

// Meta returns the value of a single meta key. The second result is false when
// the key is absent. It exposes the run-level facts (godarch_version,
// analyzed_at, schema_version, …) that are not part of the Project model.
func (s *Store) Meta(key string) (string, bool, error) {
	var value sql.NullString
	err := s.db.QueryRow(`SELECT value FROM meta WHERE key = ?`, key).Scan(&value)
	switch {
	case err == sql.ErrNoRows:
		return "", false, nil
	case err != nil:
		return "", false, fmt.Errorf("read meta %q: %w", key, err)
	}
	return value.String, true, nil
}

// migration is one embedded numbered .sql file.
type migration struct {
	version int
	name    string
	sql     string
}

// migrate applies every embedded migration whose version exceeds the version
// currently recorded in meta.schema_version, in order, recording progress as it
// goes. Each migration runs in its own transaction so a failure leaves the file
// at the last fully-applied version.
func (s *Store) migrate() error {
	migrations, err := loadMigrations()
	if err != nil {
		return err
	}

	current, err := s.schemaVersion()
	if err != nil {
		return err
	}

	for _, m := range migrations {
		if m.version <= current {
			continue
		}
		if err := s.applyMigration(m); err != nil {
			return fmt.Errorf("apply migration %s: %w", m.name, err)
		}
	}
	return nil
}

// loadMigrations reads and sorts the embedded migration files. Each file must be
// named like "0001_init.sql"; the leading integer is its version.
func loadMigrations() ([]migration, error) {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}

	var out []migration
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		prefix, _, ok := strings.Cut(e.Name(), "_")
		if !ok {
			return nil, fmt.Errorf("migration %q is not named <version>_<desc>.sql", e.Name())
		}
		version, err := strconv.Atoi(prefix)
		if err != nil {
			return nil, fmt.Errorf("migration %q has non-numeric version prefix: %w", e.Name(), err)
		}
		body, err := migrationsFS.ReadFile(path.Join("migrations", e.Name()))
		if err != nil {
			return nil, fmt.Errorf("read migration %q: %w", e.Name(), err)
		}
		out = append(out, migration{version: version, name: e.Name(), sql: string(body)})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].version < out[j].version })
	return out, nil
}

// schemaVersion returns the highest applied migration version, or 0 if the
// database has not been migrated yet (the meta table does not exist).
func (s *Store) schemaVersion() (int, error) {
	var value string
	err := s.db.QueryRow(`SELECT value FROM meta WHERE key = 'schema_version'`).Scan(&value)
	switch {
	case err == sql.ErrNoRows:
		return 0, nil
	case err != nil && strings.Contains(err.Error(), "no such table"):
		return 0, nil
	case err != nil:
		return 0, fmt.Errorf("read schema_version: %w", err)
	}
	v, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("schema_version %q is not an integer: %w", value, err)
	}
	return v, nil
}

func (s *Store) applyMigration(m migration) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(m.sql); err != nil {
		return err
	}
	if _, err := tx.Exec(
		`INSERT INTO meta(key, value) VALUES('schema_version', ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		strconv.Itoa(m.version),
	); err != nil {
		return err
	}
	return tx.Commit()
}

// SaveProject writes p to the database in a single transaction, replacing any
// previously stored project. Edges are persisted in full; Unresolved is treated
// as a derived diagnostic subset of Edges (rebuilt on load from the resolved
// flag) and is not written as separate rows.
func (s *Store) SaveProject(p *model.Project) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for _, stmt := range []string{
		`DELETE FROM nodes`, `DELETE FROM edges`, `DELETE FROM boundaries`,
	} {
		if _, err := tx.Exec(stmt); err != nil {
			return err
		}
	}

	if err := saveMeta(tx, p); err != nil {
		return err
	}
	if err := saveNodes(tx, p.Nodes); err != nil {
		return err
	}
	if err := saveEdges(tx, p.Edges); err != nil {
		return err
	}
	if err := saveBoundaries(tx, p.Boundaries); err != nil {
		return err
	}
	return tx.Commit()
}

func saveMeta(tx *sql.Tx, p *model.Project) error {
	uidJSON, err := marshalMap(stringMapToAny(p.UIDMap))
	if err != nil {
		return err
	}
	pairs := []struct {
		key string
		val any
	}{
		{"project_root", p.Root},
		{"godot_version", p.GodotVersion},
		{"main_scene", p.MainScene},
		{"uid_map", uidJSON},
		{"godarch_version", version.Version},
		{"analyzed_at", time.Now().UTC().Format(time.RFC3339)},
	}
	for _, kv := range pairs {
		if _, err := tx.Exec(
			`INSERT INTO meta(key, value) VALUES(?, ?)
			 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
			kv.key, kv.val,
		); err != nil {
			return err
		}
	}
	return nil
}

func saveNodes(tx *sql.Tx, nodes map[string]*model.Node) error {
	// Iterate deterministically so chunk boundaries don't depend on map order.
	ids := make([]string, 0, len(nodes))
	for id := range nodes {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for chunk := range chunks(len(ids)) {
		var (
			placeholders []string
			args         []any
		)
		for _, id := range ids[chunk.lo:chunk.hi] {
			n := nodes[id]
			identity, err := marshalMap(n.Identity)
			if err != nil {
				return err
			}
			properties, err := marshalMap(n.Properties)
			if err != nil {
				return err
			}
			placeholders = append(placeholders, "(?, ?, ?, ?, ?, ?)")
			args = append(args, n.ID, string(n.Kind), n.Path, n.Line, identity, properties)
		}
		if _, err := tx.Exec(
			`INSERT INTO nodes(id, kind, path, line, identity, properties) VALUES `+
				strings.Join(placeholders, ", "), args...,
		); err != nil {
			return err
		}
	}
	return nil
}

func saveEdges(tx *sql.Tx, edges []*model.Edge) error {
	for chunk := range chunks(len(edges)) {
		var (
			placeholders []string
			args         []any
		)
		for _, e := range edges[chunk.lo:chunk.hi] {
			properties, err := marshalMap(e.Properties)
			if err != nil {
				return err
			}
			placeholders = append(placeholders, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
			args = append(
				args,
				string(e.Type), e.SourceID, e.TargetID, string(e.Origin),
				boolToInt(e.Resolved), e.Confidence,
				e.Evidence.File, e.Evidence.Line, e.Evidence.Snippet, properties,
			)
		}
		if _, err := tx.Exec(
			`INSERT INTO edges(type, source_id, target_id, origin, resolved, confidence,
				ev_file, ev_line, ev_snippet, properties) VALUES `+
				strings.Join(placeholders, ", "), args...,
		); err != nil {
			return err
		}
	}
	return nil
}

func saveBoundaries(tx *sql.Tx, boundaries []*model.BoundaryPoint) error {
	for chunk := range chunks(len(boundaries)) {
		var (
			placeholders []string
			args         []any
		)
		for _, b := range boundaries[chunk.lo:chunk.hi] {
			meta, err := marshalMap(b.Meta)
			if err != nil {
				return err
			}
			placeholders = append(placeholders, "(?, ?, ?, ?, ?, ?, ?)")
			args = append(
				args,
				string(b.Direction), string(b.Type), b.NodeID, string(b.MatchKey),
				b.Evidence.File, b.Evidence.Line, meta,
			)
		}
		if _, err := tx.Exec(
			`INSERT INTO boundaries(direction, type, node_id, match_key, ev_file, ev_line, meta)
			 VALUES `+strings.Join(placeholders, ", "), args...,
		); err != nil {
			return err
		}
	}
	return nil
}

// LoadProject reconstructs the Project from the database. Edges and boundaries
// come back in insertion order; Unresolved is rebuilt as the subset of edges
// whose resolved flag is false.
func (s *Store) LoadProject() (*model.Project, error) {
	p := model.NewProject("")

	if err := s.loadMeta(p); err != nil {
		return nil, err
	}
	if err := s.loadNodes(p); err != nil {
		return nil, err
	}
	if err := s.loadEdges(p); err != nil {
		return nil, err
	}
	if err := s.loadBoundaries(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Store) loadMeta(p *model.Project) error {
	rows, err := s.db.Query(`SELECT key, value FROM meta`)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			key   string
			value sql.NullString
		)
		if err := rows.Scan(&key, &value); err != nil {
			return err
		}
		switch key {
		case "project_root":
			p.Root = value.String
		case "godot_version":
			p.GodotVersion = value.String
		case "main_scene":
			p.MainScene = value.String
		case "uid_map":
			m, err := unmarshalMap(value)
			if err != nil {
				return err
			}
			for k, v := range m {
				p.UIDMap[k] = fmt.Sprint(v)
			}
		}
	}
	return rows.Err()
}

func (s *Store) loadNodes(p *model.Project) error {
	rows, err := s.db.Query(`SELECT id, kind, path, line, identity, properties FROM nodes`)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			n          model.Node
			kind       string
			identity   sql.NullString
			properties sql.NullString
		)
		if err := rows.Scan(&n.ID, &kind, &n.Path, &n.Line, &identity, &properties); err != nil {
			return err
		}
		n.Kind = model.Kind(kind)
		if n.Identity, err = unmarshalMap(identity); err != nil {
			return err
		}
		if n.Properties, err = unmarshalMap(properties); err != nil {
			return err
		}
		node := n
		p.Nodes[n.ID] = &node
	}
	return rows.Err()
}

func (s *Store) loadEdges(p *model.Project) error {
	rows, err := s.db.Query(
		`SELECT type, source_id, target_id, origin, resolved, confidence,
			ev_file, ev_line, ev_snippet, properties
		 FROM edges ORDER BY id`,
	)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			e          model.Edge
			etype      string
			origin     string
			resolved   int
			properties sql.NullString
		)
		if err := rows.Scan(
			&etype, &e.SourceID, &e.TargetID, &origin, &resolved, &e.Confidence,
			&e.Evidence.File, &e.Evidence.Line, &e.Evidence.Snippet, &properties,
		); err != nil {
			return err
		}
		e.Type = model.EdgeType(etype)
		e.Origin = model.Origin(origin)
		e.Resolved = resolved != 0
		if e.Properties, err = unmarshalMap(properties); err != nil {
			return err
		}
		edge := e
		p.Edges = append(p.Edges, &edge)
		if !edge.Resolved {
			p.Unresolved = append(p.Unresolved, &edge)
		}
	}
	return rows.Err()
}

func (s *Store) loadBoundaries(p *model.Project) error {
	rows, err := s.db.Query(
		`SELECT direction, type, node_id, match_key, ev_file, ev_line, meta
		 FROM boundaries ORDER BY id`,
	)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			b         model.BoundaryPoint
			direction string
			btype     string
			matchKey  string
			meta      sql.NullString
		)
		if err := rows.Scan(
			&direction, &btype, &b.NodeID, &matchKey, &b.Evidence.File, &b.Evidence.Line, &meta,
		); err != nil {
			return err
		}
		b.Direction = model.Direction(direction)
		b.Type = model.BoundaryType(btype)
		b.MatchKey = model.MatchKey(matchKey)
		if b.Meta, err = unmarshalMap(meta); err != nil {
			return err
		}
		bp := b
		p.Boundaries = append(p.Boundaries, &bp)
	}
	return rows.Err()
}

// chunkRange is a half-open [lo, hi) slice window.
type chunkRange struct{ lo, hi int }

// chunks yields successive [lo, hi) windows of at most chunkSize over n items.
func chunks(n int) func(func(chunkRange) bool) {
	return func(yield func(chunkRange) bool) {
		for lo := 0; lo < n; lo += chunkSize {
			hi := min(lo+chunkSize, n)
			if !yield(chunkRange{lo, hi}) {
				return
			}
		}
	}
}

// marshalMap renders a map as a JSON string for a TEXT column, or nil (SQL NULL)
// when the map is nil so that it round-trips back to a nil map.
func marshalMap(m map[string]any) (any, error) {
	if m == nil {
		return nil, nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

// unmarshalMap is the inverse of marshalMap: SQL NULL or empty/"null" text
// becomes a nil map.
func unmarshalMap(s sql.NullString) (map[string]any, error) {
	if !s.Valid || s.String == "" || s.String == "null" {
		return nil, nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(s.String), &m); err != nil {
		return nil, err
	}
	return m, nil
}

func stringMapToAny(m map[string]string) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
