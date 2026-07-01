// Command godarch is the CLI entrypoint for godarch, a static-analysis tool
// that builds a typed dependency graph of a Godot project.
//
// The CLI is a thin orchestrator over internal/analyze's pipeline: analyze runs
// discover → extract → merge and persists the graph; graph and stats read a
// persisted database back for inspection.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/dylanblakemore/godarch/internal/analyze"
	"github.com/dylanblakemore/godarch/internal/model"
	"github.com/dylanblakemore/godarch/internal/store"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

const usage = `godarch — architecture intelligence for Godot projects

Usage:
  godarch <command> [arguments]

Commands:
  analyze <project-dir>   Analyse a Godot project, persist the graph, print a summary
  graph --file <id>       Print a node and its inbound/outbound edges
  stats                   Print graph totals and the top fan-in nodes
  help                    Show this help text
`

// run dispatches a subcommand and returns the process exit code. It takes its
// streams as arguments so it can be exercised in tests without touching os.
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}

	switch cmd := args[0]; cmd {
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usage)
		return 0
	case "analyze":
		return runAnalyze(args[1:], stdout, stderr)
	case "graph":
		return runGraph(args[1:], stdout, stderr)
	case "stats":
		return runStats(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "godarch: unknown command %q\n\n", cmd)
		fmt.Fprint(stderr, usage)
		return 2
	}
}

// runAnalyze runs the full pipeline over a project, persists the graph to a
// SQLite database, and prints the milestone-1 summary: nodes by kind, edges by
// type, boundary counts, and the unresolved/diagnostic totals.
func runAnalyze(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("analyze", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "path to the SQLite database to write (default <project-dir>/.godarch.db)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() < 1 {
		fmt.Fprintln(stderr, "godarch analyze: missing <project-dir> argument")
		return 2
	}
	projectDir := fs.Arg(0)

	p, diags, err := analyze.Run(projectDir)
	if err != nil {
		fmt.Fprintf(stderr, "godarch analyze: %v\n", err)
		return 1
	}

	dest := *dbPath
	if dest == "" {
		dest = filepath.Join(projectDir, ".godarch.db")
	}
	if err := saveProject(dest, p); err != nil {
		fmt.Fprintf(stderr, "godarch analyze: %v\n", err)
		return 1
	}

	printSummary(stdout, projectDir, dest, p, diags)
	return 0
}

// runGraph loads a persisted database and prints a single node with its inbound
// and outbound edges (text, or --json).
func runGraph(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("graph", flag.ContinueOnError)
	fs.SetOutput(stderr)
	file := fs.String("file", "", "node ID to inspect (e.g. res://player/player.gd)")
	dbPath := fs.String("db", ".godarch.db", "path to the SQLite database to read")
	asJSON := fs.Bool("json", false, "emit JSON instead of text")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *file == "" {
		fmt.Fprintln(stderr, "godarch graph: missing --file <node-id>")
		return 2
	}

	p, err := loadProject(*dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "godarch graph: %v\n", err)
		return 1
	}

	node, ok := p.Nodes[*file]
	if !ok {
		fmt.Fprintf(stderr, "godarch graph: node %q not found in %s\n", *file, *dbPath)
		return 1
	}

	view := buildNodeView(p, node)
	if *asJSON {
		return emitJSON(stdout, stderr, view)
	}
	printNodeView(stdout, view)
	return 0
}

// runStats loads a persisted database and prints graph totals plus the top
// fan-in nodes (degree only; real metrics land in M4).
func runStats(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("stats", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", ".godarch.db", "path to the SQLite database to read")
	asJSON := fs.Bool("json", false, "emit JSON instead of text")
	top := fs.Int("top", 10, "how many top fan-in nodes to list")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	p, err := loadProject(*dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "godarch stats: %v\n", err)
		return 1
	}

	s := buildStats(p, *top)
	if *asJSON {
		return emitJSON(stdout, stderr, s)
	}
	printStats(stdout, s)
	return 0
}

// saveProject opens (creating) the SQLite database at dest and writes p.
func saveProject(dest string, p *model.Project) error {
	st, err := store.Open(dest)
	if err != nil {
		return fmt.Errorf("open store %s: %w", dest, err)
	}
	defer func() { _ = st.Close() }()
	if err := st.SaveProject(p); err != nil {
		return fmt.Errorf("save project: %w", err)
	}
	return nil
}

// loadProject opens the SQLite database at path and reconstructs the Project.
func loadProject(path string) (*model.Project, error) {
	st, err := store.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open store %s: %w", path, err)
	}
	defer func() { _ = st.Close() }()
	p, err := st.LoadProject()
	if err != nil {
		return nil, fmt.Errorf("load project: %w", err)
	}
	return p, nil
}

// emitJSON marshals v as indented JSON to stdout, reporting encode failures on
// stderr with a non-zero exit code.
func emitJSON(stdout, stderr io.Writer, v any) int {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintf(stderr, "godarch: encode json: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, string(b))
	return 0
}
