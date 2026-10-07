/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package app

import (
	"context"
	"fmt"

	"github.com/nicholas-fedor/go-remove/internal/cli/cmd"
	"github.com/nicholas-fedor/go-remove/internal/cli/cmd/restore"
	"github.com/nicholas-fedor/go-remove/internal/cli/cmd/rm"
	"github.com/nicholas-fedor/go-remove/internal/cli/cmd/undo"
	"github.com/nicholas-fedor/go-remove/internal/cli/flags"
	"github.com/nicholas-fedor/go-remove/internal/fs"
	"github.com/nicholas-fedor/go-remove/internal/history"
	"github.com/nicholas-fedor/go-remove/internal/logger"
	"github.com/nicholas-fedor/go-remove/internal/tui"
	"github.com/nicholas-fedor/go-remove/internal/tui/models"
)

// Compile-time checks that services satisfies every consumer interface.
var (
	_ cmd.Browser     = (*services)(nil)
	_ rm.Remover      = (*services)(nil)
	_ undo.Undoer     = (*services)(nil)
	_ restore.Browser = (*services)(nil)
)

// services implements every service the command tree calls.
//
// Each method builds its own logger and history manager, because the TUI
// needs a capturing logger and a direct command does not. The constructor
// fields let tests replace each dependency.
type services struct {
	// fs resolves the bin directory and lists binaries.
	fs fs.FS

	// newLogger builds the logger for a direct command.
	newLogger func() (logger.Logger, error)

	// newCaptureLogger builds the logger whose output the TUI displays.
	newCaptureLogger func() (logger.Logger, error)

	// openHistory opens the history manager.
	openHistory func(log logger.Logger) (history.Manager, error)

	// runTUI runs the interactive interface.
	runTUI func(ctx context.Context, opts tui.Options) error
}

// newServices returns services backed by the real filesystem, loggers,
// history database, and terminal.
//
// Returns:
//   - *services: production services.
func newServices() *services {
	return &services{
		fs:               fs.NewRealFS(),
		newLogger:        logger.NewLogger,
		newCaptureLogger: newCaptureLogger,
		openHistory:      openHistory,
		runTUI:           tui.Run,
	}
}

// Browse runs the binary picker over the bin directory.
//
// Parameters:
//   - ctx: cancellation for the session.
//   - settings: the validated global options.
//
// Returns:
//   - error: non-nil when setup or the TUI fails.
func (s *services) Browse(ctx context.Context, settings flags.Settings) error {
	return s.browse(ctx, settings, models.Config{
		Verbose:     settings.Verbose,
		LogLevel:    settings.LogLevel,
		RestoreMode: false,
	})
}

// BrowseHistory runs the TUI in history mode.
//
// Parameters:
//   - ctx: cancellation for the session.
//   - settings: the validated global options.
//
// Returns:
//   - error: non-nil when setup or the TUI fails.
func (s *services) BrowseHistory(ctx context.Context, settings flags.Settings) error {
	return s.browse(ctx, settings, models.Config{
		Verbose:     settings.Verbose,
		LogLevel:    settings.LogLevel,
		RestoreMode: true,
	})
}

// Remove moves one binary to trash and records it in history.
//
// Parameters:
//   - ctx: cancellation for the removal.
//   - binary: the binary name, relative to the bin directory.
//   - settings: the validated global options.
//
// Returns:
//   - error: non-nil when setup or the removal fails.
func (s *services) Remove(ctx context.Context, binary string, settings flags.Settings) error {
	binDir, err := s.fs.DetermineBinDir(settings.Goroot)
	if err != nil {
		return fmt.Errorf("determining binary directory: %w", err)
	}

	log, err := s.logger(s.newLogger, settings)
	if err != nil {
		return err
	}

	manager, closeManager, err := s.history(log)
	if err != nil {
		return err
	}
	defer closeManager()

	if _, err := manager.RecordDeletion(ctx, s.fs.AdjustBinaryPath(binDir, binary)); err != nil {
		return fmt.Errorf("recording deletion: %w", err)
	}

	return nil
}

// Undo restores the most recently deleted binary.
//
// Parameters:
//   - ctx: cancellation for the restore.
//   - settings: the validated global options.
//
// Returns:
//   - *history.RestoreResult: what was restored and where.
//   - error: non-nil when setup or the restore fails.
func (s *services) Undo(
	ctx context.Context,
	settings flags.Settings,
) (*history.RestoreResult, error) {
	log, err := s.logger(s.newLogger, settings)
	if err != nil {
		return nil, err
	}

	manager, closeManager, err := s.history(log)
	if err != nil {
		return nil, err
	}
	defer closeManager()

	result, err := manager.UndoMostRecent(ctx)
	if err != nil {
		return nil, fmt.Errorf("restoring from history: %w", err)
	}

	return result, nil
}

// browse runs the TUI with the given model configuration.
//
// Parameters:
//   - ctx: cancellation for the session.
//   - settings: the validated global options.
//   - config: the model settings, including the starting view.
//
// Returns:
//   - error: non-nil when setup or the TUI fails.
func (s *services) browse(
	ctx context.Context,
	settings flags.Settings,
	config models.Config,
) error {
	binDir, err := s.fs.DetermineBinDir(settings.Goroot)
	if err != nil {
		return fmt.Errorf("determining binary directory: %w", err)
	}

	log, err := s.logger(s.newCaptureLogger, settings)
	if err != nil {
		return err
	}

	manager, closeManager, err := s.history(log)
	if err != nil {
		return err
	}
	defer closeManager()

	if err := s.runTUI(ctx, tui.Options{
		Dir:             binDir,
		Config:          config,
		Logger:          log,
		FS:              s.fs,
		HistoryManager:  manager,
		ProgramOptions:  nil,
		StdinIsTerminal: nil,
	}); err != nil {
		return fmt.Errorf("running TUI: %w", err)
	}

	return nil
}

// logger builds a logger and applies the resolved level.
//
// Parameters:
//   - build: the logger constructor to use.
//   - settings: the validated global options.
//
// Returns:
//   - logger.Logger: the configured logger.
//   - error: non-nil when the logger cannot be built.
func (s *services) logger(
	build func() (logger.Logger, error),
	settings flags.Settings,
) (logger.Logger, error) {
	log, err := build()
	if err != nil {
		return nil, fmt.Errorf("initializing logger: %w", err)
	}

	log.Level(settings.Level)

	return log, nil
}

// history opens the history manager and returns a function that closes it.
//
// Parameters:
//   - log: logger for the manager and for a close failure.
//
// Returns:
//   - history.Manager: the open manager.
//   - func(): closes the manager, logging a warning on failure.
//   - error: non-nil when the manager cannot be opened.
func (s *services) history(log logger.Logger) (history.Manager, func(), error) {
	manager, err := s.openHistory(log)
	if err != nil {
		return nil, nil, fmt.Errorf("initializing history manager: %w", err)
	}

	closeManager := func() {
		if err := manager.Close(); err != nil {
			log.Warn("Failed to close history manager", logger.Err(err))
		}
	}

	return manager, closeManager, nil
}

// newCaptureLogger builds a logger whose output the TUI can display.
//
// Returns:
//   - logger.Logger: the capturing logger.
//   - error: non-nil when the logger cannot be built.
func newCaptureLogger() (logger.Logger, error) {
	log, _, err := logger.NewLoggerWithCapture()
	if err != nil {
		return nil, fmt.Errorf("initializing capture logger: %w", err)
	}

	return log, nil
}
