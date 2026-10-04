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

// HistoryTitleLines covers the view title and the blank line beneath it.
const HistoryTitleLines = 2

// HistoryTableHeaderLines covers the header row and the rule beneath it.
const HistoryTableHeaderLines = 2

// HistoryTrashHeading is the heading of the column reporting trash availability.
const HistoryTrashHeading = "In Trash"

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
	// The frame sets Width(width-LeftPadding) and pads its content by
	// LeftPadding, so the space left for the columns is the width less both.
	// Sizing off a single subtraction over-allocates LeftPadding cells and wraps
	// every row.
	content := max(width-2*LeftPadding, 1)

	// Every row also carries a cursor prefix and a column divider, which come off
	// next.
	free := max(content-DisplayWidth(historyCursorPrefix)-historyColumnDivider, 1)

	trashWidth := len(HistoryTrashHeading)
	dateWidth := min(historyDateWidth, max(free-historyMinNameWidth, 1))
	nameWithoutTrash := max(free-dateWidth-historyColumnDivider, 1)

	// The candidate is left unclamped so the comparison below sees the
	// space that is really left rather than every candidate meeting the minimum.
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

		// Title(2) + header(2) + footer(1) + status(1) + padding(2), which
		// totals minAvailHeightAdjustment so the two views budget alike.
		reservedHeight := 8

		if state.ShowLogs {
			visibleLogCount := min(len(state.Logs), MaxVisibleLogLines)
			if visibleLogCount == 0 {
				// A line is reserved even when the panel is empty, so the panel
				// keeps its height.
				visibleLogCount = 1
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

		// Only what is left below the window is reported, on a freed indicator row.
		showMoreIndicator := indicatorRowFreed && startIdx+visibleCount < entryCount

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
				// The ellipsis takes cells of its own, and a column narrower than
				// it leaves no budget, so the result is capped to the column.
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

			s.WriteString(prefix)
			s.WriteString(row)
			s.WriteString("\n")
		}

		if showMoreIndicator {
			remaining := entryCount - startIdx - visibleCount
			moreMsg := fmt.Sprintf("...and %d more", remaining)
			s.WriteString(footerStyle.Render(moreMsg))
			s.WriteString("\n")
		}
	}

	s.WriteString("\n")

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
// Parameters:
//   - state: The snapshot supplying the terminal dimensions.
//   - body: The rendered content above the footer.
//   - footer: The rendered footer line.
//
// Returns:
//   - The framed content.
func frame(state *State, body, footer string) string {
	// The measurement uses the same style as the render, because the left padding
	// and the width change how many lines the result occupies.
	style := lipgloss.NewStyle().
		PaddingLeft(LeftPadding).
		Width(state.Width - LeftPadding)

	// Measuring what actually renders rather than counting lines keeps the total
	// from drifting as soon as the layout does.
	pad := max(state.Height-lipgloss.Height(style.Render(body+footer)), 0)
	if pad > 0 {
		// The padding goes inside the body so every line keeps its width.
		body += strings.Repeat("\n", pad)
	}

	return style.Render(body + footer)
}
