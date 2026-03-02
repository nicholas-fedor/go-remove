/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package cmd

import (
	"bytes"
	"os"
	"testing"
)

// TestRootCommand verifies the behavior of the root command.
func TestRootCommand(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantStderr string
		wantErr    bool
	}{
		{
			name:       "help flag",
			args:       []string{"-h"},
			wantStderr: "A tool to remove Go binaries\n\nUsage:\n  go-remove [binary] [flags]\n\nFlags:\n      --goroot             Target GOROOT/bin instead of GOBIN or GOPATH/bin\n  -h, --help               help for go-remove\n  -l, --log-level string   Set log level (debug, info, warn, error) (default \"info\")\n  -v, --verbose            Enable verbose output\n",
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Redirect stderr to capture output.
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			defer func() {
				os.Stderr = oldStderr

				w.Close()
			}()

			// Configure Cobra to write to stderr and set test arguments.
			rootCmd.SetOut(os.Stderr)
			rootCmd.SetArgs(tt.args)
			err := rootCmd.Execute()

			// Capture stderr output after execution.
			w.Close()

			var buf bytes.Buffer
			buf.ReadFrom(r)
			gotStderr := buf.String()

			t.Logf("Captured stderr: %q", gotStderr)

			if (err != nil) != tt.wantErr {
				t.Errorf("rootCmd.Execute() error = %v, wantErr %v", err, tt.wantErr)
			}

			if gotStderr != tt.wantStderr {
				t.Errorf("rootCmd.Execute() stderr = %q, want %q", gotStderr, tt.wantStderr)
			}
		})
	}
}
