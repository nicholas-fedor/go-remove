/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package cmd

import (
	"context"
	"io"

	"github.com/nicholas-fedor/go-remove/internal/cli/cmd/restore"
	"github.com/nicholas-fedor/go-remove/internal/cli/cmd/rm"
	"github.com/nicholas-fedor/go-remove/internal/cli/cmd/undo"
	"github.com/nicholas-fedor/go-remove/internal/cli/flags"
	buildversion "github.com/nicholas-fedor/go-remove/internal/version"
)

// Browser opens the interactive binary picker.
type Browser interface {
	// Browse runs the TUI over the bin directory until the user quits.
	//
	// Parameters:
	//   - ctx: cancellation for the session.
	//   - settings: the validated global options.
	//
	// Returns:
	//   - error: non-nil when the TUI cannot start or fails.
	Browse(ctx context.Context, settings flags.Settings) error
}

// Dependencies are the process inputs the command tree needs.
//
// The service fields are contracts declared by the packages that consume
// them, so each command depends on the narrowest interface it can use and
// this struct only collects them.
type Dependencies struct {
	// Browser runs the binary picker for the root command.
	Browser Browser

	// Remover performs removals for the rm command.
	Remover rm.Remover

	// Undoer restores the most recent removal for the undo command.
	Undoer undo.Undoer

	// HistoryBrowser runs the history view for the restore command.
	HistoryBrowser restore.Browser

	// Stdout receives command results and requested help.
	Stdout io.Writer

	// Stderr receives the usage text for an invalid invocation.
	Stderr io.Writer

	// Version is the build metadata the version command reports.
	Version buildversion.Info
}
