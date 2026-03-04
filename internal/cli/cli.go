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
	Binary      string // Binary name to remove; empty for TUI mode
	Verbose     bool   // Enable verbose logging
	Goroot      bool   // Use GOROOT/bin instead of GOBIN or GOPATH/bin
	Help        bool   // Show help; managed by Cobra
	LogLevel    string // Log level (debug, info, warn, error)
	RestoreMode bool   // Start TUI in history mode
}

// Dependencies holds runtime dependencies for CLI execution.
type Dependencies struct {
	FS             fs.FS           // Filesystem operations
	Logger         logger.Logger   // Logging interface
	HistoryManager history.Manager // History manager for undo/restore operations (optional)
}

// Run executes the CLI logic with the provided dependencies and configuration.
func Run(deps Dependencies, config Config) error {
	log := deps.Logger

	// Determine the binary directory based on GOROOT or GOPATH/GOBIN settings.
	binDir, err := deps.FS.DetermineBinDir(config.Goroot)
	if err != nil {
		_ = log.Sync() // Flush logs; errors are ignored

		return fmt.Errorf("failed to determine binary directory: %w", err)
	}

	// Execute either TUI mode or direct binary removal based on config.Binary.
	if config.Binary == "" {
		err = RunTUI(binDir, config, log, deps.FS, DefaultRunner{}, deps.HistoryManager)
	} else {
		binaryPath := deps.FS.AdjustBinaryPath(binDir, config.Binary)

		// Record deletion to history if manager is available.
		// RecordDeletion moves the binary to trash internally.
		if deps.HistoryManager != nil {
			ctx := context.Background()
			if _, recordErr := deps.HistoryManager.RecordDeletion(
				ctx,
				binaryPath,
			); recordErr != nil {
				_ = log.Sync()

				return fmt.Errorf("failed to record deletion: %w", recordErr)
			}

			// Binary was successfully moved to trash by RecordDeletion.
			if !config.Verbose {
				fmt.Fprintf(os.Stdout, "Successfully removed %s\n", config.Binary)
			}
		} else {
			// No history manager available; use direct removal as fallback.
			err = deps.FS.RemoveBinary(binaryPath, config.Binary, config.Verbose, log)
			if err != nil {
				_ = log.Sync()

				return fmt.Errorf("failed to remove binary %s: %w", config.Binary, err)
			}

			if !config.Verbose {
				fmt.Fprintf(os.Stdout, "Successfully removed %s\n", config.Binary)
			}
		}
	}

	if err != nil {
		_ = log.Sync() // Flush logs; errors are ignored

		return fmt.Errorf("failed to run TUI: %w", err)
	}

	// Sync the logger to ensure all logs are written before exit.
	_ = log.Sync() // Errors are ignored

	return nil
}
