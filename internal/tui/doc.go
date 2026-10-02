/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

// Package tui is the entry point to the interactive interface.
//
// It resolves the terminal, lists the binaries on offer, and hands the model to
// Bubble Tea. Everything the interface decides lives in the sibling models
// package, and everything it draws lives in the render package, so what remains
// here is the wiring between them and the start of the program.
//
// The interface is the whole of go-remove when no binary name is given. Without
// a terminal there is nothing to drive it, so Run refuses rather than opening a
// view the user cannot leave, and points at the argument form instead.
package tui
