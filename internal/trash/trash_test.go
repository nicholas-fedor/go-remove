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
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// platformLinux is the GOOS value for Linux systems.
const platformLinux = "linux"

// platformWindows is the GOOS value for Windows systems.
const platformWindows = "windows"

// newTestTrasher returns a Trasher rooted in a per-test temporary directory.
//
// Tests must never share the user's real trash. t.Setenv is not an option here
// because it is incompatible with t.Parallel, so the root is injected instead.
//
// Parameters:
//   - t: The test that owns the temporary directory.
//
// Returns:
//   - A Trasher backed by a temporary trash root.
func newTestTrasher(t *testing.T) Trasher {
	t.Helper()

	trasher, err := newTrasherAt(t.TempDir())
	require.NoError(t, err)

	return trasher
}

// newBenchTrasher returns a Trasher rooted in a per-benchmark temporary directory.
//
// Parameters:
//   - b: The benchmark that owns the temporary directory.
//
// Returns:
//   - A Trasher backed by a temporary trash root.
func newBenchTrasher(b *testing.B) Trasher {
	b.Helper()

	trasher, err := newTrasherAt(b.TempDir())
	require.NoError(b, err)

	return trasher
}

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

			trasher := newTestTrasher(t)

			filePath := tt.setup(t)
			ctx := t.Context()

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

	trasher := newTestTrasher(t)

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "testfile.txt")

	err := os.WriteFile(testFile, []byte("test content"), 0o644)
	require.NoError(t, err)

	// Create cancelled context
	ctx, cancel := context.WithCancel(t.Context())
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

				ctx := t.Context()
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

				ctx := t.Context()
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

			trasher := newTestTrasher(t)

			trashPath, originalPath, cleanup := tt.setup(t, trasher)
			defer cleanup()

			ctx := t.Context()
			err := trasher.RestoreFromTrash(ctx, trashPath, originalPath)

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

	trasher := newTestTrasher(t)

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "testfile.txt")

	err := os.WriteFile(testFile, []byte("test content"), 0o644)
	require.NoError(t, err)

	// File should not be in trash initially
	assert.False(t, trasher.IsInTrash(testFile))

	// Move to trash
	ctx := t.Context()
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

	trasher := newTestTrasher(t)

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "testfile.txt")

	err := os.WriteFile(testFile, []byte("test content"), 0o644)
	require.NoError(t, err)

	// Move to trash
	ctx := t.Context()
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

	trasher := newTestTrasher(t)

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "testfile.txt")

	err := os.WriteFile(testFile, []byte("test content"), 0o644)
	require.NoError(t, err)

	ctx := t.Context()
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

	trasher := newTestTrasher(t)

	// Create and trash multiple files
	tempDir := t.TempDir()
	fileNames := []string{"file1.txt", "file2.txt", "file3.txt"}

	trashedPaths := make(map[string]string, len(fileNames))
	ctx := t.Context()

	for _, name := range fileNames {
		filePath := filepath.Join(tempDir, name)

		err := os.WriteFile(filePath, []byte("content"), 0o644)
		require.NoError(t, err)

		trashPath, err := trasher.MoveToTrash(ctx, filePath)
		require.NoError(t, err)

		trashedPaths[name] = trashPath
	}

	// List trash
	entries, err := trasher.ListTrash()
	require.NoError(t, err)

	listedPaths := make(map[string]TrashEntry, len(entries))
	for _, entry := range entries {
		assert.NotEmpty(t, entry.TrashPath)
		assert.False(t, entry.DeletionTime.IsZero())

		listedPaths[entry.TrashPath] = entry
	}

	// Match on the exact path each move returned. Substring matching against the
	// name would also be satisfied by unrelated entries in a shared trash.
	for name, trashPath := range trashedPaths {
		entry, ok := listedPaths[trashPath]
		require.Truef(t, ok, "%s (%s) missing from trash listing", name, trashPath)
		assert.Equal(t, filepath.Join(tempDir, name), entry.OriginalPath)
	}
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
			expected: "/home/user/my%20file.txt",
		},
		{
			name:     "path with non-ascii bytes",
			path:     "/tmp/\xe9",
			expected: "/tmp/%E9",
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
		// The query and fragment delimiters must be escaped. A desktop splits
		// the Path value on them, so leaving them bare makes a binary named
		// tool?v2 restore to .../tool and a name containing # truncate there.
		{
			name:     "path with question mark",
			path:     "/tmp/a?b",
			expected: "/tmp/a%3Fb",
		},
		{
			name:     "path with hash",
			path:     "/tmp/a#b",
			expected: "/tmp/a%23b",
		},
		{
			name:     "path with brackets",
			path:     "/tmp/a[b]",
			expected: "/tmp/a%5Bb%5D",
		},
		{
			name:     "path with semicolon",
			path:     "/tmp/a;b",
			expected: "/tmp/a%3Bb",
		},
		{
			name:     "path with reserved characters left safe",
			path:     "/tmp/a-b_c.d~e!f$g&h'i(j)k*l+m,n=o:p@q/r",
			expected: "/tmp/a-b_c.d~e!f$g&h'i(j)k*l+m,n=o:p@q/r",
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

// TestMoveToTrash_SameNameRepeatedly tests that repeated trashing of the same
// file name keeps every copy.
//
// The old suffix had one-second resolution, so trashing the same basename twice
// within a second produced the same candidate name. os.Rename silently
// overwrote, destroying the first trashed file while both calls reported
// success.
func TestMoveToTrash_SameNameRepeatedly(t *testing.T) {
	if runtime.GOOS != platformLinux {
		t.Skip("Skipping Linux-specific test on non-Linux platform")
	}

	t.Parallel()

	trasher := newTestTrasher(t)
	ctx := t.Context()
	sourceDir := t.TempDir()

	const copies = 25

	trashPaths := make([]string, 0, copies)

	for i := range copies {
		filePath := filepath.Join(sourceDir, "duplicate.txt")

		err := os.WriteFile(filePath, fmt.Appendf(nil, "copy %d", i), 0o600)
		require.NoError(t, err)

		trashPath, err := trasher.MoveToTrash(ctx, filePath)
		require.NoError(t, err)

		trashPaths = append(trashPaths, trashPath)
	}

	// Every copy must be present and hold its own content.
	require.Len(t, trashPaths, copies)

	seen := make(map[string]struct{}, copies)

	for i, trashPath := range trashPaths {
		_, duplicate := seen[trashPath]
		require.Falsef(t, duplicate, "trash path reused: %s", trashPath)

		seen[trashPath] = struct{}{}

		data, err := os.ReadFile(trashPath)
		require.NoErrorf(t, err, "trashed copy %d is missing", i)
		assert.Equal(t, fmt.Sprintf("copy %d", i), string(data))
	}
}

// TestMoveToTrash_ConcurrentSameName tests that concurrent trashing of the same
// basename reserves a distinct entry per caller.
func TestMoveToTrash_ConcurrentSameName(t *testing.T) {
	if runtime.GOOS != platformLinux {
		t.Skip("Skipping Linux-specific test on non-Linux platform")
	}

	t.Parallel()

	trasher := newTestTrasher(t)
	ctx := t.Context()
	sourceDir := t.TempDir()

	const callers = 8

	var (
		waitGroup sync.WaitGroup
		mu        sync.Mutex
		results   = make(map[string]string, callers)
		failures  []error
	)

	for i := range callers {
		// Each caller gets its own source file, all sharing one basename, so
		// the trash entries must not collide.
		callerDir := filepath.Join(sourceDir, fmt.Sprintf("caller%d", i))
		require.NoError(t, os.MkdirAll(callerDir, 0o750))

		waitGroup.Go(func() {
			filePath := filepath.Join(callerDir, "racer.txt")

			err := os.WriteFile(filePath, fmt.Appendf(nil, "racer %d", i), 0o600)
			if err != nil {
				mu.Lock()
				defer mu.Unlock()

				failures = append(failures, err)

				return
			}

			trashPath, moveErr := trasher.MoveToTrash(ctx, filePath)
			if moveErr != nil {
				mu.Lock()
				defer mu.Unlock()

				failures = append(failures, moveErr)

				return
			}

			mu.Lock()
			defer mu.Unlock()

			results[trashPath] = fmt.Sprintf("racer %d", i)
		})
	}

	waitGroup.Wait()

	require.Empty(t, failures, "concurrent trashing reported errors")
	require.Len(t, results, callers, "each caller must own a distinct entry")

	for trashPath, content := range results {
		data, err := os.ReadFile(trashPath)
		require.NoErrorf(t, err, "trashed entry %s is missing", trashPath)
		assert.Equal(t, content, string(data))
	}
}

// TestGenerateUniqueName tests unique name generation.
func TestGenerateUniqueName(t *testing.T) {
	t.Parallel()

	// The suffix must be random, not clock-derived. Two calls in the same
	// second previously produced the same candidate, which let a second trash
	// operation silently overwrite the first.
	const draws = 1000

	seen := make(map[string]struct{}, draws)

	for range draws {
		name := generateUniqueName("test.txt")

		assert.Contains(t, name, "test.txt")
		assert.Regexp(t, `^test\.txt_[0-9a-f]{12}$`, name)

		_, duplicate := seen[name]
		require.Falsef(t, duplicate, "generateUniqueName repeated %s", name)

		seen[name] = struct{}{}
	}
}

// BenchmarkMoveToTrash benchmarks moving files to trash.
func BenchmarkMoveToTrash(b *testing.B) {
	if runtime.GOOS != platformLinux {
		b.Skip("Skipping Linux-specific benchmark on non-Linux platform")
	}

	trasher := newBenchTrasher(b)

	tempDir := b.TempDir()
	ctx := b.Context()
	i := 0

	for b.Loop() {
		b.StopTimer()

		filePath := filepath.Join(tempDir, fmt.Sprintf("benchfile_%d.txt", i))
		i++

		err := os.WriteFile(filePath, []byte("benchmark content"), 0o644)
		require.NoError(b, err)

		b.StartTimer()

		if _, err := trasher.MoveToTrash(ctx, filePath); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEncodeTrashPath benchmarks path encoding.
func BenchmarkEncodeTrashPath(b *testing.B) {
	path := "/home/user/my file with spaces and % signs.txt"

	for b.Loop() {
		_ = encodeTrashPath(path)
	}
}

// BenchmarkParseTrashInfo benchmarks trashinfo parsing.
func BenchmarkParseTrashInfo(b *testing.B) {
	content := "[Trash Info]\nPath=/home/user/test.txt\nDeletionDate=2026-03-02T12:00:00Z\n"

	for b.Loop() {
		_, _, _ = parseTrashInfo(content)
	}
}
