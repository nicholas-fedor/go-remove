//go:build linux || darwin

/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package trash

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Permission constants for file operations.
const (
	// dirPermission is the permission for creating directories.
	dirPermission = 0o700
)

// getXDGTrashPath returns the XDG trash path for the current environment.
//
// It uses $XDG_DATA_HOME/Trash when that value is an absolute path, otherwise
// ~/.local/share/Trash.
//
// Returns:
//   - Absolute trash directory path, or empty if the home directory cannot be
//     resolved.
func getXDGTrashPath() string {
	xdgDataHome := os.Getenv("XDG_DATA_HOME")
	if xdgDataHome == "" || !filepath.IsAbs(xdgDataHome) {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}

		xdgDataHome = filepath.Join(home, ".local", "share")
	}

	return filepath.Join(xdgDataHome, "Trash")
}

// xdgTrasher implements Trasher using the XDG Trash specification.
//
// The specification is shared by Linux and macOS, so one implementation
// serves both.
type xdgTrasher struct {
	trashPath string
	filesDir  string
	infoDir   string
}

var _ Trasher = (*xdgTrasher)(nil)

// newTrasher creates a trash manager using the XDG Trash specification.
//
// Returns:
//   - XDG trash implementation.
//   - An error if the trash directories cannot be created.
func newTrasher() (Trasher, error) {
	trashPath := getXDGTrashPath()
	if trashPath == "" {
		return nil, fmt.Errorf("%w: could not determine trash path", ErrTrashFull)
	}

	return newTrasherAt(trashPath)
}

// newTrasherAt creates an XDG trash manager rooted at the given path.
//
// The files and info subdirectories are created beneath root when absent.
//
// Parameters:
//   - root: Absolute path to use as the trash root.
//
// Returns:
//   - XDG trash implementation.
//   - An error if the trash directories cannot be created.
func newTrasherAt(root string) (Trasher, error) {
	trasher := &xdgTrasher{
		trashPath: root,
		filesDir:  filepath.Join(root, "files"),
		infoDir:   filepath.Join(root, "info"),
	}

	// Ensure trash directories exist
	if err := os.MkdirAll(trasher.filesDir, dirPermission); err != nil {
		return nil, fmt.Errorf("%w: creating files directory: %w", ErrTrashFull, err)
	}

	if err := os.MkdirAll(trasher.infoDir, dirPermission); err != nil {
		return nil, fmt.Errorf("%w: creating info directory: %w", ErrTrashFull, err)
	}

	return trasher, nil
}

// MoveToTrash moves a file to the XDG trash directory.
//
// Parameters:
//   - ctx: Context for cancellation.
//   - filePath: Path of the file to trash.
//
// Returns:
//   - Path of the file in trash.
//   - An error if the move fails.
func (t *xdgTrasher) MoveToTrash(ctx context.Context, filePath string) (string, error) {
	if ctx.Err() != nil {
		return "", fmt.Errorf("context cancelled: %w", ctx.Err())
	}

	// Verify source exists
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%w: %s", ErrPathNotFound, filePath)
		}

		return "", fmt.Errorf("checking file: %w", err)
	}

	// Claim a unique trash entry name. The metadata file is written first, so a
	// crash between the two writes leaves metadata with no data rather than a
	// file in trash with no way to trace it back.
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	trashFilePath, infoFilePath, err := reserveTrashEntry(
		t.filesDir,
		t.infoDir,
		filepath.Base(filePath),
		generateTrashInfo(absPath, time.Now()),
		maxTrashNameAttempts,
	)
	if err != nil {
		return "", err
	}

	// Move file to trash
	err = t.moveFile(filePath, trashFilePath)
	if err != nil {
		// Clean up info file on failure
		os.Remove(infoFilePath)

		return "", fmt.Errorf("moving file to trash: %w", err)
	}

	return trashFilePath, nil
}

// RestoreFromTrash restores a file from trash to its original location.
//
// Parameters:
//   - ctx: Context for cancellation.
//   - trashPath: Current path of the file in trash.
//   - originalPath: Destination path for restoration.
//
// Returns:
//   - An error if restoration fails.
func (t *xdgTrasher) RestoreFromTrash(ctx context.Context, trashPath, originalPath string) error {
	if ctx.Err() != nil {
		return fmt.Errorf("context cancelled: %w", ctx.Err())
	}

	// Verify file is in trash
	if !t.IsInTrash(trashPath) {
		return fmt.Errorf("%w: %s", ErrFileNotInTrash, trashPath)
	}

	// Check if destination already exists
	_, err := os.Stat(originalPath)
	if err == nil {
		return fmt.Errorf("%w: %s", ErrRestoreCollision, originalPath)
	}

	if !os.IsNotExist(err) {
		return fmt.Errorf("checking destination: %w", err)
	}

	// Ensure parent directory exists
	parentDir := filepath.Dir(originalPath)
	if err := os.MkdirAll(parentDir, dirPermission); err != nil {
		return fmt.Errorf("creating parent directory: %w", err)
	}

	// Move file from trash to original location
	if err := os.Rename(trashPath, originalPath); err != nil {
		// Try copy-and-delete for cross-device moves
		if err := t.copyAndDelete(trashPath, originalPath); err != nil {
			return fmt.Errorf("restoring file: %w", err)
		}
	}

	// Clean up trashinfo file
	infoPath := t.getInfoPath(trashPath)
	os.Remove(infoPath) // Ignore error

	return nil
}

// IsInTrash reports whether a file exists in trash.
//
// Parameters:
//   - trashPath: Path to check.
//
// Returns:
//   - True if the file is present in trash.
func (t *xdgTrasher) IsInTrash(trashPath string) bool {
	_, err := os.Stat(trashPath)
	if err != nil {
		return false
	}

	// Verify it's within the trash files directory
	rel, err := filepath.Rel(t.filesDir, trashPath)
	if err != nil {
		return false
	}

	// Only reject parent-traversal (".." or starting with "../")
	// Hidden files (starting with ".") are allowed
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}

	return rel != "" && rel != "."
}

// ListTrash returns all entries in trash.
//
// Returns:
//   - Trash entries.
//   - An error if the trash directory cannot be read.
func (t *xdgTrasher) ListTrash() ([]TrashEntry, error) {
	entries, err := os.ReadDir(t.filesDir)
	if err != nil {
		return nil, fmt.Errorf("reading trash directory: %w", err)
	}

	result := make([]TrashEntry, 0, len(entries))

	for _, entry := range entries {
		trashPath := filepath.Join(t.filesDir, entry.Name())
		infoPath := filepath.Join(t.infoDir, entry.Name()+".trashinfo")

		originalPath, deletionTime, err := t.readTrashInfo(infoPath)
		if err != nil {
			// If we can't read info, use defaults
			originalPath = ""
			deletionTime = time.Time{}
		}

		result = append(result, TrashEntry{
			Name:         entry.Name(),
			OriginalPath: originalPath,
			TrashPath:    trashPath,
			DeletionTime: deletionTime,
		})
	}

	return result, nil
}

// DeletePermanently removes a file from trash.
//
// Parameters:
//   - ctx: Context for cancellation.
//   - trashPath: Path of the file in trash.
//
// Returns:
//   - An error if deletion fails.
func (t *xdgTrasher) DeletePermanently(ctx context.Context, trashPath string) error {
	if ctx.Err() != nil {
		return fmt.Errorf("context cancelled: %w", ctx.Err())
	}

	if !t.IsInTrash(trashPath) {
		return fmt.Errorf("%w: %s", ErrFileNotInTrash, trashPath)
	}

	// Remove the file/directory
	if err := os.RemoveAll(trashPath); err != nil {
		return fmt.Errorf("deleting from trash: %w", err)
	}

	// Remove trashinfo file
	infoPath := t.getInfoPath(trashPath)
	os.Remove(infoPath) // Ignore error

	return nil
}

// GetTrashPath returns the XDG trash files directory path.
//
// Returns:
//   - Absolute path to the trash files directory.
func (t *xdgTrasher) GetTrashPath() string {
	return t.filesDir
}

// getInfoPath returns the path to the trashinfo file for a given trash file.
//
// Parameters:
//   - trashPath: Path of the file in trash.
//
// Returns:
//   - Path to the corresponding .trashinfo file.
func (t *xdgTrasher) getInfoPath(trashPath string) string {
	baseName := filepath.Base(trashPath)

	return filepath.Join(t.infoDir, baseName+".trashinfo")
}

// readTrashInfo reads and parses a trashinfo file.
//
// Parameters:
//   - infoPath: Path to the .trashinfo file.
//
// Returns:
//   - Original filesystem path.
//   - Deletion timestamp.
//   - An error if the file cannot be read or parsed.
func (t *xdgTrasher) readTrashInfo(infoPath string) (string, time.Time, error) {
	content, err := os.ReadFile(infoPath)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("reading trashinfo file: %w", err)
	}

	return parseTrashInfo(string(content))
}

// moveFile moves a file or directory, falling back to copy-and-delete across devices.
//
// Parameters:
//   - src: Source path.
//   - dst: Destination path.
//
// Returns:
//   - An error if the move fails.
func (t *xdgTrasher) moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	return t.copyAndDelete(src, dst)
}

// copyAndDelete copies a file or directory and then deletes the source.
//
// Parameters:
//   - src: Source path.
//   - dst: Destination path.
//
// Returns:
//   - An error if copy or delete fails.
func (t *xdgTrasher) copyAndDelete(src, dst string) error {
	srcInfo, err := os.Lstat(src)
	if err != nil {
		return fmt.Errorf("stating source: %w", err)
	}

	// Handle symlinks specially to preserve them
	if srcInfo.Mode()&os.ModeSymlink != 0 {
		linkTarget, err := os.Readlink(src)
		if err != nil {
			return fmt.Errorf("reading symlink: %w", err)
		}

		if err := os.Symlink(linkTarget, dst); err != nil {
			return fmt.Errorf("creating symlink: %w", err)
		}

		if err := os.Remove(src); err != nil {
			return fmt.Errorf("removing source symlink: %w", err)
		}

		return nil
	}

	if srcInfo.IsDir() {
		err = t.copyDir(src, dst)
	} else {
		err = t.copyFile(src, dst, srcInfo)
	}

	if err != nil {
		return err
	}

	if err := os.RemoveAll(src); err != nil {
		return fmt.Errorf("removing source after copy: %w", err)
	}

	return nil
}

// copyFile copies a single file and preserves its timestamps.
//
// Parameters:
//   - src: Source file path.
//   - dst: Destination file path.
//   - srcInfo: File info for the source.
//
// Returns:
//   - An error if the copy fails.
func (t *xdgTrasher) copyFile(src, dst string, srcInfo os.FileInfo) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening source file: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode()&fs.ModePerm)
	if err != nil {
		return fmt.Errorf("creating destination file: %w", err)
	}

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		dstFile.Close()
		os.Remove(dst)

		return fmt.Errorf("copying file content: %w", err)
	}

	// Close destination file before setting times
	if err := dstFile.Close(); err != nil {
		return fmt.Errorf("closing destination file: %w", err)
	}

	// Preserve modification time
	mtime := srcInfo.ModTime()
	atime := time.Now()

	if err := os.Chtimes(dst, atime, mtime); err != nil {
		return fmt.Errorf("setting file times: %w", err)
	}

	return nil
}

// copyDir recursively copies a directory.
//
// Parameters:
//   - src: Source directory path.
//   - dst: Destination directory path.
//
// Returns:
//   - An error if the copy fails.
func (t *xdgTrasher) copyDir(src, dst string) error {
	// Use Lstat to avoid following symlinks when checking source
	srcInfo, err := os.Lstat(src)
	if err != nil {
		return fmt.Errorf("stating source directory: %w", err)
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()&fs.ModePerm); err != nil {
		return fmt.Errorf("creating destination directory: %w", err)
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("reading source directory: %w", err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		// Use Lstat to detect symlinks without following them
		info, err := os.Lstat(srcPath)
		if err != nil {
			return fmt.Errorf("getting entry info: %w", err)
		}

		// Handle symlinks specially to preserve them
		if info.Mode()&os.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(srcPath)
			if err != nil {
				return fmt.Errorf("reading symlink: %w", err)
			}

			if err := os.Symlink(linkTarget, dstPath); err != nil {
				return fmt.Errorf("creating symlink: %w", err)
			}

			continue
		}

		if info.IsDir() {
			if err := t.copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := t.copyFile(srcPath, dstPath, info); err != nil {
				return err
			}
		}
	}

	return nil
}
