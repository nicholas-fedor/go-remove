/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package tui

import (
	"bytes"
	"context"
	"errors"
	"io"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	mockFS "github.com/nicholas-fedor/go-remove/internal/fs/mocks"
	"github.com/nicholas-fedor/go-remove/internal/logger"
	"github.com/nicholas-fedor/go-remove/internal/tui/models"
)

// noProgramOptions keeps a test from drawing a real view when Run never gets as
// far as running a program.
//
// Input is disabled and output is discarded, so nothing reaches a terminal
// whether or not a program is created.
func noProgramOptions() []tea.ProgramOption {
	return []tea.ProgramOption{tea.WithInput(nil), tea.WithOutput(io.Discard)}
}

// runProgramOptions starts a real program that draws into a discard sink and
// quits on the first key, so a case that must exercise the program reaches a
// clean shutdown without a terminal behind it.
func runProgramOptions() []tea.ProgramOption {
	return []tea.ProgramOption{
		tea.WithInput(strings.NewReader("q")),
		tea.WithOutput(io.Discard),
		tea.WithWindowSize(80, 24),
		tea.WithoutSignals(),
	}
}

// alwaysTerminal reports an interactive terminal, standing in for the real
// check that go test cannot satisfy.
func alwaysTerminal() bool { return true }

// expiredContext returns a context whose deadline has already passed, so the
// program is killed as soon as it starts rather than being left to run.
func expiredContext(t *testing.T) context.Context {
	t.Helper()

	ctx, cancel := context.WithDeadline(t.Context(), time.Now().Add(-time.Hour))
	t.Cleanup(cancel)

	return ctx
}

// TestRunTUI verifies the Run function's behavior under various conditions.
func TestRunTUI(t *testing.T) {
	type args struct {
		dir             string
		config          models.Config
		logger          logger.Logger
		fs              *mockFS.MockFS
		ctx             func(t *testing.T) context.Context
		programOptions  []tea.ProgramOption
		stdinIsTerminal func() bool
	}

	tests := []struct {
		name      string
		args      args
		wantErr   bool
		wantErrIs error
	}{
		{
			name: "success with binaries",
			args: args{
				dir:    "/bin",
				config: models.Config{},
				logger: logger.NopLogger(),
				fs: func() *mockFS.MockFS {
					m := mockFS.NewMockFS(t)
					m.On("ListBinaries", "/bin").Return([]string{"vhs"}, nil)

					return m
				}(),
				programOptions:  runProgramOptions(),
				stdinIsTerminal: alwaysTerminal,
			},
			wantErr: false,
		},
		{
			name: "no binaries",
			args: args{
				dir:    "/bin",
				config: models.Config{},
				logger: logger.NopLogger(),
				fs: func() *mockFS.MockFS {
					m := mockFS.NewMockFS(t)
					m.On("ListBinaries", "/bin").Return([]string{}, nil)

					return m
				}(),
				// The options stay inert because Run reports the empty listing
				// before it ever creates a program.
				programOptions:  noProgramOptions(),
				stdinIsTerminal: alwaysTerminal,
			},
			wantErr: true,
			// The listing is reported as itself rather than as a missing
			// terminal.
			wantErrIs: ErrNoBinariesFound,
		},
		{
			name: "program error",
			args: args{
				dir:    "/bin",
				config: models.Config{},
				logger: logger.NopLogger(),
				fs: func() *mockFS.MockFS {
					m := mockFS.NewMockFS(t)
					m.On("ListBinaries", "/bin").Return([]string{"vhs"}, nil)

					return m
				}(),
				ctx:             expiredContext,
				programOptions:  noProgramOptions(),
				stdinIsTerminal: alwaysTerminal,
			},
			wantErr: true,
			// Creating a program can no longer fail on its own, so the error
			// path is the one that reaches the running program.
			wantErrIs: tea.ErrProgramKilled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			if tt.args.ctx != nil {
				ctx = tt.args.ctx(t)
			}

			err := Run(ctx, Options{
				Dir:             tt.args.dir,
				Config:          tt.args.config,
				Logger:          tt.args.logger,
				FS:              tt.args.fs,
				ProgramOptions:  tt.args.programOptions,
				StdinIsTerminal: tt.args.stdinIsTerminal,
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
				t.Errorf("Run() error = %v, want it to wrap %v", err, tt.wantErrIs)
			}

			tt.args.fs.AssertExpectations(t)
		})
	}
}

// TestRunTUI_NoTerminal verifies the interactive path refuses to start without
// a terminal, and says so in terms the user can act on.
func TestRunTUI_NoTerminal(t *testing.T) {
	mockFSInstance := mockFS.NewMockFS(t)
	mockFSInstance.On("ListBinaries", "/bin").Return([]string{"vhs"}, nil).Maybe()

	err := Run(t.Context(), Options{
		Dir:             "/bin",
		Logger:          logger.NopLogger(),
		FS:              mockFSInstance,
		ProgramOptions:  noProgramOptions(),
		StdinIsTerminal: func() bool { return false },
	})

	if !errors.Is(err, ErrNotATerminal) {
		t.Fatalf("Run() error = %v, want %v", err, ErrNotATerminal)
	}

	if !strings.Contains(err.Error(), "pass a binary name") {
		t.Errorf("Run() error = %v, want it to point at the non-interactive form", err)
	}
}

// TestRunTUI_DirectoryReadFailure verifies a read failure is reported as itself
// rather than as an empty directory.
func TestRunTUI_DirectoryReadFailure(t *testing.T) {
	readErr := errors.New("permission denied")
	mockFSInstance := mockFS.NewMockFS(t)
	mockFSInstance.On("ListBinaries", "/bin").Return(nil, readErr)

	err := Run(t.Context(), Options{
		Dir:             "/bin",
		Logger:          logger.NopLogger(),
		FS:              mockFSInstance,
		ProgramOptions:  noProgramOptions(),
		StdinIsTerminal: alwaysTerminal,
	})

	if !errors.Is(err, readErr) {
		t.Fatalf("Run() error = %v, want %v", err, readErr)
	}

	if errors.Is(err, ErrNoBinariesFound) {
		t.Error("an unreadable directory must not be reported as empty")
	}
}

// sgrPattern matches an SGR sequence, the escape form that carries colour, and
// captures its parameters.
var sgrPattern = regexp.MustCompile("\x1b\\[([0-9;]*)m")

// TestRun_NO_COLOR pins the colour profile Bubble Tea derives from the
// environment, so a view never reaches a user who asked for no colour.
//
// The output is a bytes.Buffer, which is not a term.File, so detection would
// settle on NoTTY and every style would be stripped whether NO_COLOR was set or
// not. TTY_FORCE makes detection treat the stream as a terminal, which is what
// puts the two cases on the profile boundary NO_COLOR is defined against: ASCII
// with it set, ANSI256 without.
//
// A colour sequence is asserted rather than any SGR sequence, because NO_COLOR
// disables colour and not text decoration, as https://no-color.org/ requires. The
// bold title survives it.
func TestRun_NO_COLOR(t *testing.T) {
	tests := []struct {
		name      string
		env       []string
		wantColor bool
	}{
		{
			name:      "colour when NO_COLOR is unset",
			env:       []string{"TERM=xterm-256color", "TTY_FORCE=1"},
			wantColor: true,
		},
		{
			name:      "no colour with NO_COLOR set",
			env:       []string{"NO_COLOR=1", "TERM=xterm-256color", "TTY_FORCE=1"},
			wantColor: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			mockFSInstance := mockFS.NewMockFS(t)
			mockFSInstance.On("ListBinaries", "/bin").Return([]string{"vhs"}, nil)

			err := Run(t.Context(), Options{
				Dir:    "/bin",
				Logger: logger.NopLogger(),
				FS:     mockFSInstance,
				ProgramOptions: []tea.ProgramOption{
					tea.WithInput(strings.NewReader("q")),
					tea.WithOutput(&buf),
					tea.WithWindowSize(80, 24),
					tea.WithoutSignals(),
					tea.WithEnvironment(tt.env),
				},
				StdinIsTerminal: alwaysTerminal,
			})
			if err != nil {
				t.Fatalf("Run() error = %v, want nil", err)
			}

			out := buf.String()

			// Without a view there is nothing for either assertion to mean, so
			// the content is checked before the styling around it.
			if !strings.Contains(out, "Select a binary to remove:") {
				t.Fatalf("Run() wrote no view, got %q", out)
			}

			if got := hasColorSGR(out); got != tt.wantColor {
				t.Errorf(
					"Run() colour sequence present = %v, want %v, got %q",
					got,
					tt.wantColor,
					out,
				)
			}
		})
	}
}

// hasColorSGR reports whether any SGR sequence in s asks for a foreground or
// background colour.
//
// Parameters:
//   - s: Rendered program output.
//
// Returns:
//   - Whether a colour is requested anywhere in the output.
func hasColorSGR(s string) bool {
	for _, match := range sgrPattern.FindAllStringSubmatch(s, -1) {
		if paramsSetColor(match[1]) {
			return true
		}
	}

	return false
}

// paramsSetColor reports whether an SGR parameter list requests a colour.
//
// The extended forms are recognised by their 38, 48 or 58 lead, and the original
// forms by a parameter that is a standard colour number on its own.
//
// Parameters:
//   - params: The numbers between the escape and the terminating m.
//
// Returns:
//   - Whether the sequence asks for colour.
func paramsSetColor(params string) bool {
	for param := range strings.SplitSeq(params, ";") {
		n, err := strconv.Atoi(param)
		if err != nil {
			continue
		}

		if n == 38 || n == 48 || n == 58 {
			return true
		}

		switch {
		case n >= 30 && n <= 37, n >= 40 && n <= 47,
			n >= 90 && n <= 97, n >= 100 && n <= 107:
			return true
		}
	}

	return false
}
