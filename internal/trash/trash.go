/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

// Package trash provides XDG-compliant trash operations for Linux and Windows.
//
// This package implements the FreeDesktop.org Trash Specification for Linux
// and uses the Windows Shell API (SHFileOperation) for Windows. It supports
// moving files to trash, restoring files from trash, and permanent deletion.
//
// The package follows the XDG Base Directory Specification:
//   - Linux: $XDG_DATA_HOME/Trash (fallback: ~/.local/share/Trash)
//   - Windows: Uses system Recycle Bin via SHFileOperationW
//
// Usage:
//
//	trasher, err := trash.NewTrasher()
//	if err != nil {
//	    return err
//	}
//
//	// Move file to trash
//	trashPath, err := trasher.MoveToTrash(ctx, "/path/to/file")
//
//	// Restore file from trash
//	err = trasher.RestoreFromTrash(ctx, trashPath, "/original/path")
//
//	// Check if file is in trash
//	exists := trasher.IsInTrash(trashPath)
//
//	// Delete permanently
//	err = trasher.DeletePermanently(ctx, trashPath)
package trash

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Common errors for trash operations.
var (
	// ErrTrashFull indicates the trash directory is full or inaccessible.
	ErrTrashFull = errors.New("trash directory is full or inaccessible")

	// ErrFileNotInTrash indicates the file is not in the trash.
	ErrFileNotInTrash = errors.New("file not found in trash")

	// ErrRestoreCollision indicates a file already exists at the restore location.
	ErrRestoreCollision = errors.New("file already exists at restore location")

	// ErrInvalidPath indicates the provided path is invalid.
	ErrInvalidPath = errors.New("invalid path")

	// ErrPathNotFound indicates the path does not exist.
	ErrPathNotFound = errors.New("path not found")

	// ErrMissingPath indicates missing Path in trashinfo.
	ErrMissingPath = errors.New("missing Path in trashinfo")

	// ErrInvalidPercentEncoding indicates invalid percent encoding in a path.
	ErrInvalidPercentEncoding = errors.New("invalid percent encoding")

	// ErrTrashPathUnavailable indicates could not find an available trash path.
	ErrTrashPathUnavailable = errors.New("could not find available trash path")
)

// Trasher defines operations for XDG-compliant trash management.
type Trasher interface {
	// MoveToTrash moves a file to the XDG trash directory.
	// Returns the trash path and any error encountered.
	MoveToTrash(ctx context.Context, filePath string) (string, error)

	// RestoreFromTrash moves a file from trash back to its original location.
	// The original path is determined from the history record.
	RestoreFromTrash(ctx context.Context, trashPath, originalPath string) error

	// IsInTrash checks if a file exists in the trash.
	IsInTrash(trashPath string) bool

	// ListTrash returns all go-remove managed entries in trash.
	ListTrash() ([]TrashEntry, error)

	// DeletePermanently removes a file from trash permanently.
	DeletePermanently(ctx context.Context, trashPath string) error

	// GetTrashPath returns the XDG trash files directory path.
	GetTrashPath() string
}

// TrashEntry represents a single file in the trash.
type TrashEntry struct {
	// Name is the filename in trash (may include collision suffix).
	Name string

	// OriginalPath is where the file was located before deletion.
	OriginalPath string

	// TrashPath is the current location in trash.
	TrashPath string

	// DeletionTime is when the file was moved to trash.
	DeletionTime time.Time
}

// NewTrasher creates a new platform-specific trash manager.
// Returns an error if the trash directory cannot be determined or created.
func NewTrasher() (Trasher, error) {
	return newTrasher()
}

// getXDGTrashPath returns the XDG trash path based on the environment.
// On Linux: $XDG_DATA_HOME/Trash or ~/.local/share/Trash
// On Windows: Returns empty string (uses system Recycle Bin).
func getXDGTrashPath() string {
	if runtime.GOOS == "windows" {
		return ""
	}

	xdgDataHome := os.Getenv("XDG_DATA_HOME")
	if xdgDataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}

		xdgDataHome = filepath.Join(home, ".local", "share")
	}

	return filepath.Join(xdgDataHome, "Trash")
}

// encodeTrashPath encodes a path for storage in .trashinfo files.
// Special characters are percent-encoded per the XDG spec.
func encodeTrashPath(path string) string {
	// URL-encode special characters
	var result []byte

	for i := range len(path) {
		c := path[i]

		// Percent-encode non-printable characters and percent sign
		if c < 0x20 || c == '%' {
			result = append(result, fmt.Sprintf("%%%02X", c)...)
		} else {
			result = append(result, c)
		}
	}

	return string(result)
}

// decodeTrashPath decodes a path from .trashinfo file format.
func decodeTrashPath(encoded string) (string, error) {
	var result []byte

	for i := 0; i < len(encoded); i++ {
		if encoded[i] == '%' {
			// Validate that we have two following hex digits
			if i+2 >= len(encoded) {
				return "", fmt.Errorf("%w: incomplete sequence", ErrInvalidPercentEncoding)
			}

			hexChars := encoded[i+1 : i+3]

			decodedByte, err := strconv.ParseUint(hexChars, 16, 8)
			if err != nil {
				return "", fmt.Errorf("%w: %w", ErrInvalidPercentEncoding, err)
			}

			result = append(result, byte(decodedByte))
			i += 2
		} else {
			result = append(result, encoded[i])
		}
	}

	return string(result), nil
}

// generateTrashInfo creates the content for a .trashinfo file.
func generateTrashInfo(originalPath string, deletionTime time.Time) string {
	encodedPath := encodeTrashPath(originalPath)
	// Use ISO8601 without timezone per XDG spec
	timestamp := deletionTime.Local().Format("2006-01-02T15:04:05")

	return fmt.Sprintf("[Trash Info]\nPath=%s\nDeletionDate=%s\n", encodedPath, timestamp)
}

// parseTrashInfo parses a .trashinfo file content.
//
//nolint:nonamedreturns // Named returns improve clarity for this function
func parseTrashInfo(content string) (originalPath string, deletionTime time.Time, err error) {
	var (
		pathLine string
		timeLine string
	)

	_, err = fmt.Sscanf(content, "[Trash Info]\nPath=%s\nDeletionDate=%s\n", &pathLine, &timeLine)
	if err != nil {
		// Try with looser parsing
		lines := splitLines(content)

		for _, line := range lines {
			if after, ok := strings.CutPrefix(line, "Path="); ok {
				pathLine = after
			} else if after, ok := strings.CutPrefix(line, "DeletionDate="); ok {
				timeLine = after
			}
		}
	}

	if pathLine == "" {
		return "", time.Time{}, ErrMissingPath
	}

	originalPath, err = decodeTrashPath(pathLine)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("decoding path: %w", err)
	}

	if timeLine != "" {
		// Try RFC3339 format first
		deletionTime, err = time.Parse(time.RFC3339, timeLine)
		if err != nil {
			// Fall back to date-only format
			deletionTime, err = time.Parse("2006-01-02T15:04:05", timeLine)
			if err != nil {
				// Ignore time parsing errors
				deletionTime = time.Time{}
			}
		}
	}

	return originalPath, deletionTime, nil
}

// generateUniqueName creates a unique name for the trash entry.
func generateUniqueName(base string) string {
	timestamp := time.Now().Unix()

	return fmt.Sprintf("%s_%d", base, timestamp)
}

// splitLines splits a string into lines, handling various line endings.
func splitLines(s string) []string {
	var (
		lines []string
		start int
	)

	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		} else if s[i] == '\r' && i+1 < len(s) && s[i+1] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 2 //nolint:mnd // Windows line ending is 2 bytes
			i++
		}
	}

	if start < len(s) {
		lines = append(lines, s[start:])
	}

	return lines
}
