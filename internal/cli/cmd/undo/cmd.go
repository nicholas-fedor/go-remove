/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package undo

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/nicholas-fedor/go-remove/internal/cli/flags"
	"github.com/nicholas-fedor/go-remove/internal/errmsg"
	"github.com/nicholas-fedor/go-remove/internal/history"
)

// Errors reported by the undo command.
var (
	// ErrNoDeletionHistory indicates there is no deletion history to undo.
	ErrNoDeletionHistory = errors.New("no deletion history found - nothing to undo")

	// ErrBinaryNotInTrash indicates the binary is no longer available in trash.
	ErrBinaryNotInTrash = errors.New("binary is no longer in trash - cannot restore")

	// ErrBinaryAlreadyRestored indicates the binary has already been restored.
	ErrBinaryAlreadyRestored = errors.New("binary has already been restored")

	// ErrRestoreCollision indicates a file already exists at the restore location.
	ErrRestoreCollision = errors.New("a file already exists at the restore location")
)

// Undoer restores the most recently deleted binary.
type Undoer interface {
	// Undo restores the most recent deletion.
	//
	// Parameters:
	//   - ctx: cancellation for the operation.
	//   - settings: the validated global options.
	//
	// Returns:
	//   - *history.RestoreResult: what was restored and where.
	//   - error: non-nil when nothing could be restored.
	Undo(ctx context.Context, settings flags.Settings) (*history.RestoreResult, error)
}

// NewCommand returns the undo command.
//
// Parameters:
//   - ctx: cancellation for the restore.
//   - stdout: stream the result is written to.
//   - undoer: performs the restore.
//   - global: the persistent flags bound on the root command.
//
// Returns:
//   - *cobra.Command: the undo command.
func NewCommand(
	ctx context.Context,
	stdout io.Writer,
	undoer Undoer,
	global *flags.Global,
) *cobra.Command {
	return &cobra.Command{
		Use:   "undo",
		Short: "Restore the most recently removed binary",
		Long: `Restore the most recently removed binary from trash to the location it was
removed from.

Use "go-remove restore" to pick an older entry from the history instead.`,
		Example: `# Bring back the last binary removed.
go-remove undo`,
		Args: cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			// Failures from here on are about the operation, not the
			// invocation, so the usage block would not help.
			command.SilenceUsage = true

			return run(ctx, stdout, undoer, global)
		},
	}
}

// run restores the most recent deletion and reports the result.
//
// Parameters:
//   - ctx: cancellation for the restore.
//   - stdout: stream the result is written to.
//   - undoer: performs the restore.
//   - global: the persistent flags.
//
// Returns:
//   - error: one of this package's sentinels for a known history failure, or
//     a wrapped error otherwise.
func run(ctx context.Context, stdout io.Writer, undoer Undoer, global *flags.Global) error {
	settings, err := global.Resolve()
	if err != nil {
		return fmt.Errorf("resolving flags: %w", err)
	}

	result, err := undoer.Undo(ctx, settings)
	if err != nil {
		return classify(err)
	}

	if err := writeResult(stdout, result); err != nil {
		return fmt.Errorf("writing output: %w", err)
	}

	return nil
}

// classify maps a history failure to the matching sentinel.
//
// Parameters:
//   - err: the error from the undoer.
//
// Returns:
//   - error: a sentinel for a known category, or err wrapped otherwise.
func classify(err error) error {
	fallback := fmt.Errorf("undo failed: %w", err)

	switch errmsg.Classify(err) {
	case errmsg.KindNoHistory:
		return ErrNoDeletionHistory
	case errmsg.KindNotInTrash:
		return ErrBinaryNotInTrash
	case errmsg.KindAlreadyRestored:
		return ErrBinaryAlreadyRestored
	case errmsg.KindRestoreCollision:
		return ErrRestoreCollision
	case errmsg.KindUnknown:
		return fallback
	default:
		// A Kind was added without an error for this command.
		return fallback
	}
}

// writeResult reports a successful restore.
//
// Parameters:
//   - stdout: stream the result is written to.
//   - result: what was restored and where.
//
// Returns:
//   - error: non-nil when a write fails.
func writeResult(stdout io.Writer, result *history.RestoreResult) error {
	if _, err := fmt.Fprintf(
		stdout,
		"Successfully restored %s to %s\n",
		result.BinaryName,
		result.RestoredTo,
	); err != nil {
		return err //nolint:wrapcheck // The caller wraps every write error once.
	}

	if result.ModulePath != "" {
		if _, err := fmt.Fprintf(stdout, "  Module: %s\n", result.ModulePath); err != nil {
			return err //nolint:wrapcheck // The caller wraps every write error once.
		}
	}

	if result.Version != "" {
		if _, err := fmt.Fprintf(stdout, "  Version: %s\n", result.Version); err != nil {
			return err //nolint:wrapcheck // The caller wraps every write error once.
		}
	}

	return nil
}
