/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package tui

import (
	"context"
	"errors"
	"fmt"
	"os"

	"golang.org/x/term"

	tea "charm.land/bubbletea/v2"

	"github.com/nicholas-fedor/go-remove/internal/fs"
	"github.com/nicholas-fedor/go-remove/internal/history"
	"github.com/nicholas-fedor/go-remove/internal/logger"
	"github.com/nicholas-fedor/go-remove/internal/tui/models"
)

// ErrNoBinariesFound signals that no binaries were found in the target directory.
var ErrNoBinariesFound = errors.New("no binaries found in directory")

// ErrNotATerminal indicates the interactive interface was started without a terminal.
var ErrNotATerminal = errors.New("no interactive terminal available")

// Options configures the interactive interface.
type Options struct {
	// Dir is the directory holding the Go binaries.
	Dir string

	// Config holds the model settings the interface needs.
	Config models.Config

	// Logger is used by the model.
	Logger logger.Logger

	// FS is the filesystem implementation.
	FS fs.FS

	// HistoryManager backs undo and restore. It may be nil, in which case a
	// removal is a permanent delete.
	HistoryManager history.Manager

	// ProgramOptions are appended when the program is created. Tests use them
	// to inject WithInput, WithOutput, WithWindowSize and WithoutSignals.
	ProgramOptions []tea.ProgramOption

	// StdinIsTerminal reports whether standard input is an interactive
	// terminal. It is a field so a test can supply its own.
	StdinIsTerminal func() bool
}

// stdinIsTerminal reports whether the interface can be driven from standard
// input.
//
// term.IsTerminal is used rather than an os.ModeCharDevice check because
// /dev/null is itself a character device, so a redirected stdin would otherwise
// be mistaken for a terminal and fail later inside the TUI instead of here.
//
// Parameters:
//   - check: A caller-supplied terminal check, or nil to use the real one.
//
// Returns:
//   - The answer from check when it is set.
//   - Whether standard input is a terminal otherwise.
func stdinIsTerminal(check func() bool) bool {
	if check != nil {
		return check()
	}

	return term.IsTerminal(int(os.Stdin.Fd()))
}

// Run launches the interactive interface for binary selection, removal and
// restore.
//
// Parameters:
//   - ctx: Context governing the work the interface starts, carrying any
//     interrupt.
//   - opts: Settings for the interface.
//
// Returns:
//   - An error if the terminal is not interactive, no binaries are found, or
//     the program fails to run.
//
//nolint:gocritic // hugeParam: Options is a settings struct, read once per run.
func Run(ctx context.Context, opts Options) error {
	// Under a pipe or in CI the failure would otherwise surface as a nested
	// bubbletea error with no hint that a binary name is the plain form.
	if !stdinIsTerminal(opts.StdinIsTerminal) {
		return fmt.Errorf(
			"%w: go-remove needs an interactive terminal, pass a binary name to remove it directly",
			ErrNotATerminal,
		)
	}

	choices, err := opts.FS.ListBinaries(opts.Dir)
	if err != nil {
		return fmt.Errorf("listing binaries in %s: %w", opts.Dir, err)
	}

	if len(choices) == 0 && !opts.Config.RestoreMode {
		return fmt.Errorf("%w: %s", ErrNoBinariesFound, opts.Dir)
	}

	m := models.New(
		ctx,
		opts.Dir,
		opts.Config,
		opts.Logger,
		opts.FS,
		opts.HistoryManager,
		choices,
	)

	// Bubbletea's own signal handling is disabled so an interrupt reaches
	// the context installed by Execute rather than racing it.
	program := tea.NewProgram(m, append(
		[]tea.ProgramOption{tea.WithContext(ctx), tea.WithoutSignalHandler()},
		opts.ProgramOptions...,
	)...)

	_, runErr := program.Run()

	// An operation may still be running, and the caller closes the history
	// manager straight after, so wait rather than closing the store underneath.
	m.Wait()

	if runErr != nil {
		// An interrupt is the user asking to quit, so it is not a failure.
		if errors.Is(runErr, tea.ErrInterrupted) {
			return nil
		}

		// A canceled context makes Bubbletea kill the program, which is still
		// the user's interrupt. Any other kill keeps its own identity.
		if errors.Is(runErr, tea.ErrProgramKilled) && errors.Is(ctx.Err(), context.Canceled) {
			return nil
		}

		return fmt.Errorf("failed to run TUI program: %w", runErr)
	}

	return nil
}
