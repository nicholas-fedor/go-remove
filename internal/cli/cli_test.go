/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package cli

import (
	"bytes"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	mockFS "github.com/nicholas-fedor/go-remove/internal/fs/mocks"
	"github.com/nicholas-fedor/go-remove/internal/logger"
	mockLogger "github.com/nicholas-fedor/go-remove/internal/logger/mocks"
	"github.com/nicholas-fedor/go-remove/internal/tui"
)

// testCase defines the structure for TestRun test cases.
type testCase struct {
	name       string
	config     Config
	setupFS    func(t *testing.T) *mockFS.MockFS
	setupLog   func() logger.Logger
	wantErr    bool
	wantErrIs  error       // Sentinel the error must match, when set
	isTerminal func() bool // Terminal check for the interactive branch, when set
	wantOutput string      // Expected stdout output for non-verbose success
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

// runTestCase executes a single test case with the provided configuration.
// It handles stdout capture, dependency setup, execution, and assertions.
func runTestCase(t *testing.T, tt *testCase) {
	t.Helper()

	// Capture stdout for output verification.
	getOutput := captureStdout(t)

	mockFSInstance := tt.setupFS(t)
	mockLog := tt.setupLog()

	deps := Dependencies{
		FS:         mockFSInstance,
		Logger:     mockLog,
		IsTerminal: tt.isTerminal,
	}

	// Execute the run function and capture any errors.
	err := Run(t.Context(), deps, tt.config)

	// Capture stdout output after execution.
	gotOutput := getOutput()

	if (err != nil) != tt.wantErr {
		t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
	}

	if tt.wantErrIs != nil {
		//nolint:testifylint // assert keeps going so the mock expectations below still report
		assert.ErrorIs(t, err, tt.wantErrIs, "Run() must surface the branch's own error")
	}

	if tt.wantOutput != "" && gotOutput != tt.wantOutput {
		t.Errorf("Run() output = %q, want %q", gotOutput, tt.wantOutput)
	}

	// Assert that all mock expectations were met.
	mockFSInstance.AssertExpectations(t)

	// Only a generated mock has expectations to check; the nop logger is a real
	// implementation and has nothing pending.
	if ml, ok := mockLog.(*mockLogger.MockLogger); ok {
		ml.AssertExpectations(t)
	}
}

// TestRun verifies the Run function's behavior under various conditions.
func TestRun(t *testing.T) {
	tests := []testCase{
		{
			name:   "direct removal success",
			config: Config{Binary: "vhs", Verbose: false, Goroot: false},
			setupFS: func(t *testing.T) *mockFS.MockFS { //nolint:thelper // Anonymous setup function, not a test helper
				m := mockFS.NewMockFS(t)
				m.On("DetermineBinDir", false).Return("/bin", nil)
				m.On("AdjustBinaryPath", "/bin", "vhs").Return("/bin/vhs")
				m.On("RemoveBinary", "/bin/vhs", "vhs", false, mock.Anything).Return(nil)

				return m
			},
			setupLog:   logger.NopLogger,
			wantErr:    false,
			wantOutput: "Successfully removed vhs\n",
		},
		{
			name:   "direct removal failure",
			config: Config{Binary: "vhs", Verbose: false, Goroot: false},
			setupFS: func(t *testing.T) *mockFS.MockFS { //nolint:thelper // Anonymous setup function, not a test helper
				m := mockFS.NewMockFS(t)
				m.On("DetermineBinDir", false).Return("/bin", nil)
				m.On("AdjustBinaryPath", "/bin", "vhs").Return("/bin/vhs")
				m.On("RemoveBinary", "/bin/vhs", "vhs", false, mock.Anything).
					Return(errors.New("remove failed"))

				return m
			},
			setupLog: logger.NopLogger,
			wantErr:  true,
		},
		{
			name:   "tui mode surfaces the terminal guard",
			config: Config{Binary: "", Verbose: false, Goroot: false},
			setupFS: func(t *testing.T) *mockFS.MockFS { //nolint:thelper // Anonymous setup function, not a test helper
				m := mockFS.NewMockFS(t)
				// The terminal guard runs before the binary listing, so
				// ListBinaries is never reached.
				m.On("DetermineBinDir", false).Return("/bin", nil)

				return m
			},
			setupLog:  logger.NopLogger,
			wantErr:   true,
			wantErrIs: tui.ErrNotATerminal,
			// Without this the guard falls back to the real terminal check,
			// which passes when the test binary inherits a shell's stdin.
			isTerminal: func() bool { return false },
		},
		{
			name:   "bin dir error",
			config: Config{Binary: "vhs", Verbose: false, Goroot: false},
			setupFS: func(t *testing.T) *mockFS.MockFS { //nolint:thelper // Anonymous setup function, not a test helper
				m := mockFS.NewMockFS(t)
				m.On("DetermineBinDir", false).Return("", errors.New("bin dir failed"))

				return m
			},
			setupLog: logger.NopLogger,
			wantErr:  true,
		},
	}

	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			runTestCase(t, &tests[i])
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
	mockLog.EXPECT().Debug(mock.Anything).Maybe()
	mockLog.EXPECT().Debug(mock.Anything, mock.Anything).Maybe()
	mockLog.EXPECT().Info(mock.Anything).Maybe()
	mockLog.EXPECT().Info(mock.Anything, mock.Anything).Maybe()
	mockLog.EXPECT().Warn(mock.Anything).Maybe()
	mockLog.EXPECT().Warn(mock.Anything, mock.Anything).Maybe()
	mockLog.EXPECT().Error(mock.Anything).Maybe()
	mockLog.EXPECT().Error(mock.Anything, mock.Anything).Maybe()
	mockLog.EXPECT().Level(mock.Anything).Return().Maybe()

	deps := Dependencies{
		FS:     m,
		Logger: mockLog,
	}
	config := Config{Binary: "vhs", Verbose: true, Goroot: false}

	// Capture stdout to capture output.
	getOutput := captureStdout(t)

	// Execute the Run function and capture any errors.
	err := Run(t.Context(), deps, config)

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
