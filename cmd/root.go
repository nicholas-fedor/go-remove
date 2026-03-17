/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

// Package cmd contains the command-line interface logic for go-remove.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/nicholas-fedor/go-remove/internal/buildinfo"
	"github.com/nicholas-fedor/go-remove/internal/cli"
	"github.com/nicholas-fedor/go-remove/internal/fs"
	"github.com/nicholas-fedor/go-remove/internal/history"
	"github.com/nicholas-fedor/go-remove/internal/logger"
	"github.com/nicholas-fedor/go-remove/internal/storage"
	"github.com/nicholas-fedor/go-remove/internal/trash"
)

// Common errors for CLI operations.
var (
	// ErrInvalidLoggerType indicates that the logger is not of the expected *ZerologLogger type.
	ErrInvalidLoggerType = errors.New("logger is not a *ZerologLogger")

	// ErrUndoWithBinary indicates the user specified both --undo flag and a binary name.
	ErrUndoWithBinary = errors.New("cannot specify binary name with --undo flag")

	// ErrUndoWithRestore indicates the user specified both --undo and --restore flags.
	ErrUndoWithRestore = errors.New("cannot use --undo and --restore flags together")

	// ErrRestoreWithBinary indicates the user specified both --restore flag and a binary name.
	ErrRestoreWithBinary = errors.New("cannot specify binary name with --restore flag")

	// ErrNoDeletionHistory indicates there is no deletion history to undo.
	ErrNoDeletionHistory = errors.New("no deletion history found - nothing to undo")

	// ErrBinaryNotInTrash indicates the binary is no longer available in trash.
	ErrBinaryNotInTrash = errors.New("binary is no longer in trash - cannot restore")

	// ErrBinaryAlreadyRestored indicates the binary has already been restored.
	ErrBinaryAlreadyRestored = errors.New("binary has already been restored")

	// ErrRestoreCollision indicates a file already exists at the restore location.
	ErrRestoreCollisionCLI = errors.New("a file already exists at the restore location")

	// ErrNoWritableStorage indicates no writable directory was found for storage.
	ErrNoWritableStorage = errors.New("no writable directory found for storage")
)

// dirPermissions defines the permissions for creating directories.
const dirPermissions = 0o750

// getStoragePath returns the path for the history storage database.
// It uses platform-specific paths:
//   - Linux: $XDG_DATA_HOME/go-remove/history.badger or ~/.local/share/go-remove/history.badger
//   - macOS: ~/Library/Application Support/go-remove/history.badger
//   - Windows: %LOCALAPPDATA%/go-remove/history.badger
//
// The function attempts os.UserHomeDir() first, then falls back to os.Executable().
// It verifies that the chosen directory is writable before returning.
// Returns an error if no writable directory can be found.
func getStoragePath() (string, error) {
	// Try to get a writable data home directory
	dataHome, err := getWritableDataHome()
	if err != nil {
		return "", fmt.Errorf("finding writable storage directory: %w", err)
	}

	return filepath.Join(dataHome, "go-remove", "history.badger"), nil
}

// getWritableDataHome attempts to find a writable directory for storing data.
// It tries platform-specific paths first, then falls back to user home directory
// and executable directory.
// Returns an error if no writable directory can be found.
func getWritableDataHome() (string, error) {
	// Try platform-specific paths first
	candidates := getPlatformDataHomeCandidates()

	// Check each candidate for writability
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}

		if isDirWritable(candidate) {
			return candidate, nil
		}
	}

	// Fallback: try user home directory
	userHome, err := os.UserHomeDir()
	if err == nil && isDirWritable(userHome) {
		return userHome, nil
	}

	// Last resort: try executable directory
	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		if isDirWritable(exeDir) {
			return exeDir, nil
		}
	}

	return "", ErrNoWritableStorage
}

// getPlatformDataHomeCandidates returns a list of potential data home directories
// based on the current operating system.
func getPlatformDataHomeCandidates() []string {
	var candidates []string

	switch runtime.GOOS {
	case "windows":
		// Try LOCALAPPDATA first, then USERPROFILE
		if dataHome := os.Getenv("LOCALAPPDATA"); dataHome != "" {
			candidates = append(candidates, dataHome)
		}

		if userProfile := os.Getenv("USERPROFILE"); userProfile != "" {
			candidates = append(candidates, userProfile)
		}
	case "darwin":
		// Try ~/Library/Application Support
		userHome, err := os.UserHomeDir()
		if err == nil {
			candidates = append(
				candidates,
				filepath.Join(userHome, "Library", "Application Support"),
			)
		}
	default: // Linux and other Unix-like systems
		// Try $XDG_DATA_HOME first, then ~/.local/share
		if dataHome := os.Getenv("XDG_DATA_HOME"); dataHome != "" {
			candidates = append(candidates, dataHome)
		}

		userHome, err := os.UserHomeDir()
		if err == nil {
			candidates = append(candidates, filepath.Join(userHome, ".local", "share"))
		}
	}

	return candidates
}

// isDirWritable checks if a directory is writable by attempting to create
// a temporary file in it.
func isDirWritable(dir string) bool {
	// Create the directory if it doesn't exist
	if err := os.MkdirAll(dir, dirPermissions); err != nil {
		return false
	}

	// Try to create a temporary file to verify writability
	tmpFile, err := os.CreateTemp(dir, ".write_test_*")
	if err != nil {
		return false
	}

	// Clean up the temporary file
	_ = tmpFile.Close()

	_ = os.Remove(tmpFile.Name())

	return true
}

// initHistoryManager creates and initializes a history manager with all dependencies.
//
// Parameters:
//   - log: Logger instance for recording operations
//
// Returns:
//   - A history.Manager instance
//   - An error if initialization fails
func initHistoryManager(log logger.Logger) (history.Manager, error) {
	// Create trash manager
	trasher, err := trash.NewTrasher()
	if err != nil {
		return nil, fmt.Errorf("initializing trash: %w", err)
	}

	// Create storage
	dbPath, err := getStoragePath()
	if err != nil {
		return nil, fmt.Errorf("determining storage path: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), dirPermissions); err != nil {
		return nil, fmt.Errorf("creating storage directory: %w", err)
	}

	storer, err := storage.NewBadgerStore(dbPath)
	if err != nil {
		return nil, fmt.Errorf("initializing storage: %w", err)
	}

	// Ensure storer is closed if subsequent initialization fails
	defer func() {
		if err != nil {
			if closeErr := storer.Close(); closeErr != nil {
				log.Warn().Err(closeErr).Msg("Failed to close storage after initialization error")
			}
		}
	}()

	// Create build info extractor
	extractor, err := buildinfo.NewExtractor()
	if err != nil {
		return nil, fmt.Errorf("initializing build info extractor: %w", err)
	}

	// Create history manager with the provided logger
	manager := history.NewManager(trasher, storer, extractor, log)

	return manager, nil
}

// runUndo executes the undo operation to restore the most recently deleted binary.
//
// Parameters:
//   - verbose: Whether to enable verbose output
//
// Returns:
//   - An error if the undo operation fails
func runUndo(verbose bool) error {
	// Initialize logger
	log, err := logger.NewLogger()
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	if verbose {
		log.Level(zerolog.DebugLevel)
	}

	// Initialize history manager
	manager, err := initHistoryManager(log)
	if err != nil {
		return fmt.Errorf("failed to initialize history manager: %w", err)
	}

	defer func() {
		if closeErr := manager.Close(); closeErr != nil {
			log.Warn().Err(closeErr).Msg("Failed to close history manager")
		}
	}()

	// Execute undo
	ctx := context.Background()

	result, err := manager.UndoMostRecent(ctx)
	if err != nil {
		if errors.Is(err, history.ErrNoHistory) {
			return ErrNoDeletionHistory
		}

		if errors.Is(err, history.ErrNotInTrash) {
			return ErrBinaryNotInTrash
		}

		if errors.Is(err, history.ErrAlreadyRestored) {
			return ErrBinaryAlreadyRestored
		}

		if errors.Is(err, history.ErrRestoreCollision) {
			return ErrRestoreCollisionCLI
		}

		return fmt.Errorf("undo failed: %w", err)
	}

	// Print success message
	fmt.Fprintf(os.Stdout, "Successfully restored %s to %s\n", result.BinaryName, result.RestoredTo)

	if result.ModulePath != "" {
		fmt.Fprintf(os.Stdout, "  Module: %s\n", result.ModulePath)
	}

	if result.Version != "" {
		fmt.Fprintf(os.Stdout, "  Version: %s\n", result.Version)
	}

	return nil
}

// rootCmd defines the root command for go-remove.
var rootCmd = &cobra.Command{
	Use:   "go-remove [binary]",
	Short: "A tool to remove Go binaries",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Extract flag values to configure CLI behavior; defaults to TUI mode if no binary is given.
		verbose, _ := cmd.Flags().GetBool("verbose")
		goroot, _ := cmd.Flags().GetBool("goroot")
		logLevel, _ := cmd.Flags().GetString("log-level")
		undo, _ := cmd.Flags().GetBool("undo")
		restore, _ := cmd.Flags().GetBool("restore")

		// Handle undo flag - mutually exclusive with binary argument
		if undo {
			if len(args) > 0 {
				return ErrUndoWithBinary
			}

			if restore {
				return ErrUndoWithRestore
			}

			return runUndo(verbose)
		}

		// Handle restore flag - opens TUI in history mode
		if restore {
			if len(args) > 0 {
				return ErrRestoreWithBinary
			}

			// Initialize filesystem
			filesystem := fs.NewRealFS()

			// Determine the binary directory
			binDir, err := filesystem.DetermineBinDir(goroot)
			if err != nil {
				return fmt.Errorf("failed to determine binary directory: %w", err)
			}

			// Initialize the logger with capture support for TUI mode
			log, _, err := logger.NewLoggerWithCapture()
			if err != nil {
				return fmt.Errorf("failed to initialize logger: %w", err)
			}

			if verbose {
				level := logger.ParseLevel(logLevel)
				log.Level(level)
			}

			// Initialize history manager for restore mode
			manager, err := initHistoryManager(log)
			if err != nil {
				return fmt.Errorf("failed to initialize history manager: %w", err)
			}

			defer func() {
				if closeErr := manager.Close(); closeErr != nil {
					log.Warn().Err(closeErr).Msg("Failed to close history manager")
				}
			}()

			// Configure for restore mode (TUI will handle history view)
			config := cli.Config{
				Binary:      "",
				Verbose:     verbose,
				Goroot:      goroot,
				Help:        false,
				LogLevel:    logLevel,
				RestoreMode: true,
			}

			return cli.RunTUI(binDir, config, log, filesystem, cli.DefaultRunner{}, manager)
		}

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

			// Initialize history manager for recording deletions
			manager, err := initHistoryManager(log)
			if err != nil {
				return fmt.Errorf("failed to initialize history manager: %w", err)
			}

			defer func() {
				if closeErr := manager.Close(); closeErr != nil {
					log.Warn().Err(closeErr).Msg("Failed to close history manager")
				}
			}()

			// Assemble dependencies with a real filesystem, logger, and history manager.
			deps := cli.Dependencies{
				FS:             fs.NewRealFS(),
				Logger:         log,
				HistoryManager: manager,
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

		// Initialize history manager for TUI mode
		manager, err := initHistoryManager(log)
		if err != nil {
			return fmt.Errorf("failed to initialize history manager: %w", err)
		}

		defer func() {
			if closeErr := manager.Close(); closeErr != nil {
				log.Warn().Err(closeErr).Msg("Failed to close history manager")
			}
		}()

		return cli.RunTUI(binDir, config, log, filesystem, cli.DefaultRunner{}, manager)
	},
}

// init registers flags for the root command.
func init() {
	rootCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.Flags().BoolP("goroot", "", false, "Target GOROOT/bin instead of GOBIN or GOPATH/bin")
	rootCmd.Flags().StringP("log-level", "l", "info", "Set log level (debug, info, warn, error)")
	rootCmd.Flags().BoolP("undo", "u", false, "Undo the most recent deletion")
	rootCmd.Flags().BoolP("restore", "r", false, "Open history view for restoration")
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
