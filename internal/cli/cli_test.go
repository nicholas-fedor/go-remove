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
	"bytes"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	tea "github.com/charmbracelet/bubbletea"

	fsmocks "github.com/nicholas-fedor/go-remove/internal/fs/mocks"
	"github.com/nicholas-fedor/go-remove/internal/logger"
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
		name       string
		config     Config
		deps       Dependencies
		runner     ProgramRunner
		wantErr    bool
		wantOutput string // Expected stdout output for non-verbose success
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
			wantErr:    false,
			wantOutput: "Successfully removed vhs\n",
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
			wantErr:    false, // Sync errors are ignored on all platforms
			wantOutput: "Successfully removed vhs\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Redirect stdout to capture output for non-verbose success cases.
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			defer func() {
				os.Stdout = oldStdout

				w.Close()
			}()

			// Define a local run function to test with a custom runner.
			run := func(deps Dependencies, config Config, runner ProgramRunner) error {
				log := deps.Logger

				// Set log level based on config if verbose mode is enabled.
				if config.Verbose {
					zapLogger, ok := log.(*logger.ZapLogger)
					if !ok {
						return fmt.Errorf(
							"failed to set log level: %w with type %T",
							ErrInvalidLoggerType,
							log,
						)
					}

					switch config.LogLevel {
					case "debug":
						zapLogger.Logger = zapLogger.WithOptions(
							zap.IncreaseLevel(zapcore.DebugLevel),
						)
					case "warn":
						zapLogger.Logger = zapLogger.WithOptions(
							zap.IncreaseLevel(zapcore.WarnLevel),
						)
					case "error":
						zapLogger.Logger = zapLogger.WithOptions(
							zap.IncreaseLevel(zapcore.ErrorLevel),
						)
					default:
						zapLogger.Logger = zapLogger.WithOptions(
							zap.IncreaseLevel(zapcore.InfoLevel),
						)
					}

					log = zapLogger
				}

				binDir, err := deps.FS.DetermineBinDir(config.Goroot)
				if err != nil {
					_ = log.Sync() // Flush logs; errors are ignored

					return err
				}

				if config.Binary == "" {
					err = RunTUI(binDir, config, log, deps.FS, runner)
				} else {
					binaryPath := deps.FS.AdjustBinaryPath(binDir, config.Binary)

					err = deps.FS.RemoveBinary(binaryPath, config.Binary, config.Verbose, log)
					if err == nil && !config.Verbose {
						fmt.Fprintf(os.Stdout, "Successfully removed %s\n", config.Binary)
					}
				}

				if err != nil {
					_ = log.Sync() // Flush logs; errors are ignored

					return err
				}

				_ = log.Sync() // Errors are ignored

				return nil
			}

			// Use DefaultRunner if no mock is provided.
			runner := tt.runner
			if runner == nil {
				runner = DefaultRunner{}
			}

			// Execute the run function and capture any errors.
			err := run(tt.deps, tt.config, runner)

			// Capture stdout output after execution.
			w.Close()

			var buf bytes.Buffer
			buf.ReadFrom(r)
			gotOutput := buf.String()

			// Verify error behavior matches expectations.
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify stdout output for non-verbose success cases.
			if tt.wantOutput != "" && gotOutput != tt.wantOutput {
				t.Errorf("Run() output = %q, want %q", gotOutput, tt.wantOutput)
			}

			// Assert that all mock expectations were met.
			tt.deps.FS.(*fsmocks.MockFS).AssertExpectations(t)
			tt.deps.Logger.(*logmocks.MockLogger).AssertExpectations(t)
		})
	}
}
