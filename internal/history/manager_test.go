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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/go-remove/internal/buildinfo"
	buildinfomocks "github.com/nicholas-fedor/go-remove/internal/buildinfo/mocks"
	loggermocks "github.com/nicholas-fedor/go-remove/internal/logger/mocks"
	"github.com/nicholas-fedor/go-remove/internal/storage"
	storagemocks "github.com/nicholas-fedor/go-remove/internal/storage/mocks"
	"github.com/nicholas-fedor/go-remove/internal/trash"
	trashmocks "github.com/nicholas-fedor/go-remove/internal/trash/mocks"
)

// Test constants to avoid goconst warnings.
const (
	testBinaryPath  = "/usr/local/bin/test-binary"
	testTrashPath   = "/trash/test-binary"
	testBinaryName  = "test-binary"
	testModulePath  = "github.com/test/binary"
	testVersion     = "v1.0.0"
	testVCSRevision = "abc123def456"
	testGoVersion   = "go1.21"
	testChecksum    = "abc123"
	testVCSTime     = "2026-01-01T00:00:00Z"
)

// testEntryID is the expected entry ID with 20-digit zero-padded timestamp.
var testEntryID = storage.GenerateKey(1709321234, testBinaryName)

// setupManagerTest creates a new HistoryManager with mock dependencies for testing.
func setupManagerTest(
	t *testing.T,
) (*HistoryManager, *trashmocks.MockTrasher, *storagemocks.MockStorer, *buildinfomocks.MockExtractor) {
	t.Helper()

	mockTrasher := trashmocks.NewMockTrasher(t)
	mockStorer := storagemocks.NewMockStorer(t)
	mockExtractor := buildinfomocks.NewMockExtractor(t)
	mockLogger := loggermocks.NewMockLogger(t)

	// Setup logger mock to accept any calls
	mockLogger.EXPECT().Debug().Return(nil).Maybe()
	mockLogger.EXPECT().Info().Return(nil).Maybe()
	mockLogger.EXPECT().Warn().Return(nil).Maybe()
	mockLogger.EXPECT().Error().Return(nil).Maybe()

	manager := NewManager(mockTrasher, mockStorer, mockExtractor, mockLogger)

	return manager.(*HistoryManager), mockTrasher, mockStorer, mockExtractor
}

func TestNewManager(t *testing.T) {
	t.Parallel()

	mockTrasher := trashmocks.NewMockTrasher(t)
	mockStorer := storagemocks.NewMockStorer(t)
	mockExtractor := buildinfomocks.NewMockExtractor(t)
	mockLogger := loggermocks.NewMockLogger(t)

	manager := NewManager(mockTrasher, mockStorer, mockExtractor, mockLogger)

	assert.NotNil(t, manager)

	hm, ok := manager.(*HistoryManager)
	require.True(t, ok)
	assert.Equal(t, mockTrasher, hm.trasher)
	assert.Equal(t, mockStorer, hm.storer)
	assert.Equal(t, mockExtractor, hm.extractor)
	assert.Equal(t, mockLogger, hm.logger)
}

func TestHistoryManager_RecordDeletion(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	buildData := &buildinfo.BuildInfoData{
		ModulePath:  testModulePath,
		Version:     testVersion,
		VCSRevision: testVCSRevision,
		VCSTime:     testVCSTime,
		GoVersion:   testGoVersion,
		RawJSON:     []byte(`{"test": "data"}`),
	}

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		manager, mockTrasher, mockStorer, mockExtractor := setupManagerTest(t)

		// Setup expectations. The record is written before the binary moves, so
		// an interrupted deletion still leaves the binary named in history.
		mockExtractor.EXPECT().
			Extract(ctx, testBinaryPath).
			Return(buildData, nil)

		mockExtractor.EXPECT().
			CalculateChecksum(testBinaryPath).
			Return(testChecksum, nil)

		mockStorer.EXPECT().
			SaveRecord(ctx, mock.MatchedBy(func(r *storage.HistoryRecord) bool {
				return !r.TrashAvailable && r.TrashPath == ""
			})).
			Return(nil)

		mockTrasher.EXPECT().
			MoveToTrash(ctx, testBinaryPath).
			Return(testTrashPath, nil)

		mockStorer.EXPECT().
			UpdateRecord(ctx, mock.MatchedBy(func(r *storage.HistoryRecord) bool {
				return r.TrashAvailable && r.TrashPath == testTrashPath
			})).
			Return(nil)

		entry, err := manager.RecordDeletion(ctx, testBinaryPath)

		require.NoError(t, err)
		assert.NotNil(t, entry)
		assert.Equal(t, testBinaryName, entry.BinaryName)
		assert.Equal(t, testBinaryPath, entry.BinaryPath)
		assert.Equal(t, buildData.ModulePath, entry.ModulePath)
		assert.Equal(t, buildData.Version, entry.Version)
		assert.Equal(t, buildData.VCSRevision, entry.VCSRevision)
		assert.True(t, entry.InTrash)
	})

	t.Run("empty path", func(t *testing.T) {
		t.Parallel()

		manager, _, _, _ := setupManagerTest(t)

		entry, err := manager.RecordDeletion(ctx, "")

		require.ErrorIs(t, err, ErrInvalidBinaryPath)
		assert.Nil(t, entry)
	})

	t.Run("extract build info fails", func(t *testing.T) {
		t.Parallel()

		manager, _, _, mockExtractor := setupManagerTest(t)

		mockExtractor.EXPECT().
			Extract(ctx, testBinaryPath).
			Return(nil, buildinfo.ErrNotGoBinary)

		entry, err := manager.RecordDeletion(ctx, testBinaryPath)

		require.Error(t, err)
		assert.Nil(t, entry)
	})

	t.Run("calculate checksum fails", func(t *testing.T) {
		t.Parallel()

		manager, _, _, mockExtractor := setupManagerTest(t)

		mockExtractor.EXPECT().
			Extract(ctx, testBinaryPath).
			Return(buildData, nil)

		mockExtractor.EXPECT().
			CalculateChecksum(testBinaryPath).
			Return("", buildinfo.ErrPathNotFound)

		entry, err := manager.RecordDeletion(ctx, testBinaryPath)

		require.Error(t, err)
		assert.Nil(t, entry)
	})

	t.Run("move to trash fails", func(t *testing.T) {
		t.Parallel()

		manager, mockTrasher, mockStorer, mockExtractor := setupManagerTest(t)

		mockExtractor.EXPECT().
			Extract(ctx, testBinaryPath).
			Return(buildData, nil)

		mockExtractor.EXPECT().
			CalculateChecksum(testBinaryPath).
			Return(testChecksum, nil)

		// The record is written first, then discarded because nothing moved.
		mockStorer.EXPECT().
			SaveRecord(ctx, mock.AnythingOfType("*storage.HistoryRecord")).
			Return(nil)

		mockTrasher.EXPECT().
			MoveToTrash(ctx, testBinaryPath).
			Return("", trash.ErrTrashFull)

		mockStorer.EXPECT().
			DeleteRecord(mock.Anything, mock.AnythingOfType("string")).
			Return(nil)

		entry, err := manager.RecordDeletion(ctx, testBinaryPath)

		require.Error(t, err)
		assert.Nil(t, entry)
	})

	t.Run("save record fails before the move", func(t *testing.T) {
		t.Parallel()

		manager, mockTrasher, mockStorer, mockExtractor := setupManagerTest(t)

		mockExtractor.EXPECT().
			Extract(ctx, testBinaryPath).
			Return(buildData, nil)

		mockExtractor.EXPECT().
			CalculateChecksum(testBinaryPath).
			Return(testChecksum, nil)

		// Nothing has been moved yet, so no restore is attempted.
		mockStorer.EXPECT().
			SaveRecord(ctx, mock.AnythingOfType("*storage.HistoryRecord")).
			Return(storage.ErrDatabaseClosed)

		entry, err := manager.RecordDeletion(ctx, testBinaryPath)

		require.Error(t, err)
		assert.Nil(t, entry)
		mockTrasher.AssertNotCalled(t, "MoveToTrash", mock.Anything, mock.Anything)
	})

	t.Run("trash location update fails and restores", func(t *testing.T) {
		t.Parallel()

		manager, mockTrasher, mockStorer, mockExtractor := setupManagerTest(t)

		mockExtractor.EXPECT().
			Extract(ctx, testBinaryPath).
			Return(buildData, nil)

		mockExtractor.EXPECT().
			CalculateChecksum(testBinaryPath).
			Return(testChecksum, nil)

		mockStorer.EXPECT().
			SaveRecord(ctx, mock.AnythingOfType("*storage.HistoryRecord")).
			Return(nil)

		mockTrasher.EXPECT().
			MoveToTrash(ctx, testBinaryPath).
			Return(testTrashPath, nil)

		mockStorer.EXPECT().
			UpdateRecord(ctx, mock.AnythingOfType("*storage.HistoryRecord")).
			Return(storage.ErrDatabaseClosed)

		mockTrasher.EXPECT().
			RestoreFromTrash(mock.Anything, testTrashPath, testBinaryPath).
			Return(nil)

		// The binary is back in place, so the record must not survive. The
		// recovery runs on a derived context, so it is not the caller's.
		mockStorer.EXPECT().
			DeleteRecord(mock.Anything, mock.AnythingOfType("string")).
			Return(nil)

		entry, err := manager.RecordDeletion(ctx, testBinaryPath)

		require.Error(t, err)
		assert.Nil(t, entry)
	})

	t.Run("recovery fails and reports the stranded binary", func(t *testing.T) {
		t.Parallel()

		manager, mockTrasher, mockStorer, mockExtractor := setupManagerTest(t)

		mockExtractor.EXPECT().
			Extract(ctx, testBinaryPath).
			Return(buildData, nil)

		mockExtractor.EXPECT().
			CalculateChecksum(testBinaryPath).
			Return(testChecksum, nil)

		mockStorer.EXPECT().
			SaveRecord(ctx, mock.AnythingOfType("*storage.HistoryRecord")).
			Return(nil)

		mockTrasher.EXPECT().
			MoveToTrash(ctx, testBinaryPath).
			Return(testTrashPath, nil)

		mockStorer.EXPECT().
			UpdateRecord(ctx, mock.AnythingOfType("*storage.HistoryRecord")).
			Return(storage.ErrDatabaseClosed)

		mockTrasher.EXPECT().
			RestoreFromTrash(mock.Anything, testTrashPath, testBinaryPath).
			Return(trash.ErrRestoreCollision)

		entry, err := manager.RecordDeletion(ctx, testBinaryPath)

		// The caller must be able to tell that a binary is stranded with no
		// usable record, rather than seeing a generic failure.
		require.ErrorIs(t, err, ErrRecoveryRequired)
		assert.Nil(t, entry)
	})

	t.Run("recovery still runs after an interrupt", func(t *testing.T) {
		t.Parallel()

		manager, mockTrasher, mockStorer, mockExtractor := setupManagerTest(t)

		// An interrupt landing between the move and the record update leaves the
		// binary in trash, so the recovery must not inherit the cancellation that
		// caused the failure, or the binary would be stranded by the very
		// keystroke that should have stopped the work.
		interrupted, cancel := context.WithCancel(ctx)
		cancel()

		mockExtractor.EXPECT().
			Extract(mock.Anything, testBinaryPath).
			Return(buildData, nil)

		mockExtractor.EXPECT().
			CalculateChecksum(testBinaryPath).
			Return(testChecksum, nil)

		mockStorer.EXPECT().
			SaveRecord(mock.Anything, mock.AnythingOfType("*storage.HistoryRecord")).
			Return(nil)

		mockTrasher.EXPECT().
			MoveToTrash(mock.Anything, testBinaryPath).
			Return(testTrashPath, nil)

		mockStorer.EXPECT().
			UpdateRecord(mock.Anything, mock.AnythingOfType("*storage.HistoryRecord")).
			Return(storage.ErrDatabaseClosed)

		// A still-cancelled context would make this return immediately. The
		// state is captured during the call, because the recovery context is
		// cancelled once RecordDeletion returns.
		var recoveryErr error

		mockTrasher.EXPECT().
			RestoreFromTrash(mock.Anything, testTrashPath, testBinaryPath).
			//nolint:contextcheck // The point of the callback is to inspect the
			// context the code under test chose, not to propagate one.
			Run(func(ctx context.Context, _, _ string) {
				recoveryErr = ctx.Err()
			}).
			Return(nil)

		mockStorer.EXPECT().
			DeleteRecord(mock.Anything, mock.AnythingOfType("string")).
			Return(nil)

		entry, err := manager.RecordDeletion(interrupted, testBinaryPath)

		require.Error(t, err)
		assert.Nil(t, entry)

		require.NoError(t, recoveryErr,
			"recovery must not inherit the cancellation that caused the failure")
		mockTrasher.AssertNumberOfCalls(t, "RestoreFromTrash", 1)
	})

	t.Run("move failure reports a failed cleanup", func(t *testing.T) {
		t.Parallel()

		manager, mockTrasher, mockStorer, mockExtractor := setupManagerTest(t)

		mockExtractor.EXPECT().
			Extract(ctx, testBinaryPath).
			Return(buildData, nil)

		mockExtractor.EXPECT().
			CalculateChecksum(testBinaryPath).
			Return(testChecksum, nil)

		mockStorer.EXPECT().
			SaveRecord(ctx, mock.AnythingOfType("*storage.HistoryRecord")).
			Return(nil)

		mockTrasher.EXPECT().
			MoveToTrash(ctx, testBinaryPath).
			Return("", trash.ErrTrashFull)

		// The leftover record would be indistinguishable from an interrupted
		// deletion, so the cleanup failure has to reach the caller.
		mockStorer.EXPECT().
			DeleteRecord(mock.Anything, mock.AnythingOfType("string")).
			Return(storage.ErrDatabaseClosed)

		entry, err := manager.RecordDeletion(ctx, testBinaryPath)

		require.ErrorIs(t, err, trash.ErrTrashFull)
		require.ErrorIs(t, err, storage.ErrDatabaseClosed,
			"a failed cleanup must be reported alongside the move failure")
		assert.Nil(t, entry)
	})

	t.Run("cleanup completes after an interrupt", func(t *testing.T) {
		t.Parallel()

		manager, mockTrasher, mockStorer, mockExtractor := setupManagerTest(t)

		// An interrupt is what makes the move fail, and the record left behind
		// by that failure is indistinguishable from an interrupted deletion, so
		// removing it must not inherit the same cancellation.
		interrupted, cancel := context.WithCancel(ctx)
		cancel()

		mockExtractor.EXPECT().
			Extract(mock.Anything, testBinaryPath).
			Return(buildData, nil)

		mockExtractor.EXPECT().
			CalculateChecksum(testBinaryPath).
			Return(testChecksum, nil)

		mockStorer.EXPECT().
			SaveRecord(mock.Anything, mock.AnythingOfType("*storage.HistoryRecord")).
			Return(nil)

		mockTrasher.EXPECT().
			MoveToTrash(mock.Anything, testBinaryPath).
			Return("", trash.ErrTrashFull)

		var cleanupErr error

		mockStorer.EXPECT().
			DeleteRecord(mock.Anything, mock.AnythingOfType("string")).
			//nolint:contextcheck // The callback inspects the context the code
			// under test chose rather than propagating one.
			Run(func(cleanupCtx context.Context, _ string) {
				cleanupErr = cleanupCtx.Err()
			}).
			Return(nil)

		_, err := manager.RecordDeletion(interrupted, testBinaryPath)

		// Only the move is reported, because the cleanup succeeded.
		require.ErrorIs(t, err, trash.ErrTrashFull)
		require.NotErrorIs(t, err, storage.ErrDatabaseClosed)
		require.NoError(t, cleanupErr,
			"the cleanup must not inherit the cancellation that caused the failure")
	})
}

func TestHistoryManager_UndoMostRecent(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	now := time.Now()

	record := storage.HistoryRecord{
		Timestamp:      now.Unix(),
		BinaryName:     testBinaryName,
		OriginalPath:   testBinaryPath,
		TrashPath:      testTrashPath,
		ModulePath:     testModulePath,
		Version:        testVersion,
		TrashAvailable: true,
	}

	t.Run("success from trash", func(t *testing.T) {
		t.Parallel()

		manager, mockTrasher, mockStorer, _ := setupManagerTest(t)

		mockStorer.EXPECT().
			ListRecords(ctx, storage.ListOptions{Limit: undoPageSize, Offset: 0}).
			Return([]storage.HistoryRecord{record}, nil)

		mockTrasher.EXPECT().
			IsInTrash(testTrashPath).
			Return(true).
			Once()

		mockTrasher.EXPECT().
			RestoreFromTrash(ctx, testTrashPath, testBinaryPath).
			Return(nil)

		mockStorer.EXPECT().
			UpdateRecord(ctx, mock.AnythingOfType("*storage.HistoryRecord")).
			Return(nil)

		result, err := manager.UndoMostRecent(ctx)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, testBinaryName, result.BinaryName)
		assert.Equal(t, testBinaryPath, result.RestoredTo)
		assert.True(t, result.FromTrash)
	})

	t.Run("no history", func(t *testing.T) {
		t.Parallel()

		manager, _, mockStorer, _ := setupManagerTest(t)

		mockStorer.EXPECT().
			ListRecords(ctx, storage.ListOptions{Limit: undoPageSize, Offset: 0}).
			Return([]storage.HistoryRecord{}, nil)

		result, err := manager.UndoMostRecent(ctx)

		require.ErrorIs(t, err, ErrNoHistory)
		assert.Nil(t, result)
	})

	t.Run("already restored is skipped and reported as unrestorable", func(t *testing.T) {
		t.Parallel()

		manager, _, mockStorer, _ := setupManagerTest(t)

		restored := record
		restored.TrashAvailable = false

		// The first page holds the unrestorable record, the second is empty.
		mockStorer.EXPECT().
			ListRecords(ctx, storage.ListOptions{Limit: undoPageSize, Offset: 0}).
			Return([]storage.HistoryRecord{restored}, nil).
			Once()

		mockStorer.EXPECT().
			ListRecords(ctx, storage.ListOptions{Limit: undoPageSize, Offset: undoPageSize}).
			Return([]storage.HistoryRecord{}, nil).
			Once()

		result, err := manager.UndoMostRecent(ctx)

		require.ErrorIs(t, err, ErrNoRestorableHistory)
		require.NotErrorIs(t, err, ErrAlreadyRestored)
		assert.Nil(t, result)
	})

	t.Run("missing from trash is reconciled and skipped", func(t *testing.T) {
		t.Parallel()

		manager, mockTrasher, mockStorer, _ := setupManagerTest(t)

		mockStorer.EXPECT().
			ListRecords(ctx, storage.ListOptions{Limit: undoPageSize, Offset: 0}).
			Return([]storage.HistoryRecord{record}, nil).
			Once()

		mockStorer.EXPECT().
			ListRecords(ctx, storage.ListOptions{Limit: undoPageSize, Offset: undoPageSize}).
			Return([]storage.HistoryRecord{}, nil).
			Once()

		mockTrasher.EXPECT().
			IsInTrash(testTrashPath).
			Return(false).
			Once()

		// The stale flag must be corrected in storage.
		mockStorer.EXPECT().
			UpdateRecord(mock.Anything, mock.MatchedBy(func(r *storage.HistoryRecord) bool {
				return !r.TrashAvailable
			})).
			Return(nil)

		result, err := manager.UndoMostRecent(ctx)

		require.ErrorIs(t, err, ErrNoRestorableHistory)
		assert.Nil(t, result)
	})

	// This is the regression the rewrite exists for. Undo used to fetch a single
	// record and stop at the first unrestorable one, so a single successful undo
	// made every later undo fail outright.
	t.Run("skips unrestorable records and restores the next one", func(t *testing.T) {
		t.Parallel()

		manager, mockTrasher, mockStorer, _ := setupManagerTest(t)

		newerRestored := record
		newerRestored.TrashAvailable = false
		newerRestored.Timestamp = now.Unix()

		missingFromTrash := record
		missingFromTrash.Timestamp = now.Unix() - 1
		missingFromTrash.BinaryName = "vanished"
		missingFromTrash.TrashPath = "/trash/vanished"
		missingFromTrash.TrashAvailable = true

		restorable := record
		restorable.Timestamp = now.Unix() - 2
		restorable.BinaryName = "recoverable"
		restorable.TrashPath = "/trash/recoverable"
		restorable.OriginalPath = "/bin/recoverable"
		restorable.TrashAvailable = true

		// Newest first, as ListRecords guarantees.
		mockStorer.EXPECT().
			ListRecords(ctx, storage.ListOptions{Limit: undoPageSize, Offset: 0}).
			Return([]storage.HistoryRecord{newerRestored, missingFromTrash, restorable}, nil)

		mockTrasher.EXPECT().
			IsInTrash(missingFromTrash.TrashPath).
			Return(false).
			Once()

		mockStorer.EXPECT().
			UpdateRecord(mock.Anything, mock.AnythingOfType("*storage.HistoryRecord")).
			Return(nil)

		mockTrasher.EXPECT().
			IsInTrash(restorable.TrashPath).
			Return(true).
			Once()

		mockTrasher.EXPECT().
			RestoreFromTrash(ctx, restorable.TrashPath, restorable.OriginalPath).
			Return(nil)

		mockStorer.EXPECT().
			UpdateRecord(ctx, mock.AnythingOfType("*storage.HistoryRecord")).
			Return(nil)

		result, err := manager.UndoMostRecent(ctx)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "recoverable", result.BinaryName)
	})

	t.Run("restore collision", func(t *testing.T) {
		t.Parallel()

		manager, mockTrasher, mockStorer, _ := setupManagerTest(t)

		// restoreRecord checks the restore target with os.Stat, so the paths in
		// the record must point at real files for the collision to be observed.
		occupiedDir := t.TempDir()
		occupiedPath := filepath.Join(occupiedDir, "occupied")

		require.NoError(t, os.WriteFile(occupiedPath, []byte("in the way"), 0o600))

		colliding := record
		colliding.OriginalPath = occupiedPath

		mockStorer.EXPECT().
			ListRecords(ctx, storage.ListOptions{Limit: undoPageSize, Offset: 0}).
			Return([]storage.HistoryRecord{colliding}, nil)

		mockTrasher.EXPECT().
			IsInTrash(testTrashPath).
			Return(true)

		// The trash copy must not resolve, so the collision is reported rather
		// than mistaken for an already-restored file.
		_, trashErr := os.Stat(testTrashPath)
		require.Error(t, trashErr, "test trash path must not exist")

		result, err := manager.UndoMostRecent(ctx)

		require.ErrorIs(t, err, ErrRestoreCollision)
		assert.Nil(t, result)
	})

	t.Run("restore from trash fails", func(t *testing.T) {
		t.Parallel()

		manager, mockTrasher, mockStorer, _ := setupManagerTest(t)

		mockStorer.EXPECT().
			ListRecords(ctx, storage.ListOptions{Limit: undoPageSize, Offset: 0}).
			Return([]storage.HistoryRecord{record}, nil)

		mockTrasher.EXPECT().
			IsInTrash(testTrashPath).
			Return(true)

		mockTrasher.EXPECT().
			RestoreFromTrash(ctx, testTrashPath, testBinaryPath).
			Return(trash.ErrRestoreCollision)

		result, err := manager.UndoMostRecent(ctx)

		require.ErrorIs(t, err, ErrRestoreCollision)
		assert.Nil(t, result)
	})

	t.Run("list records fails", func(t *testing.T) {
		t.Parallel()

		manager, _, mockStorer, _ := setupManagerTest(t)

		mockStorer.EXPECT().
			ListRecords(ctx, storage.ListOptions{Limit: undoPageSize, Offset: 0}).
			Return(nil, errors.New("database unavailable"))

		result, err := manager.UndoMostRecent(ctx)

		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("walks pages until a restorable record is found", func(t *testing.T) {
		t.Parallel()

		manager, mockTrasher, mockStorer, _ := setupManagerTest(t)

		// A full first page of unrestorable entries forces a second page.
		firstPage := make([]storage.HistoryRecord, 0, undoPageSize)
		for i := range undoPageSize {
			entry := record
			entry.BinaryName = fmt.Sprintf("gone-%d", i)
			entry.TrashPath = fmt.Sprintf("/trash/gone-%d", i)
			entry.TrashAvailable = false
			firstPage = append(firstPage, entry)
		}

		secondPage := []storage.HistoryRecord{record}

		mockStorer.EXPECT().
			ListRecords(ctx, storage.ListOptions{Limit: undoPageSize, Offset: 0}).
			Return(firstPage, nil).
			Once()

		mockStorer.EXPECT().
			ListRecords(ctx, storage.ListOptions{Limit: undoPageSize, Offset: undoPageSize}).
			Return(secondPage, nil).
			Once()

		mockTrasher.EXPECT().
			IsInTrash(testTrashPath).
			Return(true).
			Once()

		mockTrasher.EXPECT().
			RestoreFromTrash(ctx, testTrashPath, testBinaryPath).
			Return(nil)

		mockStorer.EXPECT().
			UpdateRecord(ctx, mock.AnythingOfType("*storage.HistoryRecord")).
			Return(nil)

		result, err := manager.UndoMostRecent(ctx)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, testBinaryName, result.BinaryName)
	})
}

func TestHistoryManager_PendingAndChecksum(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	now := time.Now()

	t.Run("interrupted deletion is not reported as already restored", func(t *testing.T) {
		t.Parallel()

		manager, _, _, _ := setupManagerTest(t)

		// A record persisted before its binary moved, and never updated.
		pending := storage.HistoryRecord{
			Timestamp:      now.Unix(),
			BinaryName:     testBinaryName,
			OriginalPath:   testBinaryPath,
			TrashPath:      "",
			TrashAvailable: false,
		}

		result, err := manager.restoreRecord(ctx, &pending)

		// Nothing was ever restored, so this must not claim otherwise.
		require.ErrorIs(t, err, ErrRecoveryRequired)
		require.NotErrorIs(t, err, ErrAlreadyRestored)
		assert.Nil(t, result)
	})

	t.Run("pending deletion is flagged in the listing", func(t *testing.T) {
		t.Parallel()

		manager, _, mockStorer, _ := setupManagerTest(t)

		pending := storage.HistoryRecord{
			Timestamp:      now.Unix(),
			BinaryName:     testBinaryName,
			OriginalPath:   testBinaryPath,
			TrashPath:      "",
			TrashAvailable: false,
		}
		restored := storage.HistoryRecord{
			Timestamp:      now.Unix() - 10,
			BinaryName:     "restored",
			OriginalPath:   "/usr/local/bin/restored",
			TrashPath:      "/trash/files/restored_x",
			TrashAvailable: false,
		}

		mockStorer.EXPECT().
			ListRecords(ctx, storage.ListOptions{Limit: 10}).
			Return([]storage.HistoryRecord{pending, restored}, nil)

		entries, err := manager.GetHistory(ctx, 10)

		require.NoError(t, err)
		require.Len(t, entries, 2)

		// The interrupted deletion is distinguishable from a real restore.
		assert.True(t, entries[0].Pending, "an interrupted deletion must be flagged")
		assert.False(t, entries[1].Pending, "a completed restore is not pending")
	})

	t.Run("checksum mismatch refuses to restore", func(t *testing.T) {
		t.Parallel()

		manager, mockTrasher, mockStorer, mockExtractor := setupManagerTest(t)

		record := storage.HistoryRecord{
			Timestamp:      now.Unix(),
			BinaryName:     testBinaryName,
			OriginalPath:   testBinaryPath,
			TrashPath:      testTrashPath,
			Checksum:       testChecksum,
			TrashAvailable: true,
		}

		mockStorer.EXPECT().
			GetRecord(ctx, record.RecordKey()).
			Return(record, nil)

		mockTrasher.EXPECT().
			IsInTrash(testTrashPath).
			Return(true)

		// The trash copy no longer matches what was recorded at deletion.
		mockExtractor.EXPECT().
			CalculateChecksum(testTrashPath).
			Return("0000", nil)

		result, err := manager.Restore(ctx, record.RecordKey())

		require.ErrorIs(t, err, ErrChecksumMismatch)
		assert.Nil(t, result)

		// The copy must be left in trash rather than restored as damaged bytes,
		// and the record must be left claiming it is still recoverable.
		mockTrasher.AssertNotCalled(
			t,
			"RestoreFromTrash",
			mock.Anything,
			mock.Anything,
			mock.Anything,
		)
		mockStorer.AssertNotCalled(t, "UpdateRecord", mock.Anything, mock.Anything)
	})

	t.Run("matching checksum restores", func(t *testing.T) {
		t.Parallel()

		manager, mockTrasher, mockStorer, mockExtractor := setupManagerTest(t)

		record := storage.HistoryRecord{
			Timestamp:      now.Unix(),
			BinaryName:     testBinaryName,
			OriginalPath:   testBinaryPath,
			TrashPath:      testTrashPath,
			Checksum:       testChecksum,
			TrashAvailable: true,
		}

		mockStorer.EXPECT().
			GetRecord(ctx, record.RecordKey()).
			Return(record, nil)

		mockTrasher.EXPECT().
			IsInTrash(testTrashPath).
			Return(true)

		mockExtractor.EXPECT().
			CalculateChecksum(testTrashPath).
			Return(testChecksum, nil)

		mockTrasher.EXPECT().
			RestoreFromTrash(ctx, testTrashPath, testBinaryPath).
			Return(nil)

		mockStorer.EXPECT().
			UpdateRecord(ctx, mock.AnythingOfType("*storage.HistoryRecord")).
			Return(nil)

		result, err := manager.Restore(ctx, record.RecordKey())

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, testBinaryPath, result.RestoredTo)
	})

	t.Run("unreadable trash copy is not reported as a mismatch", func(t *testing.T) {
		t.Parallel()

		manager, mockTrasher, mockStorer, mockExtractor := setupManagerTest(t)

		record := storage.HistoryRecord{
			Timestamp:      now.Unix(),
			BinaryName:     testBinaryName,
			OriginalPath:   testBinaryPath,
			TrashPath:      testTrashPath,
			Checksum:       testChecksum,
			TrashAvailable: true,
		}

		mockStorer.EXPECT().
			GetRecord(ctx, record.RecordKey()).
			Return(record, nil)

		mockTrasher.EXPECT().
			IsInTrash(testTrashPath).
			Return(true)

		readErr := errors.New("permission denied")
		mockExtractor.EXPECT().
			CalculateChecksum(testTrashPath).
			Return("", readErr)

		result, err := manager.Restore(ctx, record.RecordKey())

		// The underlying cause must survive, and the failure must not be
		// mislabelled as corrupt content.
		require.ErrorIs(t, err, readErr)
		require.NotErrorIs(t, err, ErrChecksumMismatch)

		// The copy cannot be verified, so it stays in trash.
		mockTrasher.AssertNotCalled(
			t,
			"RestoreFromTrash",
			mock.Anything,
			mock.Anything,
			mock.Anything,
		)
		assert.Nil(t, result)
	})
}

func TestHistoryManager_Restore(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	now := time.Now()

	record := storage.HistoryRecord{
		Timestamp:      now.Unix(),
		BinaryName:     testBinaryName,
		OriginalPath:   testBinaryPath,
		TrashPath:      testTrashPath,
		ModulePath:     testModulePath,
		Version:        testVersion,
		TrashAvailable: true,
	}

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		manager, mockTrasher, mockStorer, _ := setupManagerTest(t)

		mockStorer.EXPECT().
			GetRecord(ctx, testEntryID).
			Return(record, nil)

		mockTrasher.EXPECT().
			IsInTrash(testTrashPath).
			Return(true)

		mockTrasher.EXPECT().
			RestoreFromTrash(ctx, testTrashPath, testBinaryPath).
			Return(nil)

		mockStorer.EXPECT().
			UpdateRecord(ctx, mock.AnythingOfType("*storage.HistoryRecord")).
			Return(nil)

		result, err := manager.Restore(ctx, testEntryID)

		require.NoError(t, err)
		assert.NotNil(t, result)
		// EntryID should match the generated key from the record
		assert.Equal(t, storage.GenerateKey(record.Timestamp, record.BinaryName), result.EntryID)
		assert.Equal(t, testBinaryName, result.BinaryName)
	})

	t.Run("entry not found", func(t *testing.T) {
		t.Parallel()

		manager, _, mockStorer, _ := setupManagerTest(t)

		mockStorer.EXPECT().
			GetRecord(ctx, testEntryID).
			Return(storage.HistoryRecord{}, storage.ErrRecordNotFound)

		result, err := manager.Restore(ctx, testEntryID)

		require.ErrorIs(t, err, ErrEntryNotFound)
		assert.Nil(t, result)
	})

	t.Run("already restored", func(t *testing.T) {
		t.Parallel()

		manager, _, mockStorer, _ := setupManagerTest(t)

		restoredRecord := record
		restoredRecord.TrashAvailable = false

		mockStorer.EXPECT().
			GetRecord(ctx, testEntryID).
			Return(restoredRecord, nil)

		result, err := manager.Restore(ctx, testEntryID)

		require.ErrorIs(t, err, ErrAlreadyRestored)
		assert.Nil(t, result)
	})
}

func TestHistoryManager_GetHistory(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	now := time.Now()

	records := []storage.HistoryRecord{
		{
			Timestamp:      now.Unix(),
			BinaryName:     "binary1",
			OriginalPath:   "/usr/local/bin/binary1",
			TrashAvailable: true,
		},
		{
			Timestamp:      now.Add(-time.Hour).Unix(),
			BinaryName:     "binary2",
			OriginalPath:   "/usr/local/bin/binary2",
			TrashAvailable: false,
		},
	}

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		manager, mockTrasher, mockStorer, _ := setupManagerTest(t)

		mockStorer.EXPECT().
			ListRecords(ctx, storage.ListOptions{Limit: 10}).
			Return(records, nil)

		// InTrash reflects the live trash where the platform can report it.
		if trashStateQueryable() {
			mockTrasher.EXPECT().IsInTrash("").Return(true)
		}

		entries, err := manager.GetHistory(ctx, 10)

		require.NoError(t, err)
		assert.Len(t, entries, 2)
		assert.Equal(t, "binary1", entries[0].BinaryName)
		assert.True(t, entries[0].InTrash)
		assert.Equal(t, "binary2", entries[1].BinaryName)
		assert.False(t, entries[1].InTrash,
			"a record already marked unavailable must not be probed")
	})

	t.Run("preserves the stored flag where the trash cannot report", func(t *testing.T) {
		t.Parallel()

		if trashStateQueryable() {
			t.Skip("platform trash can report membership, so the stored flag is not authoritative")
		}

		manager, mockTrasher, mockStorer, _ := setupManagerTest(t)

		mockStorer.EXPECT().
			ListRecords(ctx, storage.ListOptions{Limit: 10}).
			Return(records, nil)

		entries, err := manager.GetHistory(ctx, 10)

		require.NoError(t, err)
		require.Len(t, entries, 2)
		assert.True(t, entries[0].InTrash,
			"a stored flag must survive where membership cannot be queried")
		assert.False(t, entries[1].InTrash)

		// Asking anyway would report every entry as absent.
		mockTrasher.AssertNotCalled(t, "IsInTrash", mock.Anything)
	})

	t.Run("stale flag is reported as not in trash", func(t *testing.T) {
		t.Parallel()

		if !trashStateQueryable() {
			t.Skip("platform trash cannot report membership, so a stale flag is undetectable")
		}

		manager, mockTrasher, mockStorer, _ := setupManagerTest(t)

		stale := []storage.HistoryRecord{
			{
				Timestamp:      now.Unix(),
				BinaryName:     "purged",
				OriginalPath:   "/usr/local/bin/purged",
				TrashPath:      "/trash/files/purged",
				TrashAvailable: true,
			},
		}

		mockStorer.EXPECT().
			ListRecords(ctx, storage.ListOptions{Limit: 10}).
			Return(stale, nil)

		// The record claims to be in trash, but the system trash no longer has it.
		mockTrasher.EXPECT().IsInTrash("/trash/files/purged").Return(false)

		entries, err := manager.GetHistory(ctx, 10)

		require.NoError(t, err)
		require.Len(t, entries, 1)
		assert.False(t, entries[0].InTrash,
			"a stale availability flag must not be reported as in trash")
	})

	t.Run("empty history", func(t *testing.T) {
		t.Parallel()

		manager, _, mockStorer, _ := setupManagerTest(t)

		mockStorer.EXPECT().
			ListRecords(ctx, storage.ListOptions{Limit: 10}).
			Return([]storage.HistoryRecord{}, nil)

		entries, err := manager.GetHistory(ctx, 10)

		require.NoError(t, err)
		assert.Empty(t, entries)
	})

	t.Run("list records fails", func(t *testing.T) {
		t.Parallel()

		manager, _, mockStorer, _ := setupManagerTest(t)

		mockStorer.EXPECT().
			ListRecords(ctx, storage.ListOptions{Limit: 10}).
			Return(nil, storage.ErrDatabaseClosed)

		entries, err := manager.GetHistory(ctx, 10)

		require.Error(t, err)
		assert.Nil(t, entries)
	})
}

func TestHistoryManager_DeletePermanently(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	now := time.Now()

	record := storage.HistoryRecord{
		Timestamp:      now.Unix(),
		BinaryName:     testBinaryName,
		TrashPath:      testTrashPath,
		TrashAvailable: true,
	}

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		manager, mockTrasher, mockStorer, _ := setupManagerTest(t)

		mockStorer.EXPECT().
			GetRecord(ctx, testEntryID).
			Return(record, nil)

		mockTrasher.EXPECT().
			IsInTrash(testTrashPath).
			Return(true)

		mockTrasher.EXPECT().
			DeletePermanently(ctx, testTrashPath).
			Return(nil)

		mockStorer.EXPECT().
			DeleteRecord(ctx, testEntryID).
			Return(nil)

		err := manager.DeletePermanently(ctx, testEntryID)

		require.NoError(t, err)
	})

	t.Run("entry not found", func(t *testing.T) {
		t.Parallel()

		manager, _, mockStorer, _ := setupManagerTest(t)

		mockStorer.EXPECT().
			GetRecord(ctx, testEntryID).
			Return(storage.HistoryRecord{}, storage.ErrRecordNotFound)

		err := manager.DeletePermanently(ctx, testEntryID)

		require.ErrorIs(t, err, ErrEntryNotFound)
	})

	t.Run("delete from trash fails but continues", func(t *testing.T) {
		t.Parallel()

		manager, mockTrasher, mockStorer, _ := setupManagerTest(t)

		mockStorer.EXPECT().
			GetRecord(ctx, testEntryID).
			Return(record, nil)

		mockTrasher.EXPECT().
			IsInTrash(testTrashPath).
			Return(true)

		mockTrasher.EXPECT().
			DeletePermanently(ctx, testTrashPath).
			Return(trash.ErrFileNotInTrash)

		mockStorer.EXPECT().
			DeleteRecord(ctx, testEntryID).
			Return(nil)

		err := manager.DeletePermanently(ctx, testEntryID)

		require.NoError(t, err)
	})

	t.Run("not in trash skips trash deletion", func(t *testing.T) {
		t.Parallel()

		manager, mockTrasher, mockStorer, _ := setupManagerTest(t)

		notInTrashRecord := record
		notInTrashRecord.TrashAvailable = false

		mockStorer.EXPECT().
			GetRecord(ctx, testEntryID).
			Return(notInTrashRecord, nil)

		mockStorer.EXPECT().
			DeleteRecord(ctx, testEntryID).
			Return(nil)

		err := manager.DeletePermanently(ctx, testEntryID)

		require.NoError(t, err)
		mockTrasher.AssertNotCalled(t, "DeletePermanently")
	})
}

func TestHistoryManager_ClearHistory(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	now := time.Now()

	records := []storage.HistoryRecord{
		{
			Timestamp:      now.Unix(),
			BinaryName:     "binary1",
			TrashPath:      "/trash/binary1",
			TrashAvailable: true,
		},
		{
			Timestamp:      now.Add(-time.Hour).Unix(),
			BinaryName:     "binary2",
			TrashPath:      "/trash/binary2",
			TrashAvailable: true,
		},
	}

	t.Run("clear history only", func(t *testing.T) {
		t.Parallel()

		manager, _, mockStorer, _ := setupManagerTest(t)

		mockStorer.EXPECT().
			DeleteAllRecords(ctx).
			Return(nil)

		err := manager.ClearHistory(ctx, false)

		require.NoError(t, err)
	})

	t.Run("clear history and trash", func(t *testing.T) {
		t.Parallel()

		manager, mockTrasher, mockStorer, _ := setupManagerTest(t)

		mockStorer.EXPECT().
			ListRecords(ctx, storage.ListOptions{}).
			Return(records, nil)

		mockTrasher.EXPECT().
			IsInTrash("/trash/binary1").
			Return(true)

		mockTrasher.EXPECT().
			DeletePermanently(ctx, "/trash/binary1").
			Return(nil)

		mockTrasher.EXPECT().
			IsInTrash("/trash/binary2").
			Return(true)

		mockTrasher.EXPECT().
			DeletePermanently(ctx, "/trash/binary2").
			Return(nil)

		mockStorer.EXPECT().
			DeleteAllRecords(ctx).
			Return(nil)

		err := manager.ClearHistory(ctx, true)

		require.NoError(t, err)
	})

	t.Run("list records fails", func(t *testing.T) {
		t.Parallel()

		manager, _, mockStorer, _ := setupManagerTest(t)

		mockStorer.EXPECT().
			ListRecords(ctx, storage.ListOptions{}).
			Return(nil, storage.ErrDatabaseClosed)

		err := manager.ClearHistory(ctx, true)

		require.Error(t, err)
	})
}

func TestHistoryManager_ClearEntry(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	now := time.Now()

	record := storage.HistoryRecord{
		Timestamp:      now.Unix(),
		BinaryName:     testBinaryName,
		TrashPath:      testTrashPath,
		TrashAvailable: true,
	}

	t.Run("clear entry only", func(t *testing.T) {
		t.Parallel()

		manager, _, mockStorer, _ := setupManagerTest(t)

		mockStorer.EXPECT().
			GetRecord(ctx, testEntryID).
			Return(record, nil)

		mockStorer.EXPECT().
			DeleteRecord(ctx, testEntryID).
			Return(nil)

		err := manager.ClearEntry(ctx, testEntryID, false)

		require.NoError(t, err)
	})

	t.Run("clear entry and delete from trash", func(t *testing.T) {
		t.Parallel()

		manager, mockTrasher, mockStorer, _ := setupManagerTest(t)

		mockStorer.EXPECT().
			GetRecord(ctx, testEntryID).
			Return(record, nil)

		mockTrasher.EXPECT().
			IsInTrash(testTrashPath).
			Return(true)

		mockTrasher.EXPECT().
			DeletePermanently(ctx, testTrashPath).
			Return(nil)

		mockStorer.EXPECT().
			DeleteRecord(ctx, testEntryID).
			Return(nil)

		err := manager.ClearEntry(ctx, testEntryID, true)

		require.NoError(t, err)
	})

	t.Run("entry not found", func(t *testing.T) {
		t.Parallel()

		manager, _, mockStorer, _ := setupManagerTest(t)

		mockStorer.EXPECT().
			GetRecord(ctx, testEntryID).
			Return(storage.HistoryRecord{}, storage.ErrRecordNotFound)

		err := manager.ClearEntry(ctx, testEntryID, false)

		require.ErrorIs(t, err, ErrEntryNotFound)
	})

	t.Run("delete from trash fails", func(t *testing.T) {
		t.Parallel()

		manager, mockTrasher, mockStorer, _ := setupManagerTest(t)

		mockStorer.EXPECT().
			GetRecord(ctx, testEntryID).
			Return(record, nil)

		mockTrasher.EXPECT().
			IsInTrash(testTrashPath).
			Return(true)

		mockTrasher.EXPECT().
			DeletePermanently(ctx, testTrashPath).
			Return(errors.New("delete failed"))

		err := manager.ClearEntry(ctx, testEntryID, true)

		require.Error(t, err)
	})
}

func TestHistoryManager_Close(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		manager, _, mockStorer, _ := setupManagerTest(t)

		mockStorer.EXPECT().
			Close().
			Return(nil)

		err := manager.Close()

		require.NoError(t, err)
	})

	t.Run("close fails", func(t *testing.T) {
		t.Parallel()

		manager, _, mockStorer, _ := setupManagerTest(t)

		mockStorer.EXPECT().
			Close().
			Return(storage.ErrDatabaseClosed)

		err := manager.Close()

		require.Error(t, err)
	})
}

func TestEntryFromRecord(t *testing.T) {
	t.Parallel()

	now := time.Now()

	record := storage.HistoryRecord{
		Timestamp:      now.Unix(),
		BinaryName:     testBinaryName,
		OriginalPath:   testBinaryPath,
		ModulePath:     testModulePath,
		Version:        testVersion,
		VCSRevision:    "abc123",
		TrashAvailable: true,
	}

	entry := entryFromRecord(&record)

	assert.Equal(t, storage.GenerateKey(record.Timestamp, record.BinaryName), entry.ID)
	assert.Equal(t, now.Unix(), entry.Timestamp.Unix())
	assert.Equal(t, testBinaryName, entry.BinaryName)
	assert.Equal(t, testBinaryPath, entry.BinaryPath)
	assert.Equal(t, testModulePath, entry.ModulePath)
	assert.Equal(t, testVersion, entry.Version)
	assert.Equal(t, "abc123", entry.VCSRevision)
	assert.True(t, entry.InTrash)
}

func TestEntriesFromRecords(t *testing.T) {
	t.Parallel()

	now := time.Now()

	records := []storage.HistoryRecord{
		{
			Timestamp:      now.Unix(),
			BinaryName:     "binary1",
			OriginalPath:   "/usr/local/bin/binary1",
			TrashAvailable: true,
		},
		{
			Timestamp:      now.Add(-time.Hour).Unix(),
			BinaryName:     "binary2",
			OriginalPath:   "/usr/local/bin/binary2",
			TrashAvailable: false,
		},
	}

	entries := entriesFromRecords(records)

	require.Len(t, entries, 2)
	assert.Equal(t, "binary1", entries[0].BinaryName)
	assert.True(t, entries[0].InTrash)
	assert.Equal(t, "binary2", entries[1].BinaryName)
	assert.False(t, entries[1].InTrash)
}
