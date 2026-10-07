/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package restore

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nicholas-fedor/go-remove/internal/cli/flags"
)

// Browser opens the interactive history view.
type Browser interface {
	// BrowseHistory runs the TUI in history mode until the user quits.
	//
	// Parameters:
	//   - ctx: cancellation for the session.
	//   - settings: the validated global options.
	//
	// Returns:
	//   - error: non-nil when the TUI cannot start or fails.
	BrowseHistory(ctx context.Context, settings flags.Settings) error
}

// NewCommand returns the restore command.
//
// Parameters:
//   - ctx: cancellation for the session.
//   - browser: runs the history view.
//   - global: the persistent flags bound on the root command.
//
// Returns:
//   - *cobra.Command: the restore command.
func NewCommand(ctx context.Context, browser Browser, global *flags.Global) *cobra.Command {
	return &cobra.Command{
		Use:   "restore",
		Short: "Pick a removed binary to restore from history",
		Long: `Open the interactive history view, which lists every recorded removal and
restores the one you pick.

Use "go-remove undo" to restore the most recent removal without the view.`,
		Example: `# Browse the removal history.
go-remove restore`,
		Args: cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			// Failures from here on are about the operation, not the
			// invocation, so the usage block would not help.
			command.SilenceUsage = true

			settings, err := global.Resolve()
			if err != nil {
				return fmt.Errorf("resolving flags: %w", err)
			}

			if err := browser.BrowseHistory(ctx, settings); err != nil {
				return fmt.Errorf("opening history view: %w", err)
			}

			return nil
		},
	}
}
