/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

// Package errmsg classifies a history error into a category.
//
// A Kind describes what happened, never how it should be worded. Each interface
// maps the returned Kind to its own text, so a TUI status line and a process
// error can describe the same failure differently without either re-deriving the
// category for itself.
package errmsg
