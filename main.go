/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

// Package main provides the entry point for the go-remove command-line tool.
package main

import "github.com/nicholas-fedor/go-remove/cmd"

// main starts the go-remove CLI.
//
// It delegates to cmd.Execute for flag parsing, command dispatch, and process exit handling.
func main() {
	cmd.Execute()
}
