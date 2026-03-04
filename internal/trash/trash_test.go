/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package trash

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// platformLinux is the GOOS value for Linux systems.
const platformLinux = "linux"

// TestNewTrasher tests the creation of a new Trasher instance.
func TestNewTrasher(t *testing.T) {
	t.Parallel()

	trasher, err := NewTrasher()
	require.NoError(t, err)
	require.NotNil(t, trasher)

	// Verify we can get trash path
	trashPath := trasher.GetTrashPath()

	if runtime.GOOS == platformLinux {
		assert.NotEmpty(t, trashPath)
	}
}

// TestMoveToTrash tests moving files to trash.
func TestMoveToTrash(t *testing.T) {
	t.Parallel()

	if runtime.GOOS != platformLinux {
		t.Skip("Skipping Linux-specific test on non-Linux platform")
	}

	tests := []struct {
		name      string
		setup     func(t *testing.T) string
		wantErr   bool
		errTarget error
	}{
		{
			name: "successful file move to trash",
			setup: func(t *testing.T) string {
				t.Helper()

				tempDir := t.TempDir()
				testFile := filepath.Join(tempDir, "testfile.txt")

				err := os.WriteFile(testFile, []byte("test content"), 0o644)
				require.NoError(t, err)

				return testFile
			},
			wantErr: false,
		},
		{
			name: "successful directory move to trash",
			setup: func(t *testing.T) string {
				t.Helper()

				tempDir := t.TempDir()
				testDir := filepath.Join(tempDir, "testdir")

				err := os.Mkdir(testDir, 0o755)
				require.NoError(t, err)

				err = os.WriteFile(filepath.Join(testDir, "file.txt"), []byte("content"), 0o644)
				require.NoError(t, err)

				return testDir
			},
			wantErr: false,
		},
		{
			name: "non-existent file",
			setup: func(t *testing.T) string {
				t.Helper()

				return "/non/existent/file/path.txt"
			},
			wantErr:   true,
			errTarget: ErrPathNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			trasher, err := NewTrasher()
			require.NoError(t, err)

			filePath := tt.setup(t)
			ctx := context.Background()

			trashPath, err := trasher.MoveToTrash(ctx, filePath)

			if tt.wantErr {
				require.Error(t, err)

				if tt.errTarget != nil {
					require.ErrorIs(t, err, tt.errTarget,
						"expected error to be or wrap %v, got %v", tt.errTarget, err)
				}
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, trashPath)
				assert.True(t, trasher.IsInTrash(trashPath),
					"file should be in trash after MoveToTrash")

				// Verify original file no longer exists
				_, err = os.Stat(filePath)
				assert.True(t, os.IsNotExist(err),
					"original file should not exist after moving to trash")
			}
		})
	}
}

// TestMoveToTrash_ContextCancellation tests context cancellation.
func TestMoveToTrash_ContextCancellation(t *testing.T) {
	if runtime.GOOS != platformLinux {
		t.Skip("Skipping Linux-specific test on non-Linux platform")
	}

	t.Parallel()

	trasher, err := NewTrasher()
	require.NoError(t, err)

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "testfile.txt")

	err = os.WriteFile(testFile, []byte("test content"), 0o644)
	require.NoError(t, err)

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = trasher.MoveToTrash(ctx, testFile)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled, "expected context.Canceled error")
}

// TestRestoreFromTrash tests restoring files from trash.
func TestRestoreFromTrash(t *testing.T) {
	t.Parallel()

	if runtime.GOOS != platformLinux {
		t.Skip("Skipping Linux-specific test on non-Linux platform")
	}

	tests := []struct {
		name    string
		setup   func(t *testing.T, trasher Trasher) (trashPath, originalPath string, cleanup func())
		wantErr bool
	}{
		{
			name: "successful restore",
			setup: func(t *testing.T, trasher Trasher) (string, string, func()) {
				t.Helper()

				tempDir := t.TempDir()
				originalPath := filepath.Join(tempDir, "original", "testfile.txt")

				err := os.MkdirAll(filepath.Dir(originalPath), 0o755)
				require.NoError(t, err)

				err = os.WriteFile(originalPath, []byte("test content"), 0o644)
				require.NoError(t, err)

				ctx := context.Background()
				trashPath, err := trasher.MoveToTrash(ctx, originalPath)
				require.NoError(t, err)

				return trashPath, originalPath, func() {}
			},
			wantErr: false,
		},
		{
			name: "restore collision",
			setup: func(t *testing.T, trasher Trasher) (string, string, func()) {
				t.Helper()

				tempDir := t.TempDir()
				originalPath := filepath.Join(tempDir, "original", "testfile.txt")

				err := os.MkdirAll(filepath.Dir(originalPath), 0o755)
				require.NoError(t, err)

				err = os.WriteFile(originalPath, []byte("test content"), 0o644)
				require.NoError(t, err)

				ctx := context.Background()
				trashPath, err := trasher.MoveToTrash(ctx, originalPath)
				require.NoError(t, err)

				// Create file at original location to cause collision
				err = os.WriteFile(originalPath, []byte("blocking content"), 0o644)
				require.NoError(t, err)

				return trashPath, originalPath, func() {}
			},
			wantErr: true,
		},
		{
			name: "file not in trash",
			setup: func(t *testing.T, trasher Trasher) (string, string, func()) {
				t.Helper()

				tempDir := t.TempDir()
				fakeTrashPath := filepath.Join(tempDir, "not_in_trash.txt")
				originalPath := filepath.Join(tempDir, "original.txt")

				err := os.WriteFile(fakeTrashPath, []byte("content"), 0o644)
				require.NoError(t, err)

				return fakeTrashPath, originalPath, func() {}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			trasher, err := NewTrasher()
			require.NoError(t, err)

			trashPath, originalPath, cleanup := tt.setup(t, trasher)
			defer cleanup()

			ctx := context.Background()
			err = trasher.RestoreFromTrash(ctx, trashPath, originalPath)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				// Verify file exists at original location
				_, err = os.Stat(originalPath)
				require.NoError(t, err)

				// Verify file no longer in trash
				assert.False(t, trasher.IsInTrash(trashPath))
			}
		})
	}
}

// TestIsInTrash tests the IsInTrash method.
func TestIsInTrash(t *testing.T) {
	if runtime.GOOS != platformLinux {
		t.Skip("Skipping Linux-specific test on non-Linux platform")
	}

	t.Parallel()

	trasher, err := NewTrasher()
	require.NoError(t, err)

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "testfile.txt")

	err = os.WriteFile(testFile, []byte("test content"), 0o644)
	require.NoError(t, err)

	// File should not be in trash initially
	assert.False(t, trasher.IsInTrash(testFile))

	// Move to trash
	ctx := context.Background()
	trashPath, err := trasher.MoveToTrash(ctx, testFile)
	require.NoError(t, err)

	// File should now be in trash
	assert.True(t, trasher.IsInTrash(trashPath))

	// Non-existent path should return false
	assert.False(t, trasher.IsInTrash("/non/existent/path"))
}

// TestDeletePermanently tests permanent deletion from trash.
func TestDeletePermanently(t *testing.T) {
	if runtime.GOOS != platformLinux {
		t.Skip("Skipping Linux-specific test on non-Linux platform")
	}

	t.Parallel()

	trasher, err := NewTrasher()
	require.NoError(t, err)

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "testfile.txt")

	err = os.WriteFile(testFile, []byte("test content"), 0o644)
	require.NoError(t, err)

	// Move to trash
	ctx := context.Background()
	trashPath, err := trasher.MoveToTrash(ctx, testFile)
	require.NoError(t, err)

	// Verify file is in trash
	assert.True(t, trasher.IsInTrash(trashPath))

	// Delete permanently
	err = trasher.DeletePermanently(ctx, trashPath)
	require.NoError(t, err)

	// Verify file no longer exists
	assert.False(t, trasher.IsInTrash(trashPath))
	_, err = os.Stat(trashPath)
	assert.True(t, os.IsNotExist(err))
}

// TestDeletePermanently_NotInTrash tests deleting a file not in trash.
func TestDeletePermanently_NotInTrash(t *testing.T) {
	if runtime.GOOS != platformLinux {
		t.Skip("Skipping Linux-specific test on non-Linux platform")
	}

	t.Parallel()

	trasher, err := NewTrasher()
	require.NoError(t, err)

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "testfile.txt")

	err = os.WriteFile(testFile, []byte("test content"), 0o644)
	require.NoError(t, err)

	ctx := context.Background()
	err = trasher.DeletePermanently(ctx, testFile)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrFileNotInTrash)
}

// TestListTrash tests listing trash entries.
func TestListTrash(t *testing.T) {
	if runtime.GOOS != platformLinux {
		t.Skip("Skipping Linux-specific test on non-Linux platform")
	}

	t.Parallel()

	trasher, err := NewTrasher()
	require.NoError(t, err)

	// Create and trash multiple files
	tempDir := t.TempDir()
	fileNames := []string{"file1.txt", "file2.txt", "file3.txt"}
	originalPaths := make(map[string]string)

	ctx := context.Background()

	for _, name := range fileNames {
		filePath := filepath.Join(tempDir, name)

		err := os.WriteFile(filePath, []byte("content"), 0o644)
		require.NoError(t, err)

		absPath, _ := filepath.Abs(filePath)
		originalPaths[name] = absPath

		_, err = trasher.MoveToTrash(ctx, filePath)
		require.NoError(t, err)
	}

	// List trash
	entries, err := trasher.ListTrash()
	require.NoError(t, err)

	// Our trashed files should be in the list
	foundCount := 0

	for _, entry := range entries {
		for _, name := range fileNames {
			if strings.Contains(entry.Name, name) {
				foundCount++

				assert.NotEmpty(t, entry.TrashPath)
				assert.False(t, entry.DeletionTime.IsZero())

				break
			}
		}
	}

	assert.GreaterOrEqual(t, foundCount, len(fileNames),
		"should find all trashed files")
}

// TestGetTrashPath tests getting the trash path.
func TestGetTrashPath(t *testing.T) {
	if runtime.GOOS != platformLinux {
		t.Skip("Skipping Linux-specific test on non-Linux platform")
	}

	t.Parallel()

	trasher, err := NewTrasher()
	require.NoError(t, err)

	trashPath := trasher.GetTrashPath()
	assert.NotEmpty(t, trashPath)
	assert.Contains(t, trashPath, "Trash")
	assert.Contains(t, trashPath, "files")
}

// TestEncodeDecodeTrashPath tests path encoding/decoding.
func TestEncodeDecodeTrashPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "simple path",
			path:     "/home/user/file.txt",
			expected: "/home/user/file.txt",
		},
		{
			name:     "path with spaces",
			path:     "/home/user/my file.txt",
			expected: "/home/user/my file.txt",
		},
		{
			name:     "path with percent sign",
			path:     "/home/user/file%20name.txt",
			expected: "/home/user/file%2520name.txt",
		},
		{
			name:     "path with control characters",
			path:     "/home/user/file\x01\x02.txt",
			expected: "/home/user/file%01%02.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			encoded := encodeTrashPath(tt.path)
			assert.Equal(t, tt.expected, encoded)

			decoded, err := decodeTrashPath(encoded)
			require.NoError(t, err)
			assert.Equal(t, tt.path, decoded)
		})
	}
}

// TestDecodeTrashPath_Invalid tests decoding with invalid percent encoding.
func TestDecodeTrashPath_Invalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		encoded     string
		expected    string
		expectError bool
	}{
		{
			name:        "trailing percent",
			encoded:     "/home/user/file%",
			expectError: true,
		},
		{
			name:        "single char after percent",
			encoded:     "/home/user/file%A",
			expectError: true,
		},
		{
			name:        "invalid hex characters",
			encoded:     "/home/user/file%GG",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			decoded, err := decodeTrashPath(tt.encoded)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, decoded)
			}
		})
	}
}

// TestGenerateTrashInfo tests trashinfo generation.
func TestGenerateTrashInfo(t *testing.T) {
	t.Parallel()

	path := "/home/user/test.txt"
	deletionTime := time.Date(2026, 3, 2, 12, 0, 0, 0, time.Local)

	info := generateTrashInfo(path, deletionTime)

	assert.Contains(t, info, "[Trash Info]")
	assert.Contains(t, info, "Path=/home/user/test.txt")
	// Check format is ISO8601 without timezone (e.g., 2026-03-02T12:00:00)
	// The format should be: DeletionDate=YYYY-MM-DDTHH:MM:SS (potentially followed by newline at end)
	assert.Regexp(t, `DeletionDate=\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`, info)
}

// TestParseTrashInfo tests trashinfo parsing.
func TestParseTrashInfo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		content  string
		wantPath string
		wantTime time.Time
		wantErr  bool
	}{
		{
			name:     "valid RFC3339",
			content:  "[Trash Info]\nPath=/home/user/test.txt\nDeletionDate=2026-03-02T12:00:00Z\n",
			wantPath: "/home/user/test.txt",
			wantTime: time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC),
			wantErr:  false,
		},
		{
			name:     "valid simple format",
			content:  "[Trash Info]\nPath=/home/user/test.txt\nDeletionDate=2026-03-02T12:00:00\n",
			wantPath: "/home/user/test.txt",
			wantTime: time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC),
			wantErr:  false,
		},
		{
			name:     "missing path",
			content:  "[Trash Info]\nDeletionDate=2026-03-02T12:00:00\n",
			wantPath: "",
			wantErr:  true,
		},
		{
			name:     "encoded path",
			content:  "[Trash Info]\nPath=/home/user/file%20name.txt\nDeletionDate=2026-03-02T12:00:00Z\n",
			wantPath: "/home/user/file name.txt",
			wantTime: time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC),
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			path, deletionTime, err := parseTrashInfo(tt.content)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantPath, path)

				if !tt.wantTime.IsZero() {
					assert.Equal(t, tt.wantTime, deletionTime)
				}
			}
		})
	}
}

// TestSplitLines tests the splitLines helper.
func TestSplitLines(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Unix line endings",
			input:    "line1\nline2\nline3",
			expected: []string{"line1", "line2", "line3"},
		},
		{
			name:     "Windows line endings",
			input:    "line1\r\nline2\r\nline3",
			expected: []string{"line1", "line2", "line3"},
		},
		{
			name:     "mixed line endings",
			input:    "line1\nline2\r\nline3",
			expected: []string{"line1", "line2", "line3"},
		},
		{
			name:     "trailing newline",
			input:    "line1\nline2\n",
			expected: []string{"line1", "line2"},
		},
		{
			name:     "single line",
			input:    "line1",
			expected: []string{"line1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := splitLines(tt.input)
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
			name: "ErrTrashFull",
			err:  ErrTrashFull,
		},
		{
			name: "ErrFileNotInTrash",
			err:  ErrFileNotInTrash,
		},
		{
			name: "ErrRestoreCollision",
			err:  ErrRestoreCollision,
		},
		{
			name: "ErrInvalidPath",
			err:  ErrInvalidPath,
		},
		{
			name: "ErrPathNotFound",
			err:  ErrPathNotFound,
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

// TestTrashEntry tests the TrashEntry struct.
func TestTrashEntry(t *testing.T) {
	t.Parallel()

	entry := TrashEntry{
		Name:         "testfile.txt",
		OriginalPath: "/home/user/testfile.txt",
		TrashPath:    "/home/user/.local/share/Trash/files/testfile.txt",
		DeletionTime: time.Now(),
	}

	assert.Equal(t, "testfile.txt", entry.Name)
	assert.Equal(t, "/home/user/testfile.txt", entry.OriginalPath)
	assert.Equal(t, "/home/user/.local/share/Trash/files/testfile.txt", entry.TrashPath)
	assert.False(t, entry.DeletionTime.IsZero())
}

// TestGenerateUniqueName tests unique name generation.
func TestGenerateUniqueName(t *testing.T) {
	t.Parallel()

	name1 := generateUniqueName("test.txt")
	name2 := generateUniqueName("test.txt")

	// Names should contain the base name
	assert.Contains(t, name1, "test.txt")
	assert.Contains(t, name2, "test.txt")

	// Names should contain timestamp (underscore and digits)
	assert.Contains(t, name1, "_")
	assert.Regexp(t, `test\.txt_\d+`, name1)

	// Names may or may not be different depending on timing
	// We just verify the format is correct
	assert.NotEmpty(t, name1)
	assert.NotEmpty(t, name2)
}

// BenchmarkMoveToTrash benchmarks moving files to trash.
func BenchmarkMoveToTrash(b *testing.B) {
	if runtime.GOOS != platformLinux {
		b.Skip("Skipping Linux-specific benchmark on non-Linux platform")
	}

	trasher, err := NewTrasher()
	require.NoError(b, err)

	tempDir := b.TempDir()
	ctx := context.Background()

	b.ResetTimer()

	for i := range b.N {
		b.StopTimer()

		filePath := filepath.Join(tempDir, fmt.Sprintf("benchfile_%d.txt", i))

		err := os.WriteFile(filePath, []byte("benchmark content"), 0o644)
		require.NoError(b, err)

		b.StartTimer()

		_, err = trasher.MoveToTrash(ctx, filePath)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEncodeTrashPath benchmarks path encoding.
func BenchmarkEncodeTrashPath(b *testing.B) {
	path := "/home/user/my file with spaces and % signs.txt"

	b.ResetTimer()

	for range b.N {
		_ = encodeTrashPath(path)
	}
}

// BenchmarkParseTrashInfo benchmarks trashinfo parsing.
func BenchmarkParseTrashInfo(b *testing.B) {
	content := "[Trash Info]\nPath=/home/user/test.txt\nDeletionDate=2026-03-02T12:00:00Z\n"

	b.ResetTimer()

	for range b.N {
		_, _, _ = parseTrashInfo(content)
	}
}
