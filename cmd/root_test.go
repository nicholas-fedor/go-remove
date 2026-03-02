/*
Copyright © 2025 Nicholas Fedor <nick@nickfedor.com>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
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
