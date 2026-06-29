package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantCode int
		wantOut  string // substring expected on stdout
		wantErr  string // substring expected on stderr
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
			wantErr:  "analyze",
		},
		{
			name:     "analyze with path is recognised",
			args:     []string{"analyze", "some/project"},
			wantCode: 0,
			wantOut:  "some/project",
		},
		{
			name:     "help is recognised",
			args:     []string{"help"},
			wantCode: 0,
			wantOut:  "Usage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, errBuf bytes.Buffer
			code := run(tt.args, &out, &errBuf)
			if code != tt.wantCode {
				t.Errorf("run(%v) code = %d, want %d", tt.args, code, tt.wantCode)
			}
			if tt.wantOut != "" && !strings.Contains(out.String(), tt.wantOut) {
				t.Errorf("run(%v) stdout = %q, want substring %q", tt.args, out.String(), tt.wantOut)
			}
			if tt.wantErr != "" && !strings.Contains(errBuf.String(), tt.wantErr) {
				t.Errorf("run(%v) stderr = %q, want substring %q", tt.args, errBuf.String(), tt.wantErr)
			}
		})
	}
}
