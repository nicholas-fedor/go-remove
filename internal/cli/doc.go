/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

// Package cli holds the command-layer use case for go-remove.
//
// Config carries the parsed command-line options, Dependencies carries the
// filesystem, logger, and optional history manager the use case needs, and Run
// performs one removal: a named binary is deleted, or the interactive
// terminal interface is started when no binary was named.
package cli
