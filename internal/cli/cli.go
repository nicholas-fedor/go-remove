/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/nicholas-fedor/go-remove/internal/fs"
	"github.com/nicholas-fedor/go-remove/internal/history"
	"github.com/nicholas-fedor/go-remove/internal/logger"
	"github.com/nicholas-fedor/go-remove/internal/tui"
	"github.com/nicholas-fedor/go-remove/internal/tui/models"
)

// Config holds command-line configuration options.
type Config struct {
	Binary      string
	Verbose     bool
	Goroot      bool
	LogLevel    string
	RestoreMode bool
}

// Dependencies holds runtime dependencies for CLI execution.
type Dependencies struct {
	FS             fs.FS
	Logger         logger.Logger
	HistoryManager history.Manager

	// IsTerminal reports whether standard input is an interactive terminal.
	IsTerminal func() bool
}

// Run executes the CLI logic with the provided dependencies and configuration.
//
// Parameters:
//   - ctx: Context governing the operation, carrying any interrupt.
//   - deps: Filesystem, logger, and optional history manager.
//   - config: CLI configuration for the requested operation.
//
// Returns:
//   - An error if directory resolution, deletion, or TUI execution fails.
func Run(ctx context.Context, deps Dependencies, config Config) error {
	log := deps.Logger

	binDir, err := deps.FS.DetermineBinDir(config.Goroot)
	if err != nil {
		return fmt.Errorf("determining binary directory: %w", err)
	}

	if config.Binary == "" {
		opts := tui.Options{
			Dir: binDir,
			Config: models.Config{
				Verbose:     config.Verbose,
				LogLevel:    config.LogLevel,
				RestoreMode: config.RestoreMode,
			},
			Logger:         log,
			FS:             deps.FS,
			HistoryManager: deps.HistoryManager,
		}

		if deps.IsTerminal != nil {
			opts.StdinIsTerminal = deps.IsTerminal
		}

		if err := tui.Run(ctx, opts); err != nil {
			return fmt.Errorf("running TUI: %w", err)
		}

		return nil
	}

	binaryPath := deps.FS.AdjustBinaryPath(binDir, config.Binary)

	if deps.HistoryManager != nil {
		if _, err := deps.HistoryManager.RecordDeletion(ctx, binaryPath); err != nil {
			return fmt.Errorf("recording deletion: %w", err)
		}
	} else if err := deps.FS.RemoveBinary(binaryPath, config.Binary, config.Verbose, log); err != nil {
		return fmt.Errorf("removing binary %s: %w", config.Binary, err)
	}

	if !config.Verbose {
		fmt.Fprintf(os.Stdout, "Successfully removed %s\n", config.Binary)
	}

	return nil
}
