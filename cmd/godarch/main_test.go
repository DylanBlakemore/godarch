package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const minimalFixture = "../../testdata/fixtures/minimal"

func TestRun(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantCode int
		wantOut  []string // substrings expected on stdout
		wantErr  string   // substring expected on stderr
	}{
		{
			name:     "no args prints usage",
			args:     nil,
			wantCode: 2,
			wantErr:  "Usage",
		},
		{
			name:     "unknown command errors",
			args:     []string{"frobnicate"},
			wantCode: 2,
			wantErr:  "unknown command",
		},
		{
			name:     "analyze without path errors",
			args:     []string{"analyze"},
			wantCode: 2,
			wantErr:  "project-dir",
		},
		{
			name:     "analyze on missing dir errors",
			args:     []string{"analyze", "-db", filepath.Join(t.TempDir(), "x.db"), "does/not/exist"},
			wantCode: 1,
			wantErr:  "discover",
		},
		{
			name:     "help is recognised",
			args:     []string{"help"},
			wantCode: 0,
			wantOut:  []string{"Usage"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, errBuf bytes.Buffer
			code := run(tt.args, &out, &errBuf)
			if code != tt.wantCode {
				t.Errorf("run(%v) code = %d, want %d (stderr: %s)", tt.args, code, tt.wantCode, errBuf.String())
			}
			for _, sub := range tt.wantOut {
				if !strings.Contains(out.String(), sub) {
					t.Errorf("run(%v) stdout = %q, want substring %q", tt.args, out.String(), sub)
				}
			}
			if tt.wantErr != "" && !strings.Contains(errBuf.String(), tt.wantErr) {
				t.Errorf("run(%v) stderr = %q, want substring %q", tt.args, errBuf.String(), tt.wantErr)
			}
		})
	}
}

func TestAnalyzeMinimalFixturePrintsSummaryAndWritesDB(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "minimal.godarch.db")

	var out, errBuf bytes.Buffer
	code := run([]string{"analyze", "-db", dbPath, minimalFixture}, &out, &errBuf)
	if code != 0 {
		t.Fatalf("analyze code = %d, want 0 (stderr: %s)", code, errBuf.String())
	}

	got := out.String()
	// The summary is the pipeline's, not discovery's: it names the per-kind and
	// per-type breakdowns plus the boundary/unresolved/diagnostic tallies.
	for _, want := range []string{
		"nodes", "edges",
		"script", "scene", "autoload",
		"boundaries (ingress", "egress",
		"unresolved edges",
		"diagnostics",
		"database:",
		"4.2",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("analyze stdout missing %q\n%s", want, got)
		}
	}

	if info, err := os.Stat(dbPath); err != nil || info.Size() == 0 {
		t.Errorf("expected non-empty db at %s (err: %v)", dbPath, err)
	}
}

// analyzeToDB runs the pipeline over the minimal fixture into a fresh temp DB and
// returns its path, so the graph/stats subcommands have a database to read.
func analyzeToDB(t *testing.T) string {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "minimal.godarch.db")
	var out, errBuf bytes.Buffer
	if code := run([]string{"analyze", "-db", dbPath, minimalFixture}, &out, &errBuf); code != 0 {
		t.Fatalf("analyze code = %d, want 0 (stderr: %s)", code, errBuf.String())
	}
	return dbPath
}

func TestGraphPrintsNodeEdges(t *testing.T) {
	dbPath := analyzeToDB(t)

	var out, errBuf bytes.Buffer
	code := run([]string{"graph", "-db", dbPath, "-file", "res://player.gd"}, &out, &errBuf)
	if code != 0 {
		t.Fatalf("graph code = %d, want 0 (stderr: %s)", code, errBuf.String())
	}
	got := out.String()
	for _, want := range []string{"res://player.gd", "outbound", "inbound"} {
		if !strings.Contains(got, want) {
			t.Errorf("graph stdout missing %q\n%s", want, got)
		}
	}
}

func TestGraphJSONForKnownNode(t *testing.T) {
	dbPath := analyzeToDB(t)

	var out, errBuf bytes.Buffer
	code := run([]string{"graph", "-db", dbPath, "-file", "res://player.gd", "-json"}, &out, &errBuf)
	if code != 0 {
		t.Fatalf("graph --json code = %d, want 0 (stderr: %s)", code, errBuf.String())
	}
	var v struct {
		ID       string `json:"id"`
		Outbound []any  `json:"outbound"`
		Inbound  []any  `json:"inbound"`
	}
	if err := json.Unmarshal(out.Bytes(), &v); err != nil {
		t.Fatalf("graph --json output is not valid JSON: %v\n%s", err, out.String())
	}
	if v.ID != "res://player.gd" {
		t.Errorf("graph --json id = %q, want res://player.gd", v.ID)
	}
}

func TestGraphMissingNodeErrors(t *testing.T) {
	dbPath := analyzeToDB(t)

	var out, errBuf bytes.Buffer
	code := run([]string{"graph", "-db", dbPath, "-file", "res://nope.gd"}, &out, &errBuf)
	if code != 1 {
		t.Fatalf("graph code = %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "not found") {
		t.Errorf("graph stderr = %q, want substring %q", errBuf.String(), "not found")
	}
}

func TestGraphWithoutFileErrors(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := run([]string{"graph"}, &out, &errBuf)
	if code != 2 {
		t.Fatalf("graph code = %d, want 2", code)
	}
	if !strings.Contains(errBuf.String(), "--file") {
		t.Errorf("graph stderr = %q, want substring %q", errBuf.String(), "--file")
	}
}

func TestStatsPrintsTotals(t *testing.T) {
	dbPath := analyzeToDB(t)

	var out, errBuf bytes.Buffer
	code := run([]string{"stats", "-db", dbPath}, &out, &errBuf)
	if code != 0 {
		t.Fatalf("stats code = %d, want 0 (stderr: %s)", code, errBuf.String())
	}
	for _, want := range []string{"nodes:", "edges:", "boundaries:", "unresolved:", "top fan-in:"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("stats stdout missing %q\n%s", want, out.String())
		}
	}
}

func TestStatsJSON(t *testing.T) {
	dbPath := analyzeToDB(t)

	var out, errBuf bytes.Buffer
	code := run([]string{"stats", "-db", dbPath, "-json"}, &out, &errBuf)
	if code != 0 {
		t.Fatalf("stats --json code = %d, want 0 (stderr: %s)", code, errBuf.String())
	}
	var v struct {
		Nodes    int   `json:"nodes"`
		TopFanIn []any `json:"top_fan_in"`
	}
	if err := json.Unmarshal(out.Bytes(), &v); err != nil {
		t.Fatalf("stats --json output is not valid JSON: %v\n%s", err, out.String())
	}
	if v.Nodes == 0 {
		t.Errorf("stats --json reported 0 nodes for the minimal fixture")
	}
}
