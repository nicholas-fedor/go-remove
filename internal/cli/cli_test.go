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
	"io"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"

	tea "charm.land/bubbletea/v2"

	mockRunner "github.com/nicholas-fedor/go-remove/internal/cli/mocks"
	mockFS "github.com/nicholas-fedor/go-remove/internal/fs/mocks"
	"github.com/nicholas-fedor/go-remove/internal/logger"
	mockLogger "github.com/nicholas-fedor/go-remove/internal/logger/mocks"
)

// mockNoOpRunner provides a no-op runner for TUI tests.
func mockNoOpRunner(tea.Model, ...tea.ProgramOption) (*tea.Program, error) {
	return nil, nil //nolint:nilnil // Mock no-op runner returns nil values for test simplicity
}

// newMockLoggerWithDefaults creates a MockLogger with default expectations for all methods.
// This helper reduces boilerplate when setting up logger mocks for tests that don't need
// to verify specific logger interactions.
func newMockLoggerWithDefaults(t *testing.T) *mockLogger.MockLogger {
	t.Helper()

	m := mockLogger.NewMockLogger(t)

	// Create a nop logger and get event pointers to use as return values.
	nopLog := zerolog.New(io.Discard)

	// Use Maybe() for optional methods that may not be called during tests.
	m.On("Debug").Return(nopLog.Debug()).Maybe()
	m.On("Info").Return(nopLog.Info()).Maybe()
	m.On("Warn").Return(nopLog.Warn()).Maybe()
	m.On("Error").Return(nopLog.Error()).Maybe()
	m.On("Sync").Return(nil)
	m.On("Level", mock.Anything).Return().Maybe()
	m.On("SetCaptureFunc", mock.Anything).Return().Maybe()

	return m
}

// captureStdout redirects os.Stdout and returns a function that restores stdout
// and returns the captured output as a string.
func captureStdout(t *testing.T) func() string {
	t.Helper()

	oldStdout := os.Stdout

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}

	os.Stdout = w

	return func() string {
		// Always restore os.Stdout first
		os.Stdout = oldStdout

		// Close the write end and check for errors
		if err := w.Close(); err != nil {
			t.Errorf("Failed to close pipe writer: %v", err)
		}

		// Read from pipe and check for errors
		var buf bytes.Buffer

		if _, err := buf.ReadFrom(r); err != nil {
			t.Errorf("Failed to read from pipe: %v", err)
		}

		// Close the read end to prevent FD leak
		if err := r.Close(); err != nil {
			t.Errorf("Failed to close pipe reader: %v", err)
		}

		return buf.String()
	}
}

// makeDeps creates Dependencies from the provided setup functions.
type makeDepsConfig struct {
	setupFS     func(t *testing.T) *mockFS.MockFS
	setupLog    func(t *testing.T) *mockLogger.MockLogger
	setupRunner func(t *testing.T) *mockRunner.MockProgramRunner
}

func makeDeps(
	t *testing.T,
	cfg makeDepsConfig,
) (Dependencies, *mockFS.MockFS, *mockLogger.MockLogger, ProgramRunner) {
	t.Helper()

	mockFSInstance := cfg.setupFS(t)
	mockLog := cfg.setupLog(t)

	deps := Dependencies{
		FS:     mockFSInstance,
		Logger: mockLog,
	}

	var runner ProgramRunner = DefaultRunner{}
	if cfg.setupRunner != nil {
		runner = cfg.setupRunner(t)
	}

	return deps, mockFSInstance, mockLog, runner
}

// executeRun wraps the local run logic and returns the error.
// It mirrors the behavior of the Run function but accepts a custom runner for testing.
func executeRun(deps Dependencies, config Config, runner ProgramRunner) error {
	log := deps.Logger

	// Set log level based on config if verbose mode is enabled.
	if config.Verbose {
		level := logger.ParseLevel(config.LogLevel)
		log.Level(level)
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

// TestRun verifies the Run function's behavior under various conditions.
//
//nolint:thelper // Subtest functions in table-driven tests require *testing.T parameter for mock constructors
func TestRun(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		setupFS     func(t *testing.T) *mockFS.MockFS
		setupLog    func(t *testing.T) *mockLogger.MockLogger
		setupRunner func(t *testing.T) *mockRunner.MockProgramRunner
		wantErr     bool
		wantOutput  string // Expected stdout output for non-verbose success
	}{
		{
			name:   "direct removal success",
			config: Config{Binary: "vhs", Verbose: false, Goroot: false},
			setupFS: func(t *testing.T) *mockFS.MockFS {
				m := mockFS.NewMockFS(t)
				m.On("DetermineBinDir", false).Return("/bin", nil)
				m.On("AdjustBinaryPath", "/bin", "vhs").Return("/bin/vhs")
				m.On("RemoveBinary", "/bin/vhs", "vhs", false, mock.Anything).Return(nil)

				return m
			},
			setupLog:   newMockLoggerWithDefaults,
			wantErr:    false,
			wantOutput: "Successfully removed vhs\n",
		},
		{
			name:   "direct removal failure",
			config: Config{Binary: "vhs", Verbose: false, Goroot: false},
			setupFS: func(t *testing.T) *mockFS.MockFS {
				m := mockFS.NewMockFS(t)
				m.On("DetermineBinDir", false).Return("/bin", nil)
				m.On("AdjustBinaryPath", "/bin", "vhs").Return("/bin/vhs")
				m.On("RemoveBinary", "/bin/vhs", "vhs", false, mock.Anything).
					Return(errors.New("remove failed"))

				return m
			},
			setupLog: newMockLoggerWithDefaults,
			wantErr:  true,
		},
		{
			name:   "tui mode success",
			config: Config{Binary: "", Verbose: false, Goroot: false},
			setupFS: func(t *testing.T) *mockFS.MockFS {
				m := mockFS.NewMockFS(t)
				m.On("DetermineBinDir", false).Return("/bin", nil)
				m.On("ListBinaries", "/bin").Return([]string{"vhs"})

				return m
			},
			setupLog: newMockLoggerWithDefaults,
			setupRunner: func(t *testing.T) *mockRunner.MockProgramRunner {
				m := mockRunner.NewMockProgramRunner(t)
				m.On("RunProgram", mock.Anything, mock.Anything).Return(nil, nil)

				return m
			},
			wantErr: false,
		},
		{
			name:   "tui mode no binaries",
			config: Config{Binary: "", Verbose: false, Goroot: false},
			setupFS: func(t *testing.T) *mockFS.MockFS {
				m := mockFS.NewMockFS(t)
				m.On("DetermineBinDir", false).Return("/bin", nil)
				m.On("ListBinaries", "/bin").Return([]string{})

				return m
			},
			setupLog: newMockLoggerWithDefaults,
			setupRunner: func(t *testing.T) *mockRunner.MockProgramRunner {
				m := mockRunner.NewMockProgramRunner(t)
				// RunProgram may not be called if RunTUI returns an error early
				m.On("RunProgram", mock.Anything, mock.Anything).Return(nil, nil).Maybe()

				return m
			},
			wantErr: true,
		},
		{
			name:   "bin dir error",
			config: Config{Binary: "vhs", Verbose: false, Goroot: false},
			setupFS: func(t *testing.T) *mockFS.MockFS {
				m := mockFS.NewMockFS(t)
				m.On("DetermineBinDir", false).Return("", errors.New("bin dir failed"))

				return m
			},
			setupLog: newMockLoggerWithDefaults,
			wantErr:  true,
		},
		{
			name:   "logger sync error",
			config: Config{Binary: "vhs", Verbose: false, Goroot: false},
			setupFS: func(t *testing.T) *mockFS.MockFS {
				m := mockFS.NewMockFS(t)
				m.On("DetermineBinDir", false).Return("/bin", nil)
				m.On("AdjustBinaryPath", "/bin", "vhs").Return("/bin/vhs")
				m.On("RemoveBinary", "/bin/vhs", "vhs", false, mock.Anything).Return(nil)

				return m
			},
			setupLog: func(t *testing.T) *mockLogger.MockLogger {
				m := mockLogger.NewMockLogger(t)
				nopLog := zerolog.New(io.Discard)
				m.On("Debug").Return(nopLog.Debug()).Maybe()
				m.On("Info").Return(nopLog.Info()).Maybe()
				m.On("Warn").Return(nopLog.Warn()).Maybe()
				m.On("Error").Return(nopLog.Error()).Maybe()
				m.On("Sync").Return(errors.New("sync failed"))
				m.On("Level", mock.Anything).Return().Maybe()
				m.On("SetCaptureFunc", mock.Anything).Return().Maybe()

				return m
			},
			wantErr:    false, // Sync errors are ignored on all platforms
			wantOutput: "Successfully removed vhs\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout for output verification.
			getOutput := captureStdout(t)

			// Set up dependencies and runner.
			deps, mockFSInstance, mockLog, runner := makeDeps(t, makeDepsConfig{
				setupFS:     tt.setupFS,
				setupLog:    tt.setupLog,
				setupRunner: tt.setupRunner,
			})

			// Execute the run function and capture any errors.
			err := executeRun(deps, tt.config, runner)

			// Capture stdout output after execution.
			gotOutput := getOutput()

			// Verify error behavior matches expectations.
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify stdout output for non-verbose success cases.
			if tt.wantOutput != "" && gotOutput != tt.wantOutput {
				t.Errorf("Run() output = %q, want %q", gotOutput, tt.wantOutput)
			}

			// Assert that all mock expectations were met.
			mockFSInstance.AssertExpectations(t)
			mockLog.AssertExpectations(t)

			// If using a mock runner, assert its expectations as well.
			if mr, ok := runner.(*mockRunner.MockProgramRunner); ok {
				mr.AssertExpectations(t)
			}
		})
	}
}

// TestRun_WithLoggerSync verifies that Sync is called appropriately.
//
//nolint:thelper // Subtest functions in table-driven tests require *testing.T parameter for mock constructors
func TestRun_WithLoggerSync(t *testing.T) {
	tests := []struct {
		name       string
		config     Config
		setupFS    func(t *testing.T) *mockFS.MockFS
		setupLog   func(t *testing.T) *mockLogger.MockLogger
		wantErr    bool
		wantOutput string
	}{
		{
			name:   "sync called on success",
			config: Config{Binary: "tool", Verbose: false, Goroot: false},
			setupFS: func(t *testing.T) *mockFS.MockFS {
				m := mockFS.NewMockFS(t)
				m.On("DetermineBinDir", false).Return("/bin", nil)
				m.On("AdjustBinaryPath", "/bin", "tool").Return("/bin/tool")
				m.On("RemoveBinary", "/bin/tool", "tool", false, mock.Anything).Return(nil)

				return m
			},
			setupLog: func(t *testing.T) *mockLogger.MockLogger {
				m := mockLogger.NewMockLogger(t)
				nopLog := zerolog.New(io.Discard)
				m.On("Debug").Return(nopLog.Debug()).Maybe()
				m.On("Info").Return(nopLog.Info()).Maybe()
				m.On("Warn").Return(nopLog.Warn()).Maybe()
				m.On("Error").Return(nopLog.Error()).Maybe()
				m.On("Sync").Return(nil)

				return m
			},
			wantErr:    false,
			wantOutput: "Successfully removed tool\n",
		},
		{
			name:   "sync called on error",
			config: Config{Binary: "tool", Verbose: false, Goroot: false},
			setupFS: func(t *testing.T) *mockFS.MockFS {
				m := mockFS.NewMockFS(t)
				m.On("DetermineBinDir", false).Return("", errors.New("bin dir error"))

				return m
			},
			setupLog: func(t *testing.T) *mockLogger.MockLogger {
				m := mockLogger.NewMockLogger(t)
				nopLog := zerolog.New(io.Discard)
				m.On("Debug").Return(nopLog.Debug()).Maybe()
				m.On("Info").Return(nopLog.Info()).Maybe()
				m.On("Warn").Return(nopLog.Warn()).Maybe()
				m.On("Error").Return(nopLog.Error()).Maybe()
				m.On("Sync").Return(nil)

				return m
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout for output verification.
			getOutput := captureStdout(t)

			// Set up dependencies.
			mockFSInstance := tt.setupFS(t)
			mockLog := tt.setupLog(t)

			deps := Dependencies{
				FS:     mockFSInstance,
				Logger: mockLog,
			}

			// Execute the Run function and capture any errors.
			err := Run(deps, tt.config)

			// Capture stdout output after execution.
			gotOutput := getOutput()

			// Verify error behavior matches expectations.
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify stdout output for success cases.
			if tt.wantOutput != "" && gotOutput != tt.wantOutput {
				t.Errorf("Run() output = %q, want %q", gotOutput, tt.wantOutput)
			}

			// Assert that Sync was called on the mock logger.
			mockLog.AssertCalled(t, "Sync")

			// Assert that all mock expectations were met.
			mockFSInstance.AssertExpectations(t)
			mockLog.AssertExpectations(t)
		})
	}
}

// TestRun_VerboseMode verifies verbose mode behavior.
func TestRun_VerboseMode(t *testing.T) {
	m := mockFS.NewMockFS(t)
	m.On("DetermineBinDir", false).Return("/bin", nil)
	m.On("AdjustBinaryPath", "/bin", "vhs").Return("/bin/vhs")
	m.On("RemoveBinary", "/bin/vhs", "vhs", true, mock.Anything).Return(nil)

	mockLog := mockLogger.NewMockLogger(t)
	nopLog := zerolog.New(io.Discard)
	mockLog.On("Debug").Return(nopLog.Debug()).Maybe()
	mockLog.On("Info").Return(nopLog.Info()).Maybe()
	mockLog.On("Warn").Return(nopLog.Warn()).Maybe()
	mockLog.On("Error").Return(nopLog.Error()).Maybe()
	mockLog.On("Level", mock.Anything).Return().Maybe()
	mockLog.On("Sync").Return(nil)

	deps := Dependencies{
		FS:     m,
		Logger: mockLog,
	}
	config := Config{Binary: "vhs", Verbose: true, Goroot: false}

	// Capture stdout to capture output.
	getOutput := captureStdout(t)

	// Execute the Run function and capture any errors.
	err := Run(deps, config)

	// Restore stdout and get captured output.
	gotOutput := getOutput()

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// In verbose mode, no success message should be printed to stdout.
	if gotOutput != "" {
		t.Errorf("Expected no stdout output in verbose mode, got: %q", gotOutput)
	}

	// Assert that all mock expectations were met.
	m.AssertExpectations(t)
	mockLog.AssertExpectations(t)
}
