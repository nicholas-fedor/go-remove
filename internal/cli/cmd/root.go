/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nicholas-fedor/go-remove/internal/cli/cmd/restore"
	"github.com/nicholas-fedor/go-remove/internal/cli/cmd/rm"
	"github.com/nicholas-fedor/go-remove/internal/cli/cmd/undo"
	"github.com/nicholas-fedor/go-remove/internal/cli/cmd/version"
	"github.com/nicholas-fedor/go-remove/internal/cli/flags"
)

// NewRoot returns the go-remove command tree wired to deps.
//
// Each call builds a new tree, so no flag or silence state carries over from
// one execution to the next. Errors are not printed here. The caller reports
// them once. SilenceUsage stays off so a bad flag or argument count still
// shows usage, and each RunE switches it on for an error from the work.
//
// Parameters:
//   - ctx: cancellation for every command.
//   - deps: services and streams the commands use.
//
// Returns:
//   - *cobra.Command: the root command with every subcommand registered.
//
//nolint:gocritic // hugeParam: Dependencies is wiring, read once per tree.
func NewRoot(ctx context.Context, deps Dependencies) *cobra.Command {
	global := flags.New()

	root := &cobra.Command{
		Use:   "go-remove",
		Short: "A tool to remove Go binaries",
		Long: `go-remove removes binaries installed with "go install", moving them to
trash and recording each removal so it can be undone.

Run it with no command to pick binaries in an interactive view.`,
		Example: `# Pick binaries to remove in the interactive view.
go-remove

# Remove one binary.
go-remove rm vhs

# Bring back the last binary removed.
go-remove undo

# Pick an earlier removal to restore.
go-remove restore`,
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		RunE: func(command *cobra.Command, _ []string) error {
			// Failures from here on are about the operation, not the
			// invocation, so the usage block would not help.
			command.SilenceUsage = true

			return runRootCmd(ctx, deps.Browser, global)
		},
	}

	// Cobra prints usage for an invalid invocation to the Out stream, so Out is
	// stderr. Help the user asked for is a result, so it goes to stdout.
	root.SetOut(deps.Stderr)
	root.SetErr(deps.Stderr)

	help := root.HelpFunc()
	root.SetHelpFunc(func(command *cobra.Command, args []string) {
		command.SetOut(deps.Stdout)
		help(command, args)
	})

	flags.BindGlobal(root, global)
	registerCmds(ctx, root, deps, global)

	return root
}

// registerCmds adds every subcommand to the root command.
//
// Parameters:
//   - ctx: cancellation for every subcommand.
//   - root: the root command the subcommands are added to.
//   - deps: services and streams the subcommands use.
//   - global: the persistent flags bound on root.
//
//nolint:gocritic // hugeParam: Dependencies is wiring, read once per tree.
func registerCmds(
	ctx context.Context,
	root *cobra.Command,
	deps Dependencies,
	global *flags.Global,
) {
	root.AddCommand(
		rm.NewCommand(ctx, deps.Stdout, deps.Remover, global),
		undo.NewCommand(ctx, deps.Stdout, deps.Undoer, global),
		restore.NewCommand(ctx, deps.HistoryBrowser, global),
		version.NewCommand(deps.Stdout, deps.Version),
	)
}

// runRootCmd resolves the global flags and opens the binary picker.
//
// Parameters:
//   - ctx: cancellation for the session.
//   - browser: runs the binary picker.
//   - global: the persistent flags.
//
// Returns:
//   - error: non-nil when the flags are invalid or the picker fails.
func runRootCmd(ctx context.Context, browser Browser, global *flags.Global) error {
	settings, err := global.Resolve()
	if err != nil {
		return fmt.Errorf("resolving flags: %w", err)
	}

	if err := browser.Browse(ctx, settings); err != nil {
		return fmt.Errorf("opening binary picker: %w", err)
	}

	return nil
}
