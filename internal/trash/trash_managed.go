//go:build linux || darwin || windows

/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package trash

import (
	"context"
	"errors"
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
	// dirPermission is the permission for creating directories inside the trash.
	dirPermission = 0o700

	// restoreDirPermission is the permission for a directory recreated during a
	// restore. The trash keeps its own directories private, but a restore target
	// belongs to the user, so recreating it at 0o700 would silently change the
	// permissions of a location such as /usr/local/bin.
	restoreDirPermission = 0o755
)

// managedTrasher implements Trasher against a trash directory go-remove owns.
//
// The layout follows the XDG Trash specification on every platform, with the
// files and info subdirectories and a .trashinfo file per entry. Owning the
// directory is what makes restore, listing and permanent deletion possible,
// which the platform trash cannot offer on Windows.
type managedTrasher struct {
	trashPath string
	filesDir  string
	infoDir   string
}

var _ Trasher = (*managedTrasher)(nil)

// newTrasher creates a trash manager rooted at the platform default location.
//
// Returns:
//   - Managed trash implementation.
//   - An error if the trash directories cannot be created.
func newTrasher() (Trasher, error) {
	trashPath := platformTrashRoot()
	if trashPath == "" {
		return nil, fmt.Errorf("%w: could not determine trash path", ErrTrashFull)
	}

	return newTrasherAt(trashPath)
}

// newTrasherAt creates a managed trash manager rooted at the given path.
//
// The files and info subdirectories are created beneath root when absent.
//
// Parameters:
//   - root: Absolute path to use as the trash root.
//
// Returns:
//   - Managed trash implementation.
//   - An error if the trash directories cannot be created.
func newTrasherAt(root string) (Trasher, error) {
	trasher := &managedTrasher{
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
func (t *managedTrasher) MoveToTrash(ctx context.Context, filePath string) (string, error) {
	if ctx.Err() != nil {
		return "", fmt.Errorf("context cancelled: %w", ctx.Err())
	}

	// Verify source exists. Lstat rather than Stat, because a symlink whose
	// target is missing still exists and is what should be trashed, whereas
	// Stat would report it as absent.
	if _, err := os.Lstat(filePath); err != nil {
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
func (t *managedTrasher) RestoreFromTrash(
	ctx context.Context,
	trashPath, originalPath string,
) error {
	if ctx.Err() != nil {
		return fmt.Errorf("context cancelled: %w", ctx.Err())
	}

	// Verify file is in trash
	if !t.IsInTrash(trashPath) {
		return fmt.Errorf("%w: %s", ErrFileNotInTrash, trashPath)
	}

	// A relative or unclean destination would resolve against the working
	// directory, which is not where the file came from.
	if !filepath.IsAbs(originalPath) {
		return fmt.Errorf(
			"%w: restore target must be absolute: %s",
			ErrInvalidPath,
			originalPath,
		)
	}

	originalPath = filepath.Clean(originalPath)

	// Ensure parent directory exists
	parentDir := filepath.Dir(originalPath)
	if err := os.MkdirAll(parentDir, restoreDirPermission); err != nil {
		return fmt.Errorf("creating parent directory: %w", err)
	}

	if err := t.restoreTo(ctx, trashPath, originalPath); err != nil {
		return err
	}

	// Clean up trashinfo file
	infoPath := t.getInfoPath(trashPath)
	os.Remove(infoPath) // Ignore error

	return nil
}

// restoreTo places a trashed file at its original location without ever
// overwriting an existing file.
//
// A hard link is attempted first because the kernel refuses to create one when
// the destination already exists, which makes the collision check atomic rather
// than a check-then-act that a concurrent writer can slip through. When the two
// paths sit on different devices, or the entry is not a regular file, the copy
// path claims the destination exclusively for the same reason.
//
// Parameters:
//   - ctx: Context for cancellation.
//   - trashPath: Path of the file in trash.
//   - originalPath: Absolute destination to restore to.
//
// Returns:
//   - ErrRestoreCollision if the destination was created in the meantime.
//   - An error if the restore fails.
func (t *managedTrasher) restoreTo(ctx context.Context, trashPath, originalPath string) error {
	info, statErr := os.Lstat(trashPath)
	if statErr != nil {
		return fmt.Errorf("stating trashed entry: %w", statErr)
	}

	// Only a regular file takes the hard-link route. link() does not follow
	// symlinks, so linking one would hard-link the link itself, and every other
	// entry type is already handled explicitly by the copy path.
	if info.Mode().IsRegular() {
		linkErr := os.Link(trashPath, originalPath)

		switch {
		case linkErr == nil:
			if err := os.Remove(trashPath); err != nil {
				return fmt.Errorf("removing trashed copy after restore: %w", err)
			}

			return nil
		case errors.Is(linkErr, fs.ErrExist):
			return fmt.Errorf("%w: %s", ErrRestoreCollision, originalPath)
		case ctx.Err() != nil:
			return fmt.Errorf("context cancelled: %w", ctx.Err())
		}
	}

	// Directories, symlinks, and cross-device moves go through the copy path,
	// which recreates the entry with its original type and claims the
	// destination exclusively.
	if err := t.copyAndDelete(trashPath, originalPath); err != nil {
		if errors.Is(err, ErrRestoreCollision) {
			return fmt.Errorf("%w: %s", ErrRestoreCollision, originalPath)
		}

		return fmt.Errorf("restoring file: %w", err)
	}

	return nil
}

// IsInTrash reports whether a file exists in trash.
//
// Parameters:
//   - trashPath: Path to check.
//
// Returns:
//   - True if the file is present in trash.
func (t *managedTrasher) IsInTrash(trashPath string) bool {
	// Lstat rather than Stat. A symlink whose target has since been removed is
	// still an entry in the trash, and Stat would report it as absent, which
	// would make it neither restorable nor deletable.
	if _, err := os.Lstat(trashPath); err != nil {
		return false
	}

	rel, err := filepath.Rel(t.filesDir, trashPath)
	if err != nil {
		return false
	}

	if rel == "." || rel == ".." ||
		strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}

	// A trash entry is always a direct child of the files directory, so a
	// nested path is not one and must not be treated as restorable.
	return filepath.Dir(rel) == "."
}

// ListTrash returns all entries in trash.
//
// Returns:
//   - Trash entries.
//   - An error if the trash directory cannot be read.
func (t *managedTrasher) ListTrash() ([]TrashEntry, error) {
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
func (t *managedTrasher) DeletePermanently(ctx context.Context, trashPath string) error {
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
func (t *managedTrasher) GetTrashPath() string {
	return t.filesDir
}

// getInfoPath returns the path to the trashinfo file for a given trash file.
//
// Parameters:
//   - trashPath: Path of the file in trash.
//
// Returns:
//   - Path to the corresponding .trashinfo file.
func (t *managedTrasher) getInfoPath(trashPath string) string {
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
func (t *managedTrasher) readTrashInfo(infoPath string) (string, time.Time, error) {
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
func (t *managedTrasher) moveFile(src, dst string) error {
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}

	// Only a cross-device move is worth the copy fallback. Treating a
	// read-only or permission-denied source as a device mismatch would copy the
	// whole file, fail to unlink the original, and leave an unreferenced
	// duplicate behind while reporting failure.
	if !isCrossDevice(err) {
		return fmt.Errorf("moving %s: %w", src, err)
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
func (t *managedTrasher) copyAndDelete(src, dst string) error {
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

		// The destination is claimed the same way as for files and directories,
		// so an occupied destination is reported as a collision rather than a
		// generic symlink failure.
		if err := os.Symlink(linkTarget, dst); err != nil {
			if errors.Is(err, fs.ErrExist) {
				return fmt.Errorf("%w: %s", ErrRestoreCollision, dst)
			}

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
func (t *managedTrasher) copyFile(src, dst string, srcInfo os.FileInfo) error {
	// Opening a named pipe blocks until a writer appears, and nothing in this
	// path can be cancelled, so anything but a regular file is refused up front.
	if !srcInfo.Mode().IsRegular() {
		return fmt.Errorf(
			"%w: %s is a %s",
			ErrUnsupportedFileType,
			src,
			srcInfo.Mode().Type(),
		)
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening source file: %w", err)
	}
	defer srcFile.Close()

	// The destination is claimed exclusively so a file created after the
	// caller's collision check is never truncated.
	dstFile, err := os.OpenFile(
		dst,
		os.O_WRONLY|os.O_CREATE|os.O_EXCL,
		srcInfo.Mode()&fs.ModePerm,
	)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return fmt.Errorf("%w: %s", ErrRestoreCollision, dst)
		}

		return fmt.Errorf("creating destination file: %w", err)
	}

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		dstFile.Close()
		os.Remove(dst)

		return fmt.Errorf("copying file content: %w", err)
	}

	// Close destination file before setting times
	if err := dstFile.Close(); err != nil {
		// A close failure can mean the copy never reached disk, so the partial
		// destination has to go the same way the copy-error branch removes it.
		os.Remove(dst)

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
func (t *managedTrasher) copyDir(src, dst string) error {
	created, err := t.copyDirContents(src, dst)
	if err != nil && created {
		// The destination was made by this call, so a partial tree must not be
		// stranded. A destination this call did not create belongs to the
		// caller and is left alone, which is what stops a collision from
		// deleting the very directory it detected.
		os.RemoveAll(dst)
	}

	return err
}

// copyDirContents copies the entries of src beneath dst.
//
// Parameters:
//   - src: Source directory path.
//   - dst: Destination directory path.
//
// Returns:
//   - True if this call created the destination directory.
//   - An error if the copy fails.
func (t *managedTrasher) copyDirContents(src, dst string) (bool, error) {
	// Use Lstat to avoid following symlinks when checking source
	srcInfo, err := os.Lstat(src)
	if err != nil {
		return false, fmt.Errorf("stating source directory: %w", err)
	}

	// The destination root is claimed exclusively. MkdirAll would succeed when
	// the directory already exists and then merge the two trees, which silently
	// interleaves files instead of refusing the restore.
	if err := os.Mkdir(dst, srcInfo.Mode()&fs.ModePerm); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return false, fmt.Errorf("%w: %s", ErrRestoreCollision, dst)
		}

		return false, fmt.Errorf("creating destination directory: %w", err)
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return true, fmt.Errorf("reading source directory: %w", err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		// Use Lstat to detect symlinks without following them
		info, err := os.Lstat(srcPath)
		if err != nil {
			return true, fmt.Errorf("getting entry info: %w", err)
		}

		// Handle symlinks specially to preserve them
		if info.Mode()&os.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(srcPath)
			if err != nil {
				return true, fmt.Errorf("reading symlink: %w", err)
			}

			if err := os.Symlink(linkTarget, dstPath); err != nil {
				return true, fmt.Errorf("creating symlink: %w", err)
			}

			continue
		}

		if info.IsDir() {
			if err := t.copyDir(srcPath, dstPath); err != nil {
				return true, err
			}
		} else {
			if err := t.copyFile(srcPath, dstPath, info); err != nil {
				return true, err
			}
		}
	}

	return true, nil
}
