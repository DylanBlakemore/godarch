package main

import (
	"bytes"
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

func TestAnalyzeMinimalFixturePrintsCountsAndWritesDB(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "minimal.godarch.db")

	var out, errBuf bytes.Buffer
	code := run([]string{"analyze", "-db", dbPath, minimalFixture}, &out, &errBuf)
	if code != 0 {
		t.Fatalf("analyze code = %d, want 0 (stderr: %s)", code, errBuf.String())
	}

	got := out.String()
	for _, want := range []string{
		"scripts", "2",
		"scenes", "1",
		"resources", "0",
		"assets", "1",
		"autoloads", "1",
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
