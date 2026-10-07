/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package app

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/nicholas-fedor/go-remove/internal/cli/cmd"
	"github.com/nicholas-fedor/go-remove/internal/version"
)

// Process exit codes.
const (
	// exitOK means the command succeeded.
	exitOK = 0

	// exitFailure means the command failed.
	exitFailure = 1
)

// Run executes go-remove with the process arguments and streams.
//
// Parameters:
//   - ctx: cancellation for the whole invocation, canceled by an interrupt.
//
// Returns:
//   - int: the process exit code.
func Run(ctx context.Context) int {
	return run(ctx, os.Args[1:], os.Stdout, os.Stderr, newServices())
}

// run executes the command tree with explicit arguments, streams, and
// services, so tests can drive it without the process globals.
//
// Parameters:
//   - ctx: cancellation for the whole invocation.
//   - args: command-line arguments, without the program name.
//   - stdout: stream for results and help.
//   - stderr: stream for usage on an invalid invocation, and for the error.
//   - svc: the services the commands call.
//
// Returns:
//   - int: the process exit code.
func run(
	ctx context.Context,
	args []string,
	stdout io.Writer,
	stderr io.Writer,
	svc *services,
) int {
	root := cmd.NewRoot(ctx, cmd.Dependencies{
		Browser:        svc,
		Remover:        svc,
		Undoer:         svc,
		HistoryBrowser: svc,
		Stdout:         stdout,
		Stderr:         stderr,
		Version:        version.Current(),
	})
	root.SetArgs(args)

	if err := root.ExecuteContext(ctx); err != nil {
		// Nothing useful can be done if stderr itself cannot be written.
		_, _ = fmt.Fprintf(stderr, "Error: %v\n", err)

		return exitFailure
	}

	return exitOK
}
