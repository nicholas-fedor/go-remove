/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package history

import (
	"time"

	"github.com/nicholas-fedor/go-remove/internal/storage"
)

// HistoryEntry is a deletion record shaped for TUI and CLI display.
type HistoryEntry struct {
	ID          string
	Timestamp   time.Time
	BinaryName  string
	BinaryPath  string
	ModulePath  string
	Version     string
	VCSRevision string
	InTrash     bool
}

// RestoreResult is the outcome of a restore or undo operation.
type RestoreResult struct {
	EntryID    string
	BinaryName string
	RestoredTo string
	FromTrash  bool
	ModulePath string
	Version    string
}

// entryFromRecord converts a storage record into a display entry.
//
// Parameters:
//   - record: Persisted history record to convert.
//
// Returns:
//   - Display-oriented history entry.
func entryFromRecord(record *storage.HistoryRecord) *HistoryEntry {
	return &HistoryEntry{
		ID:          storage.GenerateKey(record.Timestamp, record.BinaryName),
		Timestamp:   time.Unix(record.Timestamp, 0),
		BinaryName:  record.BinaryName,
		BinaryPath:  record.OriginalPath,
		ModulePath:  record.ModulePath,
		Version:     record.Version,
		VCSRevision: record.VCSRevision,
		InTrash:     record.TrashAvailable,
	}
}

// entriesFromRecords converts persisted records into display entries.
//
// Parameters:
//   - records: Persisted history records to convert.
//
// Returns:
//   - Slice of display-oriented history entries.
func entriesFromRecords(records []storage.HistoryRecord) []*HistoryEntry {
	entries := make([]*HistoryEntry, len(records))
	for i := range records {
		entries[i] = entryFromRecord(&records[i])
	}

	return entries
}
