/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

// Package cli_test provides black-box integration tests for the cli package.
//
// These tests verify the orchestration layer that coordinates between the filesystem,
// logger, and history manager components. All external dependencies are mocked
// using Mockery-generated mocks to ensure no external calls are made.
package cli_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/nicholas-fedor/go-remove/internal/cli"
	fsmocks "github.com/nicholas-fedor/go-remove/internal/fs/mocks"
	"github.com/nicholas-fedor/go-remove/internal/history"
	historymocks "github.com/nicholas-fedor/go-remove/internal/history/mocks"
	loggermocks "github.com/nicholas-fedor/go-remove/internal/logger/mocks"
)

// Test constants for consistent test data.
const (
	testBinaryName = "test-binary"
	testBinaryPath = "/usr/local/bin/test-binary"
	testBinDir     = "/usr/local/bin"
	testEntryID    = "1709321234:test-binary"
)

// CLIIntegrationTestSuite provides integration tests for the CLI package.
//
// This suite tests the orchestration between multiple components:
// - fs.FS for filesystem operations
// - logger.Logger for logging
// - history.Manager for history operations (optional)
//
// All dependencies are mocked to ensure isolated, deterministic tests.
type CLIIntegrationTestSuite struct {
	suite.Suite

	// Mocks for all dependencies
	fsMock      *fsmocks.MockFS
	loggerMock  *loggermocks.MockLogger
	historyMock *historymocks.MockManager

	// Test helpers
	nopLogger zerolog.Logger
}

// SetupTest initializes the test suite before each test.
//
// This creates fresh mocks for each test to ensure test isolation.
func (s *CLIIntegrationTestSuite) SetupTest() {
	// Create mocks
	s.fsMock = fsmocks.NewMockFS(s.T())
	s.loggerMock = loggermocks.NewMockLogger(s.T())
	s.historyMock = historymocks.NewMockManager(s.T())

	// Setup logger expectations
	s.setupLoggerExpectations()

	// Initialize nop logger for mock returns
	s.nopLogger = zerolog.New(io.Discard)
}

// setupLoggerExpectations configures the logger mock to accept any log calls.
//
// This allows the CLI to log as needed without requiring explicit expectations
// in every test case.
func (s *CLIIntegrationTestSuite) setupLoggerExpectations() {
	s.loggerMock.EXPECT().Debug().Return(s.nopLogger.Debug()).Maybe()
	s.loggerMock.EXPECT().Info().Return(s.nopLogger.Info()).Maybe()
	s.loggerMock.EXPECT().Warn().Return(s.nopLogger.Warn()).Maybe()
	s.loggerMock.EXPECT().Error().Return(s.nopLogger.Error()).Maybe()
}

// captureStdout redirects os.Stdout and returns a function that restores stdout
// and returns the captured output as a string.
func captureStdout(t *testing.T) func() string {
	t.Helper()

	oldStdout := os.Stdout

	r, w, err := os.Pipe()
	require.NoError(t, err, "Failed to create pipe")

	os.Stdout = w

	return func() string {
		// Always restore os.Stdout first
		os.Stdout = oldStdout

		// Close the write end
		if err := w.Close(); err != nil {
			t.Errorf("Failed to close pipe writer: %v", err)
		}

		// Read from pipe
		var buf bytes.Buffer

		if _, err := buf.ReadFrom(r); err != nil {
			t.Errorf("Failed to read from pipe: %v", err)
		}

		// Close the read end
		if err := r.Close(); err != nil {
			t.Errorf("Failed to close pipe reader: %v", err)
		}

		return buf.String()
	}
}

// TestCLIIntegrationTestSuite runs the integration test suite.
func TestCLIIntegrationTestSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(CLIIntegrationTestSuite))
}

// TestRunDirectRemovalSuccess verifies successful binary removal workflow.
//
// This test ensures that Run properly coordinates:
// 1. Determining the binary directory
// 2. Adjusting the binary path
// 3. Removing the binary
// 4. Printing success message to stdout
//
// The workflow should complete without errors and print the expected output.
func (s *CLIIntegrationTestSuite) TestRunDirectRemovalSuccess() {
	// Capture stdout
	getOutput := captureStdout(s.T())

	// Setup expectations
	s.fsMock.EXPECT().
		DetermineBinDir(false).
		Return(testBinDir, nil)

	s.fsMock.EXPECT().
		AdjustBinaryPath(testBinDir, testBinaryName).
		Return(testBinaryPath)

	s.fsMock.EXPECT().
		RemoveBinary(testBinaryPath, testBinaryName, false, s.loggerMock).
		Return(nil)

	s.loggerMock.EXPECT().Sync().Return(nil)

	// Execute
	deps := cli.Dependencies{
		FS:     s.fsMock,
		Logger: s.loggerMock,
	}

	config := cli.Config{
		Binary:  testBinaryName,
		Verbose: false,
		Goroot:  false,
	}

	err := cli.Run(deps, config)

	// Verify
	s.Require().NoError(err)

	output := getOutput()
	s.Equal("Successfully removed "+testBinaryName+"\n", output)
}

// TestRunDirectRemovalWithHistory verifies removal with history recording.
//
// This test ensures that when a history manager is provided, Run properly:
// 1. Records the deletion to history (which moves the binary to trash internally)
// 2. Does NOT call RemoveBinary since RecordDeletion already handles it
// 3. Prints success message after successful history recording.
func (s *CLIIntegrationTestSuite) TestRunDirectRemovalWithHistory() {
	// Capture stdout
	getOutput := captureStdout(s.T())

	// Create a history entry for the mock to return
	historyEntry := &history.HistoryEntry{
		ID:         testEntryID,
		BinaryName: testBinaryName,
		BinaryPath: testBinaryPath,
	}

	// Setup expectations
	s.fsMock.EXPECT().
		DetermineBinDir(false).
		Return(testBinDir, nil)

	s.fsMock.EXPECT().
		AdjustBinaryPath(testBinDir, testBinaryName).
		Return(testBinaryPath)

	// History should be recorded (which moves binary to trash internally)
	s.historyMock.EXPECT().
		RecordDeletion(mock.Anything, testBinaryPath).
		Return(historyEntry, nil)

	// RemoveBinary is NOT called when HistoryManager is available
	// because RecordDeletion already moves the binary to trash

	s.loggerMock.EXPECT().Sync().Return(nil)

	// Execute
	deps := cli.Dependencies{
		FS:             s.fsMock,
		Logger:         s.loggerMock,
		HistoryManager: s.historyMock,
	}

	config := cli.Config{
		Binary:  testBinaryName,
		Verbose: false,
		Goroot:  false,
	}

	err := cli.Run(deps, config)

	// Verify
	s.Require().NoError(err)

	output := getOutput()
	s.Equal("Successfully removed "+testBinaryName+"\n", output)
}

// TestRunDirectRemovalVerboseMode verifies verbose mode behavior.
//
// In verbose mode, Run should:
// 1. Pass verbose=true to RemoveBinary
// 2. Not print success message to stdout (relying on verbose logs instead).
func (s *CLIIntegrationTestSuite) TestRunDirectRemovalVerboseMode() {
	// Capture stdout
	getOutput := captureStdout(s.T())

	// Setup expectations
	s.fsMock.EXPECT().
		DetermineBinDir(false).
		Return(testBinDir, nil)

	s.fsMock.EXPECT().
		AdjustBinaryPath(testBinDir, testBinaryName).
		Return(testBinaryPath)

	s.fsMock.EXPECT().
		RemoveBinary(testBinaryPath, testBinaryName, true, s.loggerMock).
		Return(nil)

	s.loggerMock.EXPECT().Sync().Return(nil)

	// Execute
	deps := cli.Dependencies{
		FS:     s.fsMock,
		Logger: s.loggerMock,
	}

	config := cli.Config{
		Binary:  testBinaryName,
		Verbose: true,
		Goroot:  false,
	}

	err := cli.Run(deps, config)

	// Verify
	s.Require().NoError(err)

	// In verbose mode, no success message should be printed to stdout
	output := getOutput()
	s.Empty(output)
}

// TestRunDirectRemovalWithGoroot verifies GOROOT mode behavior.
//
// When Goroot is true, Run should use GOROOT/bin instead of GOPATH/GOBIN.
func (s *CLIIntegrationTestSuite) TestRunDirectRemovalWithGoroot() {
	gorootBinDir := "/usr/local/go/bin"
	gorootBinaryPath := gorootBinDir + "/" + testBinaryName

	// Capture stdout
	getOutput := captureStdout(s.T())

	// Setup expectations
	s.fsMock.EXPECT().
		DetermineBinDir(true).
		Return(gorootBinDir, nil)

	s.fsMock.EXPECT().
		AdjustBinaryPath(gorootBinDir, testBinaryName).
		Return(gorootBinaryPath)

	s.fsMock.EXPECT().
		RemoveBinary(gorootBinaryPath, testBinaryName, false, s.loggerMock).
		Return(nil)

	s.loggerMock.EXPECT().Sync().Return(nil)

	// Execute
	deps := cli.Dependencies{
		FS:     s.fsMock,
		Logger: s.loggerMock,
	}

	config := cli.Config{
		Binary:  testBinaryName,
		Verbose: false,
		Goroot:  true,
	}

	err := cli.Run(deps, config)

	// Verify
	s.Require().NoError(err)

	output := getOutput()
	s.Equal("Successfully removed "+testBinaryName+"\n", output)
}

// TestRunBinDirError verifies error handling when DetermineBinDir fails.
//
// When the binary directory cannot be determined, Run should:
// 1. Sync the logger
// 2. Return an error wrapping the original failure.
func (s *CLIIntegrationTestSuite) TestRunBinDirError() {
	binDirError := errors.New("cannot determine binary directory: GOBIN and GOPATH not set")

	// Setup expectations
	s.fsMock.EXPECT().
		DetermineBinDir(false).
		Return("", binDirError)

	s.loggerMock.EXPECT().Sync().Return(nil)

	// Execute
	deps := cli.Dependencies{
		FS:     s.fsMock,
		Logger: s.loggerMock,
	}

	config := cli.Config{
		Binary:  testBinaryName,
		Verbose: false,
		Goroot:  false,
	}

	err := cli.Run(deps, config)

	// Verify
	s.Require().Error(err)
	s.Require().ErrorContains(err, "failed to determine binary directory")
	s.Require().ErrorIs(err, binDirError)
}

// TestRunRemoveBinaryError verifies error handling when RemoveBinary fails.
//
// When binary removal fails, Run should:
// 1. Sync the logger
// 2. Return an error wrapping the original failure.
func (s *CLIIntegrationTestSuite) TestRunRemoveBinaryError() {
	removeError := errors.New("permission denied")

	// Setup expectations
	s.fsMock.EXPECT().
		DetermineBinDir(false).
		Return(testBinDir, nil)

	s.fsMock.EXPECT().
		AdjustBinaryPath(testBinDir, testBinaryName).
		Return(testBinaryPath)

	s.fsMock.EXPECT().
		RemoveBinary(testBinaryPath, testBinaryName, false, s.loggerMock).
		Return(removeError)

	s.loggerMock.EXPECT().Sync().Return(nil)

	// Execute
	deps := cli.Dependencies{
		FS:     s.fsMock,
		Logger: s.loggerMock,
	}

	config := cli.Config{
		Binary:  testBinaryName,
		Verbose: false,
		Goroot:  false,
	}

	err := cli.Run(deps, config)

	// Verify
	s.Require().Error(err)
	s.Require().ErrorContains(err, "failed to remove binary")
	s.Require().ErrorContains(err, testBinaryName)
	s.Require().ErrorIs(err, removeError)
}

// TestRunHistoryRecordError verifies error handling when history recording fails.
//
// When history recording fails, Run should:
// 1. Sync the logger
// 2. Return an error wrapping the original failure
// 3. Not attempt to remove the binary.
func (s *CLIIntegrationTestSuite) TestRunHistoryRecordError() {
	recordError := errors.New("database connection failed")

	// Setup expectations
	s.fsMock.EXPECT().
		DetermineBinDir(false).
		Return(testBinDir, nil)

	s.fsMock.EXPECT().
		AdjustBinaryPath(testBinDir, testBinaryName).
		Return(testBinaryPath)

	s.historyMock.EXPECT().
		RecordDeletion(mock.Anything, testBinaryPath).
		Return(nil, recordError)

	s.loggerMock.EXPECT().Sync().Return(nil)

	// Execute
	deps := cli.Dependencies{
		FS:             s.fsMock,
		Logger:         s.loggerMock,
		HistoryManager: s.historyMock,
	}

	config := cli.Config{
		Binary:  testBinaryName,
		Verbose: false,
		Goroot:  false,
	}

	err := cli.Run(deps, config)

	// Verify
	s.Require().Error(err)
	s.Require().ErrorContains(err, "failed to record deletion")
	s.Require().ErrorIs(err, recordError)
}

// Note: TUI tests using ProgramRunner are not included in integration tests
// because RunTUI requires complex Bubbletea program initialization that is
// better tested at the unit level in internal/cli. The integration tests
// focus on the Run() function which delegates to RunTUI internally.

// TestRunConfigPropagation verifies that all config fields are properly propagated.
//
// This test uses table-driven testing to verify config propagation through Run.
func (s *CLIIntegrationTestSuite) TestRunConfigPropagation() {
	tests := []struct {
		name          string
		config        cli.Config
		expectVerbose bool
		expectGoroot  bool
	}{
		{
			name: "default config",
			config: cli.Config{
				Binary:  testBinaryName,
				Verbose: false,
				Goroot:  false,
			},
			expectVerbose: false,
			expectGoroot:  false,
		},
		{
			name: "verbose enabled",
			config: cli.Config{
				Binary:  testBinaryName,
				Verbose: true,
				Goroot:  false,
			},
			expectVerbose: true,
			expectGoroot:  false,
		},
		{
			name: "goroot enabled",
			config: cli.Config{
				Binary:  testBinaryName,
				Verbose: false,
				Goroot:  true,
			},
			expectVerbose: false,
			expectGoroot:  true,
		},
		{
			name: "both verbose and goroot enabled",
			config: cli.Config{
				Binary:  testBinaryName,
				Verbose: true,
				Goroot:  true,
			},
			expectVerbose: true,
			expectGoroot:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Capture stdout
			getOutput := captureStdout(s.T())

			// Create fresh mocks for each subtest to avoid accumulated expectations
			fsMock := fsmocks.NewMockFS(s.T())
			loggerMock := loggermocks.NewMockLogger(s.T())

			// Setup logger expectations
			nopLogger := zerolog.New(io.Discard)
			loggerMock.EXPECT().Debug().Return(nopLogger.Debug()).Maybe()
			loggerMock.EXPECT().Info().Return(nopLogger.Info()).Maybe()
			loggerMock.EXPECT().Warn().Return(nopLogger.Warn()).Maybe()
			loggerMock.EXPECT().Error().Return(nopLogger.Error()).Maybe()

			// Setup expectations based on config
			fsMock.EXPECT().
				DetermineBinDir(tt.expectGoroot).
				Return(testBinDir, nil)

			fsMock.EXPECT().
				AdjustBinaryPath(testBinDir, testBinaryName).
				Return(testBinaryPath)

			fsMock.EXPECT().
				RemoveBinary(testBinaryPath, testBinaryName, tt.expectVerbose, loggerMock).
				Return(nil)

			loggerMock.EXPECT().Sync().Return(nil)

			// Execute
			deps := cli.Dependencies{
				FS:     fsMock,
				Logger: loggerMock,
			}

			err := cli.Run(deps, tt.config)

			// Verify
			s.Require().NoError(err)

			if !tt.expectVerbose {
				output := getOutput()
				s.Equal("Successfully removed "+testBinaryName+"\n", output)
			} else {
				output := getOutput()
				s.Empty(output)
			}
		})
	}
}

// TestRunDependenciesIntegration verifies proper interaction between all dependencies.
//
// This test ensures that FS, Logger, and HistoryManager are all properly coordinated.
// When HistoryManager is available, RecordDeletion handles the binary removal internally
// and RemoveBinary is not called.
func (s *CLIIntegrationTestSuite) TestRunDependenciesIntegration() {
	// Capture stdout
	getOutput := captureStdout(s.T())

	historyEntry := &history.HistoryEntry{
		ID:         testEntryID,
		BinaryName: testBinaryName,
		BinaryPath: testBinaryPath,
	}

	// Setup mocks
	s.fsMock.EXPECT().
		DetermineBinDir(false).
		Return(testBinDir, nil)

	s.fsMock.EXPECT().
		AdjustBinaryPath(testBinDir, testBinaryName).
		Return(testBinaryPath)

	// History records the deletion (which moves binary to trash internally)
	s.historyMock.EXPECT().
		RecordDeletion(mock.Anything, testBinaryPath).
		Return(historyEntry, nil)

	// RemoveBinary is NOT called when HistoryManager is available
	// because RecordDeletion already moves the binary to trash

	s.loggerMock.EXPECT().Sync().Return(nil)

	// Execute
	deps := cli.Dependencies{
		FS:             s.fsMock,
		Logger:         s.loggerMock,
		HistoryManager: s.historyMock,
	}

	config := cli.Config{
		Binary:  testBinaryName,
		Verbose: false,
		Goroot:  false,
	}

	err := cli.Run(deps, config)

	// Verify
	s.Require().NoError(err)

	output := getOutput()
	s.Equal("Successfully removed "+testBinaryName+"\n", output)
}

// TestRunDirectRemovalWithoutHistoryManager verifies direct removal works without history.
//
// Direct removal should function normally even when no history manager is provided.
func (s *CLIIntegrationTestSuite) TestRunDirectRemovalWithoutHistoryManager() {
	// Capture stdout
	getOutput := captureStdout(s.T())

	// Setup expectations
	s.fsMock.EXPECT().
		DetermineBinDir(false).
		Return(testBinDir, nil)

	s.fsMock.EXPECT().
		AdjustBinaryPath(testBinDir, testBinaryName).
		Return(testBinaryPath)

	s.fsMock.EXPECT().
		RemoveBinary(testBinaryPath, testBinaryName, false, s.loggerMock).
		Return(nil)

	s.loggerMock.EXPECT().Sync().Return(nil)

	// Execute
	deps := cli.Dependencies{
		FS:             s.fsMock,
		Logger:         s.loggerMock,
		HistoryManager: nil, // No history manager
	}

	config := cli.Config{
		Binary:  testBinaryName,
		Verbose: false,
		Goroot:  false,
	}

	err := cli.Run(deps, config)

	// Verify
	s.Require().NoError(err)

	output := getOutput()
	s.Equal("Successfully removed "+testBinaryName+"\n", output)
}

// TestRunBinaryNotFoundError verifies error when binary path adjustment fails.
//
// This tests error propagation from the FS layer.
func (s *CLIIntegrationTestSuite) TestRunBinaryNotFoundError() {
	// Setup expectations
	s.fsMock.EXPECT().
		DetermineBinDir(false).
		Return(testBinDir, nil)

	s.fsMock.EXPECT().
		AdjustBinaryPath(testBinDir, testBinaryName).
		Return(testBinaryPath)

	removeError := errors.New("binary not found at path")

	s.fsMock.EXPECT().
		RemoveBinary(testBinaryPath, testBinaryName, false, s.loggerMock).
		Return(removeError)

	s.loggerMock.EXPECT().Sync().Return(nil)

	// Execute
	deps := cli.Dependencies{
		FS:     s.fsMock,
		Logger: s.loggerMock,
	}

	config := cli.Config{
		Binary:  testBinaryName,
		Verbose: false,
		Goroot:  false,
	}

	err := cli.Run(deps, config)

	// Verify
	s.Require().Error(err)
	s.Require().ErrorContains(err, "failed to remove binary")
	s.Require().ErrorIs(err, removeError)
}

// Additional standalone tests for edge cases.

// TestRunWithLogLevelConfig verifies LogLevel field in Config.
//
// While LogLevel is stored in Config, the actual level setting is handled
// by the caller (cmd package). This test verifies the field is accessible.
func TestRunWithLogLevelConfig(t *testing.T) {
	t.Parallel()

	config := cli.Config{
		Binary:   testBinaryName,
		LogLevel: "debug",
	}

	assert.Equal(t, "debug", config.LogLevel)
}

// TestRunWithHelpConfig verifies Help field in Config.
//
// The Help field is managed by Cobra but should be accessible.
func TestRunWithHelpConfig(t *testing.T) {
	t.Parallel()

	config := cli.Config{
		Binary: testBinaryName,
		Help:   true,
	}

	assert.True(t, config.Help)
}

// TestConfigStruct verifies all Config fields can be set.
func TestConfigStruct(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config cli.Config
	}{
		{
			name: "minimal config",
			config: cli.Config{
				Binary: testBinaryName,
			},
		},
		{
			name: "full config",
			config: cli.Config{
				Binary:      testBinaryName,
				Verbose:     true,
				Goroot:      true,
				Help:        true,
				LogLevel:    "debug",
				RestoreMode: true,
			},
		},
		{
			name: "TUI config",
			config: cli.Config{
				Binary:      "",
				Verbose:     true,
				RestoreMode: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Verify config can be created without issues
			assert.NotNil(t, tt.config)
		})
	}
}

// TestDependenciesStruct verifies Dependencies can be created with various combinations.
func TestDependenciesStruct(t *testing.T) {
	t.Parallel()

	mockFS := &fsmocks.MockFS{}
	mockLogger := &loggermocks.MockLogger{}
	mockHistory := &historymocks.MockManager{}

	tests := []struct {
		name string
		deps cli.Dependencies
	}{
		{
			name: "minimal dependencies",
			deps: cli.Dependencies{
				FS:     mockFS,
				Logger: mockLogger,
			},
		},
		{
			name: "full dependencies",
			deps: cli.Dependencies{
				FS:             mockFS,
				Logger:         mockLogger,
				HistoryManager: mockHistory,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.NotNil(t, tt.deps.FS)
			assert.NotNil(t, tt.deps.Logger)
		})
	}
}

// TestErrorWrapping verifies that errors are properly wrapped.
//
// This test ensures that the error chain can be inspected using errors.Is.
func TestErrorWrapping(t *testing.T) {
	t.Parallel()

	originalErr := errors.New("original error")

	// Create mocks
	fsMock := fsmocks.NewMockFS(t)
	loggerMock := loggermocks.NewMockLogger(t)

	// Setup logger expectations
	nopLogger := zerolog.New(io.Discard)
	loggerMock.EXPECT().Debug().Return(nopLogger.Debug()).Maybe()
	loggerMock.EXPECT().Info().Return(nopLogger.Info()).Maybe()
	loggerMock.EXPECT().Warn().Return(nopLogger.Warn()).Maybe()
	loggerMock.EXPECT().Error().Return(nopLogger.Error()).Maybe()

	// Setup mock expectations - fsMock returns the original error
	fsMock.EXPECT().
		DetermineBinDir(false).
		Return("", originalErr)

	loggerMock.EXPECT().Sync().Return(nil)

	// Execute cli.Run with mocks that return the error
	deps := cli.Dependencies{
		FS:     fsMock,
		Logger: loggerMock,
	}

	config := cli.Config{
		Binary:  "test-binary",
		Verbose: false,
		Goroot:  false,
	}

	returnedErr := cli.Run(deps, config)

	// Verify error chain - the returned error should wrap the original
	require.Error(t, returnedErr)
	assert.ErrorIs(t, returnedErr, originalErr)
}

// TestMultipleBinaryRemovals verifies multiple sequential removals.
//
// This test simulates a workflow of removing multiple binaries.
func (s *CLIIntegrationTestSuite) TestMultipleBinaryRemovals() {
	binaries := []string{"binary1", "binary2"}

	for _, binary := range binaries {
		// Capture stdout for each iteration
		getOutput := captureStdout(s.T())

		binaryPath := testBinDir + "/" + binary

		// Setup expectations for each binary
		s.fsMock.EXPECT().
			DetermineBinDir(false).
			Return(testBinDir, nil)

		s.fsMock.EXPECT().
			AdjustBinaryPath(testBinDir, binary).
			Return(binaryPath)

		s.fsMock.EXPECT().
			RemoveBinary(binaryPath, binary, false, s.loggerMock).
			Return(nil)

		s.loggerMock.EXPECT().Sync().Return(nil)

		// Execute
		deps := cli.Dependencies{
			FS:     s.fsMock,
			Logger: s.loggerMock,
		}

		config := cli.Config{
			Binary:  binary,
			Verbose: false,
			Goroot:  false,
		}

		err := cli.Run(deps, config)

		// Verify
		s.Require().NoError(err)

		output := getOutput()
		s.Equal("Successfully removed "+binary+"\n", output)
	}
}
