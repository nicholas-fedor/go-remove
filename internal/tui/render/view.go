/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package render

import (
	tea "charm.land/bubbletea/v2"
)

// newAltScreenView builds a view that owns the alternate screen.
//
// Terminal configuration is a property of the view rather than of the screen it
// draws, so it is set here once instead of at every return in every view.
//
// Parameters:
//   - content: The rendered content.
//
// Returns:
//   - A view configured for the alternate screen.
func newAltScreenView(content string) tea.View {
	view := tea.NewView(content)
	view.AltScreen = true

	return view
}
