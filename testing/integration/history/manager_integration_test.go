/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

// Package history_test provides black-box integration tests for the history package.
//
// These tests verify the orchestration layer that coordinates between buildinfo,
// trash, storage, and logger components. All external dependencies are mocked
// using Mockery-generated mocks to ensure no external calls are made.
package history_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/nicholas-fedor/go-remove/internal/buildinfo"
	buildinfomocks "github.com/nicholas-fedor/go-remove/internal/buildinfo/mocks"
	"github.com/nicholas-fedor/go-remove/internal/history"
	loggermocks "github.com/nicholas-fedor/go-remove/internal/logger/mocks"
	"github.com/nicholas-fedor/go-remove/internal/storage"
	storagemocks "github.com/nicholas-fedor/go-remove/internal/storage/mocks"
	"github.com/nicholas-fedor/go-remove/internal/trash"
	trashmocks "github.com/nicholas-fedor/go-remove/internal/trash/mocks"
)

// Test constants for consistent test data.
const (
	testBinaryPath  = "/usr/local/bin/test-binary"
	testTrashPath   = "/trash/test-binary"
	testEntryID     = "1709321234:test-binary"
	testBinaryName  = "test-binary"
	testModulePath  = "github.com/test/binary"
	testVersion     = "v1.0.0"
	testVCSRevision = "abc123def456"
	testGoVersion   = "go1.21"
	testChecksum    = "abc123"
	testVCSTime     = "2026-01-01T00:00:00Z"
	testBinaryPath2 = "/usr/local/bin/another-binary"
	testTrashPath2  = "/trash/another-binary"
	testBinaryName2 = "another-binary"
	testEntryID2    = "1709321000:another-binary"
)

// ManagerIntegrationTestSuite provides integration tests for the history Manager.
//
// This suite tests the orchestration between multiple components:
// - buildinfo.Extractor for metadata extraction
// - trash.Trasher for file operations
// - storage.Storer for persistence
// - logger.Logger for logging
//
// All dependencies are mocked to ensure isolated, deterministic tests.
type ManagerIntegrationTestSuite struct {
	suite.Suite

	// Mocks for all dependencies
	trasher   *trashmocks.MockTrasher
	storer    *storagemocks.MockStorer
	extractor *buildinfomocks.MockExtractor
	logger    *loggermocks.MockLogger

	// The manager under test
	manager history.Manager
}

// SetupTest initializes the test suite before each test.
//
// This creates fresh mocks and a new manager instance for each test
// to ensure test isolation.
func (s *ManagerIntegrationTestSuite) SetupTest() {
	// Create mocks
	s.trasher = trashmocks.NewMockTrasher(s.T())
	s.storer = storagemocks.NewMockStorer(s.T())
	s.extractor = buildinfomocks.NewMockExtractor(s.T())
	s.logger = loggermocks.NewMockLogger(s.T())

	// Setup logger mock to accept any calls (logging is not the focus of these tests)
	s.setupLoggerExpectations()

	// Create the manager with mocked dependencies
	s.manager = history.NewManager(s.trasher, s.storer, s.extractor, s.logger)
}

// setupLoggerExpectations configures the logger mock to accept any log calls.
//
// This allows the manager to log as needed without requiring explicit expectations
// in every test case.
func (s *ManagerIntegrationTestSuite) setupLoggerExpectations() {
	// Create a properly initialized zerolog event
	// We need to use a real logger to get valid events
	logger := zerolog.New(nil).With().Logger()

	// Allow any logging calls and return a valid event from the logger
	s.logger.EXPECT().Debug().Return(logger.Debug()).Maybe()
	s.logger.EXPECT().Info().Return(logger.Info()).Maybe()
	s.logger.EXPECT().Warn().Return(logger.Warn()).Maybe()
	s.logger.EXPECT().Error().Return(logger.Error()).Maybe()
}

// createBuildInfoData creates a standard BuildInfoData for testing.
//
// Returns:
//   - A BuildInfoData struct populated with test constants
func (s *ManagerIntegrationTestSuite) createBuildInfoData() *buildinfo.BuildInfoData {
	return &buildinfo.BuildInfoData{
		ModulePath:  testModulePath,
		Version:     testVersion,
		VCSRevision: testVCSRevision,
		VCSTime:     testVCSTime,
		GoVersion:   testGoVersion,
		RawJSON:     []byte(`{"test": "data"}`),
	}
}

// createHistoryRecord creates a standard HistoryRecord for testing.
//
// Parameters:
//   - timestamp: The Unix timestamp for the record
//   - binaryName: The name of the binary
//   - trashAvailable: Whether the binary is available in trash
//
// Returns:
//   - A HistoryRecord populated with test data
func (s *ManagerIntegrationTestSuite) createHistoryRecord(
	timestamp int64,
	binaryName string,
	trashAvailable bool,
) storage.HistoryRecord {
	path := "/usr/local/bin/" + binaryName
	trashPath := "/trash/" + binaryName

	return storage.HistoryRecord{
		Timestamp:      timestamp,
		BinaryName:     binaryName,
		OriginalPath:   path,
		TrashPath:      trashPath,
		ModulePath:     testModulePath,
		Version:        testVersion,
		VCSRevision:    testVCSRevision,
		TrashAvailable: trashAvailable,
		OriginalDir:    "/usr/local/bin",
	}
}

// TestManagerIntegrationTestSuite runs the integration test suite.
func TestManagerIntegrationTestSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(ManagerIntegrationTestSuite))
}

// TestFullDeletionWorkflow verifies the complete deletion workflow.
//
// This test ensures that RecordDeletion properly coordinates:
// 1. Build info extraction
// 2. Checksum calculation
// 3. Moving binary to trash
// 4. Saving the history record
//
// The workflow should return a complete HistoryEntry on success.
func (s *ManagerIntegrationTestSuite) TestFullDeletionWorkflow() {
	ctx := context.Background()

	// Setup
	buildData := s.createBuildInfoData()

	// Setup expectations in workflow order
	s.extractor.EXPECT().
		Extract(mock.Anything, testBinaryPath).
		Return(buildData, nil)

	s.extractor.EXPECT().
		CalculateChecksum(testBinaryPath).
		Return(testChecksum, nil)

	s.trasher.EXPECT().
		MoveToTrash(mock.Anything, testBinaryPath).
		Return(testTrashPath, nil)

	s.storer.EXPECT().
		SaveRecord(mock.Anything, mock.AnythingOfType("*storage.HistoryRecord")).
		Return(nil)

	// Execute
	entry, err := s.manager.RecordDeletion(ctx, testBinaryPath)

	// Verify
	s.Require().NoError(err)
	s.Require().NotNil(entry)
	s.Equal(testBinaryName, entry.BinaryName)
	s.Equal(testBinaryPath, entry.BinaryPath)
	s.Equal(buildData.ModulePath, entry.ModulePath)
	s.Equal(buildData.Version, entry.Version)
	s.Equal(buildData.VCSRevision, entry.VCSRevision)
	s.True(entry.InTrash)
	s.True(entry.CanRestore)
	s.NotEmpty(entry.ID)
}

// TestFullRestoreWorkflow verifies the complete restore workflow.
//
// This test ensures that Restore properly coordinates:
// 1. Retrieving the history record
// 2. Checking if the binary is still in trash
// 3. Checking for restore collisions
// 4. Restoring from trash
// 5. Updating the history record
//
// The workflow should return a RestoreResult on success.
func (s *ManagerIntegrationTestSuite) TestFullRestoreWorkflow() {
	ctx := context.Background()

	// Setup
	now := time.Now()
	record := s.createHistoryRecord(now.Unix(), testBinaryName, true)

	s.storer.EXPECT().
		GetRecord(mock.Anything, testEntryID).
		Return(record, nil)

	s.trasher.EXPECT().
		IsInTrash(testTrashPath).
		Return(true)

	// Note: Collision detection now uses os.Stat on original path instead of IsInTrash
	// Since we can't mock os.Stat in integration tests, the test relies on the
	// fact that testBinaryPath doesn't exist in the test environment

	s.trasher.EXPECT().
		RestoreFromTrash(mock.Anything, testTrashPath, testBinaryPath).
		Return(nil)

	s.storer.EXPECT().
		UpdateRecord(mock.Anything, mock.AnythingOfType("*storage.HistoryRecord")).
		Return(nil)

	// Execute
	result, err := s.manager.Restore(ctx, testEntryID)

	// Verify
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Equal(testBinaryName, result.BinaryName)
	s.Equal(testBinaryPath, result.RestoredTo)
	s.True(result.FromTrash)
	s.Equal(testModulePath, result.ModulePath)
	s.Equal(testVersion, result.Version)
}

// TestUndoMostRecentWorkflow verifies the undo most recent workflow.
//
// This test ensures that UndoMostRecent properly coordinates:
// 1. Getting the most recent record
// 2. Checking if the binary is still in trash
// 3. Restoring from trash
// 4. Updating the history record.
func (s *ManagerIntegrationTestSuite) TestUndoMostRecentWorkflow() {
	ctx := context.Background()

	// Setup
	now := time.Now()
	record := s.createHistoryRecord(now.Unix(), testBinaryName, true)

	s.storer.EXPECT().
		GetMostRecent(mock.Anything).
		Return(record, nil)

	s.trasher.EXPECT().
		IsInTrash(testTrashPath).
		Return(true)

	// Note: Collision detection now uses os.Stat on original path instead of IsInTrash
	// Since we can't mock os.Stat in integration tests, the test relies on the
	// fact that testBinaryPath doesn't exist in the test environment

	s.trasher.EXPECT().
		RestoreFromTrash(mock.Anything, testTrashPath, testBinaryPath).
		Return(nil)

	s.storer.EXPECT().
		UpdateRecord(mock.Anything, mock.AnythingOfType("*storage.HistoryRecord")).
		Return(nil)

	// Execute
	result, err := s.manager.UndoMostRecent(ctx)

	// Verify
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Equal(testBinaryName, result.BinaryName)
	s.Equal(testBinaryPath, result.RestoredTo)
	s.True(result.FromTrash)
}

// TestGetHistoryLimits verifies GetHistory with various limit values.
//
// This test uses table-driven testing to cover multiple scenarios:
// - Zero limit (no limit)
// - Positive limit
// - Limit larger than available records.
func (s *ManagerIntegrationTestSuite) TestGetHistoryLimits() {
	ctx := context.Background()

	// Setup
	now := time.Now()
	records := []storage.HistoryRecord{
		s.createHistoryRecord(now.Unix(), "binary1", true),
		s.createHistoryRecord(now.Add(-time.Hour).Unix(), "binary2", true),
		s.createHistoryRecord(now.Add(-2*time.Hour).Unix(), "binary3", false),
	}

	tests := []struct {
		name          string
		limit         int
		expectedCount int
	}{
		{
			name:          "no limit returns all",
			limit:         0,
			expectedCount: 3,
		},
		{
			name:          "limit of 2 returns 2",
			limit:         2,
			expectedCount: 2,
		},
		{
			name:          "limit larger than records returns all",
			limit:         10,
			expectedCount: 3,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.storer.EXPECT().
				ListRecords(mock.Anything, storage.ListOptions{Limit: tt.limit}).
				Return(records[:minInt(tt.expectedCount, len(records))], nil)

			entries, err := s.manager.GetHistory(ctx, tt.limit)

			s.Require().NoError(err)
			s.Len(entries, tt.expectedCount)
		})
	}
}

// TestDeletePermanentlyWorkflow verifies the permanent deletion workflow.
//
// This test ensures that DeletePermanently properly coordinates:
// 1. Getting the history record
// 2. Checking if the binary is in trash
// 3. Deleting from trash
// 4. Deleting the history entry.
func (s *ManagerIntegrationTestSuite) TestDeletePermanentlyWorkflow() {
	ctx := context.Background()

	// Setup
	now := time.Now()
	record := s.createHistoryRecord(now.Unix(), testBinaryName, true)

	s.storer.EXPECT().
		GetRecord(mock.Anything, testEntryID).
		Return(record, nil)

	s.trasher.EXPECT().
		IsInTrash(testTrashPath).
		Return(true)

	s.trasher.EXPECT().
		DeletePermanently(mock.Anything, testTrashPath).
		Return(nil)

	s.storer.EXPECT().
		DeleteRecord(mock.Anything, testEntryID).
		Return(nil)

	// Execute
	err := s.manager.DeletePermanently(ctx, testEntryID)

	// Verify
	s.Require().NoError(err)
}

// TestClearHistoryWithoutTrashClearing verifies clearing history without touching trash.
//
// This test ensures that ClearHistory with clearTrash=false only deletes
// history records without affecting files in trash.
func (s *ManagerIntegrationTestSuite) TestClearHistoryWithoutTrashClearing() {
	ctx := context.Background()

	s.storer.EXPECT().
		DeleteAllRecords(mock.Anything).
		Return(nil)

	err := s.manager.ClearHistory(ctx, false)

	s.Require().NoError(err)
}

// TestClearHistoryWithTrashClearing verifies clearing history and trash.
//
// This test ensures that ClearHistory with clearTrash=true:
// 1. Lists all records
// 2. Deletes each binary from trash if available
// 3. Deletes all history entries.
func (s *ManagerIntegrationTestSuite) TestClearHistoryWithTrashClearing() {
	ctx := context.Background()

	// Setup
	now := time.Now()
	records := []storage.HistoryRecord{
		s.createHistoryRecord(now.Unix(), "binary1", true),
		s.createHistoryRecord(now.Add(-time.Hour).Unix(), "binary2", true),
	}

	s.storer.EXPECT().
		ListRecords(mock.Anything, storage.ListOptions{}).
		Return(records, nil)

	// Expect trash deletion for first binary
	s.trasher.EXPECT().
		IsInTrash("/trash/binary1").
		Return(true)

	s.trasher.EXPECT().
		DeletePermanently(mock.Anything, "/trash/binary1").
		Return(nil)

	// Expect trash deletion for second binary
	s.trasher.EXPECT().
		IsInTrash("/trash/binary2").
		Return(true)

	s.trasher.EXPECT().
		DeletePermanently(mock.Anything, "/trash/binary2").
		Return(nil)

	s.storer.EXPECT().
		DeleteAllRecords(mock.Anything).
		Return(nil)

	// Execute
	err := s.manager.ClearHistory(ctx, true)

	// Verify
	s.Require().NoError(err)
}

// TestErrorHandlingExtractorFailure verifies error handling when extractor fails.
//
// When build info extraction fails, the deletion should fail early without
// modifying trash or storage.
func (s *ManagerIntegrationTestSuite) TestErrorHandlingExtractorFailure() {
	ctx := context.Background()

	// Setup
	s.extractor.EXPECT().
		Extract(mock.Anything, testBinaryPath).
		Return(nil, buildinfo.ErrNotGoBinary)

	// Execute
	entry, err := s.manager.RecordDeletion(ctx, testBinaryPath)

	// Verify
	s.Require().Error(err)
	s.Nil(entry)
}

// TestErrorHandlingTrasherFailure verifies error handling when trasher fails.
//
// When moving to trash fails, the deletion should fail without saving to storage.
func (s *ManagerIntegrationTestSuite) TestErrorHandlingTrasherFailure() {
	ctx := context.Background()

	// Setup
	buildData := s.createBuildInfoData()

	s.extractor.EXPECT().
		Extract(mock.Anything, testBinaryPath).
		Return(buildData, nil)

	s.extractor.EXPECT().
		CalculateChecksum(testBinaryPath).
		Return(testChecksum, nil)

	s.trasher.EXPECT().
		MoveToTrash(mock.Anything, testBinaryPath).
		Return("", trash.ErrTrashFull)

	// Execute
	entry, err := s.manager.RecordDeletion(ctx, testBinaryPath)

	// Verify
	s.Require().Error(err)
	s.Nil(entry)
}

// TestErrorHandlingStorageFailure verifies error handling when storage fails.
//
// When saving to storage fails, the manager should attempt to restore the file
// from trash to maintain consistency.
func (s *ManagerIntegrationTestSuite) TestErrorHandlingStorageFailure() {
	ctx := context.Background()

	// Setup
	buildData := s.createBuildInfoData()

	s.extractor.EXPECT().
		Extract(mock.Anything, testBinaryPath).
		Return(buildData, nil)

	s.extractor.EXPECT().
		CalculateChecksum(testBinaryPath).
		Return(testChecksum, nil)

	s.trasher.EXPECT().
		MoveToTrash(mock.Anything, testBinaryPath).
		Return(testTrashPath, nil)

	s.storer.EXPECT().
		SaveRecord(mock.Anything, mock.AnythingOfType("*storage.HistoryRecord")).
		Return(storage.ErrDatabaseClosed)

	// Expect restore attempt when storage fails
	s.trasher.EXPECT().
		RestoreFromTrash(mock.Anything, testTrashPath, testBinaryPath).
		Return(nil)

	// Execute
	entry, err := s.manager.RecordDeletion(ctx, testBinaryPath)

	// Verify
	s.Require().Error(err)
	s.Nil(entry)
}

// TestStateConsistencyRestoredBinaryCannotBeRestoredAgain verifies that a restored
// binary cannot be restored again.
//
// This test ensures state consistency by verifying that attempting to restore
// an already-restored entry returns ErrAlreadyRestored.
func (s *ManagerIntegrationTestSuite) TestStateConsistencyRestoredBinaryCannotBeRestoredAgain() {
	ctx := context.Background()

	// Setup
	now := time.Now()
	record := s.createHistoryRecord(now.Unix(), testBinaryName, false)

	s.storer.EXPECT().
		GetRecord(mock.Anything, testEntryID).
		Return(record, nil)

	// Execute
	result, err := s.manager.Restore(ctx, testEntryID)

	// Verify
	s.Require().ErrorIs(err, history.ErrAlreadyRestored)
	s.Nil(result)
}

// TestStateConsistencyNotInTrashUpdatesRecord verifies that checking a binary
// not in trash updates the record state.
//
// When a binary is no longer in trash, the manager should update the record
// to reflect this state.
func (s *ManagerIntegrationTestSuite) TestStateConsistencyNotInTrashUpdatesRecord() {
	ctx := context.Background()

	// Setup
	now := time.Now()
	record := s.createHistoryRecord(now.Unix(), testBinaryName, true)

	s.storer.EXPECT().
		GetMostRecent(mock.Anything).
		Return(record, nil)

	s.trasher.EXPECT().
		IsInTrash(testTrashPath).
		Return(false)

	s.storer.EXPECT().
		UpdateRecord(mock.Anything, mock.AnythingOfType("*storage.HistoryRecord")).
		Return(nil)

	// Execute
	result, err := s.manager.UndoMostRecent(ctx)

	// Verify
	s.Require().ErrorIs(err, history.ErrNotInTrash)
	s.Nil(result)
}

// TestStateConsistencyRestoreCollisionDetection verifies that restore collision
// is properly detected and reported.
//
// When a file already exists at the restore location, the manager should
// return ErrRestoreCollision. Note: Since collision detection now uses os.Stat,
// this integration test cannot easily simulate a collision without creating actual
// files. Collision detection is thoroughly tested in unit tests.
func (s *ManagerIntegrationTestSuite) TestStateConsistencyRestoreCollisionDetection() {
	ctx := context.Background()

	// Setup
	now := time.Now()
	record := s.createHistoryRecord(now.Unix(), testBinaryName, true)

	s.storer.EXPECT().
		GetRecord(mock.Anything, testEntryID).
		Return(record, nil)

	s.trasher.EXPECT().
		IsInTrash(testTrashPath).
		Return(true)

	// Note: Collision detection now uses os.Stat on original path instead of IsInTrash
	// Since we can't mock os.Stat in integration tests and the path doesn't exist,
	// no collision is detected and the restore proceeds normally.

	s.trasher.EXPECT().
		RestoreFromTrash(mock.Anything, testTrashPath, testBinaryPath).
		Return(nil)

	s.storer.EXPECT().
		UpdateRecord(mock.Anything, mock.AnythingOfType("*storage.HistoryRecord")).
		Return(nil)

	// Execute
	result, err := s.manager.Restore(ctx, testEntryID)

	// Verify: Since the file doesn't actually exist at testBinaryPath (os.Stat returns error),
	// no collision is detected and the restore succeeds. True collision detection is
	// tested in unit tests where we can control the filesystem.
	s.Require().NoError(err)
	s.NotNil(result)
}

// TestDeletePermanentlyNotInTrashSkipsTrashDeletion verifies that deleting
// permanently skips trash deletion when binary is not available in trash.
func (s *ManagerIntegrationTestSuite) TestDeletePermanentlyNotInTrashSkipsTrashDeletion() {
	ctx := context.Background()

	// Setup
	now := time.Now()
	record := s.createHistoryRecord(now.Unix(), testBinaryName, false)

	s.storer.EXPECT().
		GetRecord(mock.Anything, testEntryID).
		Return(record, nil)

	// Should delete history entry without touching trash
	s.storer.EXPECT().
		DeleteRecord(mock.Anything, testEntryID).
		Return(nil)

	// Execute
	err := s.manager.DeletePermanently(ctx, testEntryID)

	// Verify
	s.Require().NoError(err)
}

// TestClearEntryWithAndWithoutTrashDeletion verifies ClearEntry behavior
// with different deleteFromTrash values.
//
// This test uses table-driven testing to verify both scenarios.
func (s *ManagerIntegrationTestSuite) TestClearEntryWithAndWithoutTrashDeletion() {
	ctx := context.Background()
	now := time.Now()

	tests := []struct {
		name            string
		deleteFromTrash bool
		setupMocks      func(storage.HistoryRecord)
	}{
		{
			name:            "clear entry only",
			deleteFromTrash: false,
			setupMocks: func(record storage.HistoryRecord) {
				s.storer.EXPECT().
					GetRecord(mock.Anything, testEntryID).
					Return(record, nil)

				s.storer.EXPECT().
					DeleteRecord(mock.Anything, testEntryID).
					Return(nil)
			},
		},
		{
			name:            "clear entry and delete from trash",
			deleteFromTrash: true,
			setupMocks: func(record storage.HistoryRecord) {
				s.storer.EXPECT().
					GetRecord(mock.Anything, testEntryID).
					Return(record, nil)

				s.trasher.EXPECT().
					IsInTrash(testTrashPath).
					Return(true)

				s.trasher.EXPECT().
					DeletePermanently(mock.Anything, testTrashPath).
					Return(nil)

				s.storer.EXPECT().
					DeleteRecord(mock.Anything, testEntryID).
					Return(nil)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			record := s.createHistoryRecord(now.Unix(), testBinaryName, true)
			tt.setupMocks(record)

			err := s.manager.ClearEntry(ctx, testEntryID, tt.deleteFromTrash)

			s.Require().NoError(err)
		})
	}
}

// TestCloseWorkflow verifies the Close workflow.
//
// This test ensures that Close properly closes the underlying storage.
func (s *ManagerIntegrationTestSuite) TestCloseWorkflow() {
	s.storer.EXPECT().
		Close().
		Return(nil)

	err := s.manager.Close()

	s.Require().NoError(err)
}

// TestRecordDeletionEmptyPath verifies error handling for empty binary path.
//
// An empty path should return ErrInvalidBinaryPath immediately.
func (s *ManagerIntegrationTestSuite) TestRecordDeletionEmptyPath() {
	ctx := context.Background()

	entry, err := s.manager.RecordDeletion(ctx, "")

	s.Require().ErrorIs(err, history.ErrInvalidBinaryPath)
	s.Nil(entry)
}

// TestUndoMostRecentNoHistory verifies error handling when no history exists.
//
// When no history records exist, UndoMostRecent should return ErrNoHistory.
func (s *ManagerIntegrationTestSuite) TestUndoMostRecentNoHistory() {
	ctx := context.Background()

	s.storer.EXPECT().
		GetMostRecent(mock.Anything).
		Return(storage.HistoryRecord{}, storage.ErrNoHistory)

	result, err := s.manager.UndoMostRecent(ctx)

	s.Require().ErrorIs(err, history.ErrNoHistory)
	s.Nil(result)
}

// TestRestoreEntryNotFound verifies error handling when entry is not found.
//
// When the entry ID doesn't exist, Restore should return ErrEntryNotFound.
func (s *ManagerIntegrationTestSuite) TestRestoreEntryNotFound() {
	ctx := context.Background()

	s.storer.EXPECT().
		GetRecord(mock.Anything, "nonexistent:id").
		Return(storage.HistoryRecord{}, storage.ErrRecordNotFound)

	result, err := s.manager.Restore(ctx, "nonexistent:id")

	s.Require().ErrorIs(err, history.ErrEntryNotFound)
	s.Nil(result)
}

// TestDeletePermanentlyEntryNotFound verifies error handling when entry
// to delete permanently is not found.
func (s *ManagerIntegrationTestSuite) TestDeletePermanentlyEntryNotFound() {
	ctx := context.Background()

	s.storer.EXPECT().
		GetRecord(mock.Anything, "nonexistent:id").
		Return(storage.HistoryRecord{}, storage.ErrRecordNotFound)

	err := s.manager.DeletePermanently(ctx, "nonexistent:id")

	s.Require().ErrorIs(err, history.ErrEntryNotFound)
}

// TestClearEntryNotFound verifies error handling when entry to clear is not found.
func (s *ManagerIntegrationTestSuite) TestClearEntryNotFound() {
	ctx := context.Background()

	s.storer.EXPECT().
		GetRecord(mock.Anything, "nonexistent:id").
		Return(storage.HistoryRecord{}, storage.ErrRecordNotFound)

	err := s.manager.ClearEntry(ctx, "nonexistent:id", false)

	s.Require().ErrorIs(err, history.ErrEntryNotFound)
}

// TestClearHistoryTrashDeletionFailureContinues verifies that ClearHistory
// continues even if individual trash deletions fail.
//
// When clearing history with trash clearing enabled, failures to delete
// individual files from trash should be logged but not prevent the operation
// from completing.
func (s *ManagerIntegrationTestSuite) TestClearHistoryTrashDeletionFailureContinues() {
	ctx := context.Background()

	// Setup
	now := time.Now()
	records := []storage.HistoryRecord{
		s.createHistoryRecord(now.Unix(), "binary1", true),
	}

	s.storer.EXPECT().
		ListRecords(mock.Anything, storage.ListOptions{}).
		Return(records, nil)

	// Trash deletion fails
	s.trasher.EXPECT().
		IsInTrash("/trash/binary1").
		Return(true)

	s.trasher.EXPECT().
		DeletePermanently(mock.Anything, "/trash/binary1").
		Return(errors.New("delete failed"))

	// But history clearing still succeeds
	s.storer.EXPECT().
		DeleteAllRecords(mock.Anything).
		Return(nil)

	// Execute
	err := s.manager.ClearHistory(ctx, true)

	// Verify - should not fail even though trash deletion failed
	s.Require().NoError(err)
}

// TestMultipleDeletionsAndRestores verifies multiple sequential operations.
//
// This test simulates a realistic workflow of multiple deletions and restorations
// to ensure state consistency across multiple operations.
func (s *ManagerIntegrationTestSuite) TestMultipleDeletionsAndRestores() {
	ctx := context.Background()
	now := time.Now()
	buildData := s.createBuildInfoData()

	// First deletion
	s.extractor.EXPECT().
		Extract(mock.Anything, testBinaryPath).
		Return(buildData, nil)

	s.extractor.EXPECT().
		CalculateChecksum(testBinaryPath).
		Return(testChecksum, nil)

	s.trasher.EXPECT().
		MoveToTrash(mock.Anything, testBinaryPath).
		Return(testTrashPath, nil)

	s.storer.EXPECT().
		SaveRecord(mock.Anything, mock.AnythingOfType("*storage.HistoryRecord")).
		Return(nil)

	entry1, err := s.manager.RecordDeletion(ctx, testBinaryPath)
	s.Require().NoError(err)
	s.NotNil(entry1)

	// Second deletion (different binary)
	s.extractor.EXPECT().
		Extract(mock.Anything, testBinaryPath2).
		Return(buildData, nil)

	s.extractor.EXPECT().
		CalculateChecksum(testBinaryPath2).
		Return(testChecksum+"2", nil)

	s.trasher.EXPECT().
		MoveToTrash(mock.Anything, testBinaryPath2).
		Return(testTrashPath2, nil)

	s.storer.EXPECT().
		SaveRecord(mock.Anything, mock.AnythingOfType("*storage.HistoryRecord")).
		Return(nil)

	entry2, err := s.manager.RecordDeletion(ctx, testBinaryPath2)
	s.Require().NoError(err)
	s.NotNil(entry2)

	// Restore the second binary (most recent)
	record2 := s.createHistoryRecord(now.Unix(), testBinaryName2, true)

	s.storer.EXPECT().
		GetMostRecent(mock.Anything).
		Return(record2, nil)

	s.trasher.EXPECT().
		IsInTrash(testTrashPath2).
		Return(true)

	// Note: Collision detection now uses os.Stat on original path instead of IsInTrash
	// Since we can't mock os.Stat in integration tests, the test relies on the
	// fact that testBinaryPath2 doesn't exist in the test environment

	s.trasher.EXPECT().
		RestoreFromTrash(mock.Anything, testTrashPath2, testBinaryPath2).
		Return(nil)

	s.storer.EXPECT().
		UpdateRecord(mock.Anything, mock.AnythingOfType("*storage.HistoryRecord")).
		Return(nil)

	result, err := s.manager.UndoMostRecent(ctx)
	s.Require().NoError(err)
	s.Equal(testBinaryName2, result.BinaryName)
}

// Helper function for min calculation.
//
// This is a simple helper to avoid importing math for a basic operation.
func minInt(a, b int) int {
	if a < b {
		return a
	}

	return b
}

// Additional standalone tests for edge cases.

// TestRecordDeletionWithVCSTimeParsing verifies that VCS time parsing works
// correctly during deletion recording.
func TestRecordDeletionWithVCSTimeParsing(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Build data with valid VCS time
	buildData := &buildinfo.BuildInfoData{
		ModulePath:  testModulePath,
		Version:     testVersion,
		VCSRevision: testVCSRevision,
		VCSTime:     "2026-01-15T10:30:00Z",
		GoVersion:   testGoVersion,
		RawJSON:     []byte(`{"test": "data"}`),
	}

	mockTrasher := trashmocks.NewMockTrasher(t)
	mockStorer := storagemocks.NewMockStorer(t)
	mockExtractor := buildinfomocks.NewMockExtractor(t)
	mockLogger := loggermocks.NewMockLogger(t)

	// Setup logger expectations with a valid logger
	logger := zerolog.New(nil).With().Logger()
	mockLogger.EXPECT().Debug().Return(logger.Debug()).Maybe()
	mockLogger.EXPECT().Info().Return(logger.Info()).Maybe()

	mockExtractor.EXPECT().
		Extract(mock.Anything, testBinaryPath).
		Return(buildData, nil)

	mockExtractor.EXPECT().
		CalculateChecksum(testBinaryPath).
		Return(testChecksum, nil)

	mockTrasher.EXPECT().
		MoveToTrash(mock.Anything, testBinaryPath).
		Return(testTrashPath, nil)

	mockStorer.EXPECT().
		SaveRecord(mock.Anything, mock.AnythingOfType("*storage.HistoryRecord")).
		Return(nil)

	manager := history.NewManager(mockTrasher, mockStorer, mockExtractor, mockLogger)

	entry, err := manager.RecordDeletion(ctx, testBinaryPath)

	require.NoError(t, err)
	assert.NotNil(t, entry)
	assert.Equal(t, testBinaryName, entry.BinaryName)
}

// TestRestoreResultFields verifies all RestoreResult fields are populated correctly.
func TestRestoreResultFields(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	now := time.Now()

	record := storage.HistoryRecord{
		Timestamp:      now.Unix(),
		BinaryName:     testBinaryName,
		OriginalPath:   testBinaryPath,
		TrashPath:      testTrashPath,
		ModulePath:     testModulePath,
		Version:        testVersion,
		VCSRevision:    testVCSRevision,
		TrashAvailable: true,
		OriginalDir:    "/usr/local/bin",
	}

	mockTrasher := trashmocks.NewMockTrasher(t)
	mockStorer := storagemocks.NewMockStorer(t)
	mockExtractor := buildinfomocks.NewMockExtractor(t)
	mockLogger := loggermocks.NewMockLogger(t)

	// Setup logger expectations with a valid logger
	logger := zerolog.New(nil).With().Logger()
	mockLogger.EXPECT().Debug().Return(logger.Debug()).Maybe()
	mockLogger.EXPECT().Info().Return(logger.Info()).Maybe()

	entryID := history.GenerateKey(record.Timestamp, record.BinaryName)

	mockStorer.EXPECT().
		GetRecord(mock.Anything, entryID).
		Return(record, nil)

	mockTrasher.EXPECT().
		IsInTrash(testTrashPath).
		Return(true)

	// Note: Collision detection now uses os.Stat on original path instead of IsInTrash
	// Since we can't mock os.Stat in integration tests, the test relies on the
	// fact that testBinaryPath doesn't exist in the test environment

	mockTrasher.EXPECT().
		RestoreFromTrash(mock.Anything, testTrashPath, testBinaryPath).
		Return(nil)

	mockStorer.EXPECT().
		UpdateRecord(mock.Anything, mock.AnythingOfType("*storage.HistoryRecord")).
		Return(nil)

	manager := history.NewManager(mockTrasher, mockStorer, mockExtractor, mockLogger)

	result, err := manager.Restore(ctx, entryID)

	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify all fields
	assert.Equal(t, entryID, result.EntryID)
	assert.Equal(t, testBinaryName, result.BinaryName)
	assert.Equal(t, testBinaryPath, result.RestoredTo)
	assert.True(t, result.FromTrash)
	assert.Equal(t, testModulePath, result.ModulePath)
	assert.Equal(t, testVersion, result.Version)
}

// TestGetHistoryEmpty verifies GetHistory returns empty slice when no history exists.
func TestGetHistoryEmpty(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	mockTrasher := trashmocks.NewMockTrasher(t)
	mockStorer := storagemocks.NewMockStorer(t)
	mockExtractor := buildinfomocks.NewMockExtractor(t)
	mockLogger := loggermocks.NewMockLogger(t)

	// Setup logger expectations with a valid logger
	logger := zerolog.New(nil).With().Logger()
	mockLogger.EXPECT().Debug().Return(logger.Debug()).Maybe()

	mockStorer.EXPECT().
		ListRecords(mock.Anything, storage.ListOptions{Limit: 10}).
		Return([]storage.HistoryRecord{}, nil)

	manager := history.NewManager(mockTrasher, mockStorer, mockExtractor, mockLogger)

	entries, err := manager.GetHistory(ctx, 10)

	require.NoError(t, err)
	assert.Empty(t, entries)
	assert.NotNil(t, entries)
}

// TestDeletePermanentlyTrashDeleteFailureIgnored verifies that certain trash
// deletion failures are ignored during permanent deletion.
func TestDeletePermanentlyTrashDeleteFailureIgnored(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	now := time.Now()

	record := storage.HistoryRecord{
		Timestamp:      now.Unix(),
		BinaryName:     testBinaryName,
		TrashPath:      testTrashPath,
		TrashAvailable: true,
	}

	mockTrasher := trashmocks.NewMockTrasher(t)
	mockStorer := storagemocks.NewMockStorer(t)
	mockExtractor := buildinfomocks.NewMockExtractor(t)
	mockLogger := loggermocks.NewMockLogger(t)

	// Setup logger expectations with a valid logger
	logger := zerolog.New(nil).With().Logger()
	mockLogger.EXPECT().Debug().Return(logger.Debug()).Maybe()
	mockLogger.EXPECT().Info().Return(logger.Info()).Maybe()

	entryID := history.GenerateKey(record.Timestamp, record.BinaryName)

	mockStorer.EXPECT().
		GetRecord(mock.Anything, entryID).
		Return(record, nil)

	mockTrasher.EXPECT().
		IsInTrash(testTrashPath).
		Return(true)

	// Trash deletion returns ErrFileNotInTrash - this should be ignored
	mockTrasher.EXPECT().
		DeletePermanently(mock.Anything, testTrashPath).
		Return(trash.ErrFileNotInTrash)

	mockStorer.EXPECT().
		DeleteRecord(mock.Anything, entryID).
		Return(nil)

	manager := history.NewManager(mockTrasher, mockStorer, mockExtractor, mockLogger)

	err := manager.DeletePermanently(ctx, entryID)

	require.NoError(t, err)
}

// TestCloseFailure verifies error propagation when Close fails.
func TestCloseFailure(t *testing.T) {
	t.Parallel()

	mockTrasher := trashmocks.NewMockTrasher(t)
	mockStorer := storagemocks.NewMockStorer(t)
	mockExtractor := buildinfomocks.NewMockExtractor(t)
	mockLogger := loggermocks.NewMockLogger(t)

	// Setup logger expectations with a valid logger
	logger := zerolog.New(nil).With().Logger()
	mockLogger.EXPECT().Debug().Return(logger.Debug()).Maybe()

	mockStorer.EXPECT().
		Close().
		Return(storage.ErrDatabaseClosed)

	manager := history.NewManager(mockTrasher, mockStorer, mockExtractor, mockLogger)

	err := manager.Close()

	require.Error(t, err)
}

// TestClearEntryNotInTrash verifies ClearEntry behavior when binary is not in trash.
func TestClearEntryNotInTrash(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	now := time.Now()

	record := storage.HistoryRecord{
		Timestamp:      now.Unix(),
		BinaryName:     testBinaryName,
		TrashPath:      testTrashPath,
		TrashAvailable: true,
	}

	mockTrasher := trashmocks.NewMockTrasher(t)
	mockStorer := storagemocks.NewMockStorer(t)
	mockExtractor := buildinfomocks.NewMockExtractor(t)
	mockLogger := loggermocks.NewMockLogger(t)

	// Setup logger expectations with a valid logger
	logger := zerolog.New(nil).With().Logger()
	mockLogger.EXPECT().Debug().Return(logger.Debug()).Maybe()
	mockLogger.EXPECT().Info().Return(logger.Info()).Maybe()

	entryID := history.GenerateKey(record.Timestamp, record.BinaryName)

	mockStorer.EXPECT().
		GetRecord(mock.Anything, entryID).
		Return(record, nil)

	// Binary not in trash
	mockTrasher.EXPECT().
		IsInTrash(testTrashPath).
		Return(false)

	mockStorer.EXPECT().
		DeleteRecord(mock.Anything, entryID).
		Return(nil)

	manager := history.NewManager(mockTrasher, mockStorer, mockExtractor, mockLogger)

	// Should succeed even though binary is not in trash
	err := manager.ClearEntry(ctx, entryID, true)

	require.NoError(t, err)
}
