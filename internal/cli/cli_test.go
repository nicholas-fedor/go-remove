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
	"testing"

	"github.com/stretchr/testify/mock"

	tea "github.com/charmbracelet/bubbletea"

	fsmocks "github.com/nicholas-fedor/go-remove/internal/fs/mocks"
	logmocks "github.com/nicholas-fedor/go-remove/internal/logger/mocks"
)

// cliMockRunner mocks the ProgramRunner interface for CLI tests.
type cliMockRunner struct {
	runProgram func(m tea.Model, opts ...tea.ProgramOption) (*tea.Program, error)
}

// RunProgram executes the mock runner’s program function.
func (m cliMockRunner) RunProgram(
	model tea.Model,
	opts ...tea.ProgramOption,
) (*tea.Program, error) {
	return m.runProgram(model, opts...)
}

// mockNoOpRunner provides a no-op runner for TUI tests.
func mockNoOpRunner(tea.Model, ...tea.ProgramOption) (*tea.Program, error) {
	return nil, nil //nolint:nilnil // Mock no-op runner
}

// TestRun verifies the Run function’s behavior under various conditions.
func TestRun(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		deps    Dependencies
		runner  ProgramRunner
		wantErr bool
	}{
		{
			name:   "direct removal success",
			config: Config{Binary: "vhs", Verbose: false, Goroot: false},
			deps: Dependencies{
				FS: func() *fsmocks.MockFS {
					m := fsmocks.NewMockFS(t)
					m.On("DetermineBinDir", false).Return("/bin", nil)
					m.On("AdjustBinaryPath", "/bin", "vhs").Return("/bin/vhs")
					m.On("RemoveBinary", "/bin/vhs", "vhs", false, mock.Anything).Return(nil)

					return m
				}(),
				Logger: func() *logmocks.MockLogger {
					m := logmocks.NewMockLogger(t)
					m.On("Sync").Return(nil)

					return m
				}(),
			},
			wantErr: false,
		},
		{
			name:   "direct removal failure",
			config: Config{Binary: "vhs", Verbose: false, Goroot: false},
			deps: Dependencies{
				FS: func() *fsmocks.MockFS {
					m := fsmocks.NewMockFS(t)
					m.On("DetermineBinDir", false).Return("/bin", nil)
					m.On("AdjustBinaryPath", "/bin", "vhs").Return("/bin/vhs")
					m.On("RemoveBinary", "/bin/vhs", "vhs", false, mock.Anything).
						Return(errors.New("remove failed"))

					return m
				}(),
				Logger: func() *logmocks.MockLogger {
					m := logmocks.NewMockLogger(t)
					m.On("Sync").Return(nil)

					return m
				}(),
			},
			wantErr: true,
		},
		{
			name:   "tui mode success",
			config: Config{Binary: "", Verbose: false, Goroot: false},
			deps: Dependencies{
				FS: func() *fsmocks.MockFS {
					m := fsmocks.NewMockFS(t)
					m.On("DetermineBinDir", false).Return("/bin", nil)
					m.On("ListBinaries", "/bin").Return([]string{"vhs"})

					return m
				}(),
				Logger: func() *logmocks.MockLogger {
					m := logmocks.NewMockLogger(t)
					m.On("Sync").Return(nil)

					return m
				}(),
			},
			runner:  &cliMockRunner{runProgram: mockNoOpRunner},
			wantErr: false,
		},
		{
			name:   "tui mode no binaries",
			config: Config{Binary: "", Verbose: false, Goroot: false},
			deps: Dependencies{
				FS: func() *fsmocks.MockFS {
					m := fsmocks.NewMockFS(t)
					m.On("DetermineBinDir", false).Return("/bin", nil)
					m.On("ListBinaries", "/bin").Return([]string{})

					return m
				}(),
				Logger: func() *logmocks.MockLogger {
					m := logmocks.NewMockLogger(t)
					m.On("Sync").Return(nil)

					return m
				}(),
			},
			runner:  &cliMockRunner{runProgram: mockNoOpRunner},
			wantErr: true,
		},
		{
			name:   "bin dir error",
			config: Config{Binary: "vhs", Verbose: false, Goroot: false},
			deps: Dependencies{
				FS: func() *fsmocks.MockFS {
					m := fsmocks.NewMockFS(t)
					m.On("DetermineBinDir", false).Return("", errors.New("bin dir failed"))

					return m
				}(),
				Logger: func() *logmocks.MockLogger {
					m := logmocks.NewMockLogger(t)
					m.On("Sync").Return(nil)

					return m
				}(),
			},
			wantErr: true,
		},
		{
			name:   "logger sync error",
			config: Config{Binary: "vhs", Verbose: false, Goroot: false},
			deps: Dependencies{
				FS: func() *fsmocks.MockFS {
					m := fsmocks.NewMockFS(t)
					m.On("DetermineBinDir", false).Return("/bin", nil)
					m.On("AdjustBinaryPath", "/bin", "vhs").Return("/bin/vhs")
					m.On("RemoveBinary", "/bin/vhs", "vhs", false, mock.Anything).Return(nil)

					return m
				}(),
				Logger: func() *logmocks.MockLogger {
					m := logmocks.NewMockLogger(t)
					m.On("Sync").Return(errors.New("sync failed"))

					return m
				}(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Define a local run function to test with a custom runner.
			run := func(deps Dependencies, config Config, runner ProgramRunner) error {
				log := deps.Logger

				binDir, err := deps.FS.DetermineBinDir(config.Goroot)
				if err != nil {
					_ = log.Sync()

					return err
				}

				if config.Binary == "" {
					err = RunTUI(binDir, config, log, deps.FS, runner)
				} else {
					binaryPath := deps.FS.AdjustBinaryPath(binDir, config.Binary)
					err = deps.FS.RemoveBinary(binaryPath, config.Binary, config.Verbose, log)
				}

				if err != nil {
					_ = log.Sync()

					return err
				}

				return log.Sync()
			}

			// Use DefaultRunner if no mock is provided.
			runner := tt.runner
			if runner == nil {
				runner = DefaultRunner{}
			}

			err := run(tt.deps, tt.config, runner)
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}

			tt.deps.FS.(*fsmocks.MockFS).AssertExpectations(t)
			tt.deps.Logger.(*logmocks.MockLogger).AssertExpectations(t)
		})
	}
}
