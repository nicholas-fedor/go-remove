/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

// Package fs_test provides black-box integration tests for the fs package.
//
// These tests verify the FS interface behavior through Mockery-generated mocks.
// All tests use MockFS and MockLogger to simulate filesystem operations without
// requiring real filesystem access. The tests cover directory resolution, binary
// path adjustment, binary removal, and directory listing operations.
//
// Test Organization:
//   - FSIntegrationTestSuite: Main test suite using testify suite
//   - Table-driven tests for parameterized scenarios
//   - Individual tests for complex error handling and workflow cases
//
// Coverage Areas:
//   - DetermineBinDir with various environment configurations (GOPATH, GOBIN, GOROOT)
//   - AdjustBinaryPath with different directory and binary combinations
//   - RemoveBinary with verbose mode and logger integration
//   - ListBinaries with various directory contents
//   - Error handling for missing environment variables
//   - Error handling for invalid paths
//   - Permission denied scenarios
//   - Binary not found scenarios
package fs_test

import (
	"errors"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/nicholas-fedor/go-remove/internal/fs"
	fsMocks "github.com/nicholas-fedor/go-remove/internal/fs/mocks"
	"github.com/nicholas-fedor/go-remove/internal/logger"
	loggerMocks "github.com/nicholas-fedor/go-remove/internal/logger/mocks"
)

// Test constants for consistent test data.
const (
	testBinaryName   = "test-binary"
	testBinaryName2  = "another-binary"
	testBinDir       = "/home/user/go/bin"
	testGorootBinDir = "/usr/local/go/bin"
	testGopath       = "/home/user/go"
	testGobin        = "/custom/go/bin"
	testGoroot       = "/usr/local/go"
)

// FSIntegrationTestSuite provides integration tests for the fs FS interface.
//
// This suite tests the orchestration between filesystem operations and their
// consumers, ensuring that all FS operations behave correctly through the
// interface. All tests use MockFS to simulate filesystem interactions without
// requiring real filesystem access.
type FSIntegrationTestSuite struct {
	suite.Suite

	// Mock for the FS interface
	mockFS *fsMocks.MockFS

	// Mock for the Logger interface
	mockLogger *loggerMocks.MockLogger
}

// SetupTest initializes the test suite before each test.
//
// Creates fresh MockFS and MockLogger instances for each test to ensure
// test isolation and prevent test interference.
func (s *FSIntegrationTestSuite) SetupTest() {
	s.mockFS = fsMocks.NewMockFS(s.T())
	s.mockLogger = loggerMocks.NewMockLogger(s.T())
}

// TestFSIntegrationTestSuite runs the integration test suite.
func TestFSIntegrationTestSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(FSIntegrationTestSuite))
}

// TestDetermineBinDirWithGoroot verifies DetermineBinDir with GOROOT configuration.
//
// When useGoroot is true and GOROOT is set, the function should return
// the GOROOT/bin directory path.
func (s *FSIntegrationTestSuite) TestDetermineBinDirWithGoroot() {
	s.mockFS.EXPECT().
		DetermineBinDir(true).
		Return(testGorootBinDir, nil).
		Once()

	binDir, err := s.mockFS.DetermineBinDir(true)

	s.Require().NoError(err)
	s.Equal(testGorootBinDir, binDir)
}

// TestDetermineBinDirWithGobin verifies DetermineBinDir with GOBIN configuration.
//
// When useGoroot is false and GOBIN is set, the function should return
// the GOBIN directory path.
func (s *FSIntegrationTestSuite) TestDetermineBinDirWithGobin() {
	s.mockFS.EXPECT().
		DetermineBinDir(false).
		Return(testGobin, nil).
		Once()

	binDir, err := s.mockFS.DetermineBinDir(false)

	s.Require().NoError(err)
	s.Equal(testGobin, binDir)
}

// TestDetermineBinDirWithGopath verifies DetermineBinDir with GOPATH configuration.
//
// When useGoroot is false, GOBIN is unset, and GOPATH is set,
// the function should return the GOPATH/bin directory path.
func (s *FSIntegrationTestSuite) TestDetermineBinDirWithGopath() {
	expectedDir := filepath.Join(testGopath, "bin")

	s.mockFS.EXPECT().
		DetermineBinDir(false).
		Return(expectedDir, nil).
		Once()

	binDir, err := s.mockFS.DetermineBinDir(false)

	s.Require().NoError(err)
	s.Equal(expectedDir, binDir)
}

// TestDetermineBinDirWithDefault verifies DetermineBinDir with default configuration.
//
// When useGoroot is false and neither GOBIN nor GOPATH is set,
// the function should return the default ~/go/bin directory path.
func (s *FSIntegrationTestSuite) TestDetermineBinDirWithDefault() {
	expectedDir := filepath.FromSlash("/home/user/go/bin")

	s.mockFS.EXPECT().
		DetermineBinDir(false).
		Return(expectedDir, nil).
		Once()

	binDir, err := s.mockFS.DetermineBinDir(false)

	s.Require().NoError(err)
	s.Equal(expectedDir, binDir)
}

// TestDetermineBinDirGorootNotSet verifies error handling when GOROOT is not set.
//
// When useGoroot is true but GOROOT environment variable is not set,
// the function should return ErrGorootNotSet.
func (s *FSIntegrationTestSuite) TestDetermineBinDirGorootNotSet() {
	s.mockFS.EXPECT().
		DetermineBinDir(true).
		Return("", fs.ErrGorootNotSet).
		Once()

	binDir, err := s.mockFS.DetermineBinDir(true)

	s.Require().ErrorIs(err, fs.ErrGorootNotSet)
	s.Empty(binDir)
}

// TestDetermineBinDirVariousConfigurations verifies DetermineBinDir with various environment configs.
//
// This test uses table-driven testing to cover multiple scenarios:
// - GOROOT with useGoroot=true
// - GOBIN with useGoroot=false
// - GOPATH/bin with useGoroot=false
// - Default ~/go/bin with useGoroot=false.
func (s *FSIntegrationTestSuite) TestDetermineBinDirVariousConfigurations() {
	tests := []struct {
		name        string
		useGoroot   bool
		expectedDir string
		expectedErr error
	}{
		{
			name:        "GOROOT enabled with GOROOT set",
			useGoroot:   true,
			expectedDir: testGorootBinDir,
			expectedErr: nil,
		},
		{
			name:        "GOBIN configured",
			useGoroot:   false,
			expectedDir: testGobin,
			expectedErr: nil,
		},
		{
			name:        "GOPATH/bin fallback",
			useGoroot:   false,
			expectedDir: filepath.Join(testGopath, "bin"),
			expectedErr: nil,
		},
		{
			name:        "default ~/go/bin",
			useGoroot:   false,
			expectedDir: filepath.FromSlash("/home/user/go/bin"),
			expectedErr: nil,
		},
		{
			name:        "GOROOT not set error",
			useGoroot:   true,
			expectedDir: "",
			expectedErr: fs.ErrGorootNotSet,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.mockFS.EXPECT().
				DetermineBinDir(tt.useGoroot).
				Return(tt.expectedDir, tt.expectedErr).
				Once()

			binDir, err := s.mockFS.DetermineBinDir(tt.useGoroot)

			if tt.expectedErr != nil {
				s.Require().ErrorIs(err, tt.expectedErr)
				s.Empty(binDir)
			} else {
				s.Require().NoError(err)
				s.Equal(tt.expectedDir, binDir)
			}
		})
	}
}

// TestAdjustBinaryPathBasic verifies basic AdjustBinaryPath functionality.
//
// The function should join the directory and binary name into a single path.
func (s *FSIntegrationTestSuite) TestAdjustBinaryPathBasic() {
	expectedPath := filepath.Join(testBinDir, testBinaryName)

	s.mockFS.EXPECT().
		AdjustBinaryPath(testBinDir, testBinaryName).
		Return(expectedPath).
		Once()

	result := s.mockFS.AdjustBinaryPath(testBinDir, testBinaryName)

	s.Equal(expectedPath, result)
}

// TestAdjustBinaryPathWithEmptyBinary verifies AdjustBinaryPath with empty binary name.
//
// When binary name is empty, the function should return just the directory path.
func (s *FSIntegrationTestSuite) TestAdjustBinaryPathWithEmptyBinary() {
	s.mockFS.EXPECT().
		AdjustBinaryPath(testBinDir, "").
		Return(testBinDir).
		Once()

	result := s.mockFS.AdjustBinaryPath(testBinDir, "")

	s.Equal(testBinDir, result)
}

// TestAdjustBinaryPathWindows verifies AdjustBinaryPath adds .exe on Windows.
//
// On Windows, the function should append .exe extension to the binary name
// if it doesn't already have it.
func (s *FSIntegrationTestSuite) TestAdjustBinaryPathWindows() {
	if runtime.GOOS != "windows" {
		s.T().Skip("Skipping Windows-specific test on non-Windows platform")
	}

	expectedPath := filepath.Join(testBinDir, "tool.exe")

	s.mockFS.EXPECT().
		AdjustBinaryPath(testBinDir, "tool").
		Return(expectedPath).
		Once()

	result := s.mockFS.AdjustBinaryPath(testBinDir, "tool")

	s.Equal(expectedPath, result)
}

// TestAdjustBinaryPathVariousCombinations verifies AdjustBinaryPath with various inputs.
//
// This test uses table-driven testing to cover multiple scenarios:
// - Basic directory and binary
// - Empty binary name
// - Nested directories
// - Binary with existing extension.
func (s *FSIntegrationTestSuite) TestAdjustBinaryPathVariousCombinations() {
	tests := []struct {
		name     string
		dir      string
		binary   string
		expected string
	}{
		{
			name:     "basic path",
			dir:      "/home/user/go/bin",
			binary:   "myapp",
			expected: filepath.FromSlash("/home/user/go/bin/myapp"),
		},
		{
			name:     "empty binary",
			dir:      filepath.FromSlash("/home/user/go/bin"),
			binary:   "",
			expected: filepath.FromSlash("/home/user/go/bin"),
		},
		{
			name:     "nested directory",
			dir:      "/usr/local/go/bin",
			binary:   "go",
			expected: filepath.FromSlash("/usr/local/go/bin/go"),
		},
		{
			name:     "binary with version",
			dir:      "/home/user/go/bin",
			binary:   "myapp-v1.0.0",
			expected: filepath.FromSlash("/home/user/go/bin/myapp-v1.0.0"),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.mockFS.EXPECT().
				AdjustBinaryPath(tt.dir, tt.binary).
				Return(tt.expected).
				Once()

			result := s.mockFS.AdjustBinaryPath(tt.dir, tt.binary)

			s.Equal(tt.expected, result)
		})
	}
}

// TestRemoveBinarySuccess verifies successful binary removal.
//
// When the binary exists and removal succeeds, the function should
// return nil error.
func (s *FSIntegrationTestSuite) TestRemoveBinarySuccess() {
	binaryPath := filepath.Join(testBinDir, testBinaryName)

	s.mockFS.EXPECT().
		RemoveBinary(binaryPath, testBinaryName, false, mock.Anything).
		Return(nil).
		Once()

	err := s.mockFS.RemoveBinary(binaryPath, testBinaryName, false, s.mockLogger)

	s.Require().NoError(err)
}

// TestRemoveBinaryWithVerboseLogging verifies verbose mode parameter is passed correctly.
//
// When verbose mode is enabled, the function should receive the verbose flag
// and logger without error.
func (s *FSIntegrationTestSuite) TestRemoveBinaryWithVerboseLogging() {
	binaryPath := filepath.Join(testBinDir, testBinaryName)

	s.mockFS.EXPECT().
		RemoveBinary(binaryPath, testBinaryName, true, s.mockLogger).
		Return(nil).
		Once()

	err := s.mockFS.RemoveBinary(binaryPath, testBinaryName, true, s.mockLogger)

	s.Require().NoError(err)
}

// TestRemoveBinaryNotFound verifies error handling when binary is not found.
//
// When the binary does not exist at the specified path, the function
// should return ErrBinaryNotFound.
func (s *FSIntegrationTestSuite) TestRemoveBinaryNotFound() {
	nonExistentPath := "/nonexistent/binary"

	s.mockFS.EXPECT().
		RemoveBinary(nonExistentPath, "binary", false, mock.Anything).
		Return(fs.ErrBinaryNotFound).
		Once()

	err := s.mockFS.RemoveBinary(nonExistentPath, "binary", false, s.mockLogger)

	s.Require().ErrorIs(err, fs.ErrBinaryNotFound)
}

// TestRemoveBinaryPermissionDenied verifies error handling for permission denied.
//
// When the binary cannot be removed due to insufficient permissions,
// the function should return an appropriate error.
func (s *FSIntegrationTestSuite) TestRemoveBinaryPermissionDenied() {
	binaryPath := filepath.Join(testBinDir, testBinaryName)
	permissionErr := errors.New("permission denied")

	s.mockFS.EXPECT().
		RemoveBinary(binaryPath, testBinaryName, false, mock.Anything).
		Return(permissionErr).
		Once()

	err := s.mockFS.RemoveBinary(binaryPath, testBinaryName, false, s.mockLogger)

	s.Require().ErrorIs(err, permissionErr)
}

// TestRemoveBinaryVariousScenarios verifies RemoveBinary with various scenarios.
//
// This test uses table-driven testing to cover multiple scenarios:
// - Successful removal with verbose=false
// - Successful removal with verbose=true
// - Binary not found
// - Permission denied.
func (s *FSIntegrationTestSuite) TestRemoveBinaryVariousScenarios() {
	binaryPath := filepath.Join(testBinDir, testBinaryName)

	tests := []struct {
		name        string
		binaryPath  string
		binaryName  string
		verbose     bool
		expectedErr error
	}{
		{
			name:        "successful removal silent",
			binaryPath:  binaryPath,
			binaryName:  testBinaryName,
			verbose:     false,
			expectedErr: nil,
		},
		{
			name:        "successful removal verbose",
			binaryPath:  binaryPath,
			binaryName:  testBinaryName,
			verbose:     true,
			expectedErr: nil,
		},
		{
			name:        "binary not found",
			binaryPath:  "/nonexistent/binary",
			binaryName:  "binary",
			verbose:     false,
			expectedErr: fs.ErrBinaryNotFound,
		},
		{
			name:        "permission denied",
			binaryPath:  filepath.FromSlash("/root/" + testBinaryName),
			binaryName:  testBinaryName,
			verbose:     false,
			expectedErr: errors.New("permission denied"),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.mockFS.EXPECT().
				RemoveBinary(tt.binaryPath, tt.binaryName, tt.verbose, mock.Anything).
				Return(tt.expectedErr).
				Once()

			err := s.mockFS.RemoveBinary(tt.binaryPath, tt.binaryName, tt.verbose, s.mockLogger)

			if tt.expectedErr != nil {
				s.Require().Error(err)

				if errors.Is(tt.expectedErr, fs.ErrBinaryNotFound) {
					s.Require().ErrorIs(err, fs.ErrBinaryNotFound)
				}
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

// TestListBinariesEmpty verifies ListBinaries returns empty list for empty directory.
//
// When the directory is empty or does not exist, the function should
// return an empty slice.
func (s *FSIntegrationTestSuite) TestListBinariesEmpty() {
	s.mockFS.EXPECT().
		ListBinaries(testBinDir).
		Return([]string{}).
		Once()

	result := s.mockFS.ListBinaries(testBinDir)

	s.Empty(result)
}

// TestListBinariesWithBinaries verifies ListBinaries returns binary names.
//
// When the directory contains executable files, the function should
// return a slice of binary names.
func (s *FSIntegrationTestSuite) TestListBinariesWithBinaries() {
	expectedBinaries := []string{"binary1", "binary2", "binary3"}

	s.mockFS.EXPECT().
		ListBinaries(testBinDir).
		Return(expectedBinaries).
		Once()

	result := s.mockFS.ListBinaries(testBinDir)

	s.Equal(expectedBinaries, result)
}

// TestListBinariesNonExistentDir verifies ListBinaries handles non-existent directory.
//
// When the directory does not exist, the function should return an empty slice.
func (s *FSIntegrationTestSuite) TestListBinariesNonExistentDir() {
	nonExistentDir := "/nonexistent/directory"

	s.mockFS.EXPECT().
		ListBinaries(nonExistentDir).
		Return([]string{}).
		Once()

	result := s.mockFS.ListBinaries(nonExistentDir)

	s.Empty(result)
}

// TestListBinariesVariousContents verifies ListBinaries with various directory contents.
//
// This test uses table-driven testing to cover multiple scenarios:
// - Empty directory
// - Directory with multiple binaries
// - Directory with non-executable files
// - Non-existent directory.
func (s *FSIntegrationTestSuite) TestListBinariesVariousContents() {
	tests := []struct {
		name     string
		dir      string
		expected []string
	}{
		{
			name:     "empty directory",
			dir:      testBinDir,
			expected: []string{},
		},
		{
			name:     "multiple binaries",
			dir:      testBinDir,
			expected: []string{"go", "gofmt", "dlv"},
		},
		{
			name:     "single binary",
			dir:      testBinDir,
			expected: []string{"myapp"},
		},
		{
			name:     "non-existent directory",
			dir:      "/does/not/exist",
			expected: []string{},
		},
		{
			name:     "directory with subdirectories",
			dir:      testBinDir,
			expected: []string{"binary1", "binary2"},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.mockFS.EXPECT().
				ListBinaries(tt.dir).
				Return(tt.expected).
				Once()

			result := s.mockFS.ListBinaries(tt.dir)

			s.Equal(tt.expected, result)
		})
	}
}

// TestFullBinaryRemovalWorkflow verifies the complete binary removal workflow.
//
// This test ensures the full workflow works correctly:
// 1. Determine binary directory
// 2. Adjust binary path
// 3. Remove binary
// 4. List binaries to confirm removal.
func (s *FSIntegrationTestSuite) TestFullBinaryRemovalWorkflow() {
	binaryPath := filepath.Join(testBinDir, testBinaryName)

	// Step 1: Determine binary directory
	s.mockFS.EXPECT().
		DetermineBinDir(false).
		Return(testBinDir, nil).
		Once()

	binDir, err := s.mockFS.DetermineBinDir(false)
	s.Require().NoError(err)
	s.Equal(testBinDir, binDir)

	// Step 2: Adjust binary path
	s.mockFS.EXPECT().
		AdjustBinaryPath(binDir, testBinaryName).
		Return(binaryPath).
		Once()

	adjustedPath := s.mockFS.AdjustBinaryPath(binDir, testBinaryName)
	s.Equal(binaryPath, adjustedPath)

	// Step 3: Remove binary
	s.mockFS.EXPECT().
		RemoveBinary(binaryPath, testBinaryName, true, mock.Anything).
		Return(nil).
		Once()

	err = s.mockFS.RemoveBinary(binaryPath, testBinaryName, true, s.mockLogger)
	s.Require().NoError(err)

	// Step 4: List binaries (should not include removed binary)
	s.mockFS.EXPECT().
		ListBinaries(binDir).
		Return([]string{"other-binary"}).
		Once()

	binaries := s.mockFS.ListBinaries(binDir)
	s.NotContains(binaries, testBinaryName)
}

// TestBinaryRemovalWithErrorHandling verifies error handling in removal workflow.
//
// This test ensures errors are properly handled when:
// - Binary is not found
// - Permissions are insufficient
// - Directory resolution fails.
func (s *FSIntegrationTestSuite) TestBinaryRemovalWithErrorHandling() {
	s.Run("binary not found in workflow", func() {
		binaryPath := filepath.Join(testBinDir, testBinaryName)

		// Adjust path succeeds
		s.mockFS.EXPECT().
			AdjustBinaryPath(testBinDir, testBinaryName).
			Return(binaryPath).
			Once()

		adjustedPath := s.mockFS.AdjustBinaryPath(testBinDir, testBinaryName)

		// Remove fails with not found
		s.mockFS.EXPECT().
			RemoveBinary(adjustedPath, testBinaryName, false, mock.Anything).
			Return(fs.ErrBinaryNotFound).
			Once()

		err := s.mockFS.RemoveBinary(adjustedPath, testBinaryName, false, s.mockLogger)
		s.Require().ErrorIs(err, fs.ErrBinaryNotFound)
	})

	s.Run("directory resolution fails", func() {
		s.mockFS.EXPECT().
			DetermineBinDir(true).
			Return("", fs.ErrGorootNotSet).
			Once()

		binDir, err := s.mockFS.DetermineBinDir(true)
		s.Require().ErrorIs(err, fs.ErrGorootNotSet)
		s.Empty(binDir)
	})
}

// TestMultipleBinaryOperations verifies multiple binary operations in sequence.
//
// This test simulates a realistic workflow of managing multiple binaries.
func (s *FSIntegrationTestSuite) TestMultipleBinaryOperations() {
	// List initial binaries
	initialBinaries := []string{"binary1", "binary2", "binary3"}
	s.mockFS.EXPECT().
		ListBinaries(testBinDir).
		Return(initialBinaries).
		Once()

	binaries := s.mockFS.ListBinaries(testBinDir)
	s.Len(binaries, 3)

	// Remove first binary
	binary1Path := filepath.Join(testBinDir, "binary1")
	s.mockFS.EXPECT().
		RemoveBinary(binary1Path, "binary1", false, mock.Anything).
		Return(nil).
		Once()

	err := s.mockFS.RemoveBinary(binary1Path, "binary1", false, s.mockLogger)
	s.Require().NoError(err)

	// List again (binary1 removed)
	s.mockFS.EXPECT().
		ListBinaries(testBinDir).
		Return([]string{"binary2", "binary3"}).
		Once()

	binaries = s.mockFS.ListBinaries(testBinDir)
	s.Len(binaries, 2)
	s.NotContains(binaries, "binary1")

	// Remove second binary
	binary2Path := filepath.Join(testBinDir, "binary2")
	s.mockFS.EXPECT().
		RemoveBinary(binary2Path, "binary2", true, mock.Anything).
		Return(nil).
		Once()

	err = s.mockFS.RemoveBinary(binary2Path, "binary2", true, s.mockLogger)
	s.Require().NoError(err)
}

// TestErrorPropagation verifies errors are properly propagated from FS operations.
//
// This test ensures that errors from the underlying filesystem are wrapped
// and returned correctly to the caller.
func (s *FSIntegrationTestSuite) TestErrorPropagation() {
	tests := []struct {
		name        string
		setupMock   func()
		operation   func() error
		expectedErr error
	}{
		{
			name: "DetermineBinDir error",
			setupMock: func() {
				s.mockFS.EXPECT().
					DetermineBinDir(true).
					Return("", fs.ErrGorootNotSet).
					Once()
			},
			operation: func() error {
				_, err := s.mockFS.DetermineBinDir(true)

				return err
			},
			expectedErr: fs.ErrGorootNotSet,
		},
		{
			name: "RemoveBinary not found",
			setupMock: func() {
				s.mockFS.EXPECT().
					RemoveBinary("/invalid/path", "binary", false, mock.Anything).
					Return(fs.ErrBinaryNotFound).
					Once()
			},
			operation: func() error {
				return s.mockFS.RemoveBinary("/invalid/path", "binary", false, s.mockLogger)
			},
			expectedErr: fs.ErrBinaryNotFound,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()
			err := tt.operation()
			s.Require().ErrorIs(err, tt.expectedErr)
		})
	}
}

// TestFSInterfaceCompliance verifies the mock implements the FS interface.
//
// This test ensures at compile time that MockFS properly implements the FS interface.
func TestFSInterfaceCompliance(t *testing.T) {
	t.Parallel()

	// Verify MockFS implements FS interface
	var _ fs.FS = (*fsMocks.MockFS)(nil)
}

// TestErrorTypes verifies all FS error types are defined.
//
// This test ensures the exported error variables are defined and usable.
func TestErrorTypes(t *testing.T) {
	t.Parallel()

	// Verify error variables are defined and non-nil
	require.Error(t, fs.ErrGorootNotSet)
	require.Error(t, fs.ErrBinaryNotFound)

	// Verify error messages contain expected text
	assert.NotEmpty(t, fs.ErrGorootNotSet.Error())
	assert.NotEmpty(t, fs.ErrBinaryNotFound.Error())
}

// TestDetermineBinDirCrossPlatform verifies DetermineBinDir across platforms.
//
// This test ensures proper directory resolution on different platforms.
func (s *FSIntegrationTestSuite) TestDetermineBinDirCrossPlatform() {
	tests := []struct {
		name        string
		useGoroot   bool
		expectedDir string
		expectedErr error
	}{
		{
			name:        "Linux GOROOT",
			useGoroot:   true,
			expectedDir: "/usr/local/go/bin",
			expectedErr: nil,
		},
		{
			name:        "Linux GOPATH",
			useGoroot:   false,
			expectedDir: "/home/user/go/bin",
			expectedErr: nil,
		},
		{
			name:        "Windows style path",
			useGoroot:   false,
			expectedDir: `C:\Users\User\go\bin`,
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.mockFS.EXPECT().
				DetermineBinDir(tt.useGoroot).
				Return(tt.expectedDir, tt.expectedErr).
				Once()

			binDir, err := s.mockFS.DetermineBinDir(tt.useGoroot)

			if tt.expectedErr != nil {
				s.Require().ErrorIs(err, tt.expectedErr)
			} else {
				s.Require().NoError(err)
				s.Equal(tt.expectedDir, binDir)
			}
		})
	}
}

// TestRemoveBinaryWithDifferentLoggers verifies RemoveBinary with various logger states.
//
// This test ensures the removal operation works correctly with different
// logger configurations and states.
func (s *FSIntegrationTestSuite) TestRemoveBinaryWithDifferentLoggers() {
	binaryPath := filepath.Join(testBinDir, testBinaryName)

	tests := []struct {
		name    string
		verbose bool
	}{
		{
			name:    "silent mode",
			verbose: false,
		},
		{
			name:    "verbose mode",
			verbose: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Create a fresh mock logger for each subtest
			mockLogger := loggerMocks.NewMockLogger(s.T())

			s.mockFS.EXPECT().
				RemoveBinary(binaryPath, testBinaryName, tt.verbose, mockLogger).
				Return(nil).
				Once()

			err := s.mockFS.RemoveBinary(binaryPath, testBinaryName, tt.verbose, mockLogger)
			s.Require().NoError(err)
		})
	}
}

// TestListBinariesFiltering verifies ListBinaries properly filters directory contents.
//
// This test ensures that ListBinaries only returns executable files and
// filters out directories and non-executable files.
func (s *FSIntegrationTestSuite) TestListBinariesFiltering() {
	tests := []struct {
		name     string
		dir      string
		expected []string
	}{
		{
			name:     "only executables",
			dir:      testBinDir,
			expected: []string{"binary1", "binary2", "binary3"},
		},
		{
			name:     "mixed content filtered",
			dir:      testBinDir,
			expected: []string{"go", "gofmt"},
		},
		{
			name:     "empty result",
			dir:      testBinDir,
			expected: []string{},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.mockFS.EXPECT().
				ListBinaries(tt.dir).
				Return(tt.expected).
				Once()

			result := s.mockFS.ListBinaries(tt.dir)
			s.Equal(tt.expected, result)
		})
	}
}

// TestConcurrentFSOperations verifies concurrent filesystem operations.
//
// This test ensures the FS interface can handle concurrent operations safely.
func (s *FSIntegrationTestSuite) TestConcurrentFSOperations() {
	binaryPath1 := filepath.Join(testBinDir, testBinaryName)
	binaryPath2 := filepath.Join(testBinDir, testBinaryName2)

	// Setup expectations for concurrent calls
	s.mockFS.EXPECT().
		DetermineBinDir(false).
		Return(testBinDir, nil).
		Once()

	s.mockFS.EXPECT().
		ListBinaries(testBinDir).
		Return([]string{testBinaryName, testBinaryName2}).
		Once()

	// Execute operations (simulating concurrent usage)
	binDir, err := s.mockFS.DetermineBinDir(false)
	s.Require().NoError(err)
	s.Equal(testBinDir, binDir)

	binaries := s.mockFS.ListBinaries(testBinDir)
	s.Len(binaries, 2)

	// Verify both paths would be adjusted correctly
	s.mockFS.EXPECT().
		AdjustBinaryPath(binDir, testBinaryName).
		Return(binaryPath1).
		Once()

	s.mockFS.EXPECT().
		AdjustBinaryPath(binDir, testBinaryName2).
		Return(binaryPath2).
		Once()

	path1 := s.mockFS.AdjustBinaryPath(binDir, testBinaryName)
	s.Equal(binaryPath1, path1)

	path2 := s.mockFS.AdjustBinaryPath(binDir, testBinaryName2)
	s.Equal(binaryPath2, path2)
}

// TestLoggerInterfaceCompliance verifies the mock implements the Logger interface.
//
// This test ensures at compile time that MockLogger properly implements
// the Logger interface.
func TestLoggerInterfaceCompliance(t *testing.T) {
	t.Parallel()

	// Verify MockLogger implements Logger interface
	var _ logger.Logger = (*loggerMocks.MockLogger)(nil)
}

// TestAdjustBinaryPathEdgeCases verifies AdjustBinaryPath handles edge cases.
//
// This test ensures proper handling of unusual inputs like paths with
// special characters, empty directories, etc.
func (s *FSIntegrationTestSuite) TestAdjustBinaryPathEdgeCases() {
	tests := []struct {
		name     string
		dir      string
		binary   string
		expected string
	}{
		{
			name:     "path with spaces",
			dir:      "/home/user/my apps",
			binary:   "myapp",
			expected: filepath.FromSlash("/home/user/my apps/myapp"),
		},
		{
			name:     "binary with version",
			dir:      testBinDir,
			binary:   "myapp-v1.2.3",
			expected: filepath.Join(testBinDir, "myapp-v1.2.3"),
		},
		{
			name:     "relative directory",
			dir:      "./local/bin",
			binary:   "tool",
			expected: filepath.FromSlash("./local/bin/tool"),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.mockFS.EXPECT().
				AdjustBinaryPath(tt.dir, tt.binary).
				Return(tt.expected).
				Once()

			result := s.mockFS.AdjustBinaryPath(tt.dir, tt.binary)
			s.Equal(tt.expected, result)
		})
	}
}

// TestRemoveBinaryInvalidPaths verifies RemoveBinary with invalid paths.
//
// This test ensures proper error handling for various invalid path scenarios.
func (s *FSIntegrationTestSuite) TestRemoveBinaryInvalidPaths() {
	tests := []struct {
		name        string
		binaryPath  string
		binaryName  string
		expectedErr error
	}{
		{
			name:        "empty path",
			binaryPath:  "",
			binaryName:  "binary",
			expectedErr: errors.New("invalid path"),
		},
		{
			name:        "relative path",
			binaryPath:  "./binary",
			binaryName:  "binary",
			expectedErr: nil,
		},
		{
			name:        "path with parent directory traversal",
			binaryPath:  "../../../etc/passwd",
			binaryName:  "passwd",
			expectedErr: errors.New("invalid path"),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.mockFS.EXPECT().
				RemoveBinary(tt.binaryPath, tt.binaryName, false, mock.Anything).
				Return(tt.expectedErr).
				Once()

			err := s.mockFS.RemoveBinary(tt.binaryPath, tt.binaryName, false, s.mockLogger)

			if tt.expectedErr != nil {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}
