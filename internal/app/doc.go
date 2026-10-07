/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

// Package app is the composition root for go-remove.
//
// Run builds the services the command tree calls, executes the tree, reports
// any error once, and maps it to the process exit code. Loggers and the
// history database are opened only when a command needs them, so help and
// version never touch storage.
package app
