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

// Package cmd contains the command-line interface logic for go-remove.
package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/nicholas-fedor/go-remove/internal/cli"
	"github.com/nicholas-fedor/go-remove/internal/fs"
	"github.com/nicholas-fedor/go-remove/internal/logger"
)

// ErrInvalidLoggerType indicates that the logger is not of the expected *ZapLogger type.
var ErrInvalidLoggerType = errors.New("logger is not a *ZapLogger")

// rootCmd defines the root command for go-remove.
var rootCmd = &cobra.Command{
	Use:   "go-remove [binary]",
	Short: "A tool to remove Go binaries",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Initialize the logger for application-wide logging.
		log, err := logger.NewZapLogger()
		if err != nil {
			return fmt.Errorf("failed to initialize logger: %w", err)
		}

		// Assemble dependencies with a real filesystem and the logger instance.
		deps := cli.Dependencies{
			FS:     fs.NewRealFS(),
			Logger: log,
		}

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

		// Set log level based on config if verbose mode is enabled.
		if verbose {
			zapLogger, ok := log.(*logger.ZapLogger)
			if !ok {
				return fmt.Errorf(
					"failed to set log level: %w with type %T",
					ErrInvalidLoggerType,
					log,
				)
			}
			switch logLevel {
			case "debug":
				zapLogger.Logger = zapLogger.WithOptions(
					zap.IncreaseLevel(zapcore.DebugLevel),
				)
			case "warn":
				zapLogger.Logger = zapLogger.WithOptions(
					zap.IncreaseLevel(zapcore.WarnLevel),
				)
			case "error":
				zapLogger.Logger = zapLogger.WithOptions(
					zap.IncreaseLevel(zapcore.ErrorLevel),
				)
			default:
				zapLogger.Logger = zapLogger.WithOptions(
					zap.IncreaseLevel(zapcore.InfoLevel),
				)
			}
			log = zapLogger // Update the logger in deps
			deps.Logger = log
		}

		// If a binary name is provided as an argument, run in direct removal mode.
		if len(args) > 0 {
			config.Binary = args[0]

			return cli.Run(deps, config)
		}

		// Otherwise, determine the binary directory and launch the TUI for interactive selection.
		binDir, err := deps.FS.DetermineBinDir(config.Goroot)
		if err != nil {
			return fmt.Errorf("failed to determine binary directory: %w", err)
		}

		return cli.RunTUI(binDir, config, deps.Logger, deps.FS, cli.DefaultRunner{})
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
