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
//   - Linux: $XDG_DATA_HOME/Trash (fallback: ~/.local/share/Trash).
//   - Windows: Uses system Recycle Bin via SHFileOperationW.
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
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// maxTrashNameAttempts bounds the search for an unused trash entry name.
const maxTrashNameAttempts = 100

// trashInfoExt is the extension of a trash metadata file.
const trashInfoExt = ".trashinfo"

// filePermission is the permission for creating files in the trash.
const filePermission = 0o600

// Trash name length budget.
//
// A trash entry is stored as <candidate> under files and <candidate>.trashinfo
// under info, so the candidate has to leave room for its own suffix and for the
// metadata extension or the longest name cannot be created at all.
const (
	// trashNameSeparator joins the base name to its random suffix.
	trashNameSeparator = "_"

	// trashNameSuffixBytes is the length of the hexadecimal random suffix.
	trashNameSuffixBytes = 12

	// nameMaxBytes is the longest single path component most filesystems allow.
	nameMaxBytes = 255

	// maxTrashBaseBytes is the longest original file name that still leaves room
	// for the suffix and the metadata extension.
	maxTrashBaseBytes = nameMaxBytes - len(trashInfoExt) -
		len(trashNameSeparator) - trashNameSuffixBytes
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

	// ErrUnsupportedFileType indicates the entry is not a regular file.
	ErrUnsupportedFileType = errors.New("unsupported file type")

	// ErrPathNotFound indicates the path does not exist.
	ErrPathNotFound = errors.New("path not found")

	// ErrMissingPath indicates missing Path in trashinfo.
	ErrMissingPath = errors.New("missing Path in trashinfo")

	// ErrInvalidPercentEncoding indicates invalid percent encoding in a path.
	ErrInvalidPercentEncoding = errors.New("invalid percent encoding")

	// ErrTrashPathUnavailable indicates could not find an available trash path.
	ErrTrashPathUnavailable = errors.New("could not find available trash path")

	// ErrTrashRootUnsupported indicates the platform cannot use a caller-supplied trash root.
	ErrTrashRootUnsupported = errors.New("custom trash root is not supported on this platform")

	// ErrUnsupportedPlatform indicates no trash implementation exists for the platform.
	ErrUnsupportedPlatform = errors.New("trash is not supported on this platform")
)

// Trasher defines operations for XDG-compliant trash management.
type Trasher interface {
	// MoveToTrash moves a file to the trash directory.
	//
	// Parameters:
	//   - ctx: Context for cancellation.
	//   - filePath: Path of the file to trash.
	//
	// Returns:
	//   - Path of the file in trash.
	//   - An error if the move fails.
	MoveToTrash(ctx context.Context, filePath string) (string, error)

	// RestoreFromTrash moves a file from trash back to its original location.
	//
	// Parameters:
	//   - ctx: Context for cancellation.
	//   - trashPath: Current path of the file in trash.
	//   - originalPath: Destination path for restoration.
	//
	// Returns:
	//   - An error if restoration fails.
	RestoreFromTrash(ctx context.Context, trashPath, originalPath string) error

	// IsInTrash reports whether a file exists in trash.
	//
	// Parameters:
	//   - trashPath: Path to check.
	//
	// Returns:
	//   - True if the file is present in trash.
	IsInTrash(trashPath string) bool

	// ListTrash returns go-remove managed entries in trash.
	//
	// Returns:
	//   - Trash entries.
	//   - An error if the trash directory cannot be read.
	ListTrash() ([]TrashEntry, error)

	// DeletePermanently removes a file from trash.
	//
	// Parameters:
	//   - ctx: Context for cancellation.
	//   - trashPath: Path of the file in trash.
	//
	// Returns:
	//   - An error if deletion fails.
	DeletePermanently(ctx context.Context, trashPath string) error

	// GetTrashPath returns the trash files directory path.
	//
	// Returns:
	//   - Absolute path to the trash files directory.
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

// NewTrasher creates a platform-specific trash manager.
//
// Returns:
//   - Trash implementation for the current OS.
//   - An error if the trash directory cannot be determined or created.
func NewTrasher() (Trasher, error) {
	return newTrasher()
}

// encodeTrashPath encodes a path for storage in .trashinfo files.
//
// Bytes outside printable ASCII and percent signs are percent-encoded.
//
// Parameters:
//   - path: Original filesystem path.
//
// Returns:
//   - Encoded path suitable for a .trashinfo file.
func encodeTrashPath(path string) string {
	// URL-encode special characters
	var result []byte

	for i := range len(path) {
		c := path[i]

		if c <= 0x20 || c >= 0x7f || c == '%' {
			result = fmt.Appendf(result, "%%%02X", c)
		} else {
			result = append(result, c)
		}
	}

	return string(result)
}

// decodeTrashPath decodes a path from .trashinfo file format.
//
// Parameters:
//   - encoded: Percent-encoded path from a .trashinfo file.
//
// Returns:
//   - Decoded filesystem path.
//   - An error if a percent sequence is invalid.
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
//
// Parameters:
//   - originalPath: Path of the file before it was trashed.
//   - deletionTime: Time the file was moved to trash.
//
// Returns:
//   - .trashinfo file contents.
func generateTrashInfo(originalPath string, deletionTime time.Time) string {
	encodedPath := encodeTrashPath(originalPath)
	// Use ISO8601 without timezone per XDG spec
	timestamp := deletionTime.Local().Format("2006-01-02T15:04:05")

	return fmt.Sprintf("[Trash Info]\nPath=%s\nDeletionDate=%s\n", encodedPath, timestamp)
}

// parseTrashInfo parses a .trashinfo file content.
//
// Parameters:
//   - content: Raw .trashinfo file text.
//
// Returns:
//   - Original filesystem path.
//   - Deletion timestamp, or zero if it cannot be parsed.
//   - An error if the Path field is missing or invalid.
func parseTrashInfo(content string) (string, time.Time, error) {
	var pathLine, timeLine string

	_, err := fmt.Sscanf(content, "[Trash Info]\nPath=%s\nDeletionDate=%s\n", &pathLine, &timeLine)
	if err != nil {
		for line := range strings.SplitSeq(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
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

	originalPath, err := decodeTrashPath(pathLine)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("decoding path: %w", err)
	}

	var deletionTime time.Time
	if timeLine != "" {
		deletionTime, err = time.Parse(time.RFC3339, timeLine)
		if err != nil {
			deletionTime, err = time.Parse("2006-01-02T15:04:05", timeLine)
			if err != nil {
				deletionTime = time.Time{}
			}
		}
	}

	return originalPath, deletionTime, nil
}

// generateUniqueName creates a candidate trash entry name from a base file name.
//
// The suffix is random rather than clock-derived. A clock suffix of one-second
// resolution makes two calls within the same second produce the same candidate,
// which previously let a second trash operation silently overwrite the first.
//
// The base is truncated so the candidate, its suffix and the metadata extension
// all fit within the filesystem's name limit. Without that, a long original name
// fails permanently rather than being shortened.
//
// Parameters:
//   - base: Original file name.
//
// Returns:
//   - Name with a random hexadecimal suffix.
func generateUniqueName(base string) string {
	var suffix [trashNameSuffixBytes / 2]byte

	// crypto/rand.Read never fails and always fills the buffer, so the error is
	// deliberately discarded rather than falling back to a weaker source.
	_, _ = rand.Read(suffix[:])

	return truncateUTF8(base, maxTrashBaseBytes) +
		trashNameSeparator + hex.EncodeToString(suffix[:])
}

// truncateUTF8 shortens a string to at most maxBytes without splitting a rune.
//
// Parameters:
//   - value: String to shorten.
//   - maxBytes: Maximum length in bytes.
//
// Returns:
//   - The original string, or a prefix of it that ends on a rune boundary.
func truncateUTF8(value string, maxBytes int) string {
	if len(value) <= maxBytes {
		return value
	}

	cut := maxBytes
	for cut > 0 && !utf8.RuneStart(value[cut]) {
		cut--
	}

	return value[:cut]
}

// reserveTrashEntry claims an unused trash entry name and writes its metadata.
//
// The metadata file is created with O_EXCL, so claiming a name is atomic. Two
// callers racing for the same name cannot both win: the loser sees fs.ErrExist
// and retries with a fresh random candidate. A leftover data file from an
// earlier crash also blocks its own name, so an orphaned entry is never
// overwritten.
//
// Parameters:
//   - filesDir: Directory holding trashed data files.
//   - infoDir: Directory holding trash metadata files.
//   - baseName: Original file name, used as the candidate prefix.
//   - infoContent: Metadata to write for the claimed name.
//   - maxAttempts: How many candidate names to try before giving up.
//
// Returns:
//   - Path of the claimed data file.
//   - Path of the claimed metadata file.
//   - An error if no candidate could be claimed.
func reserveTrashEntry(
	filesDir, infoDir, baseName, infoContent string,
	maxAttempts int,
) (string, string, error) {
	for range maxAttempts {
		candidate := generateUniqueName(baseName)

		dataPath := filepath.Join(filesDir, candidate)
		infoPath := filepath.Join(infoDir, candidate+trashInfoExt)

		// An existing data file owns the name even if its metadata is gone.
		_, statErr := os.Lstat(dataPath)
		if statErr == nil {
			continue
		} else if !errors.Is(statErr, fs.ErrNotExist) {
			return "", "", fmt.Errorf("checking trash path: %w", statErr)
		}

		file, createErr := os.OpenFile(
			infoPath,
			os.O_WRONLY|os.O_CREATE|os.O_EXCL,
			filePermission,
		)

		switch {
		case errors.Is(createErr, fs.ErrExist):
			// Another caller claimed this name first.
			continue
		case createErr != nil:
			return "", "", fmt.Errorf("creating trashinfo file: %w", createErr)
		}

		if _, writeErr := file.WriteString(infoContent); writeErr != nil {
			file.Close()
			os.Remove(infoPath)

			return "", "", fmt.Errorf("writing trashinfo file: %w", writeErr)
		}

		if closeErr := file.Close(); closeErr != nil {
			os.Remove(infoPath)

			return "", "", fmt.Errorf("closing trashinfo file: %w", closeErr)
		}

		return dataPath, infoPath, nil
	}

	return "", "", fmt.Errorf("%w: %s", ErrTrashPathUnavailable, baseName)
}
