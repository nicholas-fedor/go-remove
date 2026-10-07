/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

// Package undo defines the undo command, which restores the most recently
// deleted binary.
//
// History failures are mapped to the sentinels in this package, so the
// message names what went wrong rather than how the history layer reports it.
package undo
