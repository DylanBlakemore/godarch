// Command godarch is the CLI entrypoint for godarch, a static-analysis tool
// that builds a typed dependency graph of a Godot project.
//
// The CLI is a thin orchestrator: it wires the analysis pipeline
// (discover → extract → resolve → build graph → analyze) and prints a summary.
// In milestone 00 the subcommands are recognised but do no real work yet.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/dylanblakemore/godarch/internal/discovery"
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
  analyze <project-dir>   Analyse a Godot project and print a summary
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
	default:
		fmt.Fprintf(stderr, "godarch: unknown command %q\n\n", cmd)
		fmt.Fprint(stderr, usage)
		return 2
	}
}

// runAnalyze discovers a Godot project, persists the (milestone-00, nodes-only)
// graph to a SQLite database, and prints a file-classification summary. It is
// the thin orchestrator the plan calls for: discover → store → summarise.
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

	p, err := discovery.Discover(projectDir)
	if err != nil {
		fmt.Fprintf(stderr, "godarch analyze: discover %s: %v\n", projectDir, err)
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

	printSummary(stdout, projectDir, dest, p)
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

// printSummary writes the headline file-classification counts the milestone-00
// deliverable specifies: scripts, scenes, resources, assets, autoloads.
func printSummary(stdout io.Writer, projectDir, dest string, p *model.Project) {
	c := discovery.Counts(p)
	version := p.GodotVersion
	if version == "" {
		version = "unknown"
	}
	fmt.Fprintf(stdout, "godarch: analyzed %s (Godot %s)\n", projectDir, version)
	fmt.Fprintf(stdout, "  scripts:   %d\n", c[model.KindScript])
	fmt.Fprintf(stdout, "  scenes:    %d\n", c[model.KindScene])
	fmt.Fprintf(stdout, "  resources: %d\n", c[model.KindResource])
	fmt.Fprintf(stdout, "  assets:    %d\n", c[model.KindAsset])
	fmt.Fprintf(stdout, "  autoloads: %d\n", c[model.KindAutoload])
	fmt.Fprintf(stdout, "  database:  %s\n", dest)
}
