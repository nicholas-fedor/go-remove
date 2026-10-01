/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package render

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGridLayout_Cursor verifies the cursor survives a resize and is clamped
// into range when it does not.
func TestGridLayout_Cursor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    GridInput
		wantX    int
		wantY    int
		wantCols int
		wantRows int
	}{
		{
			name: "a valid cursor is preserved",
			input: GridInput{
				Choices: []string{"a", "b", "c", "d", "e", "f"},
				Width:   100,
				Height:  24,
				CursorY: 3,
			},
			wantCols: 1,
			wantRows: 6,
			wantX:    0,
			wantY:    3,
		},
		{
			name: "a cursor past the last row is clamped",
			input: GridInput{
				Choices: []string{"a", "b", "c"},
				Width:   100,
				Height:  24,
				CursorY: 9,
			},
			wantCols: 1,
			wantRows: 3,
			wantX:    0,
			wantY:    2,
		},
		{
			name: "a cursor past the last item is moved onto it",
			// A 3x2 grid over 5 items leaves index 5 out of range, so the
			// cursor must land on the final item rather than past it.
			input: GridInput{
				Choices: []string{"a", "b", "c", "d", "e"},
				Width:   20,
				Height:  10,
				CursorX: 2,
				CursorY: 1,
			},
			wantCols: 3,
			wantRows: 2,
			wantX:    2,
			wantY:    0,
		},
		{
			name: "a cursor past the last column is clamped",
			input: GridInput{
				Choices: []string{"a", "b", "c", "d", "e"},
				Width:   20,
				Height:  10,
				CursorX: 9,
				CursorY: 0,
			},
			wantCols: 3,
			wantRows: 2,
			wantX:    2,
			wantY:    0,
		},
		{
			name: "no choices yields an empty grid",
			input: GridInput{
				Choices: nil,
				Width:   80,
				Height:  24,
				CursorX: 4,
				CursorY: 4,
			},
			wantCols: 0,
			wantRows: 0,
			wantX:    0,
			wantY:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := GridLayout(&tt.input)

			assert.Equal(t, tt.wantCols, got.Cols, "columns")
			assert.Equal(t, tt.wantRows, got.Rows, "rows")
			assert.Equal(t, tt.wantX, got.CursorX, "cursor x")
			assert.Equal(t, tt.wantY, got.CursorY, "cursor y")
		})
	}
}

// TestGridLayout_ColumnWidthUsesDisplayWidth verifies a wide name widens the
// column, so a CJK name is not given the space a single-cell name would take.
//
// At this width and height a four-cell name still fits two columns, while a
// six-cell name does not, which is only observable when the measurement is in
// cells. More than one item is needed, since a single item can never occupy
// more than one column however wide it is.
func TestGridLayout_ColumnWidthUsesDisplayWidth(t *testing.T) {
	t.Parallel()

	ascii := GridLayout(&GridInput{
		Choices: []string{"tool", "wget", "curl", "rg"},
		Width:   20,
		Height:  10,
	})
	cjk := GridLayout(&GridInput{
		Choices: []string{"日本語", "中国", "本語", "語文"},
		Width:   20,
		Height:  10,
	})

	assert.Equal(t, 2, ascii.Cols, "a four-cell name should fit two columns")
	assert.Equal(t, 1, cjk.Cols,
		"a six-cell name must not be given the same column width as a four-cell one")
}

// TestGridLayout_StatusLineCostsARow verifies a non-empty status line reduces
// the height available to the grid.
func TestGridLayout_StatusLineCostsARow(t *testing.T) {
	t.Parallel()

	base := GridInput{
		Choices: []string{"a", "b", "c", "d", "e", "f", "g", "h"},
		Width:   100,
		Height:  12,
	}

	without := GridLayout(&base)

	withStatus := base
	withStatus.Status = "working"
	status := GridLayout(&withStatus)

	assert.Equal(t, without.Rows-1, status.Rows,
		"a status line must take one row from the grid")
}
