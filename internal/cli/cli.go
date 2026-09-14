/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

// Package cli provides core logic for the go-remove command-line interface.
package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/nicholas-fedor/go-remove/internal/fs"
	"github.com/nicholas-fedor/go-remove/internal/history"
	"github.com/nicholas-fedor/go-remove/internal/logger"
)

// Config holds command-line configuration options.
type Config struct {
	Binary      string
	Verbose     bool
	Goroot      bool
	Help        bool
	LogLevel    string
	RestoreMode bool
}

// Dependencies holds runtime dependencies for CLI execution.
type Dependencies struct {
	FS             fs.FS
	Logger         logger.Logger
	HistoryManager history.Manager
}

// Run executes the CLI logic with the provided dependencies and configuration.
//
// Parameters:
//   - deps: Filesystem, logger, and optional history manager.
//   - config: CLI configuration for the requested operation.
//
// Returns:
//   - An error if directory resolution, deletion, or TUI execution fails.
func Run(deps Dependencies, config Config) error {
	log := deps.Logger
	defer func() { _ = log.Sync() }()

	binDir, err := deps.FS.DetermineBinDir(config.Goroot)
	if err != nil {
		return fmt.Errorf("determining binary directory: %w", err)
	}

	if config.Binary == "" {
		if err := RunTUI(
			binDir,
			config,
			log,
			deps.FS,
			DefaultRunner{},
			deps.HistoryManager,
		); err != nil {
			return fmt.Errorf("running TUI: %w", err)
		}

		return nil
	}

	binaryPath := deps.FS.AdjustBinaryPath(binDir, config.Binary)

	if deps.HistoryManager != nil {
		if _, err := deps.HistoryManager.RecordDeletion(
			context.Background(),
			binaryPath,
		); err != nil {
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
