/*
Copyright © 2025 Nicholas Fedor <nick@nickfedor.com>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/

package cli

import (
	"errors"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/mock"

	tea "charm.land/bubbletea/v2"

	fsmocks "github.com/nicholas-fedor/go-remove/internal/fs/mocks"
	logmocks "github.com/nicholas-fedor/go-remove/internal/logger/mocks"
)

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
		logger *logmocks.MockLogger
		fs     *fsmocks.MockFS
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
				logger: logmocks.NewMockLogger(t),
				fs: func() *fsmocks.MockFS {
					m := fsmocks.NewMockFS(t)
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
				logger: logmocks.NewMockLogger(t),
				fs: func() *fsmocks.MockFS {
					m := fsmocks.NewMockFS(t)
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
				logger: logmocks.NewMockLogger(t),
				fs: func() *fsmocks.MockFS {
					m := fsmocks.NewMockFS(t)
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
			err := RunTUI(tt.args.dir, tt.args.config, tt.args.logger, tt.args.fs, tt.args.runner)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunTUI() error = %v, wantErr %v", err, tt.wantErr)
			}

			tt.args.fs.AssertExpectations(t)
			tt.args.logger.AssertExpectations(t)
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
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "left":
		return tea.KeyPressMsg{Code: tea.KeyLeft}
	case "right":
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
			args:    args{msg: keyPressString("down")},
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
			args: args{msg: keyPressString("down")},
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
				logger: func() *logmocks.MockLogger {
					m := logmocks.NewMockLogger(t)

					return m
				}(),
				fs: func() *fsmocks.MockFS {
					m := fsmocks.NewMockFS(t)
					m.On("AdjustBinaryPath", "/bin", "age").Return("/bin/age")
					m.On("RemoveBinary", "/bin/age", "age", false, mock.Anything).Return(nil)
					m.On("ListBinaries", "/bin").Return([]string{"vhs"})

					return m
				}(),
			},
			args: args{msg: keyPressString("enter")},
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
				logger: func() *logmocks.MockLogger {
					m := logmocks.NewMockLogger(t)

					return m
				}(),
				fs: func() *fsmocks.MockFS {
					m := fsmocks.NewMockFS(t)
					m.On("AdjustBinaryPath", "/bin", "age").Return("/bin/age")
					m.On("RemoveBinary", "/bin/age", "age", false, mock.Anything).
						Return(errors.New("remove failed"))

					return m
				}(),
			},
			args: args{msg: keyPressString("enter")},
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
				tt.m.logger = logmocks.NewMockLogger(t)
			}

			if tt.m.fs == nil {
				tt.m.fs = fsmocks.NewMockFS(t)
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

			tt.m.fs.(*fsmocks.MockFS).AssertExpectations(t)
			tt.m.logger.(*logmocks.MockLogger).AssertExpectations(t)
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
				)
				lines = append(lines, leftPaddingStr+pad("", effectiveWidth))
				lines = append(lines, leftPaddingStr+pad("❯ vhs", effectiveWidth))

				lines = append(lines, leftPaddingStr+pad("", effectiveWidth))
				for range 19 {
					lines = append(lines, leftPaddingStr+pad("", effectiveWidth))
				}

				footerPart1 := "↑/k: up  ↓/j: down  ←/h: left  →/l: right  Enter: remove  s: toggle sort  q:"
				footerPart2 := "quit"

				lines = append(lines, leftPaddingStr+pad(footerPart1, effectiveWidth))
				lines = append(lines, leftPaddingStr+pad(footerPart2, effectiveWidth))

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
				)
				lines = append(lines, leftPaddingStr+pad("", effectiveWidth))
				lines = append(lines, leftPaddingStr+pad("❯ age", effectiveWidth))
				lines = append(lines, leftPaddingStr+pad(
					"  vhs",
					effectiveWidth,
				)) // Adjusted to match actual padding
				lines = append(lines, leftPaddingStr+pad("", effectiveWidth))

				lines = append(lines, leftPaddingStr+pad("Removed tool", effectiveWidth))
				for range 17 { // Adjusted for rows=2
					lines = append(lines, leftPaddingStr+pad("", effectiveWidth))
				}

				footerPart1 := "↑/k: up  ↓/j: down  ←/h: left  →/l: right  Enter: remove  s: toggle sort  q:"
				footerPart2 := "quit"

				lines = append(lines, leftPaddingStr+pad(footerPart1, effectiveWidth))
				lines = append(lines, leftPaddingStr+pad(footerPart2, effectiveWidth))

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
