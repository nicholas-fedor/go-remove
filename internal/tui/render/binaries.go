/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package render

import (
	"strings"

	"charm.land/lipgloss/v2"

	tea "charm.land/bubbletea/v2"
)

// Binaries renders the binary selection view.
//
// Parameters:
//   - state: The snapshot to draw.
//
// Returns:
//   - A Bubble Tea view listing the binaries in a grid.
func Binaries(state *State) tea.View {
	if len(state.Choices) == 0 {
		return newAltScreenView("No binaries found.\n")
	}

	// Apply configured styles for UI elements.
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(state.Styles.TitleColor))
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(state.Styles.CursorColor))
	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(state.Styles.FooterColor))
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(state.Styles.StatusColor))
	logStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(state.Styles.LogColor))

	// Calculate column width based on the longest binary name.
	var maxNameLen int
	for _, choice := range state.Choices {
		if DisplayWidth(choice) > maxNameLen {
			maxNameLen = DisplayWidth(choice)
		}
	}

	colWidth := maxNameLen + ColWidthPadding

	// Build the grid of binary choices with cursor highlighting.
	var grid strings.Builder

	for row := range state.Rows {
		for col := range state.Cols {
			idx := row + col*state.Rows // Column-major index (fill down columns)
			if idx >= len(state.Choices) {
				break
			}

			prefix := "  "
			if row == state.CursorY && col == state.CursorX {
				prefix = cursorStyle.Render(state.Styles.Cursor)
			}

			item := state.Choices[idx]
			visibleLen := visibleLenPrefix + DisplayWidth(item)
			padding := max(colWidth-visibleLen, 0)
			cell := prefix + item + strings.Repeat(" ", padding)
			grid.WriteString(cell)
		}

		grid.WriteString("\n")
	}

	// Assemble the full TUI layout: title, grid, logs (if visible), status, and footer.
	var s strings.Builder

	s.WriteString(titleStyle.Render("Select a binary to remove:\n"))
	s.WriteString("\n")
	s.WriteString(grid.String())
	s.WriteString("\n")

	// Render log panel if enabled
	if state.ShowLogs {
		visibleLogs := VisibleLogs(state.Logs)

		s.WriteString(logStyle.Render("─ Log Messages ─"))
		s.WriteString("\n")

		if len(visibleLogs) == 0 {
			s.WriteString(logStyle.Render("No log messages yet"))
			s.WriteString("\n")
		} else {
			for _, logEntry := range visibleLogs {
				s.WriteString(logStyle.Render(logEntry))
				s.WriteString("\n")
			}
		}

		s.WriteString("\n")
	}

	switch {
	case state.Busy != "":
		s.WriteString(statusStyle.Render("Working: " + state.Busy + " (ctrl+c to stop)"))
		s.WriteString("\n")
	case state.Status != "":
		s.WriteString(statusStyle.Render(state.Status))
		s.WriteString("\n")
	}

	// Update footer to include new key bindings
	footerText := "↑/k: up  ↓/j: down  ←/h: left  →/l: right  Enter: remove  s: sort  r: history  u: undo  L: logs  q: quit"
	footer := footerStyle.Render(footerText)

	// Pad between the content and the footer from what actually renders, rather
	// than from a hand-counted total that drifts as soon as the layout does.
	// The measurement uses the same style as the render, because left padding
	// and the width can change how many lines the result occupies, and the
	// padding goes inside the body so every line keeps its width.
	frame := lipgloss.NewStyle().
		PaddingLeft(LeftPadding).
		Width(state.Width - LeftPadding)

	body := s.String()
	pad := max(state.Height-lipgloss.Height(frame.Render(body+footer)), 0)

	if pad > 0 {
		body += strings.Repeat("\n", pad)
	}

	content := frame.Render(body + footer)

	return newAltScreenView(content)
}
