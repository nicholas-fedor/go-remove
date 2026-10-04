/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package render

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/go-remove/internal/history"
)

// TestViewHistory_AlignsColumnsWithCJKNames verifies every row lines up when a
// name contains wide runes.
//
// fmt's %-*s pads by rune count, so a CJK name is three cells too wide and the
// column drifts, wrapping the row on a terminal that had room for it.
func TestViewHistory_AlignsColumnsWithCJKNames(t *testing.T) {
	t.Parallel()

	entries := []*history.HistoryEntry{
		{ID: "1", Timestamp: time.Now(), BinaryName: "tool", InTrash: true},
		{ID: "2", Timestamp: time.Now(), BinaryName: "日本語のツール", InTrash: false},
		{ID: "3", Timestamp: time.Now(), BinaryName: "héllo-wörld", InTrash: true},
	}

	state := &State{
		HistoryEntries: entries,
		HistoryCursor:  0,
		Mode:           ModeHistory,
		Width:          80,
		Height:         24,
		Styles:         DefaultStyleConfig(),
	}

	lines := strings.Split(stripANSI(History(state).Content), "\n")

	// The rows follow the title, a blank line, the column heading and the rule.
	firstRow := HistoryTitleLines + HistoryTableHeaderLines

	// A name column sized by runes rather than cells would shift the trash
	// column, so compare the cell offset rather than the byte index.
	offsets := make([]int, 0, 3)

	for i := range 3 {
		row := lines[firstRow+i]
		require.NotEmpty(t, row)

		cell := stripANSI(row)

		at := strings.LastIndex(cell, "Yes")
		if at < 0 {
			at = strings.LastIndex(cell, "No")
		}

		require.Positivef(t, at, "row %d has no trash column: %q", i, cell)

		offsets = append(offsets, DisplayWidth(cell[:at]))
	}

	for i := 1; i < len(offsets); i++ {
		assert.Equal(t, offsets[0], offsets[i],
			"row %d has its trash column shifted by the name before it", i)
	}
}

// TestViewHistory_FitsNarrowTerminal verifies the table is sized to the window
// instead of a fixed 50 columns that wrapped on a narrow terminal.
func TestViewHistory_FitsNarrowTerminal(t *testing.T) {
	t.Parallel()

	entry := &history.HistoryEntry{
		ID:         "1",
		Timestamp:  time.Now(),
		BinaryName: "a-fairly-long-binary-name",
		InTrash:    true,
	}

	// Nine columns is the narrowest the table can render; below that the row
	// has no room to fit at all.
	for _, width := range []int{9, 10, 11, 12, 20, 24, 28, 30, 41, 42, 80, 160} {
		t.Run(fmt.Sprintf("width=%d", width), func(t *testing.T) {
			t.Parallel()

			state := &State{
				HistoryEntries: []*history.HistoryEntry{entry},
				HistoryCursor:  0,
				Mode:           ModeHistory,
				Width:          width,
				Height:         24,
				Styles:         DefaultStyleConfig(),
			}

			rendered := stripANSI(History(state).Content)
			lines := strings.Split(rendered, "\n")

			for _, line := range lines {
				assert.LessOrEqual(t, DisplayWidth(line), width,
					"no line may exceed the terminal width, got %d for %q",
					DisplayWidth(line), line)
			}

			// Every wrapped line is still within the width, so a row that spills
			// past the frame is caught as content on the line after the cursor.
			index := cursorRowIndex(t, lines)
			row := lines[index]

			if index+1 < len(lines) {
				assert.Empty(t, strings.TrimSpace(lines[index+1]),
					"the selected row spilled onto the next line at width %d, got %q after %q",
					width, lines[index+1], row)
			}
		})
	}
}

// cursorRowIndex returns the index of the rendered table row for the selected
// entry.
//
// The selected row is the one prefixed with the cursor glyph, which
// DefaultStyleConfig renders as a right-pointing triangle.
//
// Parameters:
//   - t: test handle that fails the test when no such row exists.
//   - lines: rendered output split into lines.
//
// Returns:
//   - Index of the cursor row.
func cursorRowIndex(t *testing.T, lines []string) int {
	t.Helper()

	cursor := DefaultStyleConfig().Cursor

	for index, line := range lines {
		if strings.Contains(line, cursor) {
			return index
		}
	}

	t.Fatalf("no cursor row in %q", lines)

	return 0
}

// historyEntryRows returns the rendered table rows.
//
// A row is a line whose last field is the entry's trash value, which a heading
// does not end in and neither the "...and X more" indicator nor the footer does.
func historyEntryRows(content string) []string {
	lines := strings.Split(stripANSI(content), "\n")
	rows := make([]string, 0, len(lines))

	for _, line := range lines {
		trimmed := strings.TrimRight(line, " ")
		if strings.HasSuffix(trimmed, "Yes") || strings.HasSuffix(trimmed, "No") {
			rows = append(rows, line)
		}
	}

	return rows
}

// historyEntries builds count deletion records with predictable names.
func historyEntries(count int) []*history.HistoryEntry {
	entries := make([]*history.HistoryEntry, 0, count)

	for i := range count {
		entries = append(entries, &history.HistoryEntry{
			ID:         fmt.Sprintf("entry-%d", i),
			Timestamp:  time.Now(),
			BinaryName: fmt.Sprintf("tool-%d", i),
			InTrash:    i%2 == 0,
		})
	}

	return entries
}

// TestViewHistory_SingleRowWindowRendersItsRow verifies a window one row high
// still draws that row, and that the indicator yields its row rather than
// stacking on top of it.
//
// The indicator takes its row out of the budget, so a one-row window must
// keep that row for the entry rather than have the indicator drawn over it.
func TestViewHistory_SingleRowWindowRendersItsRow(t *testing.T) {
	t.Parallel()

	// The reserved height leaves exactly one row for entries at this height.
	state := &State{
		HistoryEntries: historyEntries(3),
		HistoryCursor:  1,
		Mode:           ModeHistory,
		Width:          80,
		Height:         9,
		Styles:         DefaultStyleConfig(),
	}

	rendered := stripANSI(History(state).Content)

	rows := historyEntryRows(rendered)
	require.Len(t, rows, 1, "a one-row window must still render its row")
	assert.Contains(t, rows[0], "tool-1", "the selected entry must be the one shown")
	assert.NotContains(t, rendered, "...and",
		"the indicator has no row of its own and must be suppressed")
}

// TestViewHistory_StatusRowLeavesNoRoomForIndicator verifies a status line that
// reduces the table to a single row suppresses the indicator rather than
// drawing it over the entry, and that the footer still fits.
func TestViewHistory_StatusRowLeavesNoRoomForIndicator(t *testing.T) {
	t.Parallel()

	state := &State{
		HistoryEntries: historyEntries(4),
		HistoryCursor:  0,
		Mode:           ModeHistory,
		Width:          80,
		Height:         8,
		Status:         "Restored tool",
		Styles:         DefaultStyleConfig(),
	}

	content := History(state).Content
	rendered := stripANSI(content)

	assert.NotEmpty(t, historyEntryRows(rendered), "the entry must keep its row")
	assert.NotContains(t, rendered, "...and",
		"the indicator must be suppressed when no row is free for it")

	for line := range strings.SplitSeq(content, "\n") {
		assert.LessOrEqual(t, lipgloss.Height(line), state.Height,
			"every line must stay within the terminal, got %q", line)
	}
}

// TestViewHistory_IndicatorOnlyWhenEntriesRemain verifies the indicator reports
// only what is still below the window, and disappears once nothing is.
//
// The indicator is decided after the window scrolls, so it counts only the
// entries still below it.
func TestViewHistory_IndicatorOnlyWhenEntriesRemain(t *testing.T) {
	t.Parallel()

	entries := historyEntries(6)

	tests := []struct {
		name     string
		cursor   int
		wantRows int
		wantMore string
	}{
		{name: "at the top", cursor: 0, wantRows: 2, wantMore: "...and 4 more"},
		{name: "in the middle", cursor: 2, wantRows: 2, wantMore: "...and 3 more"},
		{name: "at the bottom", cursor: 5, wantRows: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Three rows are available, and the indicator takes one of them.
			state := &State{
				HistoryEntries: entries,
				HistoryCursor:  tt.cursor,
				Mode:           ModeHistory,
				Width:          80,
				Height:         11,
				Styles:         DefaultStyleConfig(),
			}

			rendered := stripANSI(History(state).Content)

			assert.Len(t, historyEntryRows(rendered), tt.wantRows,
				"the window must keep its rows")

			if tt.wantMore == "" {
				assert.NotContains(t, rendered, "...and",
					"nothing is left below the window, so the indicator must be gone")

				return
			}

			assert.Contains(t, rendered, tt.wantMore)
		})
	}
}

// TestViewHistory_NoDoubleReservation verifies the indicator is paid for once,
// so scrolling does not quietly shrink the window.
//
// The row is taken once, so scrolling to the end gives the same number of
// entry rows back.
func TestViewHistory_NoDoubleReservation(t *testing.T) {
	t.Parallel()

	entries := historyEntries(8)

	state := &State{
		HistoryEntries: entries,
		HistoryCursor:  0,
		Mode:           ModeHistory,
		Width:          80,
		Height:         11,
		Styles:         DefaultStyleConfig(),
	}

	rendered := stripANSI(History(state).Content)

	assert.Len(t, historyEntryRows(rendered), 2,
		"the indicator must cost one row, not two")
	assert.Contains(t, rendered, "...and 6 more")

	// At the end of the list there is nothing below the window, so the row the
	// indicator would have taken goes back to the entries.
	state.HistoryCursor = len(entries) - 1

	rendered = stripANSI(History(state).Content)

	assert.Len(t, historyEntryRows(rendered), 2,
		"scrolling must not reserve the indicator a second time")
	assert.NotContains(t, rendered, "...and",
		"the last entry leaves nothing below the window")
}

// TestViewHistory_DropsTrashColumnWhenItCannotFit verifies the trash column is
// dropped once the space left for it falls under the minimum name width.
//
// The candidate name width must be compared before it is clamped. Clamping it
// first would lift every candidate to the minimum and pin the trash column on
// at every width.
func TestViewHistory_DropsTrashColumnWhenItCannotFit(t *testing.T) {
	t.Parallel()

	entry := &history.HistoryEntry{
		ID:         "1",
		Timestamp:  time.Now(),
		BinaryName: "a-fairly-long-binary-name",
		InTrash:    true,
	}

	tests := []struct {
		name      string
		width     int
		wantTrash bool
	}{
		{name: "fits", width: 80, wantTrash: true},
		{name: "just fits", width: 41, wantTrash: true},
		{name: "one cell short", width: 40, wantTrash: false},
		{name: "narrow", width: 30, wantTrash: false},
		{name: "very narrow", width: 20, wantTrash: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			state := &State{
				HistoryEntries: []*history.HistoryEntry{entry},
				HistoryCursor:  0,
				Mode:           ModeHistory,
				Width:          tt.width,
				Height:         24,
				Styles:         DefaultStyleConfig(),
			}

			rendered := stripANSI(History(state).Content)

			assert.Equal(t, tt.wantTrash, strings.Contains(rendered, HistoryTrashHeading),
				"trash heading presence at width %d", tt.width)
			assert.Equal(t, tt.wantTrash, strings.Contains(rendered, "Yes"),
				"trash value presence at width %d", tt.width)
		})
	}
}
