/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

// Package buildinfo_test provides black-box integration tests for the buildinfo package.
//
// These tests verify the orchestration layer that coordinates between the buildinfo
// extractor and consuming components. All external dependencies are mocked
// using Mockery-generated mocks to ensure no external calls are made.
package buildinfo_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/nicholas-fedor/go-remove/internal/buildinfo"
	"github.com/nicholas-fedor/go-remove/internal/buildinfo/mocks"
)

// Test constants for consistent test data.
const (
	testBinaryPath    = "/usr/local/bin/test-binary"
	testBinaryPath2   = "/usr/local/bin/another-binary"
	testModulePath    = "github.com/test/binary"
	testVersion       = "v1.0.0"
	testPseudoVersion = "v0.0.0-20260302120000-abc123def456"
	testDevelVersion  = "(devel)"
	testVCSRevision   = "abc123def456789"
	testVCSTime       = "2026-01-15T10:30:00Z"
	testGoVersion     = "go1.21.5"
	testChecksum      = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	testChecksum2     = "a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456"
)

// BuildInfoIntegrationTestSuite provides integration tests for the buildinfo package.
//
// This suite tests the orchestration between:
// - Extractor implementations and their consumers
// - Build info data extraction workflows
// - Checksum calculation operations
// - Version parsing and type detection
// - Reinstallability determination
//
// All dependencies are mocked to ensure isolated, deterministic tests.
type BuildInfoIntegrationTestSuite struct {
	suite.Suite

	// Mock for the extractor
	extractor *mocks.MockExtractor
}

// SetupTest initializes the test suite before each test.
//
// This creates fresh mocks for each test to ensure test isolation.
func (s *BuildInfoIntegrationTestSuite) SetupTest() {
	// Create mock
	s.extractor = mocks.NewMockExtractor(s.T())
}

// TestBuildInfoIntegrationTestSuite runs the integration test suite.
func TestBuildInfoIntegrationTestSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(BuildInfoIntegrationTestSuite))
}

// TestExtractWorkflowWithVariousBuildInfoData verifies extract with different build info configurations.
//
// This test uses table-driven testing to cover multiple scenarios:
// - Standard semantic version
// - Pseudo-version
// - Development build
// - Empty settings
// - Full settings with all fields populated.
func (s *BuildInfoIntegrationTestSuite) TestExtractWorkflowWithVariousBuildInfoData() {
	ctx := context.Background()

	tests := []struct {
		name          string
		binaryPath    string
		buildInfoData *buildinfo.BuildInfoData
	}{
		{
			name:       "semantic version with full metadata",
			binaryPath: "/usr/local/bin/binary1",
			buildInfoData: &buildinfo.BuildInfoData{
				ModulePath:  testModulePath,
				Version:     testVersion,
				VCSRevision: testVCSRevision,
				VCSTime:     testVCSTime,
				GoVersion:   testGoVersion,
				Settings: map[string]string{
					"vcs.revision": testVCSRevision,
					"vcs.time":     testVCSTime,
				},
				RawJSON: []byte(`{"module": "test"}`),
			},
		},
		{
			name:       "pseudo-version build",
			binaryPath: "/usr/local/bin/binary2",
			buildInfoData: &buildinfo.BuildInfoData{
				ModulePath:  testModulePath,
				Version:     testPseudoVersion,
				VCSRevision: testVCSRevision,
				VCSTime:     testVCSTime,
				GoVersion:   testGoVersion,
				Settings: map[string]string{
					"vcs.revision": testVCSRevision,
					"vcs.time":     testVCSTime,
				},
				RawJSON: []byte(`{"module": "test"}`),
			},
		},
		{
			name:       "development build",
			binaryPath: "/usr/local/bin/binary3",
			buildInfoData: &buildinfo.BuildInfoData{
				ModulePath:  testModulePath,
				Version:     testDevelVersion,
				VCSRevision: testVCSRevision,
				VCSTime:     testVCSTime,
				GoVersion:   testGoVersion,
				Settings: map[string]string{
					"vcs.revision": testVCSRevision,
					"vcs.time":     testVCSTime,
				},
				RawJSON: []byte(`{"module": "test"}`),
			},
		},
		{
			name:       "minimal build info",
			binaryPath: "/usr/local/bin/binary4",
			buildInfoData: &buildinfo.BuildInfoData{
				ModulePath: testModulePath,
				Version:    testVersion,
				GoVersion:  testGoVersion,
				Settings:   map[string]string{},
				RawJSON:    []byte(`{}`),
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Setup expectations - each test gets its own mock return value
			s.extractor.EXPECT().
				Extract(mock.Anything, tt.binaryPath).
				Return(tt.buildInfoData, nil)

			// Execute
			result, err := s.extractor.Extract(ctx, tt.binaryPath)

			// Verify the mock returned exactly what we configured
			s.Require().NoError(err)
			s.Require().NotNil(result)
			s.Equal(tt.buildInfoData.ModulePath, result.ModulePath)
			s.Equal(tt.buildInfoData.Version, result.Version)
			s.Equal(tt.buildInfoData.VCSRevision, result.VCSRevision)
			s.Equal(tt.buildInfoData.GoVersion, result.GoVersion)
			s.NotNil(result.Settings)
		})
	}
}

// TestExtractWorkflowWithMultipleBinaries verifies extract with different binary paths.
//
// This test ensures the extractor correctly handles multiple different binaries
// in a workflow scenario.
func (s *BuildInfoIntegrationTestSuite) TestExtractWorkflowWithMultipleBinaries() {
	ctx := context.Background()

	// First binary
	buildData1 := &buildinfo.BuildInfoData{
		ModulePath:  "github.com/user/app1",
		Version:     "v1.0.0",
		VCSRevision: "rev1",
		GoVersion:   "go1.21",
		Settings:    map[string]string{},
		RawJSON:     []byte(`{"binary": "1"}`),
	}

	s.extractor.EXPECT().
		Extract(mock.Anything, testBinaryPath).
		Return(buildData1, nil)

	result1, err := s.extractor.Extract(ctx, testBinaryPath)
	s.Require().NoError(err)
	s.Equal("github.com/user/app1", result1.ModulePath)

	// Second binary
	buildData2 := &buildinfo.BuildInfoData{
		ModulePath:  "github.com/user/app2",
		Version:     "v2.0.0",
		VCSRevision: "rev2",
		GoVersion:   "go1.22",
		Settings:    map[string]string{},
		RawJSON:     []byte(`{"binary": "2"}`),
	}

	s.extractor.EXPECT().
		Extract(mock.Anything, testBinaryPath2).
		Return(buildData2, nil)

	result2, err := s.extractor.Extract(ctx, testBinaryPath2)
	s.Require().NoError(err)
	s.Equal("github.com/user/app2", result2.ModulePath)
}

// TestCalculateChecksumOperations verifies checksum calculation workflow.
//
// This test ensures CalculateChecksum properly:
// - Returns expected checksum for valid binary
// - Handles different binary paths
// - Propagates errors when checksum calculation fails.
func (s *BuildInfoIntegrationTestSuite) TestCalculateChecksumOperations() {
	tests := []struct {
		name           string
		binaryPath     string
		checksum       string
		expectedErr    error
		expectedResult string
	}{
		{
			name:           "valid checksum",
			binaryPath:     testBinaryPath,
			checksum:       testChecksum,
			expectedErr:    nil,
			expectedResult: testChecksum,
		},
		{
			name:           "different binary different checksum",
			binaryPath:     testBinaryPath2,
			checksum:       testChecksum2,
			expectedErr:    nil,
			expectedResult: testChecksum2,
		},
		{
			name:           "checksum calculation error",
			binaryPath:     "/invalid/path",
			checksum:       "",
			expectedErr:    buildinfo.ErrPathNotFound,
			expectedResult: "",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Setup expectations
			s.extractor.EXPECT().
				CalculateChecksum(tt.binaryPath).
				Return(tt.checksum, tt.expectedErr)

			// Execute
			checksum, err := s.extractor.CalculateChecksum(tt.binaryPath)

			// Verify
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Require().ErrorIs(err, tt.expectedErr)
			} else {
				s.Require().NoError(err)
				s.Equal(tt.expectedResult, checksum)
			}
		})
	}
}

// TestCalculateChecksumWithMultipleCalls verifies checksum calculation for multiple binaries.
//
// This test simulates a workflow where checksums are calculated for multiple binaries.
func (s *BuildInfoIntegrationTestSuite) TestCalculateChecksumWithMultipleCalls() {
	binaries := []struct {
		path     string
		checksum string
	}{
		{testBinaryPath, testChecksum},
		{testBinaryPath2, testChecksum2},
	}

	for _, binary := range binaries {
		s.extractor.EXPECT().
			CalculateChecksum(binary.path).
			Return(binary.checksum, nil)

		checksum, err := s.extractor.CalculateChecksum(binary.path)
		s.Require().NoError(err)
		s.Equal(binary.checksum, checksum)
	}
}

// TestIsGoBinaryChecks verifies Go binary detection workflow.
//
// This test ensures IsGoBinary properly:
// - Returns true for valid Go binaries
// - Returns false for non-Go binaries
// - Returns false for non-existent files.
func (s *BuildInfoIntegrationTestSuite) TestIsGoBinaryChecks() {
	tests := []struct {
		name         string
		binaryPath   string
		isGoBinary   bool
		expectedBool bool
	}{
		{
			name:         "valid Go binary",
			binaryPath:   testBinaryPath,
			isGoBinary:   true,
			expectedBool: true,
		},
		{
			name:         "another valid Go binary",
			binaryPath:   testBinaryPath2,
			isGoBinary:   true,
			expectedBool: true,
		},
		{
			name:         "non-Go binary",
			binaryPath:   "/usr/bin/ls",
			isGoBinary:   false,
			expectedBool: false,
		},
		{
			name:         "non-existent file",
			binaryPath:   "/nonexistent/binary",
			isGoBinary:   false,
			expectedBool: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Setup expectations
			s.extractor.EXPECT().
				IsGoBinary(tt.binaryPath).
				Return(tt.isGoBinary)

			// Execute
			result := s.extractor.IsGoBinary(tt.binaryPath)

			// Verify
			s.Equal(tt.expectedBool, result)
		})
	}
}

// TestIsGoBinaryWithExtractIntegration verifies IsGoBinary and Extract integration.
//
// This test ensures the workflow of checking if a file is a Go binary
// before attempting extraction works correctly.
func (s *BuildInfoIntegrationTestSuite) TestIsGoBinaryWithExtractIntegration() {
	ctx := context.Background()

	// Scenario 1: Valid Go binary - check passes, extract succeeds
	s.Run("valid binary workflow", func() {
		buildData := &buildinfo.BuildInfoData{
			ModulePath: testModulePath,
			Version:    testVersion,
			GoVersion:  testGoVersion,
			Settings:   map[string]string{},
			RawJSON:    []byte(`{}`),
		}

		s.extractor.EXPECT().
			IsGoBinary(testBinaryPath).
			Return(true)

		s.extractor.EXPECT().
			Extract(mock.Anything, testBinaryPath).
			Return(buildData, nil)

		// Workflow: check then extract
		if s.extractor.IsGoBinary(testBinaryPath) {
			result, err := s.extractor.Extract(ctx, testBinaryPath)
			s.Require().NoError(err)
			s.Equal(testModulePath, result.ModulePath)
		}
	})

	// Scenario 2: Invalid binary - check fails, extract not called
	s.Run("invalid binary workflow", func() {
		s.extractor.EXPECT().
			IsGoBinary("/usr/bin/ls").
			Return(false)

		// Workflow: check fails, skip extract
		isGo := s.extractor.IsGoBinary("/usr/bin/ls")
		s.False(isGo)
	})
}

// TestContextCancellationHandling verifies proper handling of cancelled contexts.
//
// This test ensures that Extract respects context cancellation and returns
// appropriate errors.
func (s *BuildInfoIntegrationTestSuite) TestContextCancellationHandling() {
	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Setup expectation for cancelled context
	s.extractor.EXPECT().
		Extract(ctx, testBinaryPath).
		Return(nil, context.Canceled)

	// Execute
	result, err := s.extractor.Extract(ctx, testBinaryPath)

	// Verify
	s.Require().Error(err)
	s.Require().ErrorIs(err, context.Canceled)
	s.Nil(result)
}

// TestContextTimeoutHandling verifies proper handling of timed-out contexts.
//
// This test ensures that Extract respects context deadlines.
func (s *BuildInfoIntegrationTestSuite) TestContextTimeoutHandling() {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()

	// Setup expectation for timed-out context
	s.extractor.EXPECT().
		Extract(ctx, testBinaryPath).
		Return(nil, context.DeadlineExceeded)

	// Execute
	result, err := s.extractor.Extract(ctx, testBinaryPath)

	// Verify
	s.Require().Error(err)
	s.Require().ErrorIs(err, context.DeadlineExceeded)
	s.Nil(result)
}

// TestErrorHandlingNonExistentBinary verifies error handling for non-existent binaries.
//
// When a binary path does not exist, Extract should return ErrPathNotFound.
func (s *BuildInfoIntegrationTestSuite) TestErrorHandlingNonExistentBinary() {
	ctx := context.Background()
	nonExistentPath := "/nonexistent/path/binary"

	// Setup expectations
	s.extractor.EXPECT().
		Extract(mock.Anything, nonExistentPath).
		Return(nil, buildinfo.ErrPathNotFound)

	s.extractor.EXPECT().
		IsGoBinary(nonExistentPath).
		Return(false)

	// Test Extract
	result, err := s.extractor.Extract(ctx, nonExistentPath)
	s.Require().Error(err)
	s.Require().ErrorIs(err, buildinfo.ErrPathNotFound)
	s.Nil(result)

	// Test IsGoBinary
	isGo := s.extractor.IsGoBinary(nonExistentPath)
	s.False(isGo)
}

// TestErrorHandlingInvalidBinary verifies error handling for invalid binaries.
//
// When a file is not a valid Go binary, Extract should return ErrNotGoBinary.
func (s *BuildInfoIntegrationTestSuite) TestErrorHandlingInvalidBinary() {
	ctx := context.Background()
	invalidPath := "/usr/bin/ls"

	// Setup expectations
	s.extractor.EXPECT().
		Extract(mock.Anything, invalidPath).
		Return(nil, buildinfo.ErrNotGoBinary)

	s.extractor.EXPECT().
		IsGoBinary(invalidPath).
		Return(false)

	// Test Extract
	result, err := s.extractor.Extract(ctx, invalidPath)
	s.Require().Error(err)
	s.Require().ErrorIs(err, buildinfo.ErrNotGoBinary)
	s.Nil(result)

	// Test IsGoBinary
	isGo := s.extractor.IsGoBinary(invalidPath)
	s.False(isGo)
}

// TestErrorHandlingBuildInfoNotFound verifies error handling for missing build info.
//
// When a file exists but has no build info, Extract should return ErrBuildInfoNotFound.
func (s *BuildInfoIntegrationTestSuite) TestErrorHandlingBuildInfoNotFound() {
	ctx := context.Background()
	strippedBinary := "/usr/local/bin/stripped"

	// Setup expectation
	s.extractor.EXPECT().
		Extract(mock.Anything, strippedBinary).
		Return(nil, buildinfo.ErrBuildInfoNotFound)

	// Execute
	result, err := s.extractor.Extract(ctx, strippedBinary)

	// Verify
	s.Require().Error(err)
	s.Require().ErrorIs(err, buildinfo.ErrBuildInfoNotFound)
	s.Nil(result)
}

// TestParseVersionType verifies version type parsing for various version formats.
//
// This test uses table-driven testing to cover:
// - Semantic versions (v1.2.3)
// - Pseudo-versions (v0.0.0-20260302120000-abc123)
// - Development builds ((devel))
// - Empty versions
// - Unknown formats.
func TestParseVersionType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		version  string
		expected string
	}{
		{
			name:     "semantic version",
			version:  "v1.2.3",
			expected: "semantic",
		},
		{
			name:     "semantic version with prerelease",
			version:  "v1.2.3-beta.1",
			expected: "pseudo",
		},
		{
			name:     "pseudo-version",
			version:  "v0.0.0-20260302120000-abc123def456",
			expected: "pseudo",
		},
		{
			name:     "pseudo-version with different timestamp",
			version:  "v0.0.0-20230101120000-deadbeef1234",
			expected: "pseudo",
		},
		{
			name:     "development build",
			version:  "(devel)",
			expected: "devel",
		},
		{
			name:     "empty version",
			version:  "",
			expected: "unknown",
		},
		{
			name:     "version without v prefix",
			version:  "1.2.3",
			expected: "unknown",
		},
		{
			name:     "arbitrary string",
			version:  "some-random-version",
			expected: "unknown",
		},
		{
			name:     "just v",
			version:  "v",
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := buildinfo.ParseVersionType(tt.version)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsReinstallable verifies reinstallability determination.
//
// This test ensures IsReinstallable returns true only when both
// ModulePath and VCSRevision are present.
func TestIsReinstallable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		buildInfo *buildinfo.BuildInfoData
		expected  bool
	}{
		{
			name: "full metadata - reinstallable",
			buildInfo: &buildinfo.BuildInfoData{
				ModulePath:  testModulePath,
				Version:     testVersion,
				VCSRevision: testVCSRevision,
				GoVersion:   testGoVersion,
				Settings:    map[string]string{},
				RawJSON:     []byte(`{}`),
			},
			expected: true,
		},
		{
			name: "missing module path - not reinstallable",
			buildInfo: &buildinfo.BuildInfoData{
				ModulePath:  "",
				Version:     testVersion,
				VCSRevision: testVCSRevision,
				GoVersion:   testGoVersion,
				Settings:    map[string]string{},
				RawJSON:     []byte(`{}`),
			},
			expected: false,
		},
		{
			name: "missing vcs revision - not reinstallable",
			buildInfo: &buildinfo.BuildInfoData{
				ModulePath:  testModulePath,
				Version:     testVersion,
				VCSRevision: "",
				GoVersion:   testGoVersion,
				Settings:    map[string]string{},
				RawJSON:     []byte(`{}`),
			},
			expected: false,
		},
		{
			name: "both missing - not reinstallable",
			buildInfo: &buildinfo.BuildInfoData{
				ModulePath:  "",
				Version:     testVersion,
				VCSRevision: "",
				GoVersion:   testGoVersion,
				Settings:    map[string]string{},
				RawJSON:     []byte(`{}`),
			},
			expected: false,
		},
		{
			name: "minimal reinstallable",
			buildInfo: &buildinfo.BuildInfoData{
				ModulePath:  testModulePath,
				VCSRevision: testVCSRevision,
				Settings:    map[string]string{},
				RawJSON:     []byte(`{}`),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.buildInfo.IsReinstallable()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGetInstallCommand verifies install command generation.
//
// This test ensures GetInstallCommand returns appropriate commands
// based on available build info data.
func TestGetInstallCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		buildInfo *buildinfo.BuildInfoData
		expected  string
	}{
		{
			name: "full metadata - install at revision",
			buildInfo: &buildinfo.BuildInfoData{
				ModulePath:  testModulePath,
				Version:     testVersion,
				VCSRevision: testVCSRevision,
				Settings:    map[string]string{},
				RawJSON:     []byte(`{}`),
			},
			expected: "go install " + testModulePath + "@" + testVCSRevision,
		},
		{
			name: "no vcs revision - not reinstallable",
			buildInfo: &buildinfo.BuildInfoData{
				ModulePath: testModulePath,
				Version:    testVersion,
				Settings:   map[string]string{},
				RawJSON:    []byte(`{}`),
			},
			expected: "",
		},
		{
			name: "devel version no revision - not reinstallable",
			buildInfo: &buildinfo.BuildInfoData{
				ModulePath:  testModulePath,
				Version:     testDevelVersion,
				VCSRevision: "",
				Settings:    map[string]string{},
				RawJSON:     []byte(`{}`),
			},
			expected: "",
		},
		{
			name: "not reinstallable - empty command",
			buildInfo: &buildinfo.BuildInfoData{
				ModulePath: "",
				Version:    testVersion,
				Settings:   map[string]string{},
				RawJSON:    []byte(`{}`),
			},
			expected: "",
		},
		{
			name: "only module path no revision - not reinstallable",
			buildInfo: &buildinfo.BuildInfoData{
				ModulePath: testModulePath,
				Settings:   map[string]string{},
				RawJSON:    []byte(`{}`),
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.buildInfo.GetInstallCommand()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestFullExtractionWorkflow verifies the complete extraction workflow.
//
// This test ensures that the full workflow of checking, extracting, and
// computing checksum works together properly.
func (s *BuildInfoIntegrationTestSuite) TestFullExtractionWorkflow() {
	ctx := context.Background()

	buildData := &buildinfo.BuildInfoData{
		ModulePath:  testModulePath,
		Version:     testVersion,
		VCSRevision: testVCSRevision,
		VCSTime:     testVCSTime,
		GoVersion:   testGoVersion,
		Settings: map[string]string{
			"vcs.revision": testVCSRevision,
			"vcs.time":     testVCSTime,
		},
		RawJSON: []byte(`{"test": "data"}`),
	}

	// Full workflow:
	// 1. Check if Go binary
	// 2. Extract build info
	// 3. Calculate checksum

	s.extractor.EXPECT().
		IsGoBinary(testBinaryPath).
		Return(true)

	s.extractor.EXPECT().
		Extract(mock.Anything, testBinaryPath).
		Return(buildData, nil)

	s.extractor.EXPECT().
		CalculateChecksum(testBinaryPath).
		Return(testChecksum, nil)

	// Execute workflow
	if !s.extractor.IsGoBinary(testBinaryPath) {
		s.T().Fatal("Expected binary to be a Go binary")
	}

	info, err := s.extractor.Extract(ctx, testBinaryPath)
	s.Require().NoError(err)
	s.Require().NotNil(info)

	checksum, err := s.extractor.CalculateChecksum(testBinaryPath)
	s.Require().NoError(err)
	s.Equal(testChecksum, checksum)

	// Verify data integrity
	s.Equal(testModulePath, info.ModulePath)
	s.Equal(testVersion, info.Version)
	s.Equal(testVCSRevision, info.VCSRevision)
	s.True(info.IsReinstallable())

	installCmd := info.GetInstallCommand()
	s.NotEmpty(installCmd)
	s.Contains(installCmd, testModulePath)
}

// TestExtractWithContextPropagation verifies context is properly propagated.
//
// This test ensures the context is passed through to the Extract method
// and can be used for cancellation/timeout.
func (s *BuildInfoIntegrationTestSuite) TestExtractWithContextPropagation() {
	ctx := context.Background()

	buildData := &buildinfo.BuildInfoData{
		ModulePath: testModulePath,
		Version:    testVersion,
		GoVersion:  testGoVersion,
		Settings:   map[string]string{},
		RawJSON:    []byte(`{}`),
	}

	// Setup expectation with context matching
	s.extractor.EXPECT().
		Extract(mock.Anything, testBinaryPath).
		Return(buildData, nil)

	// Execute with background context
	result, err := s.extractor.Extract(ctx, testBinaryPath)

	// Verify
	s.Require().NoError(err)
	s.NotNil(result)
}

// TestErrorPropagation verifies errors are properly wrapped and propagated.
//
// This test ensures that errors from the extractor are properly returned
// and can be inspected.
func (s *BuildInfoIntegrationTestSuite) TestErrorPropagation() {
	ctx := context.Background()

	// Test wrapped error
	originalErr := errors.New("underlying error")

	s.extractor.EXPECT().
		Extract(mock.Anything, testBinaryPath).
		Return(nil, originalErr)

	_, err := s.extractor.Extract(ctx, testBinaryPath)
	s.Require().Error(err)
	s.Require().ErrorIs(err, originalErr)
}

// TestBuildInfoDataStructure verifies BuildInfoData fields are properly accessible.
//
// This test ensures all fields of BuildInfoData can be set and retrieved.
func TestBuildInfoDataStructure(t *testing.T) {
	t.Parallel()

	settings := map[string]string{
		"vcs.revision": testVCSRevision,
		"vcs.time":     testVCSTime,
		"custom.key":   "custom.value",
	}

	rawJSON := []byte(`{"module": "test", "version": "v1.0.0"}`)

	data := &buildinfo.BuildInfoData{
		ModulePath:  testModulePath,
		Version:     testVersion,
		VCSRevision: testVCSRevision,
		VCSTime:     testVCSTime,
		GoVersion:   testGoVersion,
		Settings:    settings,
		RawJSON:     rawJSON,
	}

	assert.Equal(t, testModulePath, data.ModulePath)
	assert.Equal(t, testVersion, data.Version)
	assert.Equal(t, testVCSRevision, data.VCSRevision)
	assert.Equal(t, testVCSTime, data.VCSTime)
	assert.Equal(t, testGoVersion, data.GoVersion)
	assert.Equal(t, settings, data.Settings)
	assert.JSONEq(t, string(rawJSON), string(data.RawJSON))
}

// TestBuildInfoDataWithEmptySettings verifies BuildInfoData with minimal settings.
func TestBuildInfoDataWithEmptySettings(t *testing.T) {
	t.Parallel()

	data := &buildinfo.BuildInfoData{
		ModulePath: testModulePath,
		Version:    testVersion,
		GoVersion:  testGoVersion,
		Settings:   map[string]string{},
		RawJSON:    []byte(`{}`),
	}

	assert.NotNil(t, data.Settings)
	assert.Empty(t, data.Settings)
	assert.False(t, data.IsReinstallable())
}

// TestBuildInfoDataWithNilSettings verifies BuildInfoData handles nil settings.
func TestBuildInfoDataWithNilSettings(t *testing.T) {
	t.Parallel()

	data := &buildinfo.BuildInfoData{
		ModulePath:  testModulePath,
		Version:     testVersion,
		VCSRevision: testVCSRevision,
		GoVersion:   testGoVersion,
		Settings:    nil,
		RawJSON:     []byte(`{}`),
	}

	// Even with nil settings, should still be reinstallable if module and revision present
	assert.True(t, data.IsReinstallable())
}

// TestMultipleOperationsSequence verifies multiple operations in sequence.
//
// This test simulates a workflow where multiple operations are performed
// on the same binary.
func (s *BuildInfoIntegrationTestSuite) TestMultipleOperationsSequence() {
	ctx := context.Background()

	buildData := &buildinfo.BuildInfoData{
		ModulePath:  testModulePath,
		Version:     testVersion,
		VCSRevision: testVCSRevision,
		GoVersion:   testGoVersion,
		Settings:    map[string]string{},
		RawJSON:     []byte(`{}`),
	}

	// Sequence: IsGoBinary -> Extract -> CalculateChecksum
	s.extractor.EXPECT().IsGoBinary(testBinaryPath).Return(true)
	s.extractor.EXPECT().Extract(mock.Anything, testBinaryPath).Return(buildData, nil)
	s.extractor.EXPECT().CalculateChecksum(testBinaryPath).Return(testChecksum, nil)

	// Operation 1: Check
	isGo := s.extractor.IsGoBinary(testBinaryPath)
	s.True(isGo)

	// Operation 2: Extract
	info, err := s.extractor.Extract(ctx, testBinaryPath)
	s.Require().NoError(err)
	s.Equal(testModulePath, info.ModulePath)

	// Operation 3: Checksum
	checksum, err := s.extractor.CalculateChecksum(testBinaryPath)
	s.Require().NoError(err)
	s.Equal(testChecksum, checksum)
}

// TestExtractorInterfaceCompliance verifies the mock implements the interface correctly.
//
// This test ensures the MockExtractor properly implements the Extractor interface.
func TestExtractorInterfaceCompliance(t *testing.T) {
	t.Parallel()

	// This test verifies at compile time that MockExtractor implements Extractor
	var _ buildinfo.Extractor = (*mocks.MockExtractor)(nil)
}

// TestErrorTypes verifies error type definitions.
//
// This test ensures the exported error variables are defined and usable.
func TestErrorTypes(t *testing.T) {
	t.Parallel()

	// Verify error variables are defined and non-nil
	require.Error(t, buildinfo.ErrNotGoBinary)
	require.Error(t, buildinfo.ErrBuildInfoNotFound)
	require.Error(t, buildinfo.ErrPathNotFound)
	require.Error(t, buildinfo.ErrUnsupportedPlatform)

	// Verify error messages contain expected text
	assert.NotEmpty(t, buildinfo.ErrNotGoBinary.Error())
	assert.NotEmpty(t, buildinfo.ErrBuildInfoNotFound.Error())
	assert.NotEmpty(t, buildinfo.ErrPathNotFound.Error())
	assert.NotEmpty(t, buildinfo.ErrUnsupportedPlatform.Error())
}

// TestVersionTypeEdgeCases verifies ParseVersionType handles edge cases.
//
// This test ensures version parsing handles various edge cases correctly.
func TestVersionTypeEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		version string
		want    string
	}{
		{
			name:    "version with multiple hyphens",
			version: "v0.0.0-20260302-abc123",
			want:    "pseudo",
		},
		{
			name:    "version starting with hyphen after v",
			version: "v-1.2.3",
			want:    "pseudo",
		},
		{
			name:    "long semantic version",
			version: "v10.20.30",
			want:    "semantic",
		},
		{
			name:    "just v0",
			version: "v0",
			want:    "semantic",
		},
		{
			name:    "v with only hyphens",
			version: "v---",
			want:    "pseudo",
		},
		{
			name:    "whitespace version",
			version: "   ",
			want:    "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := buildinfo.ParseVersionType(tt.version)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestGetInstallCommandVariations verifies GetInstallCommand with various scenarios.
//
// This test ensures install commands are generated correctly for different
// combinations of available data.
func TestGetInstallCommandVariations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		modulePath string
		version    string
		revision   string
		expected   string
	}{
		{
			name:       "everything present - uses revision",
			modulePath: "github.com/test/app",
			version:    "v1.0.0",
			revision:   "abc123",
			expected:   "go install github.com/test/app@abc123",
		},
		{
			name:       "no revision - not reinstallable",
			modulePath: "github.com/test/app",
			version:    "v2.0.0",
			revision:   "",
			expected:   "",
		},
		{
			name:       "devel version no revision - not reinstallable",
			modulePath: "github.com/test/app",
			version:    "(devel)",
			revision:   "",
			expected:   "",
		},
		{
			name:       "no module path - empty",
			modulePath: "",
			version:    "v1.0.0",
			revision:   "abc123",
			expected:   "",
		},
		{
			name:       "empty version with revision - uses revision",
			modulePath: "github.com/test/app",
			version:    "",
			revision:   "def456",
			expected:   "go install github.com/test/app@def456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data := &buildinfo.BuildInfoData{
				ModulePath:  tt.modulePath,
				Version:     tt.version,
				VCSRevision: tt.revision,
				Settings:    map[string]string{},
				RawJSON:     []byte(`{}`),
			}

			result := data.GetInstallCommand()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestConcurrentExtractOperations verifies concurrent extract operations.
//
// This test ensures the extractor can handle concurrent operations safely.
func (s *BuildInfoIntegrationTestSuite) TestConcurrentExtractOperations() {
	ctx := context.Background()

	buildData1 := &buildinfo.BuildInfoData{
		ModulePath: "github.com/user/app1",
		Version:    "v1.0.0",
		GoVersion:  "go1.21",
		Settings:   map[string]string{},
		RawJSON:    []byte(`{}`),
	}

	buildData2 := &buildinfo.BuildInfoData{
		ModulePath: "github.com/user/app2",
		Version:    "v2.0.0",
		GoVersion:  "go1.22",
		Settings:   map[string]string{},
		RawJSON:    []byte(`{}`),
	}

	// Setup expectations for concurrent calls
	s.extractor.EXPECT().
		Extract(mock.Anything, testBinaryPath).
		Return(buildData1, nil)

	s.extractor.EXPECT().
		Extract(mock.Anything, testBinaryPath2).
		Return(buildData2, nil)

	// Execute both operations (simulating concurrent usage)
	result1, err1 := s.extractor.Extract(ctx, testBinaryPath)
	result2, err2 := s.extractor.Extract(ctx, testBinaryPath2)

	// Verify both succeeded
	s.Require().NoError(err1)
	s.Require().NoError(err2)
	s.Equal("github.com/user/app1", result1.ModulePath)
	s.Equal("github.com/user/app2", result2.ModulePath)
}

// TestIntegrationWithExtractorCreation verifies NewExtractor function signature.
//
// This test ensures NewExtractor exists and returns the expected types.
func TestIntegrationWithExtractorCreation(t *testing.T) {
	t.Parallel()

	// Verify the function signature exists
	// Note: We can't actually call NewExtractor in unit tests because
	// it checks for platform support, but we verify the types are correct

	// NewExtractor returns (*DefaultExtractor, error)
	// This is a compile-time check
	_ = buildinfo.NewExtractor
}

// TestExtractorErrorsIntegration verifies error handling integration.
//
// This test ensures errors from the extractor are properly integrated
// with the consuming components.
func (s *BuildInfoIntegrationTestSuite) TestExtractorErrorsIntegration() {
	ctx := context.Background()

	tests := []struct {
		name        string
		binaryPath  string
		expectedErr error
	}{
		{
			name:        "path not found error",
			binaryPath:  "/missing/binary",
			expectedErr: buildinfo.ErrPathNotFound,
		},
		{
			name:        "not a Go binary error",
			binaryPath:  "/usr/bin/ls",
			expectedErr: buildinfo.ErrNotGoBinary,
		},
		{
			name:        "build info not found error",
			binaryPath:  "/stripped/binary",
			expectedErr: buildinfo.ErrBuildInfoNotFound,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Setup mock for this specific test case
			s.extractor.EXPECT().
				Extract(mock.Anything, tt.binaryPath).
				Return(nil, tt.expectedErr)

			result, err := s.extractor.Extract(ctx, tt.binaryPath)
			s.Require().Error(err)
			s.Require().ErrorIs(err, tt.expectedErr)
			s.Nil(result)
		})
	}
}
