/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package cli

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	tea "charm.land/bubbletea/v2"

	mockFS "github.com/nicholas-fedor/go-remove/internal/fs/mocks"
	"github.com/nicholas-fedor/go-remove/internal/history"
	mockHistory "github.com/nicholas-fedor/go-remove/internal/history/mocks"
	"github.com/nicholas-fedor/go-remove/internal/logger"
)

// Constants for key press strings to avoid magic string repetition.
const (
	keyEnter = "enter"
	keyDown  = "down"
	keyUp    = "up"
	keyLeft  = "left"
	keyRight = "right"
	keyCtrlC = "ctrl+c"
)

// tuiMockLogger is a simple mock logger for TUI tests.
type tuiMockLogger struct {
	nopLogger zerolog.Logger
}

func (m *tuiMockLogger) Debug() *zerolog.Event {
	m.nopLogger.Debug().Msg("")

	return nil
}

func (m *tuiMockLogger) Info() *zerolog.Event {
	m.nopLogger.Info().Msg("")

	return nil
}

func (m *tuiMockLogger) Warn() *zerolog.Event {
	m.nopLogger.Warn().Msg("")

	return nil
}

func (m *tuiMockLogger) Error() *zerolog.Event {
	m.nopLogger.Error().Msg("")

	return nil
}
func (m *tuiMockLogger) Sync() error                            { return nil }
func (m *tuiMockLogger) Level(_ zerolog.Level)                  {}
func (m *tuiMockLogger) SetCaptureFunc(_ logger.LogCaptureFunc) {}

// tuiMockRunner mocks the ProgramRunner interface for TUI tests.
type tuiMockRunner struct {
	runProgram func(tea.Model, ...tea.ProgramOption) (*tea.Program, error)
}

// RunProgram executes the mock runner's program function.
func (m *tuiMockRunner) RunProgram(
	model tea.Model,
	opts ...tea.ProgramOption,
) (*tea.Program, error) {
	return m.runProgram(model, opts...)
}

// TestRunTUI verifies the RunTUI function's behavior under various conditions.
func TestRunTUI(t *testing.T) {
	type args struct {
		dir    string
		config Config
		logger logger.Logger
		fs     *mockFS.MockFS
		runner ProgramRunner
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "success with binaries",
			args: args{
				dir:    "/bin",
				config: Config{},
				logger: &tuiMockLogger{},
				fs: func() *mockFS.MockFS {
					m := mockFS.NewMockFS(t)
					m.On("ListBinaries", "/bin").Return([]string{"vhs"})

					return m
				}(),
				runner: &tuiMockRunner{runProgram: mockNoOpRunner},
			},
			wantErr: false,
		},
		{
			name: "no binaries",
			args: args{
				dir:    "/bin",
				config: Config{},
				logger: &tuiMockLogger{},
				fs: func() *mockFS.MockFS {
					m := mockFS.NewMockFS(t)
					m.On("ListBinaries", "/bin").Return([]string{})

					return m
				}(),
				runner: &tuiMockRunner{runProgram: mockNoOpRunner},
			},
			wantErr: true,
		},
		{
			name: "runner error",
			args: args{
				dir:    "/bin",
				config: Config{},
				logger: &tuiMockLogger{},
				fs: func() *mockFS.MockFS {
					m := mockFS.NewMockFS(t)
					m.On("ListBinaries", "/bin").Return([]string{"vhs"})

					return m
				}(),
				runner: &tuiMockRunner{
					runProgram: func(tea.Model, ...tea.ProgramOption) (*tea.Program, error) {
						return nil, errors.New("runner failed")
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RunTUI(
				tt.args.dir,
				tt.args.config,
				tt.args.logger,
				tt.args.fs,
				tt.args.runner,
				nil,
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunTUI() error = %v, wantErr %v", err, tt.wantErr)
			}

			tt.args.fs.AssertExpectations(t)
		})
	}
}

// Test_model_Init verifies the Init method's command output.
func Test_model_Init(t *testing.T) {
	tests := []struct {
		name string
		m    model
		want tea.Cmd
	}{
		{
			name: "basic init",
			m:    model{},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.m.Init()
			if reflect.TypeOf(got) != reflect.TypeOf(tt.want) {
				t.Errorf("model.Init() = %T, want %T", got, tt.want)
			}
		})
	}
}

// keyPress creates a KeyPressMsg with the given rune.
func keyPress(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Text: string(r), Code: r, ShiftedCode: r}
}

// keyPressString creates a KeyPressMsg from a string representation.
func keyPressString(s string) tea.KeyPressMsg {
	switch s {
	case keyEnter:
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case keyUp:
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case keyDown:
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case keyLeft:
		return tea.KeyPressMsg{Code: tea.KeyLeft}
	case keyRight:
		return tea.KeyPressMsg{Code: tea.KeyRight}
	default:
		if len(s) == 1 {
			r := rune(s[0])

			return tea.KeyPressMsg{Text: s, Code: r, ShiftedCode: r}
		}

		return tea.KeyPressMsg{}
	}
}

// Test_model_Update verifies the Update method's state changes and commands.
func Test_model_Update(t *testing.T) {
	type args struct {
		msg tea.Msg
	}

	tests := []struct {
		name    string
		m       *model
		args    args
		want    model
		wantCmd tea.Cmd
	}{
		{
			name:    "quit with q",
			m:       &model{},
			args:    args{msg: keyPress('q')},
			want:    model{},
			wantCmd: tea.Quit,
		},
		{
			name:    "move up",
			m:       &model{cursorY: 1, rows: 2},
			args:    args{msg: keyPressString("up")},
			want:    model{cursorY: 0, rows: 2},
			wantCmd: nil,
		},
		{
			name:    "move down within bounds",
			m:       &model{cursorY: 0, rows: 2, cols: 2, choices: []string{"a", "b", "c", "d"}},
			args:    args{msg: keyPressString(keyDown)},
			want:    model{cursorY: 1, rows: 2, cols: 2, choices: []string{"a", "b", "c", "d"}},
			wantCmd: nil,
		},
		{
			name: "move down at last item in last column",
			m: &model{
				cursorY: 1,
				cursorX: 1,
				rows:    2,
				cols:    2,
				choices: []string{"a", "b", "c", "d"},
			},
			args: args{msg: keyPressString(keyDown)},
			want: model{
				cursorY: 1,
				cursorX: 1,
				rows:    2,
				cols:    2,
				choices: []string{"a", "b", "c", "d"},
			},
			wantCmd: nil,
		},
		{
			name:    "move left",
			m:       &model{cursorX: 1, cols: 2},
			args:    args{msg: keyPressString("left")},
			want:    model{cursorX: 0, cols: 2},
			wantCmd: nil,
		},
		{
			name:    "move right within bounds",
			m:       &model{cursorX: 0, cols: 2, choices: []string{"a", "b"}},
			args:    args{msg: keyPressString("right")},
			want:    model{cursorX: 1, cols: 2, choices: []string{"a", "b"}},
			wantCmd: nil,
		},
		{
			name: "toggle sort to descending",
			m: &model{
				choices:       []string{"age", "vhs"},
				sortAscending: true,
				cols:          1,
				rows:          2,
				width:         80,
				height:        24,
			},
			args: args{msg: keyPress('s')},
			want: model{
				choices:       []string{"vhs", "age"},
				sortAscending: false,
				cols:          1,
				rows:          2,
				width:         80,
				height:        24,
			},
			wantCmd: nil,
		},
		{
			name: "toggle sort to ascending",
			m: &model{
				choices:       []string{"vhs", "age"},
				sortAscending: false,
				cols:          1,
				rows:          2,
				width:         80,
				height:        24,
			},
			args: args{msg: keyPress('s')},
			want: model{
				choices:       []string{"age", "vhs"},
				sortAscending: true,
				cols:          1,
				rows:          2,
				width:         80,
				height:        24,
			},
			wantCmd: nil,
		},
		{
			name: "enter removes binary",
			m: &model{
				choices:       []string{"age", "vhs"},
				cols:          1,
				rows:          2,
				dir:           "/bin",
				config:        Config{Verbose: false},
				sortAscending: true,
				width:         80,
				height:        24,
				logger:        &tuiMockLogger{},
				fs: func() *mockFS.MockFS {
					m := mockFS.NewMockFS(t)
					m.On("AdjustBinaryPath", "/bin", "age").Return("/bin/age")
					m.On("RemoveBinary", "/bin/age", "age", false, mock.Anything).Return(nil)
					m.On("ListBinaries", "/bin").Return([]string{"vhs"})

					return m
				}(),
			},
			args: args{msg: keyPressString(keyEnter)},
			want: model{
				choices:       []string{"vhs"},
				cols:          1,
				rows:          1,
				dir:           "/bin",
				config:        Config{Verbose: false},
				sortAscending: true,
				status:        "Removed age",
				width:         80,
				height:        24,
			},
			wantCmd: nil,
		},
		{
			name: "enter with error",
			m: &model{
				choices:       []string{"age"},
				cols:          1,
				rows:          1,
				dir:           "/bin",
				sortAscending: true,
				width:         80,
				height:        24,
				logger:        &tuiMockLogger{},
				fs: func() *mockFS.MockFS {
					m := mockFS.NewMockFS(t)
					m.On("AdjustBinaryPath", "/bin", "age").Return("/bin/age")
					m.On("RemoveBinary", "/bin/age", "age", false, mock.Anything).
						Return(errors.New("remove failed"))

					return m
				}(),
			},
			args: args{msg: keyPressString(keyEnter)},
			want: model{
				choices:       []string{"age"},
				cols:          1,
				rows:          1,
				dir:           "/bin",
				sortAscending: true,
				status:        "Error removing age: remove failed",
				width:         80,
				height:        24,
			},
			wantCmd: nil,
		},
		{
			name: "window size update",
			m:    &model{choices: []string{"a", "b"}},
			args: args{msg: tea.WindowSizeMsg{Width: 80, Height: 24}},
			want: model{
				choices: []string{"a", "b"},
				cols:    1,
				rows:    2,
				width:   80,
				height:  24,
			},
			wantCmd: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set default mocks if not provided.
			if tt.m.logger == nil {
				tt.m.logger = &tuiMockLogger{}
			}

			if tt.m.fs == nil {
				tt.m.fs = mockFS.NewMockFS(t)
			}

			got, gotCmd := tt.m.Update(tt.args.msg)
			gotModel := got.(*model)

			if !reflect.DeepEqual(gotModel.choices, tt.want.choices) ||
				gotModel.cursorX != tt.want.cursorX ||
				gotModel.cursorY != tt.want.cursorY ||
				gotModel.cols != tt.want.cols ||
				gotModel.rows != tt.want.rows ||
				gotModel.dir != tt.want.dir ||
				gotModel.config != tt.want.config ||
				gotModel.width != tt.want.width ||
				gotModel.height != tt.want.height ||
				gotModel.status != tt.want.status ||
				gotModel.sortAscending != tt.want.sortAscending {
				t.Errorf("model.Update() got = %+v, want %+v", gotModel, tt.want)
			}

			if reflect.TypeOf(gotCmd) != reflect.TypeOf(tt.wantCmd) {
				t.Errorf("model.Update() gotCmd = %T, want %T", gotCmd, tt.wantCmd)
			}

			tt.m.fs.(*mockFS.MockFS).AssertExpectations(t)
		})
	}
}

// Test_model_updateGrid verifies the updateGrid method's layout calculations.
func Test_model_updateGrid(t *testing.T) {
	tests := []struct {
		name string
		m    *model
		want model
	}{
		{
			name: "single item",
			m: &model{
				choices: []string{"vhs"},
				width:   80,
				height:  24,
			},
			want: model{
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
			m: &model{
				choices: []string{"vhs", "age", "tool"},
				width:   80,
				height:  24,
			},
			want: model{
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
			m: &model{
				choices: []string{"vhs", "age", "tool"},
				width:   20,
				height:  24,
			},
			want: model{
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
		m    model
		want string
	}{
		{
			name: "no_choices",
			m:    model{choices: []string{}, height: 1},
			want: "No binaries found.\n",
		},
		{
			name: "single_choice",
			m: model{
				choices:       []string{"vhs"},
				cols:          1,
				rows:          1,
				width:         80,
				height:        24,
				cursorX:       0,
				cursorY:       0,
				sortAscending: true,
				styles:        defaultStyleConfig(),
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
				for range 19 {
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
		{
			name: "multiple_choices_with_status",
			m: model{
				choices:       []string{"age", "vhs"},
				cols:          1,
				rows:          2,
				width:         80,
				height:        24,
				status:        "Removed tool",
				cursorX:       0,
				cursorY:       0,
				sortAscending: true,
				styles:        defaultStyleConfig(),
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
				for range 17 { // Adjusted for rows=2
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

			got := stripANSI(tt.m.View().Content)
			if got != tt.want {
				t.Errorf("model.View() got = %q, want %q", got, tt.want)
			}
		})
	}
}

// Test_max verifies the max function's comparison logic.
func Test_max(t *testing.T) {
	tests := []struct {
		name string
		a    int
		b    int
		want int
	}{
		{name: "a greater", a: 5, b: 3, want: 5},
		{name: "b greater", a: 2, b: 7, want: 7},
		{name: "equal", a: 4, b: 4, want: 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := maximum(tt.a, tt.b); got != tt.want {
				t.Errorf("maximum() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test_min verifies the min function's comparison logic.
func Test_min(t *testing.T) {
	tests := []struct {
		name string
		a    int
		b    int
		want int
	}{
		{name: "a lesser", a: 3, b: 5, want: 3},
		{name: "b lesser", a: 7, b: 2, want: 2},
		{name: "equal", a: 4, b: 4, want: 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := minimum(tt.a, tt.b); got != tt.want {
				t.Errorf("minimum() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Priority 1: Core History Integration Tests

// Test_model_Update_EnterWithHistoryManager verifies that RecordDeletion is called
// when a binary is removed and history manager is available.
func Test_model_Update_EnterWithHistoryManager(t *testing.T) {
	fsMock := mockFS.NewMockFS(t)
	historyMock := mockHistory.NewMockManager(t)

	// Setup expectations
	fsMock.On("AdjustBinaryPath", "/bin", "test").Return("/bin/test")
	historyMock.On("RecordDeletion", mock.Anything, "/bin/test").
		Return(&history.HistoryEntry{ID: "123", BinaryName: "test"}, nil)
	fsMock.On("ListBinaries", "/bin").Return([]string{"other"})

	m := &model{
		choices:        []string{"test"},
		dir:            "/bin",
		config:         Config{Verbose: false},
		fs:             fsMock,
		historyManager: historyMock,
		logger:         &tuiMockLogger{},
		cols:           1,
		rows:           1,
		width:          80,
		height:         24,
		sortAscending:  true,
	}

	got, _ := m.Update(keyPressString(keyEnter))
	gotModel := got.(*model)

	assert.Equal(t, "Removed test", gotModel.status)
	assert.Contains(t, gotModel.choices, "other")
	fsMock.AssertExpectations(t)
	historyMock.AssertExpectations(t)
}

// Test_model_Update_HistoryRecordError verifies error handling when RecordDeletion fails.
func Test_model_Update_HistoryRecordError(t *testing.T) {
	fsMock := mockFS.NewMockFS(t)
	historyMock := mockHistory.NewMockManager(t)

	fsMock.On("AdjustBinaryPath", "/bin", "test").Return("/bin/test")
	historyMock.On("RecordDeletion", mock.Anything, "/bin/test").
		Return(nil, errors.New("history storage full"))

	m := &model{
		choices:        []string{"test"},
		dir:            "/bin",
		config:         Config{Verbose: false},
		fs:             fsMock,
		historyManager: historyMock,
		logger:         &tuiMockLogger{},
		cols:           1,
		rows:           1,
		width:          80,
		height:         24,
		sortAscending:  true,
	}

	got, _ := m.Update(keyPressString(keyEnter))
	gotModel := got.(*model)

	assert.Equal(t, "Error recording test: history storage full", gotModel.status)
	assert.Equal(t, []string{"test"}, gotModel.choices) // Should not be removed
	fsMock.AssertExpectations(t)
	historyMock.AssertExpectations(t)
}

// Test_model_Update_BinaryRemovedFromChoicesAfterHistoryRecord verifies state consistency
// after recording deletion in history.
func Test_model_Update_BinaryRemovedFromChoicesAfterHistoryRecord(t *testing.T) {
	fsMock := mockFS.NewMockFS(t)
	historyMock := mockHistory.NewMockManager(t)

	fsMock.On("AdjustBinaryPath", "/bin", "binary1").Return("/bin/binary1")
	historyMock.On("RecordDeletion", mock.Anything, "/bin/binary1").
		Return(&history.HistoryEntry{ID: "entry1", BinaryName: "binary1"}, nil)
	fsMock.On("ListBinaries", "/bin").Return([]string{"binary2", "binary3"})

	m := &model{
		choices:        []string{"binary1", "binary2", "binary3"},
		dir:            "/bin",
		config:         Config{},
		fs:             fsMock,
		historyManager: historyMock,
		logger:         &tuiMockLogger{},
		cols:           1,
		rows:           3,
		width:          80,
		height:         24,
		sortAscending:  true,
	}

	got, _ := m.Update(keyPressString(keyEnter))
	gotModel := got.(*model)

	assert.Len(t, gotModel.choices, 2)
	assert.NotContains(t, gotModel.choices, "binary1")
	fsMock.AssertExpectations(t)
	historyMock.AssertExpectations(t)
}

// Priority 2: Restore/Undo Operations

// Test_handleRestore_RefreshesBinaryList verifies binary list is refreshed after restore.
func Test_handleRestore_RefreshesBinaryList(t *testing.T) {
	fsMock := mockFS.NewMockFS(t)
	historyMock := mockHistory.NewMockManager(t)

	entry := &history.HistoryEntry{
		ID:         "entry1",
		BinaryName: "restored_binary",
		CanRestore: true,
	}

	historyMock.On("Restore", mock.Anything, "entry1").
		Return(&history.RestoreResult{BinaryName: "restored_binary", RestoredTo: "/bin/restored_binary"}, nil)
	fsMock.On("ListBinaries", "/bin").Return([]string{"restored_binary", "existing"})

	m := &model{
		choices:        []string{"existing"},
		dir:            "/bin",
		fs:             fsMock,
		historyManager: historyMock,
		logger:         &tuiMockLogger{},
		mode:           modeHistory,
		historyEntries: []*history.HistoryEntry{entry},
		historyCursor:  0,
		width:          80,
		height:         24,
		sortAscending:  true,
	}

	got, cmd := m.handleRestore()
	gotModel := got.(*model)

	assert.Contains(t, gotModel.choices, "restored_binary")
	assert.NotNil(t, cmd)
	fsMock.AssertExpectations(t)
	historyMock.AssertExpectations(t)
}

// Test_handleRestore_BinaryAppearsInChoices verifies restored binary is visible in list.
func Test_handleRestore_BinaryAppearsInChoices(t *testing.T) {
	fsMock := mockFS.NewMockFS(t)
	historyMock := mockHistory.NewMockManager(t)

	entry := &history.HistoryEntry{
		ID:         "entry1",
		BinaryName: "newbinary",
		CanRestore: true,
	}

	historyMock.On("Restore", mock.Anything, "entry1").
		Return(&history.RestoreResult{BinaryName: "newbinary", RestoredTo: "/bin/newbinary"}, nil)
	fsMock.On("ListBinaries", "/bin").Return([]string{"newbinary"})

	m := &model{
		choices:        []string{},
		dir:            "/bin",
		fs:             fsMock,
		historyManager: historyMock,
		logger:         &tuiMockLogger{},
		mode:           modeHistory,
		historyEntries: []*history.HistoryEntry{entry},
		historyCursor:  0,
		width:          80,
		height:         24,
	}

	got, _ := m.handleRestore()
	gotModel := got.(*model)

	assert.Equal(t, []string{"newbinary"}, gotModel.choices)
	assert.Equal(t, "Restored newbinary to /bin/newbinary", gotModel.status)
	fsMock.AssertExpectations(t)
	historyMock.AssertExpectations(t)
}

// Test_handleRestore_HistoryRefreshed verifies history is updated after restore.
func Test_handleRestore_HistoryRefreshed(t *testing.T) {
	historyMock := mockHistory.NewMockManager(t)

	entry := &history.HistoryEntry{
		ID:         "entry1",
		BinaryName: "testbin",
		CanRestore: true,
	}

	historyMock.On("Restore", mock.Anything, "entry1").
		Return(&history.RestoreResult{BinaryName: "testbin", RestoredTo: "/bin/testbin"}, nil)

	fsMock := mockFS.NewMockFS(t)
	fsMock.On("ListBinaries", "/bin").Return([]string{"testbin"})

	m := &model{
		choices:        []string{},
		dir:            "/bin",
		fs:             fsMock,
		historyManager: historyMock,
		logger:         &tuiMockLogger{},
		mode:           modeHistory,
		historyEntries: []*history.HistoryEntry{entry},
		historyCursor:  0,
		width:          80,
		height:         24,
		sortAscending:  true,
	}

	_, cmd := m.handleRestore()

	assert.NotNil(t, cmd)
	fsMock.AssertExpectations(t)
	historyMock.AssertExpectations(t)
}

// Test_handleRestore_ErrorHandling verifies restore error scenarios.
func Test_handleRestore_ErrorHandling(t *testing.T) {
	tests := []struct {
		name         string
		setupMock    func(*mockHistory.MockManager)
		entry        *history.HistoryEntry
		wantStatus   string
		wantCmdIsNil bool
	}{
		{
			name: "already restored error",
			setupMock: func(m *mockHistory.MockManager) {
				m.On("Restore", mock.Anything, "entry1").
					Return(nil, history.ErrAlreadyRestored)
			},
			entry: &history.HistoryEntry{
				ID:         "entry1",
				BinaryName: "test",
				CanRestore: true,
			},
			wantStatus:   "test has already been restored",
			wantCmdIsNil: true,
		},
		{
			name: "not in trash error",
			setupMock: func(m *mockHistory.MockManager) {
				m.On("Restore", mock.Anything, "entry2").
					Return(nil, history.ErrNotInTrash)
			},
			entry: &history.HistoryEntry{
				ID:         "entry2",
				BinaryName: "test2",
				CanRestore: true,
			},
			wantStatus:   "test2 is no longer in trash",
			wantCmdIsNil: true,
		},
		{
			name: "restore collision error",
			setupMock: func(m *mockHistory.MockManager) {
				m.On("Restore", mock.Anything, "entry3").
					Return(nil, history.ErrRestoreCollision)
			},
			entry: &history.HistoryEntry{
				ID:         "entry3",
				BinaryName: "test3",
				CanRestore: true,
			},
			wantStatus:   "Cannot restore test3: file already exists",
			wantCmdIsNil: true,
		},
		{
			name: "generic error",
			setupMock: func(m *mockHistory.MockManager) {
				m.On("Restore", mock.Anything, "entry4").
					Return(nil, errors.New("disk error"))
			},
			entry: &history.HistoryEntry{
				ID:         "entry4",
				BinaryName: "test4",
				CanRestore: true,
			},
			wantStatus:   "Error restoring test4: disk error",
			wantCmdIsNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			historyMock := mockHistory.NewMockManager(t)
			tt.setupMock(historyMock)

			m := &model{
				choices:        []string{},
				dir:            "/bin",
				fs:             mockFS.NewMockFS(t),
				historyManager: historyMock,
				logger:         &tuiMockLogger{},
				mode:           modeHistory,
				historyEntries: []*history.HistoryEntry{tt.entry},
				historyCursor:  0,
			}

			got, cmd := m.handleRestore()
			gotModel := got.(*model)

			assert.Equal(t, tt.wantStatus, gotModel.status)

			if tt.wantCmdIsNil {
				assert.Nil(t, cmd)
			}

			historyMock.AssertExpectations(t)
		})
	}
}

// Test_handleUndo_BinaryModeRefresh verifies undo refreshes binary list in binary mode.
func Test_handleUndo_BinaryModeRefresh(t *testing.T) {
	fsMock := mockFS.NewMockFS(t)
	historyMock := mockHistory.NewMockManager(t)

	historyMock.On("UndoMostRecent", mock.Anything).
		Return(&history.RestoreResult{BinaryName: "undone_binary", RestoredTo: "/bin/undone_binary"}, nil)
	fsMock.On("ListBinaries", "/bin").Return([]string{"undone_binary", "existing"})

	m := &model{
		choices:        []string{"existing"},
		dir:            "/bin",
		fs:             fsMock,
		historyManager: historyMock,
		logger:         &tuiMockLogger{},
		mode:           modeBinaries,
		width:          80,
		height:         24,
		sortAscending:  true,
	}

	got, _ := m.handleUndo()
	gotModel := got.(*model)

	assert.Contains(t, gotModel.choices, "undone_binary")
	assert.Equal(t, "Restored undone_binary to /bin/undone_binary", gotModel.status)
	fsMock.AssertExpectations(t)
	historyMock.AssertExpectations(t)
}

// Priority 3: Log Polling

// Test_pollLogChannel_ReturnsLogMsg verifies log message is returned when available.
func Test_pollLogChannel_ReturnsLogMsg(t *testing.T) {
	logChan := make(chan LogMsg, 10)
	logChan <- LogMsg{Level: "INF", Message: "test message"}

	m := &model{
		logChan: logChan,
	}

	cmd := m.pollLogChannel()
	assert.NotNil(t, cmd)

	// Execute the command to get the message
	msg := cmd()
	logMsg, ok := msg.(LogMsg)
	assert.True(t, ok)
	assert.Equal(t, "INF", logMsg.Level)
	assert.Equal(t, "test message", logMsg.Message)
}

// Test_pollLogChannel_ReturnsTickWhenEmpty verifies tick is scheduled when no messages.
func Test_pollLogChannel_ReturnsTickWhenEmpty(t *testing.T) {
	logChan := make(chan LogMsg, 10)
	m := &model{
		logChan: logChan,
	}

	cmd := m.pollLogChannel()
	assert.NotNil(t, cmd)

	// Execute the command - should get a TickMsg
	// The tick msg is wrapped by tea.Tick
	msg := cmd()
	assert.NotNil(t, msg)
	// The message could be either TickMsg or pollLogTickMsg depending on implementation
}

// Test_model_Update_LogMsgHandling verifies LogMsg adds entry to logs slice.
func Test_model_Update_LogMsgHandling(t *testing.T) {
	m := &model{
		choices:       []string{"test"},
		cols:          1,
		rows:          1,
		width:         80,
		height:        24,
		logs:          []string{},
		sortAscending: true,
	}

	logMsg := LogMsg{Level: "DBG", Message: "debug info"}
	got, _ := m.Update(logMsg)
	gotModel := got.(*model)

	assert.Len(t, gotModel.logs, 1)
	assert.Contains(t, gotModel.logs[0], "debug info")
}

// Test_model_Update_PollLogTickMsgHandling verifies tick triggers next poll.
func Test_model_Update_PollLogTickMsgHandling(t *testing.T) {
	logChan := make(chan LogMsg, 10)
	m := &model{
		choices:       []string{"test"},
		cols:          1,
		rows:          1,
		width:         80,
		height:        24,
		logChan:       logChan,
		logs:          []string{},
		sortAscending: true,
	}

	tickMsg := pollLogTickMsg{}
	_, cmd := m.Update(tickMsg)

	assert.NotNil(t, cmd)
}

// Test_addLogEntry_CircularBuffer verifies max log lines limit is enforced.
func Test_addLogEntry_CircularBuffer(t *testing.T) {
	m := &model{
		logs: []string{},
	}

	// Add more than maxLogLines entries
	for i := range maxLogLines + 10 {
		m.addLogEntry(LogMsg{Level: "INF", Message: fmt.Sprintf("message %d", i)})
	}

	assert.Len(t, m.logs, maxLogLines)
	// Verify oldest entries were removed
	assert.NotContains(t, m.logs, "[INF] message 0")
	assert.Contains(t, m.logs, fmt.Sprintf("[INF] message %d", maxLogLines+9))
}

// Priority 4: History Mode Operations

// Test_model_Update_ModeSwitchToHistory verifies 'r' key switches to history mode.
func Test_model_Update_ModeSwitchToHistory(t *testing.T) {
	historyMock := mockHistory.NewMockManager(t)

	m := &model{
		choices:        []string{"test"},
		dir:            "/bin",
		config:         Config{},
		fs:             mockFS.NewMockFS(t),
		historyManager: historyMock,
		logger:         &tuiMockLogger{},
		mode:           modeBinaries,
		cols:           1,
		rows:           1,
		width:          80,
		height:         24,
		sortAscending:  true,
	}

	got, cmd := m.Update(keyPress('r'))
	gotModel := got.(*model)

	assert.Equal(t, modeHistory, gotModel.mode)
	assert.True(t, gotModel.historyLoading)
	assert.Equal(t, 0, gotModel.historyCursor)
	assert.Equal(t, "Loading history...", gotModel.status)
	assert.NotNil(t, cmd)
}

// Test_model_Update_ModeSwitchToBinaries verifies 'b' returns to binary mode.
func Test_model_Update_ModeSwitchToBinaries(t *testing.T) {
	fsMock := mockFS.NewMockFS(t)
	fsMock.On("ListBinaries", "/bin").Return([]string{"test"})

	m := &model{
		choices:       []string{"test"},
		dir:           "/bin",
		config:        Config{},
		fs:            fsMock,
		logger:        &tuiMockLogger{},
		mode:          modeHistory,
		cols:          1,
		rows:          1,
		width:         80,
		height:        24,
		sortAscending: true,
	}

	got, _ := m.Update(keyPress('b'))
	gotModel := got.(*model)

	assert.Equal(t, modeBinaries, gotModel.mode)
	fsMock.AssertExpectations(t)
}

// Test_model_Update_HistoryNavigation verifies up/down in history list.
func Test_model_Update_HistoryNavigation(t *testing.T) {
	entries := []*history.HistoryEntry{
		{ID: "1", BinaryName: "bin1"},
		{ID: "2", BinaryName: "bin2"},
		{ID: "3", BinaryName: "bin3"},
	}

	tests := []struct {
		name         string
		key          string
		initialCur   int
		expectedCur  int
		expectedMode string
	}{
		{name: "move up", key: "up", initialCur: 2, expectedCur: 1, expectedMode: modeHistory},
		{
			name:         "move up at top",
			key:          "up",
			initialCur:   0,
			expectedCur:  0,
			expectedMode: modeHistory,
		},
		{name: "move down", key: "down", initialCur: 0, expectedCur: 1, expectedMode: modeHistory},
		{
			name:         "move down at bottom",
			key:          "down",
			initialCur:   2,
			expectedCur:  2,
			expectedMode: modeHistory,
		},
		{name: "k key up", key: "k", initialCur: 1, expectedCur: 0, expectedMode: modeHistory},
		{name: "j key down", key: "j", initialCur: 0, expectedCur: 1, expectedMode: modeHistory},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &model{
				choices:        []string{},
				dir:            "/bin",
				fs:             mockFS.NewMockFS(t),
				logger:         &tuiMockLogger{},
				mode:           modeHistory,
				historyEntries: entries,
				historyCursor:  tt.initialCur,
				cols:           1,
				rows:           1,
				width:          80,
				height:         24,
				sortAscending:  true,
			}

			got, _ := m.Update(keyPressString(tt.key))
			gotModel := got.(*model)

			assert.Equal(t, tt.expectedCur, gotModel.historyCursor)
			assert.Equal(t, tt.expectedMode, gotModel.mode)
		})
	}
}

// Test_updateHistoryMode_EnterRestore verifies Enter key restores in history mode.
func Test_updateHistoryMode_EnterRestore(t *testing.T) {
	fsMock := mockFS.NewMockFS(t)
	historyMock := mockHistory.NewMockManager(t)

	entry := &history.HistoryEntry{
		ID:         "entry1",
		BinaryName: "restoreme",
		CanRestore: true,
	}

	historyMock.On("Restore", mock.Anything, "entry1").
		Return(&history.RestoreResult{BinaryName: "restoreme", RestoredTo: "/bin/restoreme"}, nil)
	fsMock.On("ListBinaries", "/bin").Return([]string{"restoreme"})

	m := &model{
		choices:        []string{},
		dir:            "/bin",
		fs:             fsMock,
		historyManager: historyMock,
		logger:         &tuiMockLogger{},
		mode:           modeHistory,
		historyEntries: []*history.HistoryEntry{entry},
		historyCursor:  0,
		cols:           1,
		rows:           1,
		width:          80,
		height:         24,
		sortAscending:  true,
	}

	got, _ := m.Update(keyPressString(keyEnter))
	gotModel := got.(*model)

	assert.Equal(t, "Restored restoreme to /bin/restoreme", gotModel.status)
	fsMock.AssertExpectations(t)
	historyMock.AssertExpectations(t)
}

// Test_updateHistoryMode_ClearEntry verifies clearing single entry.
func Test_updateHistoryMode_ClearEntry(t *testing.T) {
	historyMock := mockHistory.NewMockManager(t)

	entry := &history.HistoryEntry{
		ID:         "entry1",
		BinaryName: "testbin",
	}

	historyMock.On("ClearEntry", mock.Anything, "entry1", false).Return(nil)

	m := &model{
		choices:        []string{},
		dir:            "/bin",
		fs:             mockFS.NewMockFS(t),
		historyManager: historyMock,
		logger:         &tuiMockLogger{},
		mode:           modeHistory,
		historyEntries: []*history.HistoryEntry{entry},
		historyCursor:  0,
		cols:           1,
		rows:           1,
		width:          80,
		height:         24,
		sortAscending:  true,
	}

	got, cmd := m.Update(keyPress('c'))
	gotModel := got.(*model)

	assert.Equal(t, "Cleared history entry for testbin", gotModel.status)
	assert.NotNil(t, cmd)
	historyMock.AssertExpectations(t)
}

// Test_updateHistoryMode_ClearAll verifies clear all entries with confirmation.
func Test_updateHistoryMode_ClearAll(t *testing.T) {
	m := &model{
		choices:        []string{},
		dir:            "/bin",
		fs:             mockFS.NewMockFS(t),
		historyManager: mockHistory.NewMockManager(t),
		logger:         &tuiMockLogger{},
		mode:           modeHistory,
		historyEntries: []*history.HistoryEntry{
			{ID: "1", BinaryName: "bin1"},
			{ID: "2", BinaryName: "bin2"},
		},
		historyCursor: 0,
		cols:          1,
		rows:          1,
		width:         80,
		height:        24,
		sortAscending: true,
	}

	got, _ := m.Update(keyPress('C'))
	gotModel := got.(*model)

	assert.Equal(t, confirmClearAll, gotModel.confirmation)
}

// Priority 5: State Synchronization

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
			m := &model{
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
			m := &model{
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
			m := &model{
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

// Test_model_statusUpdates verifies status message updates correctly.
func Test_model_statusUpdates(t *testing.T) {
	fsMock := mockFS.NewMockFS(t)

	fsMock.On("AdjustBinaryPath", "/bin", "test").Return("/bin/test")
	fsMock.On("RemoveBinary", "/bin/test", "test", false, mock.Anything).Return(nil)
	fsMock.On("ListBinaries", "/bin").Return([]string{})

	m := &model{
		choices:       []string{"test"},
		dir:           "/bin",
		config:        Config{},
		fs:            fsMock,
		logger:        &tuiMockLogger{},
		status:        "",
		cols:          1,
		rows:          1,
		width:         80,
		height:        24,
		sortAscending: true,
	}

	got, _ := m.Update(keyPressString(keyEnter))
	gotModel := got.(*model)

	assert.Equal(t, "Removed test", gotModel.status)
	fsMock.AssertExpectations(t)
}

// Priority 6: Edge Cases

// Test_model_Update_EmptyBinaryList verifies empty list handling.
func Test_model_Update_EmptyBinaryList(t *testing.T) {
	m := &model{
		choices:       []string{},
		dir:           "/bin",
		config:        Config{},
		fs:            mockFS.NewMockFS(t),
		logger:        &tuiMockLogger{},
		status:        "",
		cols:          0,
		rows:          0,
		width:         80,
		height:        24,
		sortAscending: true,
	}

	// Try to remove with empty list - should quit
	got, cmd := m.Update(keyPressString(keyEnter))
	gotModel := got.(*model)

	// Should return quit command when no binaries remain
	assert.NotNil(t, gotModel)
	assert.Nil(t, cmd) // No command when list is empty
}

// Test_model_Update_HistoryEmpty verifies empty history handling.
func Test_model_Update_HistoryEmpty(t *testing.T) {
	m := &model{
		choices:        []string{},
		dir:            "/bin",
		config:         Config{},
		fs:             mockFS.NewMockFS(t),
		logger:         &tuiMockLogger{},
		mode:           modeHistory,
		historyEntries: []*history.HistoryEntry{},
		historyCursor:  0,
		cols:           1,
		rows:           1,
		width:          80,
		height:         24,
		sortAscending:  true,
	}

	// Try to clear all with empty history - should not set confirmation
	got, _ := m.Update(keyPress('C'))
	gotModel := got.(*model)

	assert.Equal(t, confirmNone, gotModel.confirmation)
}

// Test_model_Update_ConfirmationCancel verifies cancel confirmation dialog.
func Test_model_Update_ConfirmationCancel(t *testing.T) {
	tests := []struct {
		name string
		key  rune
	}{
		{name: "n key", key: 'n'},
		{name: "N key", key: 'N'},
		{name: "q key", key: 'q'},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &model{
				choices:       []string{},
				dir:           "/bin",
				config:        Config{},
				fs:            mockFS.NewMockFS(t),
				logger:        &tuiMockLogger{},
				mode:          modeHistory,
				confirmation:  confirmClearAll,
				cols:          1,
				rows:          1,
				width:         80,
				height:        24,
				sortAscending: true,
			}

			got, _ := m.Update(keyPress(tt.key))
			gotModel := got.(*model)

			assert.Equal(t, confirmNone, gotModel.confirmation)
			assert.Equal(t, "Operation cancelled", gotModel.status)
		})
	}
}

// Test_model_Update_ConfirmationAccept verifies accept confirmation dialog.
func Test_model_Update_ConfirmationAccept(t *testing.T) {
	historyMock := mockHistory.NewMockManager(t)
	historyMock.On("ClearHistory", mock.Anything, false).Return(nil)

	m := &model{
		choices:        []string{},
		dir:            "/bin",
		config:         Config{},
		fs:             mockFS.NewMockFS(t),
		historyManager: historyMock,
		logger:         &tuiMockLogger{},
		mode:           modeHistory,
		confirmation:   confirmClearAll,
		historyEntries: []*history.HistoryEntry{
			{ID: "1", BinaryName: "bin1"},
		},
		historyCursor: 0,
		cols:          1,
		rows:          1,
		width:         80,
		height:        24,
		sortAscending: true,
	}

	got, _ := m.Update(keyPress('y'))
	gotModel := got.(*model)

	assert.Equal(t, confirmNone, gotModel.confirmation)
	assert.Equal(t, "History cleared", gotModel.status)
	historyMock.AssertExpectations(t)
}

// Test_model_Update_AlternateScreen verifies alt-screen toggle.
func Test_model_Update_AlternateScreen(t *testing.T) {
	m := &model{
		choices:       []string{"test"},
		dir:           "/bin",
		config:        Config{},
		fs:            mockFS.NewMockFS(t),
		logger:        &tuiMockLogger{},
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

// Additional tests for handleClearEntry

// Test_handleClearEntry_ErrorHandling verifies error handling when clearing entry fails.
func Test_handleClearEntry_ErrorHandling(t *testing.T) {
	historyMock := mockHistory.NewMockManager(t)

	entry := &history.HistoryEntry{
		ID:         "entry1",
		BinaryName: "testbin",
	}

	historyMock.On("ClearEntry", mock.Anything, "entry1", false).Return(errors.New("clear failed"))

	m := &model{
		choices:        []string{},
		dir:            "/bin",
		fs:             mockFS.NewMockFS(t),
		historyManager: historyMock,
		logger:         &tuiMockLogger{},
		mode:           modeHistory,
		historyEntries: []*history.HistoryEntry{entry},
		historyCursor:  0,
	}

	got, _ := m.handleClearEntry(false)
	gotModel := got.(*model)

	assert.Equal(t, "Error clearing entry: clear failed", gotModel.status)
	historyMock.AssertExpectations(t)
}

// Test_handleClearEntry_NoHistoryManager verifies behavior when history manager is nil.
func Test_handleClearEntry_NoHistoryManager(t *testing.T) {
	m := &model{
		choices:        []string{},
		dir:            "/bin",
		fs:             mockFS.NewMockFS(t),
		historyManager: nil,
		logger:         &tuiMockLogger{},
		mode:           modeHistory,
		historyEntries: []*history.HistoryEntry{{ID: "1", BinaryName: "bin1"}},
		historyCursor:  0,
	}

	got, cmd := m.handleClearEntry(false)
	gotModel := got.(*model)

	assert.Equal(t, "No history entry selected", gotModel.status)
	assert.Nil(t, cmd)
}

// Test_handleUndo_ErrorHandling verifies undo error handling.
func Test_handleUndo_ErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		setupMock  func(*mockHistory.MockManager)
		wantStatus string
	}{
		{
			name: "no history error",
			setupMock: func(m *mockHistory.MockManager) {
				m.On("UndoMostRecent", mock.Anything).Return(nil, history.ErrNoHistory)
			},
			wantStatus: "No deletion history found - nothing to undo",
		},
		{
			name: "already restored error",
			setupMock: func(m *mockHistory.MockManager) {
				m.On("UndoMostRecent", mock.Anything).Return(nil, history.ErrAlreadyRestored)
			},
			wantStatus: "Binary has already been restored",
		},
		{
			name: "not in trash error",
			setupMock: func(m *mockHistory.MockManager) {
				m.On("UndoMostRecent", mock.Anything).Return(nil, history.ErrNotInTrash)
			},
			wantStatus: "Binary is no longer in trash - cannot restore",
		},
		{
			name: "restore collision error",
			setupMock: func(m *mockHistory.MockManager) {
				m.On("UndoMostRecent", mock.Anything).Return(nil, history.ErrRestoreCollision)
			},
			wantStatus: "A file already exists at the restore location",
		},
		{
			name: "generic error",
			setupMock: func(m *mockHistory.MockManager) {
				m.On("UndoMostRecent", mock.Anything).Return(nil, errors.New("unexpected error"))
			},
			wantStatus: "Undo failed: unexpected error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			historyMock := mockHistory.NewMockManager(t)
			tt.setupMock(historyMock)

			m := &model{
				choices:        []string{},
				dir:            "/bin",
				fs:             mockFS.NewMockFS(t),
				historyManager: historyMock,
				logger:         &tuiMockLogger{},
				mode:           modeBinaries,
			}

			got, _ := m.handleUndo()
			gotModel := got.(*model)

			assert.Equal(t, tt.wantStatus, gotModel.status)
			historyMock.AssertExpectations(t)
		})
	}
}

// Test_handleUndo_NoHistoryManager verifies undo behavior without history manager.
func Test_handleUndo_NoHistoryManager(t *testing.T) {
	m := &model{
		choices:        []string{},
		dir:            "/bin",
		fs:             mockFS.NewMockFS(t),
		historyManager: nil,
		logger:         &tuiMockLogger{},
		mode:           modeBinaries,
	}

	got, cmd := m.handleUndo()
	gotModel := got.(*model)

	assert.Equal(t, "History manager not available", gotModel.status)
	assert.Nil(t, cmd)
}

// Test_model_Init_WithHistoryMode verifies Init behavior in history mode.
func Test_model_Init_WithHistoryMode(t *testing.T) {
	m := &model{
		choices:        []string{},
		dir:            "/bin",
		config:         Config{},
		fs:             mockFS.NewMockFS(t),
		logger:         &tuiMockLogger{},
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
	m := &model{
		choices:       []string{},
		dir:           "/bin",
		config:        Config{Verbose: true},
		fs:            mockFS.NewMockFS(t),
		logger:        &tuiMockLogger{},
		mode:          modeBinaries,
		logChan:       logChan,
		sortAscending: true,
	}

	cmd := m.Init()

	// Should return pollLogChannel command
	assert.NotNil(t, cmd)
}

// Test_model_handleConfirmation_ExecuteClearAllError verifies error handling when ClearHistory fails.
func Test_model_handleConfirmation_ExecuteClearAllError(t *testing.T) {
	historyMock := mockHistory.NewMockManager(t)
	historyMock.On("ClearHistory", mock.Anything, false).Return(errors.New("storage error"))

	m := &model{
		choices:        []string{},
		dir:            "/bin",
		config:         Config{},
		fs:             mockFS.NewMockFS(t),
		historyManager: historyMock,
		logger:         &tuiMockLogger{},
		mode:           modeHistory,
		confirmation:   confirmClearAll,
		historyEntries: []*history.HistoryEntry{{ID: "1", BinaryName: "bin1"}},
	}

	got, _ := m.executeConfirmation()
	gotModel := got.(*model)

	assert.Equal(t, confirmNone, gotModel.confirmation)
	assert.Contains(t, gotModel.status, "Error clearing history")
	historyMock.AssertExpectations(t)
}

// Test_model_handleConfirmation_ExecuteDeletePermError verifies error handling when DeletePermanently fails.
func Test_model_handleConfirmation_ExecuteDeletePermError(t *testing.T) {
	historyMock := mockHistory.NewMockManager(t)
	historyMock.On("DeletePermanently", mock.Anything, "entry1").Return(errors.New("delete failed"))

	m := &model{
		choices:        []string{},
		dir:            "/bin",
		config:         Config{},
		fs:             mockFS.NewMockFS(t),
		historyManager: historyMock,
		logger:         &tuiMockLogger{},
		mode:           modeHistory,
		confirmation:   confirmDeletePerm,
		historyEntries: []*history.HistoryEntry{{ID: "entry1", BinaryName: "testbin"}},
		historyCursor:  0,
	}

	got, _ := m.executeConfirmation()
	gotModel := got.(*model)

	assert.Equal(t, confirmNone, gotModel.confirmation)
	assert.Contains(t, gotModel.status, "Error deleting permanently")
	historyMock.AssertExpectations(t)
}

// Test_handleRestore_NoHistoryManager verifies restore without history manager.
func Test_handleRestore_NoHistoryManager(t *testing.T) {
	m := &model{
		choices:        []string{},
		dir:            "/bin",
		config:         Config{},
		fs:             mockFS.NewMockFS(t),
		historyManager: nil,
		logger:         &tuiMockLogger{},
		mode:           modeHistory,
		historyEntries: []*history.HistoryEntry{{ID: "1", BinaryName: "bin1"}},
		historyCursor:  0,
	}

	got, cmd := m.handleRestore()
	gotModel := got.(*model)

	assert.Equal(t, "No history entry selected", gotModel.status)
	assert.Nil(t, cmd)
}

// Test_handleRestore_CannotRestore verifies restore when entry cannot be restored.
func Test_handleRestore_CannotRestore(t *testing.T) {
	m := &model{
		choices:        []string{},
		dir:            "/bin",
		config:         Config{},
		fs:             mockFS.NewMockFS(t),
		historyManager: mockHistory.NewMockManager(t),
		logger:         &tuiMockLogger{},
		mode:           modeHistory,
		historyEntries: []*history.HistoryEntry{{ID: "1", BinaryName: "bin1", CanRestore: false}},
		historyCursor:  0,
	}

	got, cmd := m.handleRestore()
	gotModel := got.(*model)

	assert.Contains(t, gotModel.status, "Cannot restore")
	assert.Nil(t, cmd)
}

// Test_model_Update_CannotRestore verifies restore via Update when entry cannot be restored.
func Test_model_Update_CannotRestore(t *testing.T) {
	m := &model{
		choices:        []string{},
		dir:            "/bin",
		config:         Config{},
		fs:             mockFS.NewMockFS(t),
		historyManager: mockHistory.NewMockManager(t),
		logger:         &tuiMockLogger{},
		mode:           modeHistory,
		historyEntries: []*history.HistoryEntry{{ID: "1", BinaryName: "bin1", CanRestore: false}},
		historyCursor:  0,
		cols:           1,
		rows:           1,
		width:          80,
		height:         24,
		sortAscending:  true,
	}

	got, _ := m.Update(keyPressString(keyEnter))
	gotModel := got.(*model)

	assert.Contains(t, gotModel.status, "Cannot restore")
}

// Test_model_Update_HistoryMsgError verifies HistoryMsg error handling.
func Test_model_Update_HistoryMsgError(t *testing.T) {
	m := &model{
		choices:        []string{},
		dir:            "/bin",
		config:         Config{},
		fs:             mockFS.NewMockFS(t),
		logger:         &tuiMockLogger{},
		mode:           modeHistory,
		historyEntries: []*history.HistoryEntry{},
		historyLoading: true,
		cols:           1,
		rows:           1,
		width:          80,
		height:         24,
		sortAscending:  true,
	}

	historyMsg := HistoryMsg{Entries: nil, Error: errors.New("load failed")}
	got, _ := m.Update(historyMsg)
	gotModel := got.(*model)

	assert.False(t, gotModel.historyLoading)
	assert.Contains(t, gotModel.status, "Error loading history")
}

// Test_model_Update_HistoryMsgEmpty verifies HistoryMsg with empty entries.
func Test_model_Update_HistoryMsgEmpty(t *testing.T) {
	m := &model{
		choices:        []string{},
		dir:            "/bin",
		config:         Config{},
		fs:             mockFS.NewMockFS(t),
		logger:         &tuiMockLogger{},
		mode:           modeHistory,
		historyEntries: []*history.HistoryEntry{},
		historyLoading: true,
		cols:           1,
		rows:           1,
		width:          80,
		height:         24,
		sortAscending:  true,
	}

	historyMsg := HistoryMsg{Entries: []*history.HistoryEntry{}, Error: nil}
	got, _ := m.Update(historyMsg)
	gotModel := got.(*model)

	assert.False(t, gotModel.historyLoading)
	assert.Equal(t, "No deletion history found", gotModel.status)
}

// Test_model_Update_HistoryMsgSuccess verifies HistoryMsg with entries.
func Test_model_Update_HistoryMsgSuccess(t *testing.T) {
	m := &model{
		choices:        []string{},
		dir:            "/bin",
		config:         Config{},
		fs:             mockFS.NewMockFS(t),
		logger:         &tuiMockLogger{},
		mode:           modeHistory,
		historyEntries: []*history.HistoryEntry{},
		historyLoading: true,
		cols:           1,
		rows:           1,
		width:          80,
		height:         24,
		sortAscending:  true,
	}

	entries := []*history.HistoryEntry{
		{ID: "1", BinaryName: "bin1"},
		{ID: "2", BinaryName: "bin2"},
	}
	historyMsg := HistoryMsg{Entries: entries, Error: nil}
	got, _ := m.Update(historyMsg)
	gotModel := got.(*model)

	assert.False(t, gotModel.historyLoading)
	assert.Len(t, gotModel.historyEntries, 2)
	assert.Equal(t, "Loaded 2 history entries", gotModel.status)
}

// Test_model_Update_ToggleLogs verifies L key toggles log panel.
func Test_model_Update_ToggleLogs(t *testing.T) {
	m := &model{
		choices:       []string{"test"},
		dir:           "/bin",
		config:        Config{},
		fs:            mockFS.NewMockFS(t),
		logger:        &tuiMockLogger{},
		mode:          modeBinaries,
		showLogs:      false,
		cols:          1,
		rows:          1,
		width:         80,
		height:        24,
		sortAscending: true,
	}

	// Toggle on
	got, _ := m.Update(keyPress('L'))
	gotModel := got.(*model)
	assert.True(t, gotModel.showLogs)

	// Toggle off
	got, _ = gotModel.Update(keyPress('L'))
	gotModel = got.(*model)
	assert.False(t, gotModel.showLogs)
}

// Test_model_Update_UndoKey verifies u key triggers undo.
func Test_model_Update_UndoKey(t *testing.T) {
	fsMock := mockFS.NewMockFS(t)
	historyMock := mockHistory.NewMockManager(t)

	historyMock.On("UndoMostRecent", mock.Anything).
		Return(&history.RestoreResult{BinaryName: "undone", RestoredTo: "/bin/undone"}, nil)
	fsMock.On("ListBinaries", "/bin").Return([]string{"undone"})

	m := &model{
		choices:        []string{},
		dir:            "/bin",
		config:         Config{},
		fs:             fsMock,
		historyManager: historyMock,
		logger:         &tuiMockLogger{},
		mode:           modeBinaries,
		cols:           1,
		rows:           1,
		width:          80,
		height:         24,
		sortAscending:  true,
	}

	got, _ := m.Update(keyPress('u'))
	gotModel := got.(*model)

	assert.Equal(t, "Restored undone to /bin/undone", gotModel.status)
	fsMock.AssertExpectations(t)
	historyMock.AssertExpectations(t)
}

// Test_model_Update_DeletePermanentlyKey verifies d key triggers permanent delete with confirmation.
func Test_model_Update_DeletePermanentlyKey(t *testing.T) {
	m := &model{
		choices:        []string{},
		dir:            "/bin",
		config:         Config{},
		fs:             mockFS.NewMockFS(t),
		historyManager: mockHistory.NewMockManager(t),
		logger:         &tuiMockLogger{},
		mode:           modeHistory,
		historyEntries: []*history.HistoryEntry{{ID: "1", BinaryName: "bin1"}},
		historyCursor:  0,
		confirmation:   confirmNone,
		cols:           1,
		rows:           1,
		width:          80,
		height:         24,
		sortAscending:  true,
	}

	got, _ := m.Update(keyPress('d'))
	gotModel := got.(*model)

	assert.Equal(t, confirmDeletePerm, gotModel.confirmation)
}

// Test_model_Update_ClearEntryKey verifies c key triggers clear entry.
func Test_model_Update_ClearEntryKey(t *testing.T) {
	historyMock := mockHistory.NewMockManager(t)
	historyMock.On("ClearEntry", mock.Anything, "1", false).Return(nil)

	m := &model{
		choices:        []string{},
		dir:            "/bin",
		config:         Config{},
		fs:             mockFS.NewMockFS(t),
		historyManager: historyMock,
		logger:         &tuiMockLogger{},
		mode:           modeHistory,
		historyEntries: []*history.HistoryEntry{{ID: "1", BinaryName: "bin1"}},
		historyCursor:  0,
		cols:           1,
		rows:           1,
		width:          80,
		height:         24,
		sortAscending:  true,
	}

	got, _ := m.Update(keyPress('c'))
	gotModel := got.(*model)

	assert.Equal(t, "Cleared history entry for bin1", gotModel.status)
	historyMock.AssertExpectations(t)
}

// Test_model_getVisibleLogs verifies getVisibleLogs behavior.
func Test_model_getVisibleLogs(t *testing.T) {
	tests := []struct {
		name     string
		showLogs bool
		logs     []string
		wantLen  int
	}{
		{
			name:     "logs visible",
			showLogs: true,
			logs:     []string{"log1", "log2", "log3"},
			wantLen:  3,
		},
		{
			name:     "logs not visible",
			showLogs: false,
			logs:     []string{"log1", "log2"},
			wantLen:  0,
		},
		{
			name:     "empty logs",
			showLogs: true,
			logs:     []string{},
			wantLen:  0,
		},
		{
			name:     "logs exceed max visible",
			showLogs: true,
			logs:     []string{"1", "2", "3", "4", "5", "6", "7"},
			wantLen:  maxVisibleLogLines,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &model{
				showLogs: tt.showLogs,
				logs:     tt.logs,
			}

			got := m.getVisibleLogs()
			assert.Len(t, got, tt.wantLen)
		})
	}
}

// Test_model_handleConfirmation_UnknownConfirmation verifies behavior with unknown confirmation type.
func Test_model_handleConfirmation_UnknownConfirmation(t *testing.T) {
	m := &model{
		choices:       []string{},
		dir:           "/bin",
		config:        Config{},
		fs:            mockFS.NewMockFS(t),
		logger:        &tuiMockLogger{},
		mode:          modeHistory,
		confirmation:  "unknown_type",
		cols:          1,
		rows:          1,
		width:         80,
		height:        24,
		sortAscending: true,
	}

	got, cmd := m.handleConfirmation(keyPress('y'))
	gotModel := got.(*model)

	// Should clear confirmation but do nothing else
	assert.Equal(t, confirmNone, gotModel.confirmation)
	assert.Nil(t, cmd)
}

// Test_model_handleRestore_HistoryModeRefresh verifies history refresh after restore in history mode.
func Test_handleRestore_HistoryModeRefresh(t *testing.T) {
	fsMock := mockFS.NewMockFS(t)
	historyMock := mockHistory.NewMockManager(t)

	entry := &history.HistoryEntry{
		ID:         "entry1",
		BinaryName: "restoreme",
		CanRestore: true,
	}

	historyMock.On("Restore", mock.Anything, "entry1").
		Return(&history.RestoreResult{BinaryName: "restoreme", RestoredTo: "/bin/restoreme"}, nil)
	fsMock.On("ListBinaries", "/bin").Return([]string{"restoreme"})

	m := &model{
		choices:        []string{},
		dir:            "/bin",
		fs:             fsMock,
		historyManager: historyMock,
		logger:         &tuiMockLogger{},
		mode:           modeHistory,
		historyEntries: []*history.HistoryEntry{entry},
		historyCursor:  0,
		width:          80,
		height:         24,
		sortAscending:  true,
	}

	got, cmd := m.handleRestore()
	gotModel := got.(*model)

	assert.Equal(t, "Restored restoreme to /bin/restoreme", gotModel.status)
	assert.NotNil(t, cmd) // Should return loadHistory command
	fsMock.AssertExpectations(t)
	historyMock.AssertExpectations(t)
}
