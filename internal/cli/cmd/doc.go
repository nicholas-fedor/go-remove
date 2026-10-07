/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

// Package cmd builds the go-remove command tree.
//
// NewRoot establishes the root command, which opens the interactive binary
// picker, and registers each subcommand. Every subcommand lives in its own
// package under this one and declares the narrow interface it calls, so this
// package only assembles the tree. It parses flags and delegates to injected
// services. It does not open storage, build loggers, or exit the process.
package cmd
