/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package history

import (
	"time"

	"github.com/nicholas-fedor/go-remove/internal/storage"
)

// GenerateKey creates a composite key for history entries.
// This is a convenience wrapper around storage.GenerateKey.
//
// Parameters:
//   - timestamp: Unix timestamp (seconds since epoch)
//   - binaryName: Name of the binary file
//
// Returns:
//   - A composite key string
func GenerateKey(timestamp int64, binaryName string) string {
	return storage.GenerateKey(timestamp, binaryName)
}

// HistoryEntry represents a deletion record for UI display.
//
// This type is optimized for presentation in the TUI and CLI interfaces,
// providing a flattened view of the underlying storage.HistoryRecord.
type HistoryEntry struct {
	// ID is the unique entry identifier (composite key: "timestamp:binary_name").
	ID string

	// Timestamp is when the binary was deleted.
	Timestamp time.Time

	// BinaryName is the name of the binary file (e.g., "golangci-lint").
	BinaryName string

	// BinaryPath is the original full path where the binary was located.
	BinaryPath string

	// ModulePath is the Go module path (e.g., "github.com/golangci/golangci-lint").
	ModulePath string

	// Version is the semantic version of the binary (e.g., "v1.55.2").
	Version string

	// VCSRevision is the git commit SHA used to build the binary.
	VCSRevision string

	// InTrash indicates whether the binary is still available in trash.
	InTrash bool

	// CanRestore indicates whether the binary can be restored.
	// This is true when InTrash is true.
	CanRestore bool
}

// RestoreResult contains the outcome of a restore operation.
//
// This type provides comprehensive information about what was restored,
// where it was restored to, and how the restoration was performed.
type RestoreResult struct {
	// EntryID is the history entry ID that was restored.
	EntryID string

	// BinaryName is the name of the restored binary.
	BinaryName string

	// RestoredTo is the path where the binary was restored.
	RestoredTo string

	// FromTrash is true if restored from trash, false if reinstalled.
	FromTrash bool

	// ModulePath is the module used for reinstall (empty if restored from trash).
	ModulePath string

	// Version is the version used for reinstall (empty if restored from trash).
	Version string
}

// entryFromRecord converts a storage.HistoryRecord to a HistoryEntry.
//
// Parameters:
//   - record: The storage record to convert (passed by pointer for efficiency)
//
// Returns:
//   - A HistoryEntry populated from the record
func entryFromRecord(record *storage.HistoryRecord) *HistoryEntry {
	return &HistoryEntry{
		ID:          GenerateKey(record.Timestamp, record.BinaryName),
		Timestamp:   time.Unix(record.Timestamp, 0),
		BinaryName:  record.BinaryName,
		BinaryPath:  record.OriginalPath,
		ModulePath:  record.ModulePath,
		Version:     record.Version,
		VCSRevision: record.VCSRevision,
		InTrash:     record.TrashAvailable,
		CanRestore:  record.TrashAvailable,
	}
}

// entriesFromRecords converts a slice of storage.HistoryRecord to HistoryEntry slice.
//
// Parameters:
//   - records: The storage records to convert
//
// Returns:
//   - A slice of HistoryEntry populated from the records
func entriesFromRecords(records []storage.HistoryRecord) []*HistoryEntry {
	entries := make([]*HistoryEntry, len(records))

	for i := range records {
		entries[i] = entryFromRecord(&records[i])
	}

	return entries
}
