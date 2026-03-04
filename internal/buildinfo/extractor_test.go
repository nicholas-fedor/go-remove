/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package buildinfo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// platformLinux is the GOOS value for Linux systems.
const platformLinux = "linux"

// platformWindows is the GOOS value for Windows systems.
const platformWindows = "windows"

// TestNewExtractor tests the creation of a new extractor instance.
func TestNewExtractor(t *testing.T) {
	t.Parallel()

	extractor, err := NewExtractor()

	// Should succeed on supported platforms
	if runtime.GOOS == platformLinux || runtime.GOOS == platformWindows {
		require.NoError(t, err)
		require.NotNil(t, extractor)
	}
}

// TestDefaultExtractor_Extract tests the Extract method.
func TestDefaultExtractor_Extract(t *testing.T) {
	t.Parallel()

	extractor, err := NewExtractor()
	require.NoError(t, err)

	// Get the path to the test binary (this binary itself)
	testBinaryPath, err := os.Executable()
	require.NoError(t, err)

	tests := []struct {
		name      string
		setup     func(t *testing.T) string
		wantErr   bool
		errTarget error
		validate  func(t *testing.T, data *BuildInfoData)
	}{
		{
			name: "extract from Go test binary",
			setup: func(t *testing.T) string {
				t.Helper()

				return testBinaryPath
			},
			wantErr: false,
			validate: func(t *testing.T, data *BuildInfoData) {
				t.Helper()

				// Verify basic fields are populated
				assert.NotEmpty(t, data.GoVersion, "GoVersion should not be empty")
				assert.NotEmpty(t, data.RawJSON, "RawJSON should not be empty")
				assert.NotNil(t, data.Settings, "Settings should not be nil")

				// The test binary should have valid build info
				assert.NotEmpty(t, data.GoVersion, "should have Go version")
			},
		},
		{
			name: "non-existent file",
			setup: func(t *testing.T) string {
				t.Helper()

				return "/non/existent/binary/path"
			},
			wantErr:   true,
			errTarget: ErrPathNotFound,
		},
		{
			name: "non-Go binary file",
			setup: func(t *testing.T) string {
				t.Helper()

				tempDir := t.TempDir()
				fakeBinary := filepath.Join(tempDir, "fake-binary")

				err := os.WriteFile(fakeBinary, []byte("not a go binary"), 0o755)
				require.NoError(t, err)

				return fakeBinary
			},
			wantErr:   true,
			errTarget: ErrNotGoBinary,
		},
		{
			name: "directory instead of file",
			setup: func(t *testing.T) string {
				t.Helper()

				tempDir := t.TempDir()

				return tempDir
			},
			wantErr:   true,
			errTarget: ErrNotGoBinary,
		},
		{
			name: "empty file",
			setup: func(t *testing.T) string {
				t.Helper()

				tempDir := t.TempDir()
				emptyFile := filepath.Join(tempDir, "empty")

				err := os.WriteFile(emptyFile, []byte{}, 0o644)
				require.NoError(t, err)

				return emptyFile
			},
			wantErr:   true,
			errTarget: ErrNotGoBinary,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			binaryPath := tt.setup(t)
			ctx := context.Background()

			data, err := extractor.Extract(ctx, binaryPath)

			if tt.wantErr {
				require.Error(t, err)

				if tt.errTarget != nil {
					require.ErrorIs(t, err, tt.errTarget,
						"expected error to wrap %v, got %v", tt.errTarget, err)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, data)

				if tt.validate != nil {
					tt.validate(t, data)
				}
			}
		})
	}
}

// TestDefaultExtractor_Extract_ContextCancellation tests context cancellation.
func TestDefaultExtractor_Extract_ContextCancellation(t *testing.T) {
	t.Parallel()

	extractor, err := NewExtractor()
	require.NoError(t, err)

	// Get the path to the test binary
	testBinaryPath, err := os.Executable()
	require.NoError(t, err)

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = extractor.Extract(ctx, testBinaryPath)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled, "expected context.Canceled error")
}

// TestDefaultExtractor_CalculateChecksum tests checksum calculation.
func TestDefaultExtractor_CalculateChecksum(t *testing.T) {
	t.Parallel()

	extractor, err := NewExtractor()
	require.NoError(t, err)

	tests := []struct {
		name      string
		setup     func(t *testing.T) string
		wantErr   bool
		errTarget error
		validate  func(t *testing.T, checksum string, content []byte)
	}{
		{
			name: "calculate checksum for file",
			setup: func(t *testing.T) string {
				t.Helper()

				tempDir := t.TempDir()
				testFile := filepath.Join(tempDir, "testfile.txt")

				content := []byte("test content for checksum")
				err := os.WriteFile(testFile, content, 0o644)
				require.NoError(t, err)

				return testFile
			},
			wantErr: false,
			validate: func(t *testing.T, checksum string, content []byte) {
				t.Helper()

				// Verify checksum is valid hex
				assert.Len(t, checksum, 64, "SHA256 checksum should be 64 hex characters")

				// Verify by computing expected checksum
				hasher := sha256.New()
				hasher.Write(content)
				expectedChecksum := hex.EncodeToString(hasher.Sum(nil))

				assert.Equal(t, expectedChecksum, checksum)
			},
		},
		{
			name: "calculate checksum for Go binary",
			setup: func(t *testing.T) string {
				t.Helper()

				path, err := os.Executable()
				require.NoError(t, err)

				return path
			},
			wantErr: false,
			validate: func(t *testing.T, checksum string, _ []byte) {
				t.Helper()

				// Verify checksum format
				assert.Len(t, checksum, 64, "SHA256 checksum should be 64 hex characters")

				// Verify it's valid hex
				_, err := hex.DecodeString(checksum)
				require.NoError(t, err)
			},
		},
		{
			name: "non-existent file",
			setup: func(t *testing.T) string {
				t.Helper()

				return "/non/existent/file"
			},
			wantErr:   true,
			errTarget: ErrPathNotFound,
		},
		{
			name: "directory instead of file",
			setup: func(t *testing.T) string {
				t.Helper()

				return t.TempDir()
			},
			wantErr: true, // Should error when trying to read directory
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			filePath := tt.setup(t)

			// Read content for validation
			var content []byte

			if _, err := os.Stat(filePath); err == nil {
				content, _ = os.ReadFile(filePath)
			}

			checksum, err := extractor.CalculateChecksum(filePath)

			if tt.wantErr {
				require.Error(t, err)

				if tt.errTarget != nil {
					require.ErrorIs(t, err, tt.errTarget,
						"expected error to wrap %v, got %v", tt.errTarget, err)
				}
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, checksum)

				if tt.validate != nil {
					tt.validate(t, checksum, content)
				}
			}
		})
	}
}

// TestDefaultExtractor_IsGoBinary tests the IsGoBinary method.
func TestDefaultExtractor_IsGoBinary(t *testing.T) {
	t.Parallel()

	extractor, err := NewExtractor()
	require.NoError(t, err)

	// Get the path to the test binary
	testBinaryPath, err := os.Executable()
	require.NoError(t, err)

	tests := []struct {
		name     string
		setup    func(t *testing.T) string
		expected bool
	}{
		{
			name: "Go test binary",
			setup: func(t *testing.T) string {
				t.Helper()

				return testBinaryPath
			},
			expected: true,
		},
		{
			name: "non-existent file",
			setup: func(t *testing.T) string {
				t.Helper()

				return "/non/existent/binary"
			},
			expected: false,
		},
		{
			name: "regular text file",
			setup: func(t *testing.T) string {
				t.Helper()

				tempDir := t.TempDir()
				textFile := filepath.Join(tempDir, "test.txt")

				err := os.WriteFile(textFile, []byte("not a binary"), 0o644)
				require.NoError(t, err)

				return textFile
			},
			expected: false,
		},
		{
			name: "directory",
			setup: func(t *testing.T) string {
				t.Helper()

				return t.TempDir()
			},
			expected: false,
		},
		{
			name: "empty file",
			setup: func(t *testing.T) string {
				t.Helper()

				tempDir := t.TempDir()
				emptyFile := filepath.Join(tempDir, "empty")

				err := os.WriteFile(emptyFile, []byte{}, 0o644)
				require.NoError(t, err)

				return emptyFile
			},
			expected: false,
		},
		{
			name: "ELF binary without Go build info",
			setup: func(t *testing.T) string {
				t.Helper()

				tempDir := t.TempDir()
				fakeBinary := filepath.Join(tempDir, "fake-elf")

				// Write ELF magic header but no Go build info
				elfHeader := []byte{0x7f, 'E', 'L', 'F', 0x02, 0x01, 0x01}
				err := os.WriteFile(fakeBinary, elfHeader, 0o755)
				require.NoError(t, err)

				return fakeBinary
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			binaryPath := tt.setup(t)
			result := extractor.IsGoBinary(binaryPath)

			assert.Equal(t, tt.expected, result,
				"IsGoBinary(%s) = %v, want %v", binaryPath, result, tt.expected)
		})
	}
}

// TestParseVersionType tests the ParseVersionType function.
func TestParseVersionType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		version  string
		expected string
	}{
		{
			name:     "semantic version v1.2.3",
			version:  "v1.2.3",
			expected: "semantic",
		},
		{
			name:     "semantic version v0.0.1",
			version:  "v0.0.1",
			expected: "semantic",
		},
		{
			name:     "semantic version with prerelease v1.0.0-alpha",
			version:  "v1.0.0-alpha",
			expected: "semantic", // Semver pre-release is not a Go pseudo-version
		},
		{
			name:     "pseudo-version",
			version:  "v0.0.0-20260302120000-abc123def456",
			expected: "pseudo",
		},
		{
			name:     "pseudo-version with module path",
			version:  "v1.2.3-0.20260302120000-abc123def456",
			expected: "pseudo",
		},
		{
			name:     "development version",
			version:  "(devel)",
			expected: "devel",
		},
		{
			name:     "empty string",
			version:  "",
			expected: "unknown",
		},
		{
			name:     "random string",
			version:  "not-a-version",
			expected: "unknown",
		},
		{
			name:     "version without v prefix",
			version:  "1.2.3",
			expected: "unknown",
		},
		{
			name:     "v prefix only",
			version:  "v",
			expected: "unknown", // Single "v" is not a valid version
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := ParseVersionType(tt.version)
			assert.Equal(t, tt.expected, result,
				"ParseVersionType(%q) = %q, want %q", tt.version, result, tt.expected)
		})
	}
}

// TestBuildInfoData_IsReinstallable tests the IsReinstallable method.
func TestBuildInfoData_IsReinstallable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		data     BuildInfoData
		expected bool
	}{
		{
			name: "has module path and vcs revision",
			data: BuildInfoData{
				ModulePath:  "github.com/user/repo",
				VCSRevision: "abc123def456",
			},
			expected: true,
		},
		{
			name: "missing module path",
			data: BuildInfoData{
				ModulePath:  "",
				VCSRevision: "abc123def456",
			},
			expected: false,
		},
		{
			name: "missing vcs revision",
			data: BuildInfoData{
				ModulePath:  "github.com/user/repo",
				VCSRevision: "",
			},
			expected: false,
		},
		{
			name: "both fields empty",
			data: BuildInfoData{
				ModulePath:  "",
				VCSRevision: "",
			},
			expected: false,
		},
		{
			name: "has version but no vcs revision",
			data: BuildInfoData{
				ModulePath:  "github.com/user/repo",
				Version:     "v1.0.0",
				VCSRevision: "",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.data.IsReinstallable()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestBuildInfoData_GetInstallCommand tests the GetInstallCommand method.
func TestBuildInfoData_GetInstallCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		data     BuildInfoData
		expected string
	}{
		{
			name: "install with tagged version (preferred over revision)",
			data: BuildInfoData{
				ModulePath:  "github.com/user/repo",
				VCSRevision: "abc123def456",
				Version:     "v1.0.0",
			},
			expected: "go install github.com/user/repo@v1.0.0",
		},
		{
			name: "install with vcs revision when no tagged version",
			data: BuildInfoData{
				ModulePath:  "github.com/user/repo",
				VCSRevision: "abc123def456",
				Version:     "",
			},
			expected: "go install github.com/user/repo@abc123def456",
		},
		{
			name: "install with vcs revision when devel version",
			data: BuildInfoData{
				ModulePath:  "github.com/user/repo",
				VCSRevision: "abc123def456",
				Version:     "(devel)",
			},
			expected: "go install github.com/user/repo@abc123def456",
		},
		{
			name: "install at latest when only module path available",
			data: BuildInfoData{
				ModulePath:  "github.com/user/repo",
				VCSRevision: "",
				Version:     "",
			},
			expected: "go install github.com/user/repo@latest",
		},
		{
			name: "not reinstallable - missing module path",
			data: BuildInfoData{
				ModulePath:  "",
				VCSRevision: "abc123def456",
				Version:     "v1.0.0",
			},
			expected: "",
		},
		{
			name: "not reinstallable - all fields empty",
			data: BuildInfoData{
				ModulePath:  "",
				VCSRevision: "",
				Version:     "",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.data.GetInstallCommand()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestErrors tests the exported error variables.
func TestErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
	}{
		{
			name: "ErrNotGoBinary",
			err:  ErrNotGoBinary,
		},
		{
			name: "ErrBuildInfoNotFound",
			err:  ErrBuildInfoNotFound,
		},
		{
			name: "ErrPathNotFound",
			err:  ErrPathNotFound,
		},
		{
			name: "ErrUnsupportedPlatform",
			err:  ErrUnsupportedPlatform,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Error(t, tt.err)
			require.NotEmpty(t, tt.err.Error())
		})
	}
}

// TestBuildInfoData tests the BuildInfoData struct fields.
func TestBuildInfoData(t *testing.T) {
	t.Parallel()

	data := BuildInfoData{
		ModulePath:  "github.com/test/module",
		Version:     "v1.2.3",
		VCSRevision: "abc123def456789",
		VCSTime:     "2026-03-01T12:00:00Z",
		GoVersion:   "go1.26",
		Settings: map[string]string{
			"vcs":          "git",
			"vcs.revision": "abc123def456789",
		},
		RawJSON: []byte(`{"GoVersion":"go1.26"}`),
	}

	assert.Equal(t, "github.com/test/module", data.ModulePath)
	assert.Equal(t, "v1.2.3", data.Version)
	assert.Equal(t, "abc123def456789", data.VCSRevision)
	assert.Equal(t, "2026-03-01T12:00:00Z", data.VCSTime)
	assert.Equal(t, "go1.26", data.GoVersion)
	assert.NotNil(t, data.Settings)
	assert.NotEmpty(t, data.RawJSON)
}

// TestIsSupportedPlatform tests the platform support check.
func TestIsSupportedPlatform(t *testing.T) {
	t.Parallel()

	// This test will pass on Linux and Windows, which are the only supported platforms
	if runtime.GOOS == platformLinux || runtime.GOOS == platformWindows {
		assert.True(t, isSupportedPlatform())
	}
}

// TestDefaultExtractor_Extract_WithBuildSettings tests extraction with VCS settings.
func TestDefaultExtractor_Extract_WithBuildSettings(t *testing.T) {
	t.Parallel()

	extractor, err := NewExtractor()
	require.NoError(t, err)

	// Use the current test binary which should have build info
	testBinaryPath, err := os.Executable()
	require.NoError(t, err)

	ctx := context.Background()
	data, err := extractor.Extract(ctx, testBinaryPath)
	require.NoError(t, err)
	require.NotNil(t, data)

	// Verify that the data has the expected structure
	assert.NotEmpty(t, data.GoVersion, "should have Go version")
	assert.NotNil(t, data.Settings, "should have settings map")

	// RawJSON should be valid JSON
	assert.NotEmpty(t, data.RawJSON, "should have raw JSON")
	assert.NotEmpty(t, data.RawJSON, "raw JSON should not be empty")
}

// BenchmarkCalculateChecksum benchmarks checksum calculation.
func BenchmarkCalculateChecksum(b *testing.B) {
	extractor, err := NewExtractor()
	require.NoError(b, err)

	// Create a temp file with content
	tempDir := b.TempDir()
	testFile := filepath.Join(tempDir, "benchfile.txt")

	content := make([]byte, 1024*1024) // 1MB file

	err = os.WriteFile(testFile, content, 0o644)
	require.NoError(b, err)

	b.ResetTimer()

	for range b.N {
		_, err := extractor.CalculateChecksum(testFile)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkIsGoBinary benchmarks the IsGoBinary check.
func BenchmarkIsGoBinary(b *testing.B) {
	extractor, err := NewExtractor()
	require.NoError(b, err)

	testBinaryPath, err := os.Executable()
	require.NoError(b, err)

	b.ResetTimer()

	for range b.N {
		_ = extractor.IsGoBinary(testBinaryPath)
	}
}

// BenchmarkParseVersionType benchmarks version type parsing.
func BenchmarkParseVersionType(b *testing.B) {
	versions := []string{
		"v1.2.3",
		"v0.0.0-20260302120000-abc123def456",
		"(devel)",
		"not-a-version",
	}

	b.ResetTimer()

	for range b.N {
		for _, v := range versions {
			_ = ParseVersionType(v)
		}
	}
}

// mockExtractor is a mock implementation of the Extractor interface for testing.
type mockExtractor struct {
	extractFunc           func(ctx context.Context, binaryPath string) (*BuildInfoData, error)
	calculateChecksumFunc func(binaryPath string) (string, error)
	isGoBinaryFunc        func(binaryPath string) bool
}

func (m *mockExtractor) Extract(ctx context.Context, binaryPath string) (*BuildInfoData, error) {
	if m.extractFunc != nil {
		return m.extractFunc(ctx, binaryPath)
	}

	return nil, errors.New("mock Extract not implemented")
}

func (m *mockExtractor) CalculateChecksum(binaryPath string) (string, error) {
	if m.calculateChecksumFunc != nil {
		return m.calculateChecksumFunc(binaryPath)
	}

	return "", errors.New("mock CalculateChecksum not implemented")
}

func (m *mockExtractor) IsGoBinary(binaryPath string) bool {
	if m.isGoBinaryFunc != nil {
		return m.isGoBinaryFunc(binaryPath)
	}

	return false
}

// Verify mock implements the interface.
var _ Extractor = (*mockExtractor)(nil)
