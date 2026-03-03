//go:build linux

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
	"time"
)

// Permission constants for file operations.
const (
	// dirPermission is the permission for creating directories.
	dirPermission = 0o700
	// filePermission is the permission for creating files.
	filePermission = 0o600
)

// linuxTrasher implements Trasher for Linux using XDG Trash specification.
type linuxTrasher struct {
	trashPath string
	filesDir  string
	infoDir   string
}

// newTrasher creates a new Linux-specific trash manager.
func newTrasher() (Trasher, error) {
	trashPath := getXDGTrashPath()
	if trashPath == "" {
		return nil, fmt.Errorf("%w: could not determine trash path", ErrTrashFull)
	}

	trasher := &linuxTrasher{
		trashPath: trashPath,
		filesDir:  filepath.Join(trashPath, "files"),
		infoDir:   filepath.Join(trashPath, "info"),
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
// Returns the path to the file in trash.
func (t *linuxTrasher) MoveToTrash(ctx context.Context, filePath string) (string, error) {
	if ctx.Err() != nil {
		return "", fmt.Errorf("context cancelled: %w", ctx.Err())
	}

	// Verify source exists
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%w: %s", ErrPathNotFound, filePath)
		}

		return "", fmt.Errorf("checking file: %w", err)
	}

	// Generate unique trash entry name using timestamp
	baseName := filepath.Base(filePath)
	uniqueName := generateUniqueName(baseName)

	trashFilePath := filepath.Join(t.filesDir, uniqueName)
	infoFilePath := filepath.Join(t.infoDir, uniqueName+".trashinfo")

	// Check for collisions and retry if necessary
	const maxRetries = 100

	for i := range maxRetries {
		_, err = os.Stat(trashFilePath)
		if err != nil {
			break // Path is available
		}

		uniqueName = generateUniqueName(fmt.Sprintf("%s_%d", baseName, i))
		trashFilePath = filepath.Join(t.filesDir, uniqueName)
		infoFilePath = filepath.Join(t.infoDir, uniqueName+".trashinfo")
	}

	// Create trashinfo file first
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	infoContent := generateTrashInfo(absPath, time.Now())

	err = os.WriteFile(infoFilePath, []byte(infoContent), filePermission)
	if err != nil {
		return "", fmt.Errorf("creating trashinfo file: %w", err)
	}

	// Move file to trash
	err = t.moveFile(filePath, trashFilePath, fileInfo)
	if err != nil {
		// Clean up info file on failure
		os.Remove(infoFilePath)

		return "", fmt.Errorf("moving file to trash: %w", err)
	}

	return trashFilePath, nil
}

// RestoreFromTrash restores a file from trash to its original location.
func (t *linuxTrasher) RestoreFromTrash(ctx context.Context, trashPath, originalPath string) error {
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

// IsInTrash checks if a file exists in trash.
func (t *linuxTrasher) IsInTrash(trashPath string) bool {
	_, err := os.Stat(trashPath)
	if err != nil {
		return false
	}

	// Verify it's within the trash files directory
	rel, err := filepath.Rel(t.filesDir, trashPath)
	if err != nil {
		return false
	}

	// Should not start with ".." to be inside trash
	return rel != "" && rel != "." && rel[0] != '.'
}

// ListTrash returns all entries in trash.
func (t *linuxTrasher) ListTrash() ([]TrashEntry, error) {
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

// DeletePermanently removes a file from trash permanently.
func (t *linuxTrasher) DeletePermanently(ctx context.Context, trashPath string) error {
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
func (t *linuxTrasher) GetTrashPath() string {
	return t.filesDir
}

// getInfoPath returns the path to the trashinfo file for a given trash file.
func (t *linuxTrasher) getInfoPath(trashPath string) string {
	baseName := filepath.Base(trashPath)

	return filepath.Join(t.infoDir, baseName+".trashinfo")
}

// readTrashInfo reads and parses a trashinfo file.
func (t *linuxTrasher) readTrashInfo(infoPath string) (string, time.Time, error) {
	content, err := os.ReadFile(infoPath)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("reading trashinfo file: %w", err)
	}

	return parseTrashInfo(string(content))
}

// moveFile moves a file or directory to trash, handling cross-device moves.
//
//nolint:unparam // info parameter is kept for API consistency
func (t *linuxTrasher) moveFile(src, dst string, info os.FileInfo) error {
	// Try simple rename first
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}

	// If rename failed (likely cross-device), copy and delete
	return t.copyAndDelete(src, dst)
}

// copyAndDelete copies a file or directory and then deletes the source.
func (t *linuxTrasher) copyAndDelete(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stating source: %w", err)
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

// copyFile copies a single file.
func (t *linuxTrasher) copyFile(src, dst string, srcInfo os.FileInfo) error {
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
func (t *linuxTrasher) copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
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

		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("getting entry info: %w", err)
		}

		if entry.IsDir() {
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

// generateUniqueName creates a unique name for the trash entry.
func generateUniqueName(base string) string {
	timestamp := time.Now().Unix()

	return fmt.Sprintf("%s_%d", base, timestamp)
}
