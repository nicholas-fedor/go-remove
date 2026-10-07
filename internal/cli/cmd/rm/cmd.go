/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package rm

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/nicholas-fedor/go-remove/internal/cli/flags"
)

// Remover removes one binary from the bin directory.
type Remover interface {
	// Remove moves the named binary to trash and records it in history.
	//
	// Parameters:
	//   - ctx: cancellation for the operation.
	//   - binary: the binary name, relative to the bin directory.
	//   - settings: the validated global options.
	//
	// Returns:
	//   - error: non-nil when the binary cannot be removed.
	Remove(ctx context.Context, binary string, settings flags.Settings) error
}

// NewCommand returns the rm command.
//
// Parameters:
//   - ctx: cancellation for the removal.
//   - stdout: stream the success line is written to.
//   - remover: performs the removal.
//   - global: the persistent flags bound on the root command.
//
// Returns:
//   - *cobra.Command: the rm command.
func NewCommand(
	ctx context.Context,
	stdout io.Writer,
	remover Remover,
	global *flags.Global,
) *cobra.Command {
	return &cobra.Command{
		Use:     "rm <binary>",
		Aliases: []string{"remove"},
		Short:   "Remove a Go binary",
		Long: `Remove the named binary from GOBIN, or GOPATH/bin when GOBIN is not set.

The binary is moved to trash and recorded in history, so "go-remove undo"
or "go-remove restore" can bring it back. Pass --goroot to remove a binary
from GOROOT/bin instead.`,
		Example: `# Remove a binary from GOBIN or GOPATH/bin.
go-remove rm vhs

# Remove a binary from GOROOT/bin, with debug logging.
go-remove --goroot -v rm gofmt`,
		Args: cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			// Failures from here on are about the operation, not the
			// invocation, so the usage block would not help.
			command.SilenceUsage = true

			return run(ctx, stdout, remover, global, args[0])
		},
	}
}

// run validates the arguments and performs one removal.
//
// Parameters:
//   - ctx: cancellation for the removal.
//   - stdout: stream the success line is written to.
//   - remover: performs the removal.
//   - global: the persistent flags.
//   - arg: the raw binary name argument.
//
// Returns:
//   - error: non-nil when validation, removal, or writing the result fails.
func run(
	ctx context.Context,
	stdout io.Writer,
	remover Remover,
	global *flags.Global,
	arg string,
) error {
	binary, err := flags.ValidateBinary(arg)
	if err != nil {
		return fmt.Errorf("validating binary name: %w", err)
	}

	settings, err := global.Resolve()
	if err != nil {
		return fmt.Errorf("resolving flags: %w", err)
	}

	if err := remover.Remove(ctx, binary, settings); err != nil {
		return fmt.Errorf("removing %s: %w", binary, err)
	}

	// Verbose output already reports the removal through the logger.
	if settings.Verbose {
		return nil
	}

	if _, err := fmt.Fprintf(stdout, "Successfully removed %s\n", binary); err != nil {
		return fmt.Errorf("writing output: %w", err)
	}

	return nil
}
