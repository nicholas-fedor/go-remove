/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package version

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	buildversion "github.com/nicholas-fedor/go-remove/internal/version"
)

// NewCommand returns the version command.
//
// Parameters:
//   - stdout: stream the build metadata is written to.
//   - info: the build metadata to report.
//
// Returns:
//   - *cobra.Command: the version command.
func NewCommand(stdout io.Writer, info buildversion.Info) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long: `Print the version, commit, and build time of the installed binary.

Release builds are stamped at link time. A build that was never stamped
reports Go's module version instead, such as a pseudo-version or (devel).`,
		Args: cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			command.SilenceUsage = true

			if _, err := fmt.Fprintf(
				stdout,
				"go-remove %s\n  Commit: %s\n  Built:  %s\n",
				info.Version,
				info.CommitSHA,
				info.BuildTime,
			); err != nil {
				return fmt.Errorf("writing output: %w", err)
			}

			return nil
		},
	}
}
