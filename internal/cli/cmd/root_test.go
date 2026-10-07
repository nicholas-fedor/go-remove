/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/go-remove/internal/cli/cmd/mocks"
	restoremocks "github.com/nicholas-fedor/go-remove/internal/cli/cmd/restore/mocks"
	rmmocks "github.com/nicholas-fedor/go-remove/internal/cli/cmd/rm/mocks"
	undomocks "github.com/nicholas-fedor/go-remove/internal/cli/cmd/undo/mocks"
	"github.com/nicholas-fedor/go-remove/internal/cli/flags"
	"github.com/nicholas-fedor/go-remove/internal/logger"
	buildversion "github.com/nicholas-fedor/go-remove/internal/version"
)

// fixture holds a command tree and the mocks and streams behind it.
type fixture struct {
	// browser backs the root command.
	browser *mocks.MockBrowser

	// remover backs the rm command.
	remover *rmmocks.MockRemover

	// undoer backs the undo command.
	undoer *undomocks.MockUndoer

	// history backs the restore command.
	history *restoremocks.MockBrowser

	// stdout captures results and help.
	stdout *bytes.Buffer

	// stderr captures usage for an invalid invocation.
	stderr *bytes.Buffer
}

// newFixture returns mocks with no expectations, so any unexpected service
// call fails the test.
//
// Parameters:
//   - t: the running test.
//
// Returns:
//   - *fixture: the mocks and streams.
func newFixture(t *testing.T) *fixture {
	t.Helper()

	return &fixture{
		browser: mocks.NewMockBrowser(t),
		remover: rmmocks.NewMockRemover(t),
		undoer:  undomocks.NewMockUndoer(t),
		history: restoremocks.NewMockBrowser(t),
		stdout:  &bytes.Buffer{},
		stderr:  &bytes.Buffer{},
	}
}

// execute builds a fresh tree and runs it with args.
//
// Parameters:
//   - t: the running test.
//   - args: the command-line arguments.
//
// Returns:
//   - error: the command error.
func (f *fixture) execute(t *testing.T, args ...string) error {
	t.Helper()

	root := NewRoot(t.Context(), Dependencies{
		Browser:        f.browser,
		Remover:        f.remover,
		Undoer:         f.undoer,
		HistoryBrowser: f.history,
		Stdout:         f.stdout,
		Stderr:         f.stderr,
		Version: buildversion.Info{
			Version:   "v1.2.3",
			CommitSHA: "abc123",
			BuildTime: "now",
		},
	})
	root.SetArgs(args)

	return root.Execute()
}

// TestNewRoot_Help verifies the help text lists every command and the global
// flags, and goes to stdout.
func TestNewRoot_Help(t *testing.T) {
	t.Parallel()

	f := newFixture(t)

	require.NoError(t, f.execute(t, "-h"))

	help := f.stdout.String()
	for _, want := range []string{
		"Usage:\n  go-remove [flags]\n  go-remove [command]",
		"rm          Remove a Go binary",
		"undo        Restore the most recently removed binary",
		"restore     Pick a removed binary to restore from history",
		"version     Print version information",
		"--goroot             Target GOROOT/bin instead of GOBIN or GOPATH/bin",
		"-l, --log-level string   Set log level (debug, info, warn, error) (default \"info\")",
		"-v, --verbose            Enable verbose output",
	} {
		assert.Contains(t, help, want)
	}

	assert.Empty(t, f.stderr.String())
}

// TestNewRoot_SubcommandHelp verifies help for a subcommand also goes to
// stdout, since the help func is inherited from the root.
func TestNewRoot_SubcommandHelp(t *testing.T) {
	t.Parallel()

	f := newFixture(t)

	require.NoError(t, f.execute(t, "rm", "-h"))

	assert.Contains(t, f.stdout.String(), "Usage:\n  go-remove rm <binary> [flags]")
	assert.Contains(t, f.stdout.String(), "Aliases:\n  rm, remove")
	assert.Empty(t, f.stderr.String())
}

// TestNewRoot_BareOpensPicker verifies go-remove with no command opens the
// binary picker with the resolved global settings.
func TestNewRoot_BareOpensPicker(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.browser.EXPECT().
		Browse(mock.Anything, flags.Settings{
			LogLevel: "warn",
			Level:    logger.WarnLevel,
			Verbose:  false,
			Goroot:   true,
		}).
		Return(nil)

	require.NoError(t, f.execute(t, "--goroot", "-l", "warn"))
}

// TestNewRoot_PickerError verifies a failed picker is wrapped and does not
// print usage.
func TestNewRoot_PickerError(t *testing.T) {
	t.Parallel()

	errBrowse := errors.New("no terminal")

	f := newFixture(t)
	f.browser.EXPECT().Browse(mock.Anything, mock.Anything).Return(errBrowse)

	err := f.execute(t)

	require.ErrorIs(t, err, errBrowse)
	assert.NotContains(t, f.stderr.String(), "Usage:")
}

// TestNewRoot_RoutesSubcommands verifies each command, and the remove alias,
// reaches its own service with the global flags applied.
func TestNewRoot_RoutesSubcommands(t *testing.T) {
	t.Parallel()

	verbose := flags.Settings{
		LogLevel: "info",
		Level:    logger.DebugLevel,
		Verbose:  true,
		Goroot:   false,
	}

	tests := []struct {
		expect func(f *fixture)
		name   string
		args   []string
	}{
		{
			name: "rm",
			args: []string{"-v", "rm", "vhs"},
			expect: func(f *fixture) {
				f.remover.EXPECT().Remove(mock.Anything, "vhs", verbose).Return(nil)
			},
		},
		{
			name: "remove alias",
			args: []string{"remove", "-v", "vhs"},
			expect: func(f *fixture) {
				f.remover.EXPECT().Remove(mock.Anything, "vhs", verbose).Return(nil)
			},
		},
		{
			name: "undo",
			args: []string{"undo", "--verbose"},
			expect: func(f *fixture) {
				f.undoer.EXPECT().Undo(mock.Anything, verbose).Return(nil, errors.New("stop"))
			},
		},
		{
			name: "restore",
			args: []string{"restore", "-v"},
			expect: func(f *fixture) {
				f.history.EXPECT().BrowseHistory(mock.Anything, verbose).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newFixture(t)
			tt.expect(f)

			// The undo case returns an error to keep the result out of the
			// way. Only the routing is under test here.
			_ = f.execute(t, tt.args...)
		})
	}
}

// TestNewRoot_Version verifies the version command reports the injected
// build metadata.
func TestNewRoot_Version(t *testing.T) {
	t.Parallel()

	f := newFixture(t)

	require.NoError(t, f.execute(t, "version"))
	assert.Equal(t, "go-remove v1.2.3\n  Commit: abc123\n  Built:  now\n", f.stdout.String())
}

// TestNewRoot_InvocationErrorsShowUsage verifies an unknown command, an
// unknown flag, or a bad argument count prints usage once, and calls nothing.
func TestNewRoot_InvocationErrorsShowUsage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		wantDetail string
		args       []string
	}{
		{
			name:       "positional binary on root",
			args:       []string{"vhs"},
			wantDetail: `unknown command "vhs"`,
		},
		{
			name:       "unknown flag",
			args:       []string{"--bogus"},
			wantDetail: "unknown flag",
		},
		{
			name:       "removed undo flag",
			args:       []string{"--undo"},
			wantDetail: "unknown flag",
		},
		{
			name:       "rm without a binary",
			args:       []string{"rm"},
			wantDetail: "accepts 1 arg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newFixture(t)

			err := f.execute(t, tt.args...)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantDetail)
			assert.Equal(t, 1, strings.Count(f.stderr.String(), "Usage:"),
				"usage must be printed once for an invalid invocation")
			assert.NotContains(t, f.stderr.String(), "Error:",
				"the caller reports the error, so cobra must not")
		})
	}
}

// TestNewRoot_RejectsUnknownLogLevel verifies an unrecognized level fails
// before any service is called, and names the accepted values.
func TestNewRoot_RejectsUnknownLogLevel(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{
		{"-l", "banana"},
		{"-l", "banana", "rm", "vhs"},
		{"--log-level", "banana", "undo"},
		{"-l", "banana", "restore"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			t.Parallel()

			f := newFixture(t)

			err := f.execute(t, args...)

			require.ErrorIs(t, err, logger.ErrInvalidLogLevel)
			assert.Contains(t, err.Error(), "debug, info, warn, error")
		})
	}
}

// TestNewRoot_FreshTreeEachCall verifies an operation failure in one tree
// does not silence usage for an invalid invocation in the next.
//
// RunE switches SilenceUsage on for a failure of the work. A shared tree would
// carry that into a later run and hide the valid flags for a mistyped one.
func TestNewRoot_FreshTreeEachCall(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.browser.EXPECT().Browse(mock.Anything, mock.Anything).Return(errors.New("no terminal")).Once()

	require.Error(t, f.execute(t))
	require.Error(t, f.execute(t, "--bogus"))

	assert.Equal(t, 1, strings.Count(f.stderr.String(), "Usage:"))
}
