/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

// Package cli provides core logic for the go-remove command-line interface.
package cli

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

// stdinIsTerminal reports whether standard input is an interactive terminal.
//
// term.IsTerminal is used rather than an os.ModeCharDevice check because
// /dev/null is itself a character device, so a redirected stdin would otherwise
// be mistaken for a terminal and fail later inside the TUI instead of here. It
// is a variable so the TUI can be exercised in tests, which have no terminal.
var stdinIsTerminal = func() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// ProgramRunner defines an interface for running Bubbletea programs.
type ProgramRunner interface {
	// RunProgram launches a Bubble Tea program.
	//
	// Parameters:
	//   - m: Initial TUI model.
	//   - opts: Optional Bubble Tea program options.
	//
	// Returns:
	//   - Started program.
	//   - An error if the program cannot be created.
	RunProgram(m tea.Model, opts ...tea.ProgramOption) (*tea.Program, error)
}

// DefaultRunner provides the default Bubbletea program runner.
type DefaultRunner struct{}

var _ ProgramRunner = DefaultRunner{}

// RunTUI launches the interactive TUI for binary selection, removal, and restore.
//
// Parameters:
//   - dir: Directory containing Go binaries.
//   - config: CLI configuration.
//   - log: Logger used by the TUI.
//   - filesystem: Filesystem implementation.
//   - runner: Bubble Tea program runner.
//   - historyMgr: History manager for undo and restore.
//
// Returns:
//   - An error if no binaries are found or the program fails to run.
func RunTUI(
	ctx context.Context,
	dir string,
	config Config,
	log logger.Logger,
	filesystem fs.FS,
	runner ProgramRunner,
	historyMgr history.Manager,
) error {
	// A TUI needs a terminal. Under a pipe or in CI the failure otherwise
	// surfaces as a nested bubbletea error with no hint that the non-interactive
	// form is a plain argument.
	if !stdinIsTerminal() {
		return fmt.Errorf(
			"%w: go-remove needs an interactive terminal, pass a binary name to remove it directly",
			ErrNotATerminal,
		)
	}

	// Fetch available binaries from the specified directory.
	choices, err := filesystem.ListBinaries(dir)
	if err != nil {
		return fmt.Errorf("listing binaries in %s: %w", dir, err)
	}

	if len(choices) == 0 && !config.RestoreMode {
		return fmt.Errorf("%w: %s", ErrNoBinariesFound, dir)
	}

	m := models.New(
		ctx,
		dir,
		models.Config{
			Verbose:     config.Verbose,
			LogLevel:    config.LogLevel,
			RestoreMode: config.RestoreMode,
		},
		log,
		filesystem,
		historyMgr,
		choices,
	)

	// Start the TUI program with the caller's context. Bubbletea's own signal
	// handling is disabled so an interrupt reaches the context installed by
	// Execute rather than racing it, leaving a single interruption path.
	program, err := runner.RunProgram(
		m,
		tea.WithContext(ctx),
		tea.WithoutSignalHandler(),
	)
	if err != nil {
		return fmt.Errorf("failed to start TUI program: %w", err)
	}

	// Allow mocked runners to return nil for testing purposes.
	if program == nil {
		return nil
	}

	// Run the program and capture any runtime errors.
	_, runErr := program.Run()

	// An operation may still be running, possibly part way through a recovery.
	// The history manager is closed by the caller straight after this returns,
	// so wait rather than closing the store underneath it.
	m.Wait()

	if runErr != nil {
		// An interrupt is the user asking to quit, so it is not a failure.
		if errors.Is(runErr, tea.ErrInterrupted) {
			return nil
		}

		// A cancelled context makes Bubbletea kill the program, which is still
		// the user's interrupt rather than a real failure. Anything else keeps
		// its own identity, so an unexpected kill is still reported.
		if errors.Is(runErr, tea.ErrProgramKilled) && errors.Is(ctx.Err(), context.Canceled) {
			return nil
		}

		return fmt.Errorf("failed to run TUI program: %w", runErr)
	}

	return nil
}

// RunProgram launches a Bubble Tea program with the given model and options.
//
// Parameters:
//   - m: Initial TUI model.
//   - opts: Optional Bubble Tea program options.
//
// Returns:
//   - Started program.
//   - Always nil error.
func (r DefaultRunner) RunProgram(m tea.Model, opts ...tea.ProgramOption) (*tea.Program, error) {
	program := tea.NewProgram(m, opts...)

	return program, nil
}
