/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package models

import (
	"errors"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mockFS "github.com/nicholas-fedor/go-remove/internal/fs/mocks"
	"github.com/nicholas-fedor/go-remove/internal/history"
	"github.com/nicholas-fedor/go-remove/internal/logger"
	"github.com/nicholas-fedor/go-remove/internal/tui/render"
)

// TestRefreshChoices_ReportsReadFailure verifies a failed rescan is visible
// rather than silently leaving a stale list on screen.
func TestRefreshChoices_ReportsReadFailure(t *testing.T) {
	t.Parallel()

	readErr := errors.New("permission denied")
	fsMock := mockFS.NewMockFS(t)
	fsMock.On("ListBinaries", "/bin").Return(nil, readErr)

	m := &Model{
		dir:     "/bin",
		choices: []string{"existing"},
		logger:  logger.NopLogger(),
		styles:  render.DefaultStyleConfig(),
		fs:      fsMock,
	}

	m.refreshChoices()

	assert.Contains(t, m.status, "Could not read", "a failed rescan must be visible")
	assert.Contains(t, m.status, "permission denied")
	assert.Equal(t, []string{"existing"}, m.choices, "the existing list must be preserved")
}

// TestRefreshChoices_ReplacesListOnSuccess verifies a successful rescan
// replaces the list, which is the success counterpart to the failure case.
func TestRefreshChoices_ReplacesListOnSuccess(t *testing.T) {
	t.Parallel()

	fsMock := mockFS.NewMockFS(t)
	fsMock.On("ListBinaries", "/bin").Return([]string{"tool"}, nil)

	m := &Model{
		dir:     "/bin",
		choices: []string{"stale"},
		status:  "Could not read /bin: permission denied",
		logger:  logger.NopLogger(),
		styles:  render.DefaultStyleConfig(),
		fs:      fsMock,
	}

	m.refreshChoices()

	assert.Equal(t, []string{"tool"}, m.choices)
}

// Test_model_Init verifies Init always starts log polling.
//
// Polling is not conditional on verbose mode, so a model that has no log
// channel of its own still receives a command. Returning nil here would leave
// the capture callback writing into a channel nobody drains.
func Test_model_Init(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		m    Model
	}{
		{
			name: "binaries mode polls",
			m:    Model{},
		},
		{
			name: "history mode batches the history load with polling",
			m:    Model{mode: modeHistory},
		},
		{
			name: "an existing log channel still polls",
			m:    Model{logChan: make(chan LogMsg, maxLogLines)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.m.Init()

			require.NotNil(t, got, "Init must start log polling")
		})
	}
}

// Test_model_updateGrid verifies the updateGrid method's layout calculations.
func Test_model_updateGrid(t *testing.T) {
	tests := []struct {
		name string
		m    *Model
		want Model
	}{
		{
			name: "single item",
			m: &Model{
				choices: []string{"vhs"},
				width:   80,
				height:  24,
			},
			want: Model{
				choices: []string{"vhs"},
				width:   80,
				height:  24,
				cols:    1,
				rows:    1,
				cursorX: 0,
				cursorY: 0,
			},
		},
		{
			name: "multiple items",
			m: &Model{
				choices: []string{"vhs", "age", "tool"},
				width:   80,
				height:  24,
			},
			want: Model{
				choices: []string{"vhs", "age", "tool"},
				width:   80,
				height:  24,
				cols:    1,
				rows:    3,
				cursorX: 0,
				cursorY: 0,
			},
		},
		{
			name: "tall narrow window",
			m: &Model{
				choices: []string{"vhs", "age", "tool"},
				width:   20,
				height:  24,
			},
			want: Model{
				choices: []string{"vhs", "age", "tool"},
				width:   20,
				height:  24,
				cols:    1,
				rows:    3,
				cursorX: 0,
				cursorY: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.m.updateGrid()

			if !reflect.DeepEqual(*tt.m, tt.want) {
				t.Errorf("model.updateGrid() = %+v, want %+v", *tt.m, tt.want)
			}
		})
	}
}

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(str string) string {
	return ansiRegex.ReplaceAllString(str, "")
}

// withoutBlankLines drops blank lines so a comparison covers content and order
// only.
//
// The view measures and inserts the padding between its content and the footer
// to fill the terminal, so the padding is asserted as a height property by
// render.TestView_FillsTerminalHeight rather than being hand-counted in the
// expected output.
func withoutBlankLines(s string) string {
	lines := strings.Split(s, "\n")
	kept := make([]string, 0, len(lines))

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		kept = append(kept, line)
	}

	return strings.Join(kept, "\n")
}

// Test_model_View verifies the View method's rendered output.
func Test_model_View(t *testing.T) {
	const contentWidth = 78 // Max visible width for content (excluding leftPadding)

	const leftPaddingStr = "  " // The left padding string

	const effectiveWidth = contentWidth - len(leftPaddingStr) // Effective width for content padding

	displayWidth := func(s string) int {
		return utf8.RuneCountInString(stripANSI(s))
	}

	pad := func(s string, w int) string {
		currentWidth := displayWidth(s)
		if currentWidth >= w {
			return s
		}

		return s + strings.Repeat(" ", w-currentWidth)
	}

	tests := []struct {
		name string
		m    Model
		want string
	}{
		{
			name: "no_choices",
			m:    Model{choices: []string{}, height: 1},
			want: "No binaries found.\n",
		},
		{
			name: "single_choice",
			m: Model{
				choices:       []string{"vhs"},
				cols:          1,
				rows:          1,
				width:         80,
				height:        24,
				cursorX:       0,
				cursorY:       0,
				sortAscending: true,
				styles:        render.DefaultStyleConfig(),
			},
			want: func() string {
				lines := make([]string, 0, 25)

				lines = append(
					lines,
					leftPaddingStr+pad("Select a binary to remove:", effectiveWidth),
					leftPaddingStr+pad("", effectiveWidth),
					leftPaddingStr+pad("❯ vhs", effectiveWidth),
					leftPaddingStr+pad("", effectiveWidth),
				)

				footerPart1 := "↑/k: up  ↓/j: down  ←/h: left  →/l: right  Enter: remove  s: sort  r:"
				footerPart2 := "history  u: undo  L: logs  q: quit"

				lines = append(
					lines,
					leftPaddingStr+pad(footerPart1, effectiveWidth),
					leftPaddingStr+pad(footerPart2, effectiveWidth),
				)

				return strings.Join(lines, "\n")
			}(),
		},
		{
			name: "multiple_choices_with_status",
			m: Model{
				choices:       []string{"age", "vhs"},
				cols:          1,
				rows:          2,
				width:         80,
				height:        24,
				status:        "Removed tool",
				cursorX:       0,
				cursorY:       0,
				sortAscending: true,
				styles:        render.DefaultStyleConfig(),
			},
			want: func() string {
				lines := make([]string, 0, 25)

				lines = append(
					lines,
					leftPaddingStr+pad("Select a binary to remove:", effectiveWidth),
					leftPaddingStr+pad("", effectiveWidth),
					leftPaddingStr+pad("❯ age", effectiveWidth),
					leftPaddingStr+pad("  vhs", effectiveWidth), // Adjusted to match actual padding
					leftPaddingStr+pad("", effectiveWidth),
					leftPaddingStr+pad("Removed tool", effectiveWidth),
				)
				for range 13 { // Adjusted for rows=2, totalHeightBase=8, and status line
					lines = append(lines, leftPaddingStr+pad("", effectiveWidth))
				}

				footerPart1 := "↑/k: up  ↓/j: down  ←/h: left  →/l: right  Enter: remove  s: sort  r:"
				footerPart2 := "history  u: undo  L: logs  q: quit"

				lines = append(
					lines,
					leftPaddingStr+pad(footerPart1, effectiveWidth),
					leftPaddingStr+pad(footerPart2, effectiveWidth),
				)

				return strings.Join(lines, "\n")
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.m.sortChoices()
			tt.m.updateGrid()

			view := tt.m.View()
			got := stripANSI(view.Content)

			// The view measures and fills the terminal itself, so the padding
			// between the content and the footer is asserted as a height
			// property rather than hand-counted here.
			if content, want := withoutBlankLines(
				got,
			), withoutBlankLines(
				tt.want,
			); content != want {
				t.Errorf("model.View() content = %q, want %q", content, want)
			}
		})
	}
}

// Test_model_Update_AlternateScreen verifies alt-screen toggle.
func Test_model_Update_AlternateScreen(t *testing.T) {
	m := &Model{
		choices:       []string{"test"},
		dir:           "/bin",
		config:        Config{},
		fs:            mockFS.NewMockFS(t),
		logger:        logger.NopLogger(),
		cols:          1,
		rows:          1,
		width:         80,
		height:        24,
		sortAscending: true,
	}

	view := m.View()

	// View should have AltScreen enabled
	assert.True(t, view.AltScreen)
}

// Test_model_sortChoices verifies sorting logic.
func Test_model_sortChoices(t *testing.T) {
	tests := []struct {
		name          string
		choices       []string
		sortAscending bool
		wantFirst     string
		wantLast      string
	}{
		{
			name:          "ascending sort",
			choices:       []string{"zebra", "apple", "mango"},
			sortAscending: true,
			wantFirst:     "apple",
			wantLast:      "zebra",
		},
		{
			name:          "descending sort",
			choices:       []string{"zebra", "apple", "mango"},
			sortAscending: false,
			wantFirst:     "zebra",
			wantLast:      "apple",
		},
		{
			name:          "empty choices",
			choices:       []string{},
			sortAscending: true,
			wantFirst:     "",
			wantLast:      "",
		},
		{
			name:          "single choice",
			choices:       []string{"only"},
			sortAscending: true,
			wantFirst:     "only",
			wantLast:      "only",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Model{
				choices:       tt.choices,
				sortAscending: tt.sortAscending,
			}

			m.sortChoices()

			if len(m.choices) > 0 {
				assert.Equal(t, tt.wantFirst, m.choices[0])
				assert.Equal(t, tt.wantLast, m.choices[len(m.choices)-1])
			}
		})
	}
}

// Test_model_updateGrid verifies grid recalculation.
func Test_model_updateGrid_VerifyState(t *testing.T) {
	tests := []struct {
		name      string
		choices   []string
		width     int
		height    int
		wantCols  int
		wantRows  int
		wantValid bool
	}{
		{
			name:      "many items wide window",
			choices:   []string{"a", "b", "c", "d", "e", "f"},
			width:     100,
			height:    24,
			wantCols:  1,
			wantRows:  6,
			wantValid: true,
		},
		{
			name:      "few items",
			choices:   []string{"a", "b"},
			width:     80,
			height:    24,
			wantCols:  1,
			wantRows:  2,
			wantValid: true,
		},
		{
			name:      "empty choices",
			choices:   []string{},
			width:     80,
			height:    24,
			wantCols:  0,
			wantRows:  0,
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Model{
				choices: tt.choices,
				width:   tt.width,
				height:  tt.height,
				cols:    0,
				rows:    0,
			}

			m.updateGrid()

			if tt.wantValid {
				assert.Equal(t, tt.wantCols, m.cols)
				assert.Equal(t, tt.wantRows, m.rows)
			}
		})
	}
}

// Test_model_cursorBounds verifies cursor stays within bounds after list changes.
func Test_model_cursorBounds(t *testing.T) {
	tests := []struct {
		name       string
		initialX   int
		initialY   int
		cols       int
		rows       int
		choicesLen int
		wantX      int
		wantY      int
	}{
		{
			name:       "cursor within bounds",
			initialX:   0,
			initialY:   0,
			cols:       2,
			rows:       3,
			choicesLen: 6,
			wantX:      0,
			wantY:      0,
		},
		{
			name:       "cursor beyond col bounds",
			initialX:   3,
			initialY:   0,
			cols:       2,
			rows:       3,
			choicesLen: 6,
			wantX:      1, // Should be clamped to cols-1
			wantY:      0,
		},
		{
			name:       "cursor beyond row bounds",
			initialX:   0,
			initialY:   5,
			cols:       2,
			rows:       3,
			choicesLen: 6,
			wantX:      0,
			wantY:      2, // Should be clamped to rows-1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Model{
				cursorX: tt.initialX,
				cursorY: tt.initialY,
				cols:    tt.cols,
				rows:    tt.rows,
				choices: make([]string, tt.choicesLen),
				width:   80,
				height:  24,
			}

			m.updateGrid()

			// After updateGrid, cursor should be within bounds
			if m.cols > 0 && m.cursorX >= m.cols {
				t.Errorf("cursorX %d >= cols %d", m.cursorX, m.cols)
			}

			if m.rows > 0 && m.cursorY >= m.rows {
				t.Errorf("cursorY %d >= rows %d", m.cursorY, m.rows)
			}
		})
	}
}

// Test_model_Init_WithHistoryMode verifies Init behavior in history mode.
func Test_model_Init_WithHistoryMode(t *testing.T) {
	m := &Model{
		choices:        []string{},
		dir:            "/bin",
		config:         Config{},
		fs:             mockFS.NewMockFS(t),
		logger:         logger.NopLogger(),
		mode:           modeHistory,
		historyEntries: []*history.HistoryEntry{},
		sortAscending:  true,
	}

	cmd := m.Init()

	// Should return a batch command with loadHistory and pollLogChannel
	assert.NotNil(t, cmd)
}

// Test_model_Init_WithVerboseMode verifies Init behavior in verbose mode.
func Test_model_Init_WithVerboseMode(t *testing.T) {
	logChan := make(chan LogMsg, 10)
	m := &Model{
		choices:       []string{},
		dir:           "/bin",
		config:        Config{Verbose: true},
		fs:            mockFS.NewMockFS(t),
		logger:        logger.NopLogger(),
		mode:          modeBinaries,
		logChan:       logChan,
		sortAscending: true,
	}

	cmd := m.Init()

	// Should return pollLogChannel command
	assert.NotNil(t, cmd)
}
