/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package render

import (
	"github.com/nicholas-fedor/go-remove/internal/history"
)

const (
	// ModeBinaries renders the binary selection grid.
	ModeBinaries Mode = "binaries"

	// ModeHistory renders the deletion history table.
	ModeHistory Mode = "history"
)

// Mode selects which screen a State renders.
type Mode string

// State is everything the views need in order to draw a frame.
//
// It is a snapshot rather than a reference to the model, so a view cannot
// observe a change made while it renders, and so the views stay pure.
type State struct {
	// Mode selects the screen.
	Mode Mode

	// Width and Height are the terminal dimensions in cells.
	Width  int
	Height int

	// Styles carries the appearance settings.
	Styles StyleConfig

	// Choices are the binary names to lay out, in display order.
	Choices []string

	// Rows and Cols are the grid dimensions, and CursorX and CursorY locate the
	// selected item within it.
	Rows    int
	Cols    int
	CursorX int
	CursorY int

	// HistoryEntries are the deletion records to display, and HistoryCursor is
	// the selected index within them.
	HistoryEntries []*history.HistoryEntry
	HistoryCursor  int

	// HistoryLoading reports that the entries are still being fetched.
	HistoryLoading bool

	// Status is the current status line, and Busy names the running operation
	// when one is in flight.
	Status string
	Busy   string

	// Confirmation names the destructive action awaiting acknowledgement, or is
	// empty when none is pending.
	Confirmation string

	// ShowLogs reports whether the log panel is visible, and Logs holds the
	// full buffer, which the view trims to what fits.
	Logs     []string
	ShowLogs bool
}

// VisibleLogs returns the log lines that fit in the panel.
//
// Whether the panel is drawn at all is the caller's decision, made before this
// is called.
//
// Parameters:
//   - logs: The full log buffer.
//
// Returns:
//   - The last MaxVisibleLogLines entries, or all of them when there are fewer.
func VisibleLogs(logs []string) []string {
	start := max(len(logs)-MaxVisibleLogLines, 0)

	return logs[start:]
}
