// Command godarch is the CLI entrypoint for godarch, a static-analysis tool
// that builds a typed dependency graph of a Godot project.
//
// The CLI is a thin orchestrator: it wires the analysis pipeline
// (discover → extract → resolve → build graph → analyze) and prints a summary.
// In milestone 00 the subcommands are recognised but do no real work yet.
package main

import (
	"fmt"
	"io"
	"os"
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

// runAnalyze is a no-op placeholder in milestone 00: it validates arguments and
// echoes the target, proving the plumbing without any analysis logic yet.
func runAnalyze(args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "godarch analyze: missing <project-dir> argument")
		return 2
	}
	projectDir := args[0]
	fmt.Fprintf(stdout, "godarch: analyze %s (not yet implemented)\n", projectDir)
	return 0
}
