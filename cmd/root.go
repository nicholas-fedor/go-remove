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
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

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
	// ErrUndoWithBinary indicates the user specified both --undo flag and a binary name.
	ErrUndoWithBinary = errors.New("cannot specify binary name with --undo flag")

	// ErrUndoWithRestore indicates the user specified both --undo and --restore flags.
	ErrUndoWithRestore = errors.New("cannot use --undo and --restore flags together")

	// ErrRestoreWithBinary indicates the user specified both --restore flag and a binary name.
	ErrRestoreWithBinary = errors.New("cannot specify binary name with --restore flag")

	// ErrNoDeletionHistory indicates there is no deletion history to undo.
	ErrNoDeletionHistory = errors.New("no deletion history found - nothing to undo")

	// ErrEmptyBinaryName indicates the binary name argument was blank.
	ErrEmptyBinaryName = errors.New("binary name cannot be empty")

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
//
// It uses platform-specific data directories:
//   - Linux: $XDG_DATA_HOME/go-remove/history.badger or ~/.local/share/go-remove/history.badger.
//   - macOS: ~/Library/Application Support/go-remove/history.badger.
//   - Windows: %LOCALAPPDATA%/go-remove/history.badger.
//
// Returns:
//   - Absolute path to the Badger database directory.
//   - An error if no writable storage directory can be found.
func getStoragePath() (string, error) {
	// Try to get a writable data home directory
	dataHome, err := getWritableDataHome()
	if err != nil {
		return "", fmt.Errorf("finding writable storage directory: %w", err)
	}

	return filepath.Join(dataHome, "go-remove", "history.badger"), nil
}

// getWritableDataHome finds a writable directory for application data.
//
// It tries platform-specific paths first, then the user home directory,
// then the executable directory.
//
// Returns:
//   - Writable data directory path.
//   - An error if no writable directory can be found.
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

// getPlatformDataHomeCandidates returns candidate data directories for the current OS.
//
// Returns:
//   - Ordered list of potential data home directories.
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
		// Try $XDG_DATA_HOME first, then ~/.local/share. The XDG specification
		// requires the value to be absolute, and a relative one would build a
		// data directory against the working directory. The trash layer
		// validates this too.
		if dataHome := os.Getenv("XDG_DATA_HOME"); filepath.IsAbs(dataHome) {
			candidates = append(candidates, dataHome)
		}

		userHome, err := os.UserHomeDir()
		if err == nil {
			candidates = append(candidates, filepath.Join(userHome, ".local", "share"))
		}
	}

	return candidates
}

// isDirWritable reports whether a directory can be written to.
//
// Parameters:
//   - dir: Directory path to test.
//
// Returns:
//   - True if a temporary file can be created in the directory.
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
//   - log: Logger instance for recording operations.
//
// Returns:
//   - A history.Manager instance.
//   - An error if initialization fails.
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
//   - verbose: Whether to enable verbose output.
//
// Returns:
//   - An error if the undo operation fails.
func runUndo(ctx context.Context, verbose bool, logLevel string) error {
	// Initialize logger
	log, err := logger.NewLogger()
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	// Undo honours the same level flags as removal, so --log-level is not
	// silently ignored on this path.
	level, err := logger.ParseLogLevel(logLevel)
	if err != nil {
		return fmt.Errorf("parsing log level: %w", err)
	}

	if verbose {
		level = zerolog.DebugLevel
	}

	log.Level(level)

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

// runRemove initializes dependencies and either removes a binary or starts the TUI.
//
// Parameters:
//   - config: CLI configuration for the requested operation.
//
// Returns:
//   - An error if initialization or execution fails.
func runRemove(ctx context.Context, config cli.Config) error {
	var (
		log logger.Logger
		err error
	)

	if config.Binary == "" {
		log, _, err = logger.NewLoggerWithCapture()
	} else {
		log, err = logger.NewLogger()
	}

	if err != nil {
		return fmt.Errorf("initializing logger: %w", err)
	}

	// The level applies on its own, so --log-level does something without
	// also requiring --verbose. An unrecognised name is rejected rather than
	// silently becoming info, which left a user believing they had enabled
	// debug output.
	level, err := logger.ParseLogLevel(config.LogLevel)
	if err != nil {
		return fmt.Errorf("parsing log level: %w", err)
	}

	if config.Verbose {
		level = zerolog.DebugLevel
	}

	log.Level(level)

	manager, err := initHistoryManager(log)
	if err != nil {
		return fmt.Errorf("initializing history manager: %w", err)
	}

	defer func() {
		if closeErr := manager.Close(); closeErr != nil {
			log.Warn().Err(closeErr).Msg("failed to close history manager")
		}
	}()

	filesystem := fs.NewRealFS()
	if config.Binary != "" {
		if err := cli.Run(
			ctx,
			cli.Dependencies{
				FS:             filesystem,
				Logger:         log,
				HistoryManager: manager,
			}, config); err != nil {
			return fmt.Errorf("running CLI: %w", err)
		}

		return nil
	}

	binDir, err := filesystem.DetermineBinDir(config.Goroot)
	if err != nil {
		return fmt.Errorf("determining binary directory: %w", err)
	}

	if err := cli.RunTUI(
		ctx,
		binDir,
		config,
		log,
		filesystem,
		cli.DefaultRunner{},
		manager,
	); err != nil {
		return fmt.Errorf("running TUI: %w", err)
	}

	return nil
}

// rootCmd defines the root command for go-remove.
var rootCmd = &cobra.Command{
	Use:   "go-remove [binary]",
	Short: "A tool to remove Go binaries",
	Args:  cobra.MaximumNArgs(1),
	// Errors are reported once, by Execute. SilenceErrors stops cobra printing a
	// second copy. SilenceUsage stays off so cobra prints the usage block itself
	// for a mistyped flag or a bad argument count, and RunE switches it on so an
	// error from the work itself, such as a flag combination, does not. A
	// SetFlagErrorFunc calling Usage here would print that block a second time.
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// From here on, failures are about the operation rather than the
		// invocation, so the usage block is no longer useful.
		cmd.SilenceUsage = true

		// The context cobra was given carries the signal handling installed by
		// Execute, so an interrupt can reach the work in progress.
		ctx := cmd.Context()

		verbose, err := cmd.Flags().GetBool("verbose")
		if err != nil {
			return fmt.Errorf("reading --verbose: %w", err)
		}

		goroot, err := cmd.Flags().GetBool("goroot")
		if err != nil {
			return fmt.Errorf("reading --goroot: %w", err)
		}

		logLevel, err := cmd.Flags().GetString("log-level")
		if err != nil {
			return fmt.Errorf("reading --log-level: %w", err)
		}

		undo, err := cmd.Flags().GetBool("undo")
		if err != nil {
			return fmt.Errorf("reading --undo: %w", err)
		}

		restore, err := cmd.Flags().GetBool("restore")
		if err != nil {
			return fmt.Errorf("reading --restore: %w", err)
		}

		if undo && restore {
			return ErrUndoWithRestore
		}

		if undo && len(args) > 0 {
			return ErrUndoWithBinary
		}

		if restore && len(args) > 0 {
			return ErrRestoreWithBinary
		}

		if undo {
			return runUndo(ctx, verbose, logLevel)
		}

		config := cli.Config{
			Verbose:     verbose,
			Goroot:      goroot,
			LogLevel:    logLevel,
			RestoreMode: restore,
		}

		if len(args) > 0 {
			// An empty argument used to pass the empty name through, which
			// silently opened the TUI instead of reporting the problem.
			if strings.TrimSpace(args[0]) == "" {
				return ErrEmptyBinaryName
			}

			config.Binary = args[0]
		}

		return runRemove(ctx, config)
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

// notifyInterrupt installs signal handling where the first interrupt cancels the
// returned context and a second one terminates the process.
//
// signal.NotifyContext leaves its handler registered after the first signal and
// nothing reads it thereafter, so a later signal is swallowed rather than
// reaching the default behaviour. Restoring the default once the context closes
// restores the usual convention, and also bounds the cancellation-free recovery
// of a half-finished deletion.
//
// Returns:
//   - A context cancelled by the first interrupt or terminate signal.
//   - A function that releases the handler. It is safe to call more than once.
func notifyInterrupt() (context.Context, context.CancelFunc) {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)

	go func() {
		<-ctx.Done()
		stop()
	}()

	return ctx, stop
}

// Execute runs the root command.
//
// A cancellable context is installed for the interrupt and terminate signals, so
// a long move or copy can be stopped rather than killed part way through.
// Errors are written to stderr and the process exits with status 1.
func Execute() {
	ctx, stop := notifyInterrupt()

	err := execute(ctx)

	// Released explicitly rather than deferred, so the handler is restored
	// before os.Exit skips any deferred call.
	stop()

	if err != nil {
		// Report errors to stderr and exit with a non-zero status to signal failure.
		os.Stderr.WriteString("Error: " + err.Error() + "\n")
		os.Exit(1)
	}
}

// execute runs the root command and returns its error.
//
// It is separate from Execute so the exit path stays out of the way of tests.
//
// Parameters:
//   - ctx: Context governing the whole invocation.
//
// Returns:
//   - The error reported by the command, if any.
func execute(ctx context.Context) error {
	// RunE turns SilenceUsage on so a failure of the operation does not print
	// the flag list. It stays set on the command, so it is cleared here to give
	// each execution the default where a mistyped flag or a bad argument count
	// still shows the valid flags.
	rootCmd.SilenceUsage = false

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		return fmt.Errorf("running go-remove: %w", err)
	}

	return nil
}
