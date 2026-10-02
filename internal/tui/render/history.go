/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package render

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	tea "charm.land/bubbletea/v2"
)

// Confirmation identifies a destructive action awaiting acknowledgement.
const (
	// ConfirmNone means nothing is pending.
	ConfirmNone = ""

	// ConfirmClearAll is the prompt for clearing every history entry.
	ConfirmClearAll = "clear_all"

	// ConfirmDeletePerm is the prompt for permanently deleting one entry.
	ConfirmDeletePerm = "delete_permanent"
)

// History table layout constants. These must stay consistent with the rows the
// table writes.
const (
	dateTimeFormat      = "2006-01-02 15:04" // Format for displaying timestamps
	separatorAdjustment = 2                  // Extra width for the column separator

	historyDateHeading   = "Date/Time"
	historyDateWidth     = 16    // Widest date the table shows before yielding space
	historyMinNameWidth  = 8     // Narrowest the name column may become before it is useless
	historyColumnDivider = 1     // Spaces between two columns
	historyEllipsis      = "..." // Trailing ellipsis on a name shortened to fit its column

	// historyCursorPrefix is the gutter every row carries, selected or not.
	historyCursorPrefix = "  "
)

// historyColumns holds the widths chosen for one frame's table.
type historyColumns struct {
	date      int
	name      int
	trash     int
	showTrash bool
	content   int
}

// sizeHistoryColumns derives the table widths from the terminal width.
//
// Columns are sized from the space the frame actually leaves rather than a
// fixed total. The frame sets Width(width-leftPadding) and pads its content by
// leftPadding, so the space left for the columns is the width less both. Sizing
// off a single subtraction over-allocates leftPadding cells and wraps every
// row. Every row also carries a cursor prefix, which comes off next.
//
// The trash column is dropped first because it is the least informative, and
// neither remaining column is forced wider than the space available, so a
// narrow window truncates rather than wrapping.
//
// Parameters:
//   - width: Terminal width in cells.
//
// Returns:
//   - The column widths for this frame.
func sizeHistoryColumns(width int) historyColumns {
	content := max(width-2*LeftPadding, 1)
	free := max(content-DisplayWidth(historyCursorPrefix)-historyColumnDivider, 1)

	trashWidth := len(HistoryTrashHeading)
	dateWidth := min(historyDateWidth, max(free-historyMinNameWidth, 1))
	nameWithoutTrash := max(free-dateWidth-historyColumnDivider, 1)

	// The candidate is deliberately left unclamped so it reflects the space
	// that is really left. Clamping it before the comparison would make every
	// candidate meet the minimum and pin the trash column on even when it does
	// not fit.
	nameWithTrashCandidate := max(
		free-dateWidth-2*historyColumnDivider-trashWidth,
		1,
	)

	showTrash := nameWithTrashCandidate >= historyMinNameWidth
	nameWidth := nameWithoutTrash

	if showTrash {
		nameWidth = max(nameWithTrashCandidate, historyMinNameWidth)
	}

	return historyColumns{
		date:      dateWidth,
		name:      nameWidth,
		trash:     trashWidth,
		showTrash: showTrash,
		content:   content,
	}
}

// History renders the deletion history view.
//
// Parameters:
//   - state: The snapshot to draw.
//
// Returns:
//   - A Bubble Tea view listing deletion history.
func History(state *State) tea.View {
	// Apply configured styles for UI elements.
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(state.Styles.TitleColor))
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(state.Styles.HistoryColor))
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(state.Styles.CursorColor))
	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(state.Styles.FooterColor))
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(state.Styles.StatusColor))
	logStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(state.Styles.LogColor))
	trashYesStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(state.Styles.TrashYesColor))
	trashNoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(state.Styles.TrashNoColor))

	var s strings.Builder

	s.WriteString(titleStyle.Render("Deletion History\n"))
	s.WriteString("\n")

	// Calculate visible count first for use in both rendering and height calculation
	var (
		visibleCount      int
		entryCount        int
		maxVisibleEntries int
	)

	switch {
	case state.HistoryLoading:
		s.WriteString("Loading history...\n")
	case len(state.HistoryEntries) == 0:
		s.WriteString("No deletion history found.\n")
	default:
		columns := sizeHistoryColumns(state.Width)

		// Table header
		dateHeading := TruncateToWidth(historyDateHeading, columns.date)

		header := PadToWidth(dateHeading, columns.date) +
			strings.Repeat(" ", historyColumnDivider) +
			PadToWidth("Binary", columns.name)

		if columns.showTrash {
			header += strings.Repeat(" ", historyColumnDivider) +
				PadToWidth(HistoryTrashHeading, columns.trash)
		}

		s.WriteString(headerStyle.Render(header))
		s.WriteString("\n")
		s.WriteString(strings.Repeat("─", min(
			columns.date+columns.name+separatorAdjustment,
			columns.content,
		)))
		s.WriteString("\n")

		// Calculate available height for history entries
		// Reserve space for: title(2) + header(2) + footer(1) + status(1) + padding(2)
		reservedHeight := 8

		if state.ShowLogs {
			// Reserve additional space for log panel (header + separator + lines)
			visibleLogCount := min(len(state.Logs), MaxVisibleLogLines)
			if visibleLogCount == 0 {
				visibleLogCount = 1 // Placeholder line
			}

			reservedHeight += visibleLogCount + LogPanelSeparatorLines
		}

		maxVisibleEntries = max(state.Height-reservedHeight, 1)
		entryCount = len(state.HistoryEntries)

		// The last row of the budget always goes to an entry: an indicator on its
		// own says nothing about what it is reporting.
		rowBudget := maxVisibleEntries

		indicatorRowFreed := entryCount > maxVisibleEntries && rowBudget > 1
		if indicatorRowFreed {
			rowBudget--
		}

		// Ensure the cursor is within the visible range, scrolling the window
		// when it is not.
		startIdx := 0

		visibleCount = min(entryCount, rowBudget)

		if state.HistoryCursor >= visibleCount {
			startIdx = state.HistoryCursor - visibleCount + 1
			visibleCount = min(entryCount-startIdx, rowBudget)
		}

		// What is left below the window is all the indicator can honestly
		// report, so it appears only while something remains and there is a row
		// to show it on.
		showMoreIndicator := indicatorRowFreed && startIdx+visibleCount < entryCount

		// Table rows - display only visible entries
		for i := range visibleCount {
			entryIdx := startIdx + i
			if entryIdx >= entryCount {
				break
			}

			entry := state.HistoryEntries[entryIdx]

			prefix := historyCursorPrefix
			if entryIdx == state.HistoryCursor {
				prefix = cursorStyle.Render(state.Styles.Cursor)
			}

			dateStr := entry.Timestamp.Format(dateTimeFormat)

			nameStr := entry.BinaryName
			if DisplayWidth(nameStr) > columns.name {
				// The ellipsis occupies cells of its own, so the name is
				// shortened into what the column has left once they are paid
				// for. A column narrower than the ellipsis leaves no budget at
				// all, which would put the ellipsis past the column edge, so the
				// result is capped to the column afterwards.
				budget := max(columns.name-DisplayWidth(historyEllipsis), 0)

				nameStr = TruncateToWidth(nameStr, budget) + historyEllipsis
				nameStr = TruncateToWidth(nameStr, columns.name)
			}

			var trashStr string
			if entry.InTrash {
				trashStr = trashYesStyle.Render("Yes")
			} else {
				trashStr = trashNoStyle.Render("No")
			}

			row := PadToWidth(TruncateToWidth(dateStr, columns.date), columns.date) +
				strings.Repeat(" ", historyColumnDivider) +
				PadToWidth(nameStr, columns.name)

			if columns.showTrash {
				row += strings.Repeat(" ", historyColumnDivider) +
					PadToWidth(trashStr, columns.trash)
			}

			s.WriteString(prefix + row)
			s.WriteString("\n")
		}

		// Show indicator if there are more entries
		if showMoreIndicator {
			remaining := entryCount - startIdx - visibleCount
			moreMsg := fmt.Sprintf("...and %d more", remaining)
			s.WriteString(footerStyle.Render(moreMsg))
			s.WriteString("\n")
		}
	}

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

	// Show confirmation dialog if active
	switch state.Confirmation {
	case ConfirmClearAll:
		s.WriteString(statusStyle.Render("Clear all history? This cannot be undone. (y/n)"))
		s.WriteString("\n")
	case ConfirmDeletePerm:
		if state.HistoryCursor < len(state.HistoryEntries) {
			entry := state.HistoryEntries[state.HistoryCursor]
			s.WriteString(
				statusStyle.Render(fmt.Sprintf("Permanently delete %s? (y/n)", entry.BinaryName)),
			)
			s.WriteString("\n")
		}
	default:
		if state.Status != "" {
			s.WriteString(statusStyle.Render(state.Status))
			s.WriteString("\n")
		}
	}

	// Footer with history-specific key bindings
	var footerText string

	switch {
	case state.Confirmation != ConfirmNone:
		footerText = "y: confirm  n: cancel"
	default:
		footerText = "↑/k: up  ↓/j: down  Enter: restore  d: delete  c: clear entry  C: clear all  b: back  u: undo  L: logs  q: quit"
	}

	return newAltScreenView(frame(state, s.String(), footerStyle.Render(footerText)))
}

// frame pads the body to fill the terminal and wraps it in the left-padded frame.
//
// Padding is measured from what actually renders rather than from a hand-counted
// total that drifts as soon as the layout does. The measurement uses the same
// style as the render, because left padding and the width can change how many
// lines the result occupies, and the padding goes inside the body so every line
// keeps its width.
//
// Parameters:
//   - state: The snapshot supplying the terminal dimensions.
//   - body: The rendered content above the footer.
//   - footer: The rendered footer line.
//
// Returns:
//   - The framed content.
func frame(state *State, body, footer string) string {
	style := lipgloss.NewStyle().
		PaddingLeft(LeftPadding).
		Width(state.Width - LeftPadding)

	pad := max(state.Height-lipgloss.Height(style.Render(body+footer)), 0)
	if pad > 0 {
		body += strings.Repeat("\n", pad)
	}

	return style.Render(body + footer)
}

// HistoryTitleLines covers the view title and the blank line beneath it.
const HistoryTitleLines = 2

// HistoryTableHeaderLines covers the header row and the rule beneath it.
const HistoryTableHeaderLines = 2

// HistoryTrashHeading is the heading of the column reporting trash availability.
const HistoryTrashHeading = "In Trash"
