/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			wantStderr: "A tool to remove Go binaries\n\nUsage:\n  go-remove [binary] [flags]\n\nFlags:\n      --goroot             Target GOROOT/bin instead of GOBIN or GOPATH/bin\n  -h, --help               help for go-remove\n  -l, --log-level string   Set log level (debug, info, warn, error) (default \"info\")\n  -r, --restore            Open history view for restoration\n  -u, --undo               Undo the most recent deletion\n  -v, --verbose            Enable verbose output\n",
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

// TestExecute_ResetsSilenceUsageEachRun verifies the usage suppression set by
// RunE does not leak into a later execution.
//
// RunE turns SilenceUsage on so a failure of the operation does not print the
// flag list. That state lives on the shared command, so without a reset a
// subsequent unknown flag or bad argument count would stop showing usage.
func TestExecute_ResetsSilenceUsageEachRun(t *testing.T) {
	oldStderr := os.Stderr

	r, w, _ := os.Pipe()
	os.Stderr = w

	defer func() {
		os.Stderr = oldStderr

		w.Close()
	}()

	rootCmd.SetOut(os.Stderr)
	rootCmd.SetErr(os.Stderr)
	rootCmd.InitDefaultHelpFlag()

	// Simulate a previous RunE having suppressed usage.
	rootCmd.SilenceUsage = true

	if err := rootCmd.Flags().Set("help", "false"); err != nil {
		t.Fatalf("resetting the help flag: %v", err)
	}

	rootCmd.SetArgs([]string{"--bogus"})

	err := execute()
	require.Error(t, err, "an unknown flag must fail")

	assert.False(
		t,
		rootCmd.SilenceUsage,
		"each execution must start from the default, or a later flag error would hide the valid flags",
	)

	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)

	assert.Equal(t, 1, strings.Count(buf.String(), "Usage:"),
		"usage must be printed once for the flag error")
}

// TestFlagErrorsShowUsage verifies that a flag parse error still prints the
// valid flags.
//
// SilenceUsage suppresses the usage block for every error, so without
// SetFlagErrorFunc a mistyped flag was reported with no hint of what is valid.
func TestFlagErrorsShowUsage(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantErr    bool
		wantUsage  bool
		wantDetail string
	}{
		{
			name:       "unknown flag",
			args:       []string{"--bogus"},
			wantErr:    true,
			wantUsage:  true,
			wantDetail: "unknown flag",
		},
		{
			name:       "too many arguments",
			args:       []string{"one", "two"},
			wantErr:    true,
			wantUsage:  true,
			wantDetail: "accepts at most 1 arg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldStderr := os.Stderr

			r, w, _ := os.Pipe()
			os.Stderr = w

			defer func() {
				os.Stderr = oldStderr

				w.Close()
			}()

			rootCmd.SetOut(os.Stderr)
			rootCmd.SetErr(os.Stderr)
			// A previous RunE leaves SilenceUsage set on the shared command, and
			// an earlier -h leaves the help flag true, which makes cobra short
			// circuit before validating arguments. Restore what a fresh process
			// would have. The help flag is added lazily, so force it to exist.
			rootCmd.SilenceUsage = false
			rootCmd.InitDefaultHelpFlag()

			if err := rootCmd.Flags().Set("help", "false"); err != nil {
				t.Fatalf("resetting the help flag: %v", err)
			}

			rootCmd.SetArgs(tt.args)
			err := rootCmd.Execute()

			w.Close()

			var buf bytes.Buffer
			buf.ReadFrom(r)
			got := buf.String()

			if (err != nil) != tt.wantErr {
				t.Fatalf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && !strings.Contains(err.Error(), tt.wantDetail) {
				t.Errorf("error must explain the failure, got %q", err.Error())
			}

			// SilenceErrors means cobra does not print the message here; the
			// top-level Execute does, so only usage is expected on stderr.
			if gotCount := strings.Count(got, "Usage:"); gotCount != 1 {
				t.Errorf(
					"usage must be printed exactly once, got %d occurrences: %q",
					gotCount,
					got,
				)
			}

			if !strings.Contains(got, "--goroot") {
				t.Errorf("usage must list the valid flags, got %q", got)
			}
		})
	}
}

// TestGetStoragePath verifies the storage path calculation.
func TestGetStoragePath(t *testing.T) {
	// This test verifies that getStoragePath returns a non-empty string and no error.
	// The actual path depends on environment variables, so we just verify
	// it doesn't return empty or panic.
	path, err := getStoragePath()
	if err != nil {
		t.Errorf("getStoragePath() returned error: %v", err)
	}

	if path == "" {
		t.Error("getStoragePath() returned empty string")
	}

	// Verify it contains the expected components
	if !strings.Contains(path, "go-remove") {
		t.Errorf("getStoragePath() = %q, expected to contain 'go-remove'", path)
	}
}
