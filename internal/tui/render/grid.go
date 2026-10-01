/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package render

// Layout constants for the choice grid. These must stay consistent with the
// view functions that draw the result.
const (
	ColWidthPadding          = 3 // Padding added to column width for spacing
	availWidthAdjustment     = 4 // Adjustment to width for border and padding
	minAvailHeightAdjustment = 8 // Minimum height adjustment for title, footer and padding
	MaxVisibleLogLines       = 5 // Maximum number of log lines to display
	LogPanelSeparatorLines   = 2 // Number of separator lines for log panel
)

// GridInput describes what a grid layout is computed from.
type GridInput struct {
	// Choices are the names to lay out, in display order.
	Choices []string

	// Width and Height are the terminal dimensions in cells.
	Width  int
	Height int

	// Status is the current status line. A non-empty status costs one row.
	Status string

	// ShowLogs reports whether the log panel is visible, and LogCount is how
	// many lines it currently holds.
	ShowLogs bool
	LogCount int

	// CursorX and CursorY are the current cursor position, which survives a
	// resize when it still addresses a valid item.
	CursorX int
	CursorY int
}

// Grid is a computed column-major layout for a choice list.
type Grid struct {
	// Cols and Rows are the grid dimensions.
	Cols int
	Rows int

	// CursorX and CursorY are the cursor clamped to the grid.
	CursorX int
	CursorY int
}

// GridLayout computes the grid dimensions for a set of choices in a terminal.
//
// Parameters:
//   - input: The choices and the space available for them.
//
// Returns:
//   - The layout, with the cursor clamped inside it.
func GridLayout(input *GridInput) Grid {
	// Column width is driven by the longest name, measured in cells so a CJK
	// or emoji name is not under-counted.
	maxNameLen := 0
	for _, choice := range input.Choices {
		if DisplayWidth(choice) > maxNameLen {
			maxNameLen = DisplayWidth(choice)
		}
	}

	colWidth := maxNameLen + ColWidthPadding
	availWidth := input.Width - availWidthAdjustment

	// The status line takes one row when present. The constant height adjustment
	// does not account for it dynamically.
	statusAdjustment := 0
	if input.Status != "" {
		statusAdjustment = 1
	}

	availHeight := max(input.Height-minAvailHeightAdjustment-statusAdjustment, 1)

	if input.ShowLogs {
		// Reserve up to the visible limit for the panel, plus its separator.
		visibleLogCount := min(input.LogCount, MaxVisibleLogLines)
		if visibleLogCount == 0 {
			// Empty log panel: header and placeholder still occupy a row.
			visibleLogCount = 1
		}

		availHeight = max(availHeight-visibleLogCount-LogPanelSeparatorLines, 1)
	}

	if len(input.Choices) == 0 {
		return Grid{}
	}

	// Maximise rows, then bound columns by the width available.
	maxCols := max(availWidth/colWidth, 1)

	rows := min(availHeight, len(input.Choices))
	if rows == 0 {
		rows = 1
	}

	cols := min(maxCols, (len(input.Choices)+rows-1)/rows)

	grid := Grid{Cols: cols, Rows: rows, CursorX: input.CursorX, CursorY: input.CursorY}

	return clampCursor(grid, len(input.Choices))
}

// clampCursor moves the cursor inside the grid after a resize.
//
// A column-major grid addresses item index as y + x*rows, so a cursor left over
// from a wider layout can point past the last item.
//
// Parameters:
//   - grid: The computed layout.
//   - count: Number of items laid out.
//
// Returns:
//   - The grid with its cursor inside the item range.
func clampCursor(grid Grid, count int) Grid {
	if grid.CursorX >= grid.Cols {
		grid.CursorX = grid.Cols - 1
	}

	if grid.CursorY >= grid.Rows {
		grid.CursorY = grid.Rows - 1
	}

	if grid.CursorY+grid.CursorX*grid.Rows >= count {
		lastIdx := count - 1
		grid.CursorX = lastIdx / grid.Rows
		grid.CursorY = lastIdx % grid.Rows
	}

	return grid
}
