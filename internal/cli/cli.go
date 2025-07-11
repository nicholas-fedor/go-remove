/*
Copyright © 2025 Nicholas Fedor <nick@nickfedor.com>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/

// Package cli provides core logic for the go-remove command-line interface.
package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/nicholas-fedor/go-remove/internal/fs"
	"github.com/nicholas-fedor/go-remove/internal/logger"
)

// ErrInvalidLoggerType indicates that the logger is not of the expected *ZapLogger type.
var ErrInvalidLoggerType = errors.New("logger is not a *ZapLogger")

// Config holds command-line configuration options.
type Config struct {
	Binary   string // Binary name to remove; empty for TUI mode
	Verbose  bool   // Enable verbose logging
	Goroot   bool   // Use GOROOT/bin instead of GOBIN or GOPATH/bin
	Help     bool   // Show help; managed by Cobra
	LogLevel string // Log level (debug, info, warn, error)
}

// Dependencies holds runtime dependencies for CLI execution.
type Dependencies struct {
	FS     fs.FS         // Filesystem operations
	Logger logger.Logger // Logging interface
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
		err = RunTUI(binDir, config, log, deps.FS, DefaultRunner{})
	} else {
		binaryPath := deps.FS.AdjustBinaryPath(binDir, config.Binary)

		err = deps.FS.RemoveBinary(binaryPath, config.Binary, config.Verbose, log)
		if err == nil && !config.Verbose {
			// Print success message to stdout for non-verbose mode.
			fmt.Fprintf(os.Stdout, "Successfully removed %s\n", config.Binary)
		}
	}

	if err != nil {
		_ = log.Sync() // Flush logs; errors are ignored

		if config.Binary == "" {
			return fmt.Errorf("failed to run TUI: %w", err)
		}

		return fmt.Errorf("failed to remove binary %s: %w", config.Binary, err)
	}

	// Sync the logger to ensure all logs are written before exit.
	_ = log.Sync() // Errors are ignored

	return nil
}
