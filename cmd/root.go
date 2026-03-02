/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

// Package cmd contains the command-line interface logic for go-remove.
package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/nicholas-fedor/go-remove/internal/cli"
	"github.com/nicholas-fedor/go-remove/internal/fs"
	"github.com/nicholas-fedor/go-remove/internal/logger"
)

// ErrInvalidLoggerType indicates that the logger is not of the expected *ZerologLogger type.
var ErrInvalidLoggerType = errors.New("logger is not a *ZerologLogger")

// rootCmd defines the root command for go-remove.
var rootCmd = &cobra.Command{
	Use:   "go-remove [binary]",
	Short: "A tool to remove Go binaries",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Extract flag values to configure CLI behavior; defaults to TUI mode if no binary is given.
		verbose, _ := cmd.Flags().GetBool("verbose")
		goroot, _ := cmd.Flags().GetBool("goroot")
		logLevel, _ := cmd.Flags().GetString("log-level")
		config := cli.Config{
			Binary:   "",
			Verbose:  verbose,
			Goroot:   goroot,
			Help:     false, // Cobra manages help output automatically
			LogLevel: logLevel,
		}

		// If a binary name is provided as an argument, run in direct removal mode.
		if len(args) > 0 {
			config.Binary = args[0]

			// Initialize the standard logger for direct removal mode.
			log, err := logger.NewLogger()
			if err != nil {
				return fmt.Errorf("failed to initialize logger: %w", err)
			}

			// Set log level based on config if verbose mode is enabled.
			if verbose {
				level := logger.ParseLevel(logLevel)
				log.Level(level)
			}

			// Assemble dependencies with a real filesystem and the logger instance.
			deps := cli.Dependencies{
				FS:     fs.NewRealFS(),
				Logger: log,
			}

			return cli.Run(deps, config)
		}

		// Otherwise, determine the binary directory and launch the TUI for interactive selection.
		// For TUI mode, we use a logger with capture support to display logs within the interface.
		filesystem := fs.NewRealFS()

		binDir, err := filesystem.DetermineBinDir(config.Goroot)
		if err != nil {
			return fmt.Errorf("failed to determine binary directory: %w", err)
		}

		// Initialize the logger with capture support for TUI mode.
		log, _, err := logger.NewLoggerWithCapture()
		if err != nil {
			return fmt.Errorf("failed to initialize logger: %w", err)
		}

		// Set log level based on config if verbose mode is enabled.
		if verbose {
			level := logger.ParseLevel(logLevel)
			log.Level(level)
		}

		return cli.RunTUI(binDir, config, log, filesystem, cli.DefaultRunner{})
	},
}

// init registers flags for the root command.
func init() {
	rootCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.Flags().BoolP("goroot", "", false, "Target GOROOT/bin instead of GOBIN or GOPATH/bin")
	rootCmd.Flags().StringP("log-level", "l", "info", "Set log level (debug, info, warn, error)")
}

// Execute runs the root command and handles any execution errors.
func Execute() {
	// Execute the command, capturing any errors for reporting and exit handling.
	if err := rootCmd.Execute(); err != nil {
		// Report errors to stderr and exit with a non-zero status to signal failure.
		os.Stderr.WriteString("Error: " + err.Error() + "\n")
		os.Exit(1)
	}
}
