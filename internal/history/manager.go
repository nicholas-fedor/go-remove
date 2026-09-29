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
	"runtime"
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

	// ErrNoRestorableHistory indicates history exists but nothing in it can be restored.
	ErrNoRestorableHistory = errors.New("no restorable deletion found in history")

	// ErrInvalidRecord indicates a history record is missing required data.
	ErrInvalidRecord = errors.New("invalid history record")

	// ErrRecoveryRequired indicates a binary is stranded in trash with no usable
	// history record, so it must be recovered by hand.
	ErrRecoveryRequired = errors.New("manual recovery required")

	// ErrChecksumMismatch indicates the trash copy does not match the checksum
	// recorded when the binary was deleted.
	ErrChecksumMismatch = errors.New("trash copy does not match the recorded checksum")
)

// Log field keys used across history operations.
const (
	logFieldBinary  = "binary"
	logFieldEntryID = "entry_id"
	logFieldPath    = "path"
	logFieldTrash   = "trash"
)

// undoPageSize bounds how many history records UndoMostRecent reads at a time.
//
// Each persisted record embeds the binary's full build information, so loading
// the whole history to restore one entry would be needlessly expensive.
const undoPageSize = 50

// platformWindows is the GOOS value for Windows systems.
const platformWindows = "windows"

// trashStateQueryable reports whether the platform's trash can confirm that a
// path is still present.
//
// The Windows Recycle Bin is owned by the shell and exposes no reliable
// membership check, so IsInTrash always reports false there. Consulting it
// would mark every entry as absent straight after a successful deletion.
func trashStateQueryable() bool {
	return runtime.GOOS != platformWindows
}

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

var _ Manager = (*HistoryManager)(nil)

// NewManager creates a new history manager instance.
//
// Parameters:
//   - trasher: The trash.Trasher implementation for file operations.
//   - storer: The storage.Storer implementation for persistence.
//   - extractor: The buildinfo.Extractor implementation for metadata.
//   - log: The logger.Logger for logging operations.
//
// Returns:
//   - A Manager instance.
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
//  3. Persist the record, before the binary is moved, marked not yet in trash
//  4. Move binary to trash
//  5. Update the record with the trash location
//
// Recording before moving is what makes an interrupted deletion recoverable.
// Moving first leaves a binary in trash that no record points at if the process
// dies before the write, and the trash directory is not indexed by anything else,
// so that binary is unrecoverable through go-remove. Writing first leaves at
// worst a record missing its trash location, which still names the binary and
// leaves the file discoverable in the trash.
//
// Parameters:
//   - ctx: Context for cancellation.
//   - binaryPath: Full path to the binary being deleted.
//
// Returns:
//   - The created history entry.
//   - An error if the operation fails.
func (m *HistoryManager) RecordDeletion(
	ctx context.Context,
	binaryPath string,
) (*HistoryEntry, error) {
	if binaryPath == "" {
		return nil, ErrInvalidBinaryPath
	}

	m.logger.Debug().
		Str(logFieldPath, binaryPath).
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

	// Get original directory
	originalDir := filepath.Dir(binaryPath)

	// Create history record
	now := time.Now()

	record := storage.HistoryRecord{
		Timestamp:      now.Unix(),
		BinaryName:     filepath.Base(binaryPath),
		OriginalPath:   binaryPath,
		TrashPath:      "",
		ModulePath:     buildData.ModulePath,
		Version:        buildData.Version,
		VCSRevision:    buildData.VCSRevision,
		VCSTime:        time.Time{}, // Will be empty if parsing fails
		GoVersion:      buildData.GoVersion,
		BuildInfo:      string(buildData.RawJSON),
		Checksum:       checksum,
		TrashAvailable: false, // The binary has not moved yet.
		OriginalDir:    originalDir,
	}

	// Try to parse VCS time if available
	if buildData.VCSTime != "" {
		if parsedTime, err := time.Parse(time.RFC3339, buildData.VCSTime); err == nil {
			record.VCSTime = parsedTime
		}
	}

	// Persist the intent before touching the binary.
	if err := m.storer.SaveRecord(ctx, &record); err != nil {
		return nil, fmt.Errorf("saving history record: %w", err)
	}

	// Move binary to trash
	trashPath, moveErr := m.trasher.MoveToTrash(ctx, binaryPath)
	if moveErr != nil {
		// Nothing was moved, so the record describes a deletion that did not
		// happen. Drop it rather than leave history claiming otherwise.
		if delErr := m.storer.DeleteRecord(ctx, record.RecordKey()); delErr != nil {
			m.logger.Warn().
				Err(delErr).
				Str(logFieldPath, binaryPath).
				Msg("Failed to remove history record for a deletion that did not occur")
		}

		return nil, fmt.Errorf("moving to trash: %w", moveErr)
	}

	// Record where the binary went, so the entry becomes restorable.
	record.TrashPath = trashPath
	record.TrashAvailable = true

	if err := m.storer.UpdateRecord(ctx, &record); err != nil {
		m.logger.Warn().
			Err(err).
			Str(logFieldPath, binaryPath).
			Str(logFieldTrash, trashPath).
			Msg("Failed to record trash location, attempting to restore from trash")

		restoreErr := m.trasher.RestoreFromTrash(ctx, trashPath, binaryPath)
		if restoreErr == nil {
			// The binary is back where it started, so the record no longer
			// describes reality and should not remain in the history.
			if delErr := m.storer.DeleteRecord(ctx, record.RecordKey()); delErr != nil {
				m.logger.Warn().
					Err(delErr).
					Str(logFieldPath, binaryPath).
					Msg("Failed to remove history record after recovery")
			}

			return nil, fmt.Errorf("recording trash location: %w", err)
		}

		// The binary is in trash and cannot be put back, and the record does
		// not point at it. Report both facts, because the caller otherwise
		// cannot tell that a binary is stranded with no usable index.
		return nil, errors.Join(
			fmt.Errorf("recording trash location: %w", err),
			fmt.Errorf(
				"%w: binary is stranded in trash at %s and must be recovered by hand",
				ErrRecoveryRequired,
				trashPath,
			),
		)
	}

	m.logger.Info().
		Str(logFieldBinary, record.BinaryName).
		Str(logFieldPath, binaryPath).
		Str(logFieldTrash, trashPath).
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
//   - ctx: Context for cancellation.
//
// Returns:
//   - The result of the restore operation.
//   - An error if the operation fails or no history exists.
//
// UndoMostRecent restores the most recent deletion that can still be restored.
//
// Records are examined newest first, one bounded page at a time. One that is
// already restored, or whose trash copy has since been removed, is reconciled
// and skipped rather than ending the search, because those are expected states
// rather than failures. Previously a single restored record made every later
// undo fail outright.
//
// Parameters:
//   - ctx: Context for cancellation.
//
// Returns:
//   - The result of the restore operation.
//   - ErrNoHistory if no records exist.
//   - ErrNoRestorableHistory if no record can be restored.
//   - An error if the operation fails.
func (m *HistoryManager) UndoMostRecent(ctx context.Context) (*RestoreResult, error) {
	m.logger.Debug().Msg("Undoing most recent deletion")

	for offset := 0; ; offset += undoPageSize {
		records, err := m.storer.ListRecords(ctx, storage.ListOptions{
			Limit:  undoPageSize,
			Offset: offset,
		})
		if err != nil {
			if errors.Is(err, storage.ErrNoHistory) {
				return nil, ErrNoHistory
			}

			return nil, fmt.Errorf("listing history records: %w", err)
		}

		// An empty page means the history is exhausted. Distinguish a history
		// that never had entries from one where everything was unrestorable.
		if len(records) == 0 {
			if offset == 0 {
				return nil, ErrNoHistory
			}

			return nil, ErrNoRestorableHistory
		}

		for i := range records {
			record := &records[i]

			result, restoreErr := m.restoreRecord(ctx, record)
			if restoreErr == nil {
				return result, nil
			}

			// Only an unrestorable record is worth skipping. Any other failure is
			// real and must reach the caller rather than being masked by an older
			// entry. An interrupted deletion is unrestorable, so it is skipped like
			// the other states, and remains visible in the history listing.
			if !errors.Is(restoreErr, ErrAlreadyRestored) &&
				!errors.Is(restoreErr, ErrNotInTrash) &&
				!errors.Is(restoreErr, ErrRecoveryRequired) {
				return nil, restoreErr
			}

			m.logger.Debug().
				Str(logFieldBinary, record.BinaryName).
				Err(restoreErr).
				Msg("Skipping unrestorable history entry during undo")
		}
	}
}

// reconcileTrashState checks a record against the filesystem and persists the
// correction when the stored availability flag is stale.
//
// Parameters:
//   - ctx: Context for cancellation.
//   - record: Record to reconcile. Its TrashAvailable field may be corrected.
//
// Returns:
//   - True if the record's binary is still in trash and can be restored.
func (m *HistoryManager) reconcileTrashState(
	ctx context.Context,
	record *storage.HistoryRecord,
) bool {
	if !record.TrashAvailable {
		return false
	}

	if m.trasher.IsInTrash(record.TrashPath) {
		return true
	}

	record.TrashAvailable = false

	if err := m.storer.UpdateRecord(ctx, record); err != nil {
		m.logger.Warn().
			Err(err).
			Str(logFieldBinary, record.BinaryName).
			Msg("Failed to persist reconciled trash state")
	}

	return false
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
//   - ctx: Context for cancellation.
//   - entryID: The history entry ID (format: "timestamp:binary_name").
//
// Returns:
//   - The result of the restore operation.
//   - An error if the operation fails.
func (m *HistoryManager) Restore(ctx context.Context, entryID string) (*RestoreResult, error) {
	m.logger.Debug().
		Str(logFieldEntryID, entryID).
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

// deletionPending reports whether a record describes a deletion that was
// interrupted rather than one that finished.
//
// A record is persisted before its binary moves, so it carries no trash path
// until the move completes. An earlier version always wrote a trash path, so an
// empty one identifies exactly the window between recording the intent and
// recording where the binary went.
func deletionPending(record *storage.HistoryRecord) bool {
	return !record.TrashAvailable && record.TrashPath == ""
}

// restoreRecord performs the actual restoration of a binary.
//
// Parameters:
//   - ctx: Context for cancellation.
//   - record: The history record to restore (passed by pointer for efficiency).
//
// Returns:
//   - The result of the restore operation.
//   - An error if the operation fails.
func (m *HistoryManager) restoreRecord(
	ctx context.Context,
	record *storage.HistoryRecord,
) (*RestoreResult, error) {
	// A deletion that never finished is neither restored nor restorable, so it
	// must not be reported as though it had been restored.
	if deletionPending(record) {
		return nil, fmt.Errorf(
			"%w: deletion of %s was interrupted and must be recovered by hand",
			ErrRecoveryRequired,
			record.BinaryName,
		)
	}

	// Check if already restored
	if !record.TrashAvailable {
		return nil, fmt.Errorf("%w: binary has already been restored", ErrAlreadyRestored)
	}

	// Check if still in trash
	if !m.reconcileTrashState(ctx, record) {
		return nil, fmt.Errorf("%w: %s", ErrNotInTrash, record.BinaryName)
	}

	// Check if file exists at original location
	if record.OriginalPath == "" {
		return nil, fmt.Errorf("%w: original path is empty", ErrInvalidRecord)
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

	// Verify the trash copy against the checksum captured at deletion time,
	// before anything is moved. Restoring altered or damaged bytes would report
	// a success and put an unusable binary back where the user expects a working
	// one. Refusing here leaves the copy in trash, where it stays recoverable,
	// and leaves the record untouched.
	if record.Checksum != "" {
		actual, err := m.extractor.CalculateChecksum(record.TrashPath)
		if err != nil {
			return nil, fmt.Errorf(
				"%w: cannot read the trashed copy at %s: %w",
				ErrChecksumMismatch,
				record.TrashPath,
				err,
			)
		}

		if actual != record.Checksum {
			m.logger.Error().
				Str(logFieldBinary, record.BinaryName).
				Str(logFieldTrash, record.TrashPath).
				Str("expected_checksum", record.Checksum).
				Str("actual_checksum", actual).
				Msg("Trashed copy does not match the checksum recorded at deletion")

			return nil, fmt.Errorf(
				"%w: trashed copy of %s is left in place and must be recovered by hand",
				ErrChecksumMismatch,
				record.BinaryName,
			)
		}
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
		Str(logFieldBinary, record.BinaryName).
		Str(logFieldPath, record.OriginalPath).
		Msg("Binary restored from trash")

	return &RestoreResult{
		EntryID:    record.RecordKey(),
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
//   - ctx: Context for cancellation.
//   - limit: Maximum number of entries to return (0 = no limit).
//
// Returns:
//   - A slice of history entries.
//   - An error if the operation fails.
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

	// Report the live trash state rather than the stored flag. The flag is only
	// a record of what was true when the binary was deleted, and the system
	// trash can be emptied independently. Reconciling here is read-only, so
	// opening the view never rewrites history.
	//
	// On platforms that cannot answer the question, the stored flag is the only
	// signal available and is kept.
	queryable := trashStateQueryable()

	for i := range entries {
		inTrash := records[i].TrashAvailable
		if queryable {
			inTrash = inTrash && m.trasher.IsInTrash(records[i].TrashPath)
		}

		entries[i].InTrash = inTrash
	}

	m.logger.Debug().
		Int("count", len(entries)).
		Msg("Retrieved deletion history")

	return entries, nil
}

// DeletePermanently removes a binary from trash and deletes the history entry.
//
// Parameters:
//   - ctx: Context for cancellation.
//   - entryID: The history entry ID.
//
// Returns:
//   - An error if the operation fails.
func (m *HistoryManager) DeletePermanently(ctx context.Context, entryID string) error {
	m.logger.Debug().
		Str(logFieldEntryID, entryID).
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
		Str(logFieldBinary, record.BinaryName).
		Str(logFieldEntryID, entryID).
		Msg("Binary permanently deleted")

	return nil
}

// ClearHistory removes all history entries.
//
// Parameters:
//   - ctx: Context for cancellation.
//   - clearTrash: If true, also clears all binaries from trash.
//
// Returns:
//   - An error if the operation fails.
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
						Str(logFieldBinary, records[i].BinaryName).
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
//   - ctx: Context for cancellation.
//   - entryID: The history entry ID.
//   - deleteFromTrash: If true, also deletes the binary from trash.
//
// Returns:
//   - An error if the operation fails.
func (m *HistoryManager) ClearEntry(
	ctx context.Context,
	entryID string,
	deleteFromTrash bool,
) error {
	m.logger.Debug().
		Str(logFieldEntryID, entryID).
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
		Str(logFieldBinary, record.BinaryName).
		Str(logFieldEntryID, entryID).
		Msg("History entry cleared")

	return nil
}

// Close closes all underlying resources.
//
// Returns:
//   - An error if closing fails.
func (m *HistoryManager) Close() error {
	m.logger.Debug().Msg("Closing history manager")

	if err := m.storer.Close(); err != nil {
		return fmt.Errorf("closing storage: %w", err)
	}

	m.logger.Info().Msg("History manager closed")

	return nil
}
