/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package undo

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/go-remove/internal/cli/cmd/undo/mocks"
	"github.com/nicholas-fedor/go-remove/internal/cli/flags"
	"github.com/nicholas-fedor/go-remove/internal/history"
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

// execute runs a standalone undo command.
//
// Parameters:
//   - t: the running test.
//   - undoer: the service the command calls.
//   - global: the persistent flag values.
//   - args: the command arguments.
//
// Returns:
//   - string: what the command wrote to stdout.
//   - error: the command error.
func execute(t *testing.T, undoer Undoer, global *flags.Global, args ...string) (string, error) {
	t.Helper()

	var stdout bytes.Buffer

	command := NewCommand(t.Context(), &stdout, undoer, global)
	command.SetOut(&bytes.Buffer{})
	command.SetErr(&bytes.Buffer{})
	command.SetArgs(args)

	err := command.Execute()

	return stdout.String(), err
}

// TestNewCommand_ReportsResult verifies the restored binary is reported, with
// the module and version lines only when they are known.
func TestNewCommand_ReportsResult(t *testing.T) {
	t.Parallel()

	tests := []struct {
		result *history.RestoreResult
		name   string
		want   string
	}{
		{
			name: "with module and version",
			result: &history.RestoreResult{
				EntryID:    "1",
				BinaryName: "vhs",
				RestoredTo: "/go/bin/vhs",
				FromTrash:  true,
				ModulePath: "github.com/charmbracelet/vhs",
				Version:    "v0.9.0",
			},
			want: "Successfully restored vhs to /go/bin/vhs\n" +
				"  Module: github.com/charmbracelet/vhs\n" +
				"  Version: v0.9.0\n",
		},
		{
			name: "without build info",
			result: &history.RestoreResult{
				EntryID:    "1",
				BinaryName: "vhs",
				RestoredTo: "/go/bin/vhs",
				FromTrash:  true,
				ModulePath: "",
				Version:    "",
			},
			want: "Successfully restored vhs to /go/bin/vhs\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			undoer := mocks.NewMockUndoer(t)
			undoer.EXPECT().
				Undo(mock.Anything, flags.Settings{
					LogLevel: "info",
					Level:    logger.InfoLevel,
					Verbose:  false,
					Goroot:   false,
				}).
				Return(tt.result, nil)

			stdout, err := execute(t, undoer, flags.New())

			require.NoError(t, err)
			assert.Equal(t, tt.want, stdout)
		})
	}
}

// TestNewCommand_MapsErrors verifies each history failure category reaches the
// user as this package's sentinel, and an unknown one keeps its cause.
func TestNewCommand_MapsErrors(t *testing.T) {
	t.Parallel()

	errOther := errors.New("disk on fire")

	tests := []struct {
		cause error
		want  error
		name  string
	}{
		{
			name:  "no history",
			cause: fmt.Errorf("wrapped: %w", history.ErrNoHistory),
			want:  ErrNoDeletionHistory,
		},
		{
			name:  "not in trash",
			cause: history.ErrNotInTrash,
			want:  ErrBinaryNotInTrash,
		},
		{
			name:  "already restored",
			cause: history.ErrAlreadyRestored,
			want:  ErrBinaryAlreadyRestored,
		},
		{
			name:  "restore collision",
			cause: history.ErrRestoreCollision,
			want:  ErrRestoreCollision,
		},
		{
			name:  "unknown",
			cause: errOther,
			want:  errOther,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			undoer := mocks.NewMockUndoer(t)
			undoer.EXPECT().Undo(mock.Anything, mock.Anything).Return(nil, tt.cause)

			stdout, err := execute(t, undoer, flags.New())

			require.ErrorIs(t, err, tt.want)
			assert.Empty(t, stdout)
		})
	}
}

// TestNewCommand_RejectsBeforeUndoing verifies invalid input fails without
// calling the undoer, which has no expectations and fails the test if called.
func TestNewCommand_RejectsBeforeUndoing(t *testing.T) {
	t.Parallel()

	t.Run("argument", func(t *testing.T) {
		t.Parallel()

		_, err := execute(t, mocks.NewMockUndoer(t), flags.New(), "vhs")

		require.Error(t, err)
	})

	t.Run("unknown log level", func(t *testing.T) {
		t.Parallel()

		global := &flags.Global{LogLevel: "banana", Verbose: false, Goroot: false}

		_, err := execute(t, mocks.NewMockUndoer(t), global)

		require.ErrorIs(t, err, logger.ErrInvalidLogLevel)
	})
}

// TestNewCommand_WriteError verifies a failed write of the result is reported.
func TestNewCommand_WriteError(t *testing.T) {
	t.Parallel()

	undoer := mocks.NewMockUndoer(t)
	undoer.EXPECT().Undo(mock.Anything, mock.Anything).Return(&history.RestoreResult{
		EntryID:    "1",
		BinaryName: "vhs",
		RestoredTo: "/go/bin/vhs",
		FromTrash:  true,
		ModulePath: "",
		Version:    "",
	}, nil)

	command := NewCommand(t.Context(), errWriter{}, undoer, flags.New())
	command.SetOut(&bytes.Buffer{})
	command.SetErr(&bytes.Buffer{})
	command.SetArgs([]string{})

	require.ErrorIs(t, command.Execute(), errWrite)
}
