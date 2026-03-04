/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package history

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/nicholas-fedor/go-remove/internal/buildinfo"
	"github.com/nicholas-fedor/go-remove/internal/logger"
	"github.com/nicholas-fedor/go-remove/internal/storage"
	"github.com/nicholas-fedor/go-remove/internal/trash"
)

// Common errors for history operations.
var (
	// ErrEntryNotFound indicates the requested history entry does not exist.
	ErrEntryNotFound = errors.New("history entry not found")

	// ErrCannotRestore indicates the binary cannot be restored.
	ErrCannotRestore = errors.New("binary cannot be restored")

	// ErrNotInTrash indicates the binary is no longer in trash.
	ErrNotInTrash = errors.New("binary is no longer in trash")

	// ErrAlreadyRestored indicates the binary has already been restored.
	ErrAlreadyRestored = errors.New("binary has already been restored")

	// ErrInvalidBinaryPath indicates the binary path is invalid.
	ErrInvalidBinaryPath = errors.New("invalid binary path")

	// ErrRestoreCollision indicates a file already exists at the restore location.
	ErrRestoreCollision = errors.New("file already exists at restore location")

	// ErrNoHistory indicates no deletion history exists.
	ErrNoHistory = errors.New("no deletion history found")
)

// Manager defines high-level history operations.
//
// This interface provides the orchestration layer that coordinates trash,
// storage, and buildinfo packages to provide undo/restore functionality.
type Manager interface {
	// RecordDeletion captures a binary deletion to history.
	//
	// Parameters:
	//   - ctx: Context for cancellation
	//   - binaryPath: Full path to the binary being deleted
	//
	// Returns:
	//   - The created history entry
	//   - An error if the operation fails
	RecordDeletion(ctx context.Context, binaryPath string) (*HistoryEntry, error)

	// UndoMostRecent restores the most recently deleted binary.
	//
	// Parameters:
	//   - ctx: Context for cancellation
	//
	// Returns:
	//   - The result of the restore operation
	//   - An error if the operation fails or no history exists
	UndoMostRecent(ctx context.Context) (*RestoreResult, error)

	// Restore restores a specific binary from history by ID.
	//
	// Parameters:
	//   - ctx: Context for cancellation
	//   - entryID: The history entry ID (format: "timestamp:binary_name")
	//
	// Returns:
	//   - The result of the restore operation
	//   - An error if the operation fails
	Restore(ctx context.Context, entryID string) (*RestoreResult, error)

	// GetHistory retrieves the deletion history (newest first).
	//
	// Parameters:
	//   - ctx: Context for cancellation
	//   - limit: Maximum number of entries to return (0 = no limit)
	//
	// Returns:
	//   - A slice of history entries
	//   - An error if the operation fails
	GetHistory(ctx context.Context, limit int) ([]*HistoryEntry, error)

	// DeletePermanently removes a binary from trash and deletes the history entry.
	//
	// Parameters:
	//   - ctx: Context for cancellation
	//   - entryID: The history entry ID
	//
	// Returns:
	//   - An error if the operation fails
	DeletePermanently(ctx context.Context, entryID string) error

	// ClearHistory removes all history entries.
	//
	// Parameters:
	//   - ctx: Context for cancellation
	//   - clearTrash: If true, also clears all binaries from trash
	//
	// Returns:
	//   - An error if the operation fails
	ClearHistory(ctx context.Context, clearTrash bool) error

	// ClearEntry removes a single history entry.
	//
	// Parameters:
	//   - ctx: Context for cancellation
	//   - entryID: The history entry ID
	//   - deleteFromTrash: If true, also deletes the binary from trash
	//
	// Returns:
	//   - An error if the operation fails
	ClearEntry(ctx context.Context, entryID string, deleteFromTrash bool) error

	// Close closes all underlying resources.
	//
	// Returns:
	//   - An error if closing fails
	Close() error
}

// HistoryManager implements the Manager interface.
//
// It coordinates trash operations, storage persistence, and build info
// extraction to provide a unified history management interface.
type HistoryManager struct {
	trasher   trash.Trasher
	storer    storage.Storer
	extractor buildinfo.Extractor
	logger    logger.Logger
}

// NewManager creates a new history manager instance.
//
// Parameters:
//   - trasher: The trash.Trasher implementation for file operations
//   - storer: The storage.Storer implementation for persistence
//   - extractor: The buildinfo.Extractor implementation for metadata
//   - log: The logger.Logger for logging operations
//
// Returns:
//   - A Manager instance
//
// Example:
//
//	trasher, _ := trash.NewTrasher()
//	store, _ := storage.NewBadgerStore(dbPath)
//	extractor, _ := buildinfo.NewExtractor()
//	log, _ := logger.NewLogger()
//	manager := history.NewManager(trasher, store, extractor, log)
func NewManager(
	trasher trash.Trasher,
	storer storage.Storer,
	extractor buildinfo.Extractor,
	log logger.Logger,
) Manager {
	return &HistoryManager{
		trasher:   trasher,
		storer:    storer,
		extractor: extractor,
		logger:    log,
	}
}

// RecordDeletion captures a binary deletion to history.
//
// The workflow is:
//  1. Extract build info from binary
//  2. Calculate checksum
//  3. Move binary to trash
//  4. Create HistoryRecord with metadata
//  5. Save to storage
//
// Parameters:
//   - ctx: Context for cancellation
//   - binaryPath: Full path to the binary being deleted
//
// Returns:
//   - The created history entry
//   - An error if the operation fails
func (m *HistoryManager) RecordDeletion(
	ctx context.Context,
	binaryPath string,
) (*HistoryEntry, error) {
	if binaryPath == "" {
		return nil, ErrInvalidBinaryPath
	}

	m.logger.Debug().
		Str("path", binaryPath).
		Msg("Recording binary deletion")

	// Extract build info from binary
	buildData, err := m.extractor.Extract(ctx, binaryPath)
	if err != nil {
		return nil, fmt.Errorf("extracting build info: %w", err)
	}

	// Calculate checksum
	checksum, err := m.extractor.CalculateChecksum(binaryPath)
	if err != nil {
		return nil, fmt.Errorf("calculating checksum: %w", err)
	}

	// Move binary to trash
	trashPath, err := m.trasher.MoveToTrash(ctx, binaryPath)
	if err != nil {
		return nil, fmt.Errorf("moving to trash: %w", err)
	}

	// Get original directory
	originalDir := filepath.Dir(binaryPath)

	// Create history record
	now := time.Now()

	record := storage.HistoryRecord{
		Timestamp:      now.Unix(),
		BinaryName:     filepath.Base(binaryPath),
		OriginalPath:   binaryPath,
		TrashPath:      trashPath,
		ModulePath:     buildData.ModulePath,
		Version:        buildData.Version,
		VCSRevision:    buildData.VCSRevision,
		VCSTime:        time.Time{}, // Will be empty if parsing fails
		GoVersion:      buildData.GoVersion,
		BuildInfo:      string(buildData.RawJSON),
		Checksum:       checksum,
		TrashAvailable: true,
		OriginalDir:    originalDir,
	}

	// Try to parse VCS time if available
	if buildData.VCSTime != "" {
		if parsedTime, err := time.Parse(time.RFC3339, buildData.VCSTime); err == nil {
			record.VCSTime = parsedTime
		}
	}

	// Save to storage
	if err := m.storer.SaveRecord(ctx, &record); err != nil {
		// Attempt to restore from trash if storage fails
		m.logger.Warn().
			Err(err).
			Msg("Failed to save record, attempting to restore from trash")

		if restoreErr := m.trasher.RestoreFromTrash(ctx, trashPath, binaryPath); restoreErr != nil {
			m.logger.Error().
				Err(restoreErr).
				Msg("Failed to restore from trash after storage failure")
		}

		return nil, fmt.Errorf("saving history record: %w", err)
	}

	m.logger.Info().
		Str("binary", record.BinaryName).
		Str("path", binaryPath).
		Str("trash", trashPath).
		Msg("Binary deletion recorded")

	return entryFromRecord(&record), nil
}

// UndoMostRecent restores the most recently deleted binary.
//
// The workflow is:
//  1. Get most recent record from storage
//  2. Check if in trash (TrashAvailable flag)
//  3. If in trash: RestoreFromTrash
//  4. If not in trash: Cannot restore (return error)
//  5. Update storage: TrashAvailable=false
//
// Parameters:
//   - ctx: Context for cancellation
//
// Returns:
//   - The result of the restore operation
//   - An error if the operation fails or no history exists
func (m *HistoryManager) UndoMostRecent(ctx context.Context) (*RestoreResult, error) {
	m.logger.Debug().Msg("Undoing most recent deletion")

	// Get most recent record
	record, err := m.storer.GetMostRecent(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrNoHistory) {
			return nil, ErrNoHistory
		}

		return nil, fmt.Errorf("getting most recent record: %w", err)
	}

	return m.restoreRecord(ctx, &record)
}

// Restore restores a specific binary from history by ID.
//
// The workflow is:
//  1. Get specific record from storage
//  2. Check if in trash
//  3. If in trash: RestoreFromTrash
//  4. If not in trash: Return error (cannot restore deleted binary)
//  5. Update storage: TrashAvailable=false
//
// Parameters:
//   - ctx: Context for cancellation
//   - entryID: The history entry ID (format: "timestamp:binary_name")
//
// Returns:
//   - The result of the restore operation
//   - An error if the operation fails
func (m *HistoryManager) Restore(ctx context.Context, entryID string) (*RestoreResult, error) {
	m.logger.Debug().
		Str("entry_id", entryID).
		Msg("Restoring binary by entry ID")

	// Get specific record
	record, err := m.storer.GetRecord(ctx, entryID)
	if err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			return nil, ErrEntryNotFound
		}

		return nil, fmt.Errorf("getting record: %w", err)
	}

	return m.restoreRecord(ctx, &record)
}

// restoreRecord performs the actual restoration of a binary.
//
// Parameters:
//   - ctx: Context for cancellation
//   - record: The history record to restore (passed by pointer for efficiency)
//
// Returns:
//   - The result of the restore operation
//   - An error if the operation fails
func (m *HistoryManager) restoreRecord(
	ctx context.Context,
	record *storage.HistoryRecord,
) (*RestoreResult, error) {
	// Check if already restored
	if !record.TrashAvailable {
		return nil, fmt.Errorf("%w: binary has already been restored", ErrAlreadyRestored)
	}

	// Check if still in trash
	if !m.trasher.IsInTrash(record.TrashPath) {
		// Update storage to reflect trash status
		record.TrashAvailable = false

		if err := m.storer.UpdateRecord(ctx, record); err != nil {
			m.logger.Warn().
				Err(err).
				Msg("Failed to update record after trash check")
		}

		return nil, fmt.Errorf("%w: %s", ErrNotInTrash, record.BinaryName)
	}

	// Check if file exists at original location
	if record.OriginalPath == "" {
		return nil, fmt.Errorf("%w: original path is empty", ErrRestoreCollision)
	}

	// Check if a file already exists at the original location
	// Only treat as collision if the file is different from the one in trash
	if stat, err := os.Stat(record.OriginalPath); err == nil {
		// File exists at original location - check if it's the same file (already restored)
		trashStat, trashErr := os.Stat(record.TrashPath)
		if trashErr != nil {
			// Can't access trash file, assume collision to be safe
			return nil, fmt.Errorf("%w: %s", ErrRestoreCollision, record.OriginalPath)
		}

		// Compare device and inode to determine if files are the same
		// If they are the same file, it's already restored, not a collision
		if !os.SameFile(stat, trashStat) {
			return nil, fmt.Errorf("%w: %s", ErrRestoreCollision, record.OriginalPath)
		}

		// Files are the same - already restored, update record and return error
		record.TrashAvailable = false

		if updateErr := m.storer.UpdateRecord(ctx, record); updateErr != nil {
			m.logger.Warn().
				Err(updateErr).
				Msg("Failed to update record after detecting already restored file")
		}

		return nil, fmt.Errorf("%w: %s", ErrAlreadyRestored, record.BinaryName)
	}

	// Restore from trash
	if err := m.trasher.RestoreFromTrash(ctx, record.TrashPath, record.OriginalPath); err != nil {
		if errors.Is(err, trash.ErrRestoreCollision) {
			return nil, fmt.Errorf("%w: %s", ErrRestoreCollision, record.OriginalPath)
		}

		return nil, fmt.Errorf("restoring from trash: %w", err)
	}

	// Update record to mark as restored
	record.TrashAvailable = false

	if err := m.storer.UpdateRecord(ctx, record); err != nil {
		m.logger.Warn().
			Err(err).
			Msg("Failed to update record after restore")
	}

	m.logger.Info().
		Str("binary", record.BinaryName).
		Str("path", record.OriginalPath).
		Msg("Binary restored from trash")

	return &RestoreResult{
		EntryID:    GenerateKey(record.Timestamp, record.BinaryName),
		BinaryName: record.BinaryName,
		RestoredTo: record.OriginalPath,
		FromTrash:  true,
		ModulePath: record.ModulePath,
		Version:    record.Version,
	}, nil
}

// GetHistory retrieves the deletion history (newest first).
//
// Parameters:
//   - ctx: Context for cancellation
//   - limit: Maximum number of entries to return (0 = no limit)
//
// Returns:
//   - A slice of history entries
//   - An error if the operation fails
func (m *HistoryManager) GetHistory(ctx context.Context, limit int) ([]*HistoryEntry, error) {
	m.logger.Debug().
		Int("limit", limit).
		Msg("Getting deletion history")

	opts := storage.ListOptions{
		Limit: limit,
	}

	records, err := m.storer.ListRecords(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("listing records: %w", err)
	}

	entries := entriesFromRecords(records)

	m.logger.Debug().
		Int("count", len(entries)).
		Msg("Retrieved deletion history")

	return entries, nil
}

// DeletePermanently removes a binary from trash and deletes the history entry.
//
// Parameters:
//   - ctx: Context for cancellation
//   - entryID: The history entry ID
//
// Returns:
//   - An error if the operation fails
func (m *HistoryManager) DeletePermanently(ctx context.Context, entryID string) error {
	m.logger.Debug().
		Str("entry_id", entryID).
		Msg("Permanently deleting binary")

	// Get the record
	record, err := m.storer.GetRecord(ctx, entryID)
	if err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			return ErrEntryNotFound
		}

		return fmt.Errorf("getting record: %w", err)
	}

	// Delete from trash if available
	if record.TrashAvailable && m.trasher.IsInTrash(record.TrashPath) {
		if err := m.trasher.DeletePermanently(ctx, record.TrashPath); err != nil {
			if !errors.Is(err, trash.ErrFileNotInTrash) {
				return fmt.Errorf("deleting from trash: %w", err)
			}
		}
	}

	// Delete the history entry
	if err := m.storer.DeleteRecord(ctx, entryID); err != nil {
		return fmt.Errorf("deleting history entry: %w", err)
	}

	m.logger.Info().
		Str("binary", record.BinaryName).
		Str("entry_id", entryID).
		Msg("Binary permanently deleted")

	return nil
}

// ClearHistory removes all history entries.
//
// Parameters:
//   - ctx: Context for cancellation
//   - clearTrash: If true, also clears all binaries from trash
//
// Returns:
//   - An error if the operation fails
func (m *HistoryManager) ClearHistory(ctx context.Context, clearTrash bool) error {
	m.logger.Debug().
		Bool("clear_trash", clearTrash).
		Msg("Clearing history")

	if clearTrash {
		// Get all records to clear from trash
		records, err := m.storer.ListRecords(ctx, storage.ListOptions{})
		if err != nil {
			return fmt.Errorf("listing records for trash clearing: %w", err)
		}

		// Delete each binary from trash
		for i := range records {
			if records[i].TrashAvailable && m.trasher.IsInTrash(records[i].TrashPath) {
				if err := m.trasher.DeletePermanently(ctx, records[i].TrashPath); err != nil {
					m.logger.Warn().
						Err(err).
						Str("binary", records[i].BinaryName).
						Msg("Failed to delete binary from trash")
				}
			}
		}
	}

	// Delete all history entries
	if err := m.storer.DeleteAllRecords(ctx); err != nil {
		return fmt.Errorf("deleting all records: %w", err)
	}

	m.logger.Info().
		Bool("cleared_trash", clearTrash).
		Msg("History cleared")

	return nil
}

// ClearEntry removes a single history entry.
//
// Parameters:
//   - ctx: Context for cancellation
//   - entryID: The history entry ID
//   - deleteFromTrash: If true, also deletes the binary from trash
//
// Returns:
//   - An error if the operation fails
func (m *HistoryManager) ClearEntry(
	ctx context.Context,
	entryID string,
	deleteFromTrash bool,
) error {
	m.logger.Debug().
		Str("entry_id", entryID).
		Bool("delete_from_trash", deleteFromTrash).
		Msg("Clearing history entry")

	// Get the record
	record, err := m.storer.GetRecord(ctx, entryID)
	if err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			return ErrEntryNotFound
		}

		return fmt.Errorf("getting record: %w", err)
	}

	// Delete from trash if requested
	if deleteFromTrash && record.TrashAvailable && m.trasher.IsInTrash(record.TrashPath) {
		if err := m.trasher.DeletePermanently(ctx, record.TrashPath); err != nil {
			if !errors.Is(err, trash.ErrFileNotInTrash) {
				return fmt.Errorf("deleting from trash: %w", err)
			}
		}
	}

	// Delete the history entry
	if err := m.storer.DeleteRecord(ctx, entryID); err != nil {
		return fmt.Errorf("deleting history entry: %w", err)
	}

	m.logger.Info().
		Str("binary", record.BinaryName).
		Str("entry_id", entryID).
		Msg("History entry cleared")

	return nil
}

// Close closes all underlying resources.
//
// Returns:
//   - An error if closing fails
func (m *HistoryManager) Close() error {
	m.logger.Debug().Msg("Closing history manager")

	if err := m.storer.Close(); err != nil {
		return fmt.Errorf("closing storage: %w", err)
	}

	m.logger.Info().Msg("History manager closed")

	return nil
}
