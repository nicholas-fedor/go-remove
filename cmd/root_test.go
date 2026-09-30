/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/go-remove/internal/logger"
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

// runCommand executes the root command with the given arguments after restoring
// the state a previous run leaves behind, and returns the error plus anything
// written to stderr.
//
// Parameters:
//   - t: Test that owns the run.
//   - args: Command line arguments.
//
// Returns:
//   - What the command wrote to stderr.
//   - The error reported by the command, if any.
func runCommand(t *testing.T, args []string) (string, error) {
	t.Helper()

	oldStderr := os.Stderr

	r, w, _ := os.Pipe()
	os.Stderr = w

	t.Cleanup(func() {
		os.Stderr = oldStderr

		w.Close()
	})

	rootCmd.SetOut(os.Stderr)
	rootCmd.SetErr(os.Stderr)
	rootCmd.InitDefaultHelpFlag()
	rootCmd.SilenceUsage = false

	// Undo and restore keep their values on the shared command between runs.
	for _, name := range []string{"undo", "restore"} {
		if err := rootCmd.Flags().Set(name, "false"); err != nil {
			t.Fatalf("resetting --%s: %v", name, err)
		}
	}

	if err := rootCmd.Flags().Set("help", "false"); err != nil {
		t.Fatalf("resetting the help flag: %v", err)
	}

	rootCmd.SetArgs(args)

	err := execute(t.Context())

	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)

	return buf.String(), err
}

// TestNotifyInterrupt_ReleaseIsSafeToRepeat verifies the signal handler can be
// released both from the watcher goroutine and from the caller.
//
// Releasing cancels the context, which wakes the watcher, which releases again,
// and Execute releases a third time. Two concurrent releases touching the same
// signal registration would race or panic, so this is run under the race
// detector. A second real signal is not delivered here because the default
// behaviour restored after the first one would terminate this process.
func TestNotifyInterrupt_ReleaseIsSafeToRepeat(t *testing.T) {
	ctx, release := notifyInterrupt()

	require.NoError(t, ctx.Err())

	release()
	release()

	require.ErrorIs(t, ctx.Err(), context.Canceled)
}

// TestRunE_FlagValidation verifies the argument combinations that are rejected
// before any work is done.
//
// These branches had no coverage at all, which is how a duplicated usage block
// and a blank binary name reaching the TUI went unnoticed. Each message is
// self explanatory, so the flag list is deliberately not repeated for them.
func TestRunE_FlagValidation(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want error
	}{
		{
			name: "undo and restore together",
			args: []string{"--undo", "--restore"},
			want: ErrUndoWithRestore,
		},
		{
			name: "undo with a binary name",
			args: []string{"--undo", "vhs"},
			want: ErrUndoWithBinary,
		},
		{
			name: "restore with a binary name",
			args: []string{"--restore", "vhs"},
			want: ErrRestoreWithBinary,
		},
		{
			name: "empty binary name",
			args: []string{""},
			want: ErrEmptyBinaryName,
		},
		{
			name: "whitespace binary name",
			args: []string{"   "},
			want: ErrEmptyBinaryName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := runCommand(t, tt.args)

			require.ErrorIs(t, err, tt.want)
			assert.NotContains(t, out, "Usage:",
				"a rejected combination explains itself, so the flag list is not repeated")
		})
	}
}

// TestRunE_RejectsUnknownLogLevel verifies an unrecognised level is reported.
//
// It used to fall back to info silently, so someone who asked for debug output
// was told nothing and saw none. The check happens before the history database
// is opened, so it has no side effects.
func TestRunE_RejectsUnknownLogLevel(t *testing.T) {
	_, err := runCommand(t, []string{"--log-level", "banana", "vhs"})

	require.Error(t, err)
	require.ErrorIs(t, err, logger.ErrInvalidLogLevel)
	assert.Contains(t, err.Error(), "debug, info, warn, error",
		"the error must list the accepted values")
}

// TestGetWritableDataHome_RejectsRelativeXDG verifies a relative XDG_DATA_HOME
// is ignored rather than used to build a data directory.
//
// Only a path can be trusted, and the XDG specification says a relative value
// must be treated as unset.
func TestGetWritableDataHome_RejectsRelativeXDG(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	// A relative value must not be used, and the platform default is selected.
	t.Setenv("XDG_DATA_HOME", "relative/data")

	dir, err := getWritableDataHome()
	require.NoError(t, err)

	require.NotEmpty(t, dir)
	assert.True(t, filepath.IsAbs(dir), "the data home must be absolute, got %q", dir)
	assert.NotContains(t, dir, filepath.Join("relative", "data"),
		"a relative XDG_DATA_HOME must be ignored")
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

	err := execute(t.Context())
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
