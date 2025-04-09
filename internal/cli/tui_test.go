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
	"testing"

	"github.com/stretchr/testify/mock"

	tea "github.com/charmbracelet/bubbletea"

	fsmocks "github.com/nicholas-fedor/go-remove/internal/fs/mocks"
	logmocks "github.com/nicholas-fedor/go-remove/internal/logger/mocks"
)

// tuiMockRunner mocks the ProgramRunner interface for TUI tests.
type tuiMockRunner struct {
	runProgram func(tea.Model, ...tea.ProgramOption) (*tea.Program, error)
}

// RunProgram executes the mock runner’s program function.
func (m *tuiMockRunner) RunProgram(
	model tea.Model,
	opts ...tea.ProgramOption,
) (*tea.Program, error) {
	return m.runProgram(model, opts...)
}

// TestRunTUI verifies the RunTUI function’s behavior under various conditions.
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

// Test_model_Init verifies the Init method’s command output.
func Test_model_Init(t *testing.T) {
	tests := []struct {
		name string
		m    model
		want tea.Cmd
	}{
		{
			name: "basic init",
			m:    model{},
			want: tea.EnterAltScreen,
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

// Test_model_Update verifies the Update method’s state changes and commands.
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
			args:    args{msg: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}},
			want:    model{},
			wantCmd: tea.Quit,
		},
		{
			name:    "move up",
			m:       &model{cursorY: 1, rows: 2},
			args:    args{msg: tea.KeyMsg{Type: tea.KeyUp}},
			want:    model{cursorY: 0, rows: 2},
			wantCmd: nil,
		},
		{
			name:    "move down within bounds",
			m:       &model{cursorY: 0, rows: 2, cols: 2, choices: []string{"a", "b", "c", "d"}},
			args:    args{msg: tea.KeyMsg{Type: tea.KeyDown}},
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
			args: args{msg: tea.KeyMsg{Type: tea.KeyDown}},
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
			args:    args{msg: tea.KeyMsg{Type: tea.KeyLeft}},
			want:    model{cursorX: 0, cols: 2},
			wantCmd: nil,
		},
		{
			name:    "move right within bounds",
			m:       &model{cursorX: 0, cols: 2, choices: []string{"a", "b"}},
			args:    args{msg: tea.KeyMsg{Type: tea.KeyRight}},
			want:    model{cursorX: 1, cols: 2, choices: []string{"a", "b"}},
			wantCmd: nil,
		},
		{
			name: "enter removes binary",
			m: &model{
				choices: []string{"vhs", "age"},
				cols:    2,
				rows:    1,
				dir:     "/bin",
				config:  Config{Verbose: false},
				logger: func() *logmocks.MockLogger {
					m := logmocks.NewMockLogger(t)

					return m
				}(),
				fs: func() *fsmocks.MockFS {
					m := fsmocks.NewMockFS(t)
					m.On("AdjustBinaryPath", "/bin", "vhs").Return("/bin/vhs")
					m.On("RemoveBinary", "/bin/vhs", "vhs", false, mock.Anything).Return(nil)
					m.On("ListBinaries", "/bin").Return([]string{"age"})

					return m
				}(),
			},
			args: args{msg: tea.KeyMsg{Type: tea.KeyEnter}},
			want: model{
				choices: []string{"age"},
				cols:    1,
				rows:    1,
				dir:     "/bin",
				config:  Config{Verbose: false},
				status:  "Removed vhs",
			},
			wantCmd: nil,
		},
		{
			name: "enter with error",
			m: &model{
				choices: []string{"vhs"},
				cols:    1,
				rows:    1,
				dir:     "/bin",
				logger: func() *logmocks.MockLogger {
					m := logmocks.NewMockLogger(t)

					return m
				}(),
				fs: func() *fsmocks.MockFS {
					m := fsmocks.NewMockFS(t)
					m.On("AdjustBinaryPath", "/bin", "vhs").Return("/bin/vhs")
					m.On("RemoveBinary", "/bin/vhs", "vhs", false, mock.Anything).
						Return(errors.New("remove failed"))

					return m
				}(),
			},
			args: args{msg: tea.KeyMsg{Type: tea.KeyEnter}},
			want: model{
				choices: []string{"vhs"},
				cols:    1,
				rows:    1,
				dir:     "/bin",
				status:  "Error removing vhs: remove failed",
			},
			wantCmd: nil,
		},
		{
			name: "window size update",
			m:    &model{choices: []string{"a", "b"}},
			args: args{msg: tea.WindowSizeMsg{Width: 80, Height: 24}},
			want: model{
				choices: []string{"a", "b"},
				cols:    2,
				rows:    1,
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
				gotModel.status != tt.want.status {
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

// Test_model_updateGrid verifies the updateGrid method’s layout calculations.
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
				cols:    3,
				rows:    1,
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

// Test_model_View verifies the View method’s rendered output.
func Test_model_View(t *testing.T) {
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
				choices: []string{"vhs"},
				cols:    1,
				rows:    1,
				width:   80,
				height:  24,
				cursorX: 0,
				cursorY: 0,
				styles:  defaultStyleConfig(),
			},
			want: "  Select a binary to remove:                                       \n" +
				"                                                                   \n" +
				"  ❯ vhs                                                            \n" +
				"                                                                   \n" +
				"                                                                   \n" +
				"                                                                   \n" +
				"                                                                   \n" +
				"                                                                   \n" +
				"                                                                   \n" +
				"                                                                   \n" +
				"                                                                   \n" +
				"                                                                   \n" +
				"                                                                   \n" +
				"                                                                   \n" +
				"                                                                   \n" +
				"                                                                   \n" +
				"                                                                   \n" +
				"                                                                   \n" +
				"                                                                   \n" +
				"                                                                   \n" +
				"  ↑/k: up  ↓/j: down  ←/h: left  →/l: right  Enter: remove  q: quit",
		},
		{
			name: "multiple_choices_with_status",
			m: model{
				choices: []string{"vhs", "age"},
				cols:    2,
				rows:    1,
				width:   80,
				height:  24,
				status:  "Removed tool",
				cursorX: 0,
				cursorY: 0,
				styles:  defaultStyleConfig(),
			},
			want: "  Select a binary to remove:                                       \n" +
				"                                                                   \n" +
				"  ❯ vhs   age                                                      \n" +
				"                                                                   \n" +
				"  Removed tool                                                     \n" +
				"                                                                   \n" +
				"                                                                   \n" +
				"                                                                   \n" +
				"                                                                   \n" +
				"                                                                   \n" +
				"                                                                   \n" +
				"                                                                   \n" +
				"                                                                   \n" +
				"                                                                   \n" +
				"                                                                   \n" +
				"                                                                   \n" +
				"                                                                   \n" +
				"                                                                   \n" +
				"                                                                   \n" +
				"                                                                   \n" +
				"                                                                   \n" +
				"  ↑/k: up  ↓/j: down  ←/h: left  →/l: right  Enter: remove  q: quit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.m.updateGrid()

			got := tt.m.View()
			want := tt.want

			if got != want {
				t.Errorf("model.View() got = %q, want %q", got, want)
			}
		})
	}
}

// Test_max verifies the max function’s comparison logic.
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
			if got := max(tt.a, tt.b); got != tt.want {
				t.Errorf("max() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test_min verifies the min function’s comparison logic.
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
			if got := min(tt.a, tt.b); got != tt.want {
				t.Errorf("min() = %v, want %v", got, tt.want)
			}
		})
	}
}
