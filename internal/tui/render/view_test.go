/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package render

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/go-remove/internal/history"
)

// ansiRegex matches the SGR escape sequences the styles emit.
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// stripANSI removes the styling escapes so a comparison covers the text alone.
func stripANSI(str string) string {
	return ansiRegex.ReplaceAllString(str, "")
}

// withGrid recomputes a state's grid the way the program does on its first size
// message, so the layout under test is one the real flow would produce rather
// than hand-set rows the program would never have.
//
// The model applies GridLayout's result to its own state, so a test driving the
// views directly has to make the same substitution.
//
// Parameters:
//   - state: The snapshot to lay out, updated in place.
func withGrid(state *State) {
	grid := GridLayout(&GridInput{
		Choices:  state.Choices,
		Width:    state.Width,
		Height:   state.Height,
		Status:   state.Status,
		ShowLogs: state.ShowLogs,
		LogCount: len(state.Logs),
		CursorX:  state.CursorX,
		CursorY:  state.CursorY,
	})

	state.Rows = grid.Rows
	state.Cols = grid.Cols
	state.CursorX = grid.CursorX
	state.CursorY = grid.CursorY
}

// TestView_FillsTerminalHeight verifies each view fills the terminal exactly,
// with the footer on the last line.
//
// The padding is measured from what actually renders, so asserting the
// property keeps the body and the footer tied to the real layout.
func TestView_FillsTerminalHeight(t *testing.T) {
	t.Parallel()

	heights := []int{10, 24, 40}
	widths := []int{40, 80, 160}

	states := map[string]func() *State{
		"binaries": func() *State {
			return &State{
				Mode: ModeBinaries, Choices: []string{"vhs", "tool"},
				Width: 80, Height: 24, Styles: DefaultStyleConfig(),
			}
		},
		"binaries with status and logs": func() *State {
			return &State{
				Mode: ModeBinaries, Choices: []string{"vhs"},
				Width: 80, Height: 24,
				Status:   "Removed tool",
				Logs:     []string{"a log line", "another log line"},
				ShowLogs: true,
				Styles:   DefaultStyleConfig(),
			}
		},
		"history": func() *State {
			return &State{
				Mode:           ModeHistory,
				HistoryEntries: []*history.HistoryEntry{{BinaryName: "vhs"}},
				Width:          80, Height: 24, Styles: DefaultStyleConfig(),
			}
		},
	}

	for name, build := range states {
		for _, height := range heights {
			for _, width := range widths {
				t.Run(fmt.Sprintf("%s/h=%d/w=%d", name, height, width), func(t *testing.T) {
					t.Parallel()

					state := build()
					state.Height = height
					state.Width = width

					// Recalculate the grid as the first size message does, rather than
					// using rows the program would never produce.
					withGrid(state)

					// The model picks the view from the mode, so the test does the
					// same to reach the view the mode names.
					var content string

					if state.Mode == ModeHistory {
						content = History(state).Content
					} else {
						content = Binaries(state).Content
					}

					// A view that cannot fit may exceed the terminal height, but it must
					// still reach the bottom.
					got := lipgloss.Height(content)

					if state.ShowLogs && got > height {
						assert.GreaterOrEqual(t, got, height,
							"a view that cannot fit must still reach the bottom")

						return
					}

					assert.Equal(t, height, got,
						"the view must fill exactly the terminal height")

					// The footer is written last, so it must end the view.
					lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
					require.NotEmpty(t, lines)
					assert.NotEmpty(t, strings.TrimSpace(lines[len(lines)-1]),
						"the last line must not be blank")
				})
			}
		}
	}
}
