/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

// Package models holds the Bubble Tea model behind the interactive interface.
//
// The model owns the state the TUI reads and the keys it acts on. Every
// rendering decision belongs to the sibling render package, so what is left
// here is the state, the transitions between views, and the operations that run
// outside the update loop so the view stays responsive and interruptible.
package models
