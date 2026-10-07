/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package rm

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/go-remove/internal/cli/cmd/rm/mocks"
	"github.com/nicholas-fedor/go-remove/internal/cli/flags"
	"github.com/nicholas-fedor/go-remove/internal/logger"
)

// errWrite is the error errWriter returns.
var errWrite = errors.New("write failed")

// errWriter is an io.Writer whose every write fails.
type errWriter struct{}

// Write fails every write.
//
// Returns:
//   - int: always 0.
//   - error: always errWrite.
func (errWriter) Write([]byte) (int, error) { return 0, errWrite }

// execute runs a standalone rm command with the given flags and arguments.
//
// Parameters:
//   - t: the running test.
//   - remover: the service the command calls.
//   - global: the persistent flag values.
//   - args: the command arguments.
//
// Returns:
//   - string: what the command wrote to stdout.
//   - string: what cobra wrote to its error stream.
//   - error: the command error.
func execute(
	t *testing.T,
	remover Remover,
	global *flags.Global,
	args ...string,
) (string, string, error) {
	t.Helper()

	var stdout, stderr bytes.Buffer

	command := NewCommand(t.Context(), &stdout, remover, global)
	command.SetOut(&stderr)
	command.SetErr(&stderr)
	command.SetArgs(args)

	err := command.Execute()

	return stdout.String(), stderr.String(), err
}

// TestNewCommand_Removes verifies the binary and resolved settings reach the
// remover and the success line is printed unless verbose is on.
func TestNewCommand_Removes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		global     flags.Global
		wantLevel  logger.Level
		wantStdout string
	}{
		{
			name:       "quiet",
			global:     flags.Global{LogLevel: "warn", Verbose: false, Goroot: true},
			wantLevel:  logger.WarnLevel,
			wantStdout: "Successfully removed vhs\n",
		},
		{
			name:       "verbose leaves reporting to the logger",
			global:     flags.Global{LogLevel: "info", Verbose: true, Goroot: false},
			wantLevel:  logger.DebugLevel,
			wantStdout: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			remover := mocks.NewMockRemover(t)
			remover.EXPECT().
				Remove(mock.Anything, "vhs", flags.Settings{
					LogLevel: tt.global.LogLevel,
					Level:    tt.wantLevel,
					Verbose:  tt.global.Verbose,
					Goroot:   tt.global.Goroot,
				}).
				Return(nil)

			stdout, _, err := execute(t, remover, &tt.global, "vhs")

			require.NoError(t, err)
			assert.Equal(t, tt.wantStdout, stdout)
		})
	}
}

// TestNewCommand_RejectsBeforeRemoving verifies invalid input fails without
// calling the remover, which has no expectations and fails the test if called.
func TestNewCommand_RejectsBeforeRemoving(t *testing.T) {
	t.Parallel()

	tests := []struct {
		wantErr   error
		name      string
		args      []string
		logLevel  string
		wantUsage bool
	}{
		{
			name:      "empty name",
			args:      []string{""},
			logLevel:  "info",
			wantErr:   flags.ErrEmptyBinaryName,
			wantUsage: false,
		},
		{
			name:      "whitespace name",
			args:      []string{"   "},
			logLevel:  "info",
			wantErr:   flags.ErrEmptyBinaryName,
			wantUsage: false,
		},
		{
			name:      "unknown log level",
			args:      []string{"vhs"},
			logLevel:  "banana",
			wantErr:   logger.ErrInvalidLogLevel,
			wantUsage: false,
		},
		{
			name:      "no binary",
			args:      []string{},
			logLevel:  "info",
			wantErr:   nil,
			wantUsage: true,
		},
		{
			name:      "two binaries",
			args:      []string{"one", "two"},
			logLevel:  "info",
			wantErr:   nil,
			wantUsage: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			global := &flags.Global{LogLevel: tt.logLevel, Verbose: false, Goroot: false}

			stdout, stderr, err := execute(t, mocks.NewMockRemover(t), global, tt.args...)

			require.Error(t, err)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			}

			assert.Empty(t, stdout)
			assert.Equal(t, tt.wantUsage, strings.Contains(stderr, "Usage:"),
				"usage is shown for a bad argument count only")
		})
	}
}

// TestNewCommand_RemoverError verifies a failed removal is wrapped with the
// binary name and prints nothing.
func TestNewCommand_RemoverError(t *testing.T) {
	t.Parallel()

	errRemove := errors.New("remove failed")

	remover := mocks.NewMockRemover(t)
	remover.EXPECT().Remove(mock.Anything, "vhs", mock.Anything).Return(errRemove)

	stdout, stderr, err := execute(t, remover, flags.New(), "vhs")

	require.ErrorIs(t, err, errRemove)
	assert.Contains(t, err.Error(), "removing vhs")
	assert.Empty(t, stdout)
	assert.NotContains(t, stderr, "Usage:", "an operation failure does not repeat the usage")
}

// TestNewCommand_WriteError verifies a failed write of the success line is
// reported.
func TestNewCommand_WriteError(t *testing.T) {
	t.Parallel()

	remover := mocks.NewMockRemover(t)
	remover.EXPECT().Remove(mock.Anything, "vhs", mock.Anything).Return(nil)

	command := NewCommand(t.Context(), errWriter{}, remover, flags.New())
	command.SetOut(&bytes.Buffer{})
	command.SetErr(&bytes.Buffer{})
	command.SetArgs([]string{"vhs"})

	require.ErrorIs(t, command.Execute(), errWrite)
}
