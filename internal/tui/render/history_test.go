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

	// Each row ends with its trash value. A name column sized by runes rather
	// than cells shifts that column, so compare the cell offset of the value
	// rather than its byte index, which differs for a wide name anyway.
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

	// Nine columns is the narrowest the table can render: a row carries the cursor
	// gutter, the date, a divider and the name, and the frame pads both sides.
	// Below that the row has no room to fit at all.
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

			// A row wider than the frame wraps onto a second line, and every
			// wrapped line is still within the width, so the check above misses
			// it. The entry row is the one carrying the cursor glyph, and
			// whatever follows it is blank padding, so any content on the next
			// line is the tail of a row that spilled past the frame.
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
//   - t: test handle used to fail the test when no such row exists.
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
