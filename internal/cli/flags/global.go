/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package flags

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nicholas-fedor/go-remove/internal/logger"
)

// Global flag names, shorthands, and defaults.
const (
	// flagVerbose is the verbose flag name.
	flagVerbose = "verbose"

	// flagVerboseShort is the shorthand for --verbose.
	flagVerboseShort = "v"

	// flagLogLevel is the log level flag name.
	flagLogLevel = "log-level"

	// flagLogLevelShort is the shorthand for --log-level.
	flagLogLevelShort = "l"

	// flagGoroot is the GOROOT flag name.
	flagGoroot = "goroot"

	// defaultLogLevel is the log level used when --log-level is not given.
	defaultLogLevel = "info"
)

// Global holds the raw values of the persistent flags shared by every command.
type Global struct {
	// LogLevel is the log level name given with --log-level.
	LogLevel string

	// Verbose enables verbose output and debug logging.
	Verbose bool

	// Goroot targets GOROOT/bin instead of GOBIN or GOPATH/bin.
	Goroot bool
}

// Settings are the validated global options for one command execution.
type Settings struct {
	// LogLevel is the log level name, kept for the TUI's own log view.
	LogLevel string

	// Level is the parsed log level, raised to debug by Verbose.
	Level logger.Level

	// Verbose enables verbose output and debug logging.
	Verbose bool

	// Goroot targets GOROOT/bin instead of GOBIN or GOPATH/bin.
	Goroot bool
}

// New returns the global flag defaults before cobra parses arguments.
//
// Returns:
//   - *Global: info level, with verbose and goroot off.
func New() *Global {
	return &Global{
		LogLevel: defaultLogLevel,
		Verbose:  false,
		Goroot:   false,
	}
}

// BindGlobal registers the persistent flags on the root command.
//
// Parameters:
//   - command: the root command that owns the persistent flag set.
//   - global: destination for the parsed values.
func BindGlobal(command *cobra.Command, global *Global) {
	command.PersistentFlags().BoolVarP(
		&global.Verbose,
		flagVerbose,
		flagVerboseShort,
		false,
		"Enable verbose output",
	)
	command.PersistentFlags().StringVarP(
		&global.LogLevel,
		flagLogLevel,
		flagLogLevelShort,
		defaultLogLevel,
		"Set log level (debug, info, warn, error)",
	)
	command.PersistentFlags().BoolVar(
		&global.Goroot,
		flagGoroot,
		false,
		"Target GOROOT/bin instead of GOBIN or GOPATH/bin",
	)
}

// Resolve validates the global flags.
//
// The level applies on its own, so --log-level does something without
// --verbose, and --verbose raises it to debug.
//
// Returns:
//   - Settings: the validated options.
//   - error: wraps [logger.ErrInvalidLogLevel] for an unrecognized level.
func (g *Global) Resolve() (Settings, error) {
	level, err := logger.ParseLogLevel(g.LogLevel)
	if err != nil {
		return Settings{}, fmt.Errorf("parsing --%s: %w", flagLogLevel, err)
	}

	if g.Verbose {
		level = logger.DebugLevel
	}

	return Settings{
		LogLevel: g.LogLevel,
		Level:    level,
		Verbose:  g.Verbose,
		Goroot:   g.Goroot,
	}, nil
}
