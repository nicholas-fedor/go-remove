//go:build linux || darwin

/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package trash

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRestoreFromTrash_RejectsExistingFile verifies that an existing file at the
// restore location is never overwritten.
//
// The previous implementation checked for the destination with os.Stat and then
// renamed, so a file created inside that window was silently destroyed. The
// restore now claims the destination atomically.
func TestRestoreFromTrash_RejectsExistingFile(t *testing.T) {
	t.Parallel()

	trasher := newTestTrasher(t)
	ctx := t.Context()

	sourceDir := t.TempDir()
	source := filepath.Join(sourceDir, "payload")
	require.NoError(t, os.WriteFile(source, []byte("from trash"), 0o600))

	trashPath, err := trasher.MoveToTrash(ctx, source)
	require.NoError(t, err)

	// Occupy the restore target.
	occupied := filepath.Join(sourceDir, "restored")
	require.NoError(t, os.WriteFile(occupied, []byte("already here"), 0o600))

	err = trasher.RestoreFromTrash(ctx, trashPath, occupied)
	require.ErrorIs(t, err, ErrRestoreCollision)

	// The occupying file must be intact and the trashed copy must survive.
	data, readErr := os.ReadFile(occupied)
	require.NoError(t, readErr)
	assert.Equal(t, "already here", string(data))
	assert.True(t, trasher.IsInTrash(trashPath),
		"a rejected restore must not consume the trashed copy")
}

// TestRestoreFromTrash_RejectsExistingDirectory verifies that restoring a
// trashed directory onto an existing directory is refused rather than merged.
func TestRestoreFromTrash_RejectsExistingDirectory(t *testing.T) {
	t.Parallel()

	trasher := newTestTrasher(t)
	ctx := t.Context()

	sourceDir := t.TempDir()
	source := filepath.Join(sourceDir, "payload")
	require.NoError(t, os.MkdirAll(source, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(source, "inner"), []byte("inner"), 0o600))

	trashPath, err := trasher.MoveToTrash(ctx, source)
	require.NoError(t, err)

	// Occupy the restore target with a directory of its own.
	occupied := filepath.Join(sourceDir, "restored")
	require.NoError(t, os.MkdirAll(occupied, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(occupied, "existing"), []byte("existing"), 0o600))

	err = trasher.RestoreFromTrash(ctx, trashPath, occupied)
	require.ErrorIs(t, err, ErrRestoreCollision)

	// The occupying directory must be untouched, and the trashed one intact.
	assertDirContainsOnly(t, occupied, "existing")

	inner, readErr := os.ReadFile(filepath.Join(trashPath, "inner"))
	require.NoError(t, readErr, "the trashed directory must survive a refused restore")
	assert.Equal(t, "inner", string(inner))
}

// TestCopyAndDelete_RejectsExistingDirectory verifies the exclusive create in
// the copy path itself.
//
// Restore normally notices an occupied destination first, because os.Link
// reports EEXIST for an existing directory. That check is skipped when the
// entry is not a regular file or the two paths are on different devices, so
// the copy path has to refuse an existing destination on its own. MkdirAll
// would have succeeded and merged the two trees.
func TestCopyAndDelete_RejectsExistingDirectory(t *testing.T) {
	t.Parallel()

	trasher, ok := newTestTrasher(t).(*xdgTrasher)
	require.True(t, ok)

	source := filepath.Join(t.TempDir(), "src")
	require.NoError(t, os.MkdirAll(source, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(source, "inner"), []byte("inner"), 0o600))

	occupied := filepath.Join(t.TempDir(), "dst")
	require.NoError(t, os.MkdirAll(occupied, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(occupied, "existing"), []byte("existing"), 0o600))

	err := trasher.copyAndDelete(source, occupied)
	require.ErrorIs(t, err, ErrRestoreCollision)

	assertDirContainsOnly(t, occupied, "existing")
	assert.DirExists(t, source, "a refused copy must not consume the source")
}

// assertDirContainsOnly asserts the directory holds exactly the named entries.
func assertDirContainsOnly(t *testing.T, dir string, names ...string) {
	t.Helper()

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	got := make([]string, 0, len(entries))
	for _, entry := range entries {
		got = append(got, entry.Name())
	}

	assert.ElementsMatch(t, names, got, "directory contents must be unchanged")
}

// TestRestoreFromTrash_RestoresFile verifies the successful restore path still
// moves the file and clears the metadata.
func TestRestoreFromTrash_RestoresFile(t *testing.T) {
	t.Parallel()

	trasher := newTestTrasher(t)
	ctx := t.Context()

	sourceDir := t.TempDir()
	source := filepath.Join(sourceDir, "payload")
	require.NoError(t, os.WriteFile(source, []byte("restored content"), 0o600))

	trashPath, err := trasher.MoveToTrash(ctx, source)
	require.NoError(t, err)
	require.NoFileExists(t, source, "trashing must move the file out of the source location")

	require.NoError(t, trasher.RestoreFromTrash(ctx, trashPath, source))

	data, readErr := os.ReadFile(source)
	require.NoError(t, readErr)
	assert.Equal(t, "restored content", string(data))
	assert.False(t, trasher.IsInTrash(trashPath), "the trashed copy must be consumed")
	assert.NoFileExists(t, trashPath+trashInfoExt,
		"the metadata must be removed alongside the restored file")
}

// TestRestoreFromTrash_RejectsRelativePath verifies that a relative restore
// target is refused rather than resolved against the working directory.
func TestRestoreFromTrash_RejectsRelativePath(t *testing.T) {
	// Not parallel: t.Chdir is incompatible with t.Parallel.
	trasher := newTestTrasher(t)
	ctx := t.Context()

	sourceDir := t.TempDir()
	source := filepath.Join(sourceDir, "payload")
	require.NoError(t, os.WriteFile(source, []byte("content"), 0o600))

	trashPath, err := trasher.MoveToTrash(ctx, source)
	require.NoError(t, err)

	// Run from a scratch directory. If the validation ever regresses, the
	// restore resolves against the working directory and would otherwise create
	// the target inside the package source tree.
	t.Chdir(t.TempDir())

	err = trasher.RestoreFromTrash(ctx, trashPath, "relative/target")
	require.ErrorIs(t, err, ErrInvalidPath)
}

// TestMoveToTrash_LongName verifies that a name too long for the trash is
// shortened rather than failing permanently.
//
// The candidate is stored as <candidate> and <candidate>.trashinfo, so an
// original name near the limit left no room and could never be trashed at all.
func TestMoveToTrash_LongName(t *testing.T) {
	t.Parallel()

	trasher := newTestTrasher(t)
	dir := t.TempDir()

	// Creatable on its own, but too long for a trash entry once the random
	// suffix and the metadata extension are added.
	longName := filepath.Join(dir, strings.Repeat("a", 250)+".txt")
	require.NoError(t, os.WriteFile(longName, []byte("long"), 0o600))

	trashPath, err := trasher.MoveToTrash(t.Context(), longName)
	require.NoError(t, err, "a long name must be truncated, not rejected")

	data, readErr := os.ReadFile(trashPath)
	require.NoError(t, readErr)
	assert.Equal(t, "long", string(data))
}

// TestMoveToTrash_LongMultiByteName verifies that truncation does not split a
// multi-byte rune, which would leave invalid UTF-8 in the stored name.
func TestMoveToTrash_LongMultiByteName(t *testing.T) {
	t.Parallel()

	trasher := newTestTrasher(t)
	dir := t.TempDir()

	// 124 two-byte runes is 248 bytes: creatable alone, too long for a trash
	// entry, and a byte-wise truncation would split a rune.
	longName := filepath.Join(dir, strings.Repeat("é", 124)+".txt")
	require.NoError(t, os.WriteFile(longName, []byte("long"), 0o600))

	trashPath, err := trasher.MoveToTrash(t.Context(), longName)
	require.NoError(t, err)

	assert.True(t, utf8.ValidString(filepath.Base(trashPath)),
		"trash name must remain valid UTF-8")
}

// TestMoveToTrash_DanglingSymlink verifies that a symlink whose target is
// missing is trashed rather than reported as not found.
//
// The rest of the package uses Lstat deliberately, so a Stat here would make
// go-remove refuse a file that plainly exists.
func TestMoveToTrash_DanglingSymlink(t *testing.T) {
	t.Parallel()

	trasher := newTestTrasher(t)
	ctx := t.Context()

	dir := t.TempDir()
	link := filepath.Join(dir, "dangling")
	require.NoError(t, os.Symlink(filepath.Join(dir, "absent-target"), link))

	trashPath, err := trasher.MoveToTrash(ctx, link)
	require.NoError(t, err, "a dangling symlink still exists and must be trashed")

	target, err := os.Readlink(trashPath)
	require.NoError(t, err, "the symlink itself must be trashed, not its target")
	assert.Equal(t, filepath.Join(dir, "absent-target"), target)
}

// TestMoveToTrash_FifoIsRefused verifies that a named pipe is rejected rather
// than opened, which would block forever waiting for a writer.
func TestMoveToTrash_FifoIsRefused(t *testing.T) {
	t.Parallel()

	trasher, ok := newTestTrasher(t).(*xdgTrasher)
	require.True(t, ok)

	fifo := filepath.Join(t.TempDir(), "pipe")
	require.NoError(t, syscall.Mkfifo(fifo, 0o600))

	// The copy path is what would hang, so drive it directly rather than
	// through MoveToTrash, whose rename succeeds on a single filesystem.
	err := trasher.copyAndDelete(fifo, filepath.Join(t.TempDir(), "pipe-copy"))
	require.ErrorIs(t, err, ErrUnsupportedFileType)
}

// TestCopyFile_RejectsExistingDestination verifies that the copy path claims
// its destination exclusively rather than truncating whatever is there.
func TestCopyFile_RejectsExistingDestination(t *testing.T) {
	t.Parallel()

	trasher, ok := newTestTrasher(t).(*xdgTrasher)
	require.True(t, ok)

	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	require.NoError(t, os.WriteFile(src, []byte("source"), 0o600))

	dst := filepath.Join(dir, "dst")
	require.NoError(t, os.WriteFile(dst, []byte("existing"), 0o600))

	srcInfo, err := os.Lstat(src)
	require.NoError(t, err)

	err = trasher.copyFile(src, dst, srcInfo)
	require.ErrorIs(t, err, ErrRestoreCollision)

	data, readErr := os.ReadFile(dst)
	require.NoError(t, readErr)
	assert.Equal(t, "existing", string(data), "an existing destination must not be truncated")
}

// TestCopyAndDelete_SymlinkSourceIsPreserved verifies that a symlink survives
// the copy path as a symlink rather than being dereferenced.
func TestCopyAndDelete_SymlinkSourceIsPreserved(t *testing.T) {
	t.Parallel()

	trasher, ok := newTestTrasher(t).(*xdgTrasher)
	require.True(t, ok)

	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	require.NoError(t, os.WriteFile(target, []byte("payload"), 0o600))

	link := filepath.Join(dir, "link")
	require.NoError(t, os.Symlink(target, link))

	dst := filepath.Join(t.TempDir(), "copied-link")
	require.NoError(t, trasher.copyAndDelete(link, dst))

	resolved, err := os.Readlink(dst)
	require.NoError(t, err, "the copy must remain a symlink")
	assert.Equal(t, target, resolved)
	assert.FileExists(t, target, "dereferencing the copy would have moved the target")
}
