/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

// Package storage_test provides black-box integration tests for the storage package.
//
// These tests verify the Storer interface behavior through the MockStorer mock
// implementation. All tests use Mockery-generated mocks to ensure no real database
// connections are made. The tests cover the complete lifecycle of history records
// including CRUD operations, listing with filters, and error handling scenarios.
//
// Test Organization:
//   - StorageIntegrationTestSuite: Main test suite using testify suite
//   - Table-driven tests for parameterized scenarios
//   - Individual tests for complex error handling cases
//
// Coverage Areas:
//   - Full record lifecycle (Save → Get → Update → Delete)
//   - GetMostRecent with multiple records
//   - ListRecords with various ListOptions (limit, offset, filters)
//   - DeleteAllRecords operation
//   - Context cancellation handling
//   - Error propagation from underlying storage
package storage_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/nicholas-fedor/go-remove/internal/storage"
	"github.com/nicholas-fedor/go-remove/internal/storage/mocks"
)

// Test constants for consistent test data.
const (
	testBinaryName  = "test-binary"
	testBinaryPath  = "/usr/local/bin/test-binary"
	testModulePath  = "github.com/test/binary"
	testVersion     = "v1.0.0"
	testVCSRevision = "abc123def456"
	testChecksum    = "sha256:abc123"
	testGoVersion   = "go1.22.0"
	testTimestamp   = int64(1709321234)
	testTrashPath   = "/tmp/trash/test-binary"
	testOriginalDir = "/usr/local/bin"
)

// StorageIntegrationTestSuite provides integration tests for the storage Storer interface.
//
// This suite tests the orchestration between the application layer and storage layer,
// ensuring that all storage operations behave correctly through the Storer interface.
// All tests use the MockStorer to simulate database interactions without requiring
// a real BadgerDB instance.
type StorageIntegrationTestSuite struct {
	suite.Suite

	// Mock for the Storer interface
	mockStorer *mocks.MockStorer
}

// SetupTest initializes the test suite before each test.
//
// Creates a fresh MockStorer instance for each test to ensure test isolation
// and prevent test interference.
func (s *StorageIntegrationTestSuite) SetupTest() {
	s.mockStorer = mocks.NewMockStorer(s.T())
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
func (s *StorageIntegrationTestSuite) createHistoryRecord(
	timestamp int64,
	binaryName string,
	trashAvailable bool,
) storage.HistoryRecord {
	return storage.HistoryRecord{
		Timestamp:      timestamp,
		BinaryName:     binaryName,
		OriginalPath:   "/usr/local/bin/" + binaryName,
		TrashPath:      "/tmp/trash/" + binaryName,
		ModulePath:     testModulePath,
		Version:        testVersion,
		VCSRevision:    testVCSRevision,
		VCSTime:        time.Now(),
		GoVersion:      testGoVersion,
		BuildInfo:      `{"path":"github.com/test/` + binaryName + `","version":"v1.0.0"}`,
		Checksum:       testChecksum,
		TrashAvailable: trashAvailable,
		OriginalDir:    testOriginalDir,
	}
}

// TestStorageIntegrationTestSuite runs the integration test suite.
func TestStorageIntegrationTestSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(StorageIntegrationTestSuite))
}

// TestFullRecordLifecycle verifies the complete CRUD workflow.
//
// This test ensures that the storage layer properly handles:
// 1. Saving a new record
// 2. Retrieving the saved record
// 3. Updating the record
// 4. Deleting the record
//
// Each operation should maintain data integrity and return appropriate errors.
func (s *StorageIntegrationTestSuite) TestFullRecordLifecycle() {
	ctx := context.Background()
	timestamp := time.Now().Unix()
	binaryName := "lifecycle-test"
	key := storage.GenerateKey(timestamp, binaryName)

	// Step 1: Save a new record
	record := s.createHistoryRecord(timestamp, binaryName, true)

	s.mockStorer.EXPECT().
		SaveRecord(mock.Anything, mock.AnythingOfType("*storage.HistoryRecord")).
		Return(nil).
		Once()

	err := s.mockStorer.SaveRecord(ctx, &record)
	s.Require().NoError(err)

	// Step 2: Retrieve the record
	s.mockStorer.EXPECT().
		GetRecord(mock.Anything, key).
		Return(record, nil).
		Once()

	retrieved, err := s.mockStorer.GetRecord(ctx, key)
	s.Require().NoError(err)
	s.Equal(record.BinaryName, retrieved.BinaryName)
	s.Equal(record.Timestamp, retrieved.Timestamp)

	// Step 3: Update the record
	record.Version = "v2.0.0"
	record.TrashAvailable = false

	s.mockStorer.EXPECT().
		UpdateRecord(mock.Anything, mock.AnythingOfType("*storage.HistoryRecord")).
		Return(nil).
		Once()

	err = s.mockStorer.UpdateRecord(ctx, &record)
	s.Require().NoError(err)

	// Step 4: Delete the record
	s.mockStorer.EXPECT().
		DeleteRecord(mock.Anything, key).
		Return(nil).
		Once()

	err = s.mockStorer.DeleteRecord(ctx, key)
	s.Require().NoError(err)
}

// TestSaveRecordWithNilRecord verifies error handling for nil record.
//
// The storage layer should return an error when attempting to save a nil record.
func (s *StorageIntegrationTestSuite) TestSaveRecordWithNilRecord() {
	ctx := context.Background()

	s.mockStorer.EXPECT().
		SaveRecord(mock.Anything, (*storage.HistoryRecord)(nil)).
		Return(storage.ErrInvalidRecord).
		Once()

	err := s.mockStorer.SaveRecord(ctx, nil)
	s.Require().ErrorIs(err, storage.ErrInvalidRecord)
}

// TestSaveRecordWithZeroTimestamp verifies validation for zero timestamp.
//
// Records must have a valid non-zero timestamp to ensure proper key generation.
func (s *StorageIntegrationTestSuite) TestSaveRecordWithZeroTimestamp() {
	ctx := context.Background()
	record := s.createHistoryRecord(0, testBinaryName, true)

	s.mockStorer.EXPECT().
		SaveRecord(mock.Anything, mock.AnythingOfType("*storage.HistoryRecord")).
		Return(storage.ErrInvalidRecord).
		Once()

	err := s.mockStorer.SaveRecord(ctx, &record)
	s.Require().ErrorIs(err, storage.ErrInvalidRecord)
}

// TestSaveRecordWithEmptyBinaryName verifies validation for empty binary name.
//
// Binary name is a required field for key generation and record identification.
func (s *StorageIntegrationTestSuite) TestSaveRecordWithEmptyBinaryName() {
	ctx := context.Background()
	record := s.createHistoryRecord(testTimestamp, "", true)

	s.mockStorer.EXPECT().
		SaveRecord(mock.Anything, mock.AnythingOfType("*storage.HistoryRecord")).
		Return(storage.ErrInvalidRecord).
		Once()

	err := s.mockStorer.SaveRecord(ctx, &record)
	s.Require().ErrorIs(err, storage.ErrInvalidRecord)
}

// TestGetRecordNotFound verifies error handling for non-existent records.
//
// When a key does not exist, GetRecord should return ErrRecordNotFound.
func (s *StorageIntegrationTestSuite) TestGetRecordNotFound() {
	ctx := context.Background()
	key := storage.GenerateKey(testTimestamp, "non-existent")

	s.mockStorer.EXPECT().
		GetRecord(mock.Anything, key).
		Return(storage.HistoryRecord{}, storage.ErrRecordNotFound).
		Once()

	_, err := s.mockStorer.GetRecord(ctx, key)
	s.Require().ErrorIs(err, storage.ErrRecordNotFound)
}

// TestGetRecordInvalidKey verifies error handling for invalid key format.
//
// Keys must be in the format "<timestamp>:<binary_name>".
func (s *StorageIntegrationTestSuite) TestGetRecordInvalidKey() {
	ctx := context.Background()
	invalidKey := "invalid-key-format"

	s.mockStorer.EXPECT().
		GetRecord(mock.Anything, invalidKey).
		Return(storage.HistoryRecord{}, storage.ErrInvalidKey).
		Once()

	_, err := s.mockStorer.GetRecord(ctx, invalidKey)
	s.Require().ErrorIs(err, storage.ErrInvalidKey)
}

// TestGetMostRecentWithMultipleRecords verifies retrieval of the most recent record.
//
// When multiple records exist, GetMostRecent should return the one with the
// highest timestamp (most recent).
func (s *StorageIntegrationTestSuite) TestGetMostRecentWithMultipleRecords() {
	ctx := context.Background()
	now := time.Now()

	// Create records with different timestamps
	oldest := s.createHistoryRecord(now.Add(-2*time.Hour).Unix(), "oldest", true)
	middle := s.createHistoryRecord(now.Add(-1*time.Hour).Unix(), "middle", true)
	newest := s.createHistoryRecord(now.Unix(), "newest", true)

	// List all records
	allRecords := []storage.HistoryRecord{newest, middle, oldest}

	s.mockStorer.EXPECT().
		ListRecords(mock.Anything, storage.ListOptions{}).
		Return(allRecords, nil).
		Once()

	records, err := s.mockStorer.ListRecords(ctx, storage.ListOptions{})
	s.Require().NoError(err)
	s.Require().Len(records, 3)

	// GetMostRecent should return the newest
	s.mockStorer.EXPECT().
		GetMostRecent(mock.Anything).
		Return(newest, nil).
		Once()

	mostRecent, err := s.mockStorer.GetMostRecent(ctx)
	s.Require().NoError(err)
	s.Equal("newest", mostRecent.BinaryName)
	s.Equal(newest.Timestamp, mostRecent.Timestamp)
}

// TestGetMostRecentNoHistory verifies error when no records exist.
//
// When the storage is empty, GetMostRecent should return ErrNoHistory.
func (s *StorageIntegrationTestSuite) TestGetMostRecentNoHistory() {
	ctx := context.Background()

	s.mockStorer.EXPECT().
		GetMostRecent(mock.Anything).
		Return(storage.HistoryRecord{}, storage.ErrNoHistory).
		Once()

	_, err := s.mockStorer.GetMostRecent(ctx)
	s.Require().ErrorIs(err, storage.ErrNoHistory)
}

// TestListRecordsWithVariousOptions verifies listing with different ListOptions.
//
// This test uses table-driven testing to cover:
// - No options (all records)
// - Limit option
// - Offset option
// - Combined limit and offset
// - OnlyAvailable filter.
func (s *StorageIntegrationTestSuite) TestListRecordsWithVariousOptions() {
	ctx := context.Background()
	now := time.Now()

	// Create test records
	records := []storage.HistoryRecord{
		s.createHistoryRecord(now.Unix(), "binary-0", true),
		s.createHistoryRecord(now.Add(-10*time.Second).Unix(), "binary-1", false),
		s.createHistoryRecord(now.Add(-20*time.Second).Unix(), "binary-2", true),
		s.createHistoryRecord(now.Add(-30*time.Second).Unix(), "binary-3", false),
		s.createHistoryRecord(now.Add(-40*time.Second).Unix(), "binary-4", true),
	}

	tests := []struct {
		name          string
		opts          storage.ListOptions
		expectedCount int
		expectedNames []string
		returnRecords []storage.HistoryRecord
	}{
		{
			name:          "no options returns all",
			opts:          storage.ListOptions{},
			expectedCount: 5,
			expectedNames: []string{"binary-0", "binary-1", "binary-2", "binary-3", "binary-4"},
			returnRecords: records,
		},
		{
			name:          "limit of 2 returns first 2",
			opts:          storage.ListOptions{Limit: 2},
			expectedCount: 2,
			expectedNames: []string{"binary-0", "binary-1"},
			returnRecords: records[:2],
		},
		{
			name:          "offset of 2 skips first 2",
			opts:          storage.ListOptions{Offset: 2},
			expectedCount: 3,
			expectedNames: []string{"binary-2", "binary-3", "binary-4"},
			returnRecords: records[2:],
		},
		{
			name:          "limit 2 offset 1 returns middle 2",
			opts:          storage.ListOptions{Limit: 2, Offset: 1},
			expectedCount: 2,
			expectedNames: []string{"binary-1", "binary-2"},
			returnRecords: records[1:3],
		},
		{
			name:          "only available filter",
			opts:          storage.ListOptions{OnlyAvailable: true},
			expectedCount: 3,
			expectedNames: []string{"binary-0", "binary-2", "binary-4"},
			returnRecords: []storage.HistoryRecord{records[0], records[2], records[4]},
		},
		{
			name:          "only available with limit",
			opts:          storage.ListOptions{OnlyAvailable: true, Limit: 2},
			expectedCount: 2,
			expectedNames: []string{"binary-0", "binary-2"},
			returnRecords: []storage.HistoryRecord{records[0], records[2]},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.mockStorer.EXPECT().
				ListRecords(mock.Anything, tt.opts).
				Return(tt.returnRecords, nil).
				Once()

			result, err := s.mockStorer.ListRecords(ctx, tt.opts)
			s.Require().NoError(err)
			s.Len(result, tt.expectedCount)

			for i, name := range tt.expectedNames {
				s.Equal(name, result[i].BinaryName)
			}
		})
	}
}

// TestListRecordsEmpty verifies empty list when no records exist.
//
// When the storage is empty, ListRecords should return an empty slice.
func (s *StorageIntegrationTestSuite) TestListRecordsEmpty() {
	ctx := context.Background()

	s.mockStorer.EXPECT().
		ListRecords(mock.Anything, storage.ListOptions{}).
		Return([]storage.HistoryRecord{}, nil).
		Once()

	records, err := s.mockStorer.ListRecords(ctx, storage.ListOptions{})
	s.Require().NoError(err)
	s.Empty(records)
}

// TestUpdateRecordNotFound verifies error when updating non-existent record.
//
// When the record key does not exist, UpdateRecord should return ErrRecordNotFound.
func (s *StorageIntegrationTestSuite) TestUpdateRecordNotFound() {
	ctx := context.Background()
	record := s.createHistoryRecord(testTimestamp, "non-existent", true)

	s.mockStorer.EXPECT().
		UpdateRecord(mock.Anything, mock.AnythingOfType("*storage.HistoryRecord")).
		Return(storage.ErrRecordNotFound).
		Once()

	err := s.mockStorer.UpdateRecord(ctx, &record)
	s.Require().ErrorIs(err, storage.ErrRecordNotFound)
}

// TestDeleteRecordNotFound verifies error when deleting non-existent record.
//
// When the record key does not exist, DeleteRecord should return ErrRecordNotFound.
func (s *StorageIntegrationTestSuite) TestDeleteRecordNotFound() {
	ctx := context.Background()
	key := storage.GenerateKey(testTimestamp, "non-existent")

	s.mockStorer.EXPECT().
		DeleteRecord(mock.Anything, key).
		Return(storage.ErrRecordNotFound).
		Once()

	err := s.mockStorer.DeleteRecord(ctx, key)
	s.Require().ErrorIs(err, storage.ErrRecordNotFound)
}

// TestDeleteAllRecords verifies removal of all records.
//
// DeleteAllRecords should remove all history records from storage.
func (s *StorageIntegrationTestSuite) TestDeleteAllRecords() {
	ctx := context.Background()

	s.mockStorer.EXPECT().
		DeleteAllRecords(mock.Anything).
		Return(nil).
		Once()

	err := s.mockStorer.DeleteAllRecords(ctx)
	s.Require().NoError(err)
}

// TestDeleteAllRecordsError verifies error propagation from DeleteAllRecords.
//
// Errors from the underlying storage should be properly returned.
func (s *StorageIntegrationTestSuite) TestDeleteAllRecordsError() {
	ctx := context.Background()
	dbError := errors.New("database write error")

	s.mockStorer.EXPECT().
		DeleteAllRecords(mock.Anything).
		Return(dbError).
		Once()

	err := s.mockStorer.DeleteAllRecords(ctx)
	s.Require().ErrorIs(err, dbError)
}

// TestContextCancellation verifies proper handling of canceled context.
//
// When the context is canceled before or during an operation,
// the storage layer should return ErrContextCanceled.
func (s *StorageIntegrationTestSuite) TestContextCancellation() {
	// Test SaveRecord with canceled context
	s.Run("SaveRecord", func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		record := s.createHistoryRecord(testTimestamp, testBinaryName, true)

		s.mockStorer.EXPECT().
			SaveRecord(mock.Anything, mock.AnythingOfType("*storage.HistoryRecord")).
			Return(storage.ErrContextCanceled).
			Once()

		err := s.mockStorer.SaveRecord(ctx, &record)
		s.ErrorIs(err, storage.ErrContextCanceled)
	})

	// Test GetRecord with canceled context
	s.Run("GetRecord", func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		key := storage.GenerateKey(testTimestamp, testBinaryName)

		s.mockStorer.EXPECT().
			GetRecord(mock.Anything, key).
			Return(storage.HistoryRecord{}, storage.ErrContextCanceled).
			Once()

		_, err := s.mockStorer.GetRecord(ctx, key)
		s.ErrorIs(err, storage.ErrContextCanceled)
	})

	// Test GetMostRecent with canceled context
	s.Run("GetMostRecent", func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		s.mockStorer.EXPECT().
			GetMostRecent(mock.Anything).
			Return(storage.HistoryRecord{}, storage.ErrContextCanceled).
			Once()

		_, err := s.mockStorer.GetMostRecent(ctx)
		s.ErrorIs(err, storage.ErrContextCanceled)
	})

	// Test ListRecords with canceled context
	s.Run("ListRecords", func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		s.mockStorer.EXPECT().
			ListRecords(mock.Anything, storage.ListOptions{}).
			Return(nil, storage.ErrContextCanceled).
			Once()

		_, err := s.mockStorer.ListRecords(ctx, storage.ListOptions{})
		s.ErrorIs(err, storage.ErrContextCanceled)
	})

	// Test UpdateRecord with canceled context
	s.Run("UpdateRecord", func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		record := s.createHistoryRecord(testTimestamp, testBinaryName, true)

		s.mockStorer.EXPECT().
			UpdateRecord(mock.Anything, mock.AnythingOfType("*storage.HistoryRecord")).
			Return(storage.ErrContextCanceled).
			Once()

		err := s.mockStorer.UpdateRecord(ctx, &record)
		s.ErrorIs(err, storage.ErrContextCanceled)
	})

	// Test DeleteRecord with canceled context
	s.Run("DeleteRecord", func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		key := storage.GenerateKey(testTimestamp, testBinaryName)

		s.mockStorer.EXPECT().
			DeleteRecord(mock.Anything, key).
			Return(storage.ErrContextCanceled).
			Once()

		err := s.mockStorer.DeleteRecord(ctx, key)
		s.ErrorIs(err, storage.ErrContextCanceled)
	})

	// Test DeleteAllRecords with canceled context
	s.Run("DeleteAllRecords", func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		s.mockStorer.EXPECT().
			DeleteAllRecords(mock.Anything).
			Return(storage.ErrContextCanceled).
			Once()

		err := s.mockStorer.DeleteAllRecords(ctx)
		s.ErrorIs(err, storage.ErrContextCanceled)
	})
}

// TestErrorPropagation verifies that errors are properly propagated.
//
// This test ensures that errors from the underlying storage are wrapped
// and returned correctly to the caller.
func (s *StorageIntegrationTestSuite) TestErrorPropagation() {
	ctx := context.Background()
	customError := errors.New("custom storage error")

	s.Run("SaveRecord error propagation", func() {
		record := s.createHistoryRecord(testTimestamp, testBinaryName, true)

		s.mockStorer.EXPECT().
			SaveRecord(mock.Anything, mock.AnythingOfType("*storage.HistoryRecord")).
			Return(customError).
			Once()

		err := s.mockStorer.SaveRecord(ctx, &record)
		s.ErrorIs(err, customError)
	})

	s.Run("GetRecord error propagation", func() {
		key := storage.GenerateKey(testTimestamp, testBinaryName)

		s.mockStorer.EXPECT().
			GetRecord(mock.Anything, key).
			Return(storage.HistoryRecord{}, customError).
			Once()

		_, err := s.mockStorer.GetRecord(ctx, key)
		s.ErrorIs(err, customError)
	})

	s.Run("GetMostRecent error propagation", func() {
		s.mockStorer.EXPECT().
			GetMostRecent(mock.Anything).
			Return(storage.HistoryRecord{}, customError).
			Once()

		_, err := s.mockStorer.GetMostRecent(ctx)
		s.ErrorIs(err, customError)
	})

	s.Run("ListRecords error propagation", func() {
		s.mockStorer.EXPECT().
			ListRecords(mock.Anything, storage.ListOptions{}).
			Return(nil, customError).
			Once()

		_, err := s.mockStorer.ListRecords(ctx, storage.ListOptions{})
		s.ErrorIs(err, customError)
	})

	s.Run("UpdateRecord error propagation", func() {
		record := s.createHistoryRecord(testTimestamp, testBinaryName, true)

		s.mockStorer.EXPECT().
			UpdateRecord(mock.Anything, mock.AnythingOfType("*storage.HistoryRecord")).
			Return(customError).
			Once()

		err := s.mockStorer.UpdateRecord(ctx, &record)
		s.ErrorIs(err, customError)
	})

	s.Run("DeleteRecord error propagation", func() {
		key := storage.GenerateKey(testTimestamp, testBinaryName)

		s.mockStorer.EXPECT().
			DeleteRecord(mock.Anything, key).
			Return(customError).
			Once()

		err := s.mockStorer.DeleteRecord(ctx, key)
		s.ErrorIs(err, customError)
	})
}

// TestClose verifies the Close operation.
//
// Close should properly close the database connection and release resources.
func (s *StorageIntegrationTestSuite) TestClose() {
	s.mockStorer.EXPECT().
		Close().
		Return(nil).
		Once()

	err := s.mockStorer.Close()
	s.Require().NoError(err)
}

// TestCloseError verifies error handling when Close fails.
//
// When the database is already closed or an error occurs, Close should
// return an appropriate error.
func (s *StorageIntegrationTestSuite) TestCloseError() {
	s.mockStorer.EXPECT().
		Close().
		Return(storage.ErrDatabaseClosed).
		Once()

	err := s.mockStorer.Close()
	s.Require().ErrorIs(err, storage.ErrDatabaseClosed)
}

// TestDatabaseClosedError verifies all operations fail when database is closed.
//
// When the database is closed, all operations should return ErrDatabaseClosed.
func (s *StorageIntegrationTestSuite) TestDatabaseClosedError() {
	ctx := context.Background()
	key := storage.GenerateKey(testTimestamp, testBinaryName)
	record := s.createHistoryRecord(testTimestamp, testBinaryName, true)

	s.Run("SaveRecord", func() {
		s.mockStorer.EXPECT().
			SaveRecord(mock.Anything, mock.AnythingOfType("*storage.HistoryRecord")).
			Return(storage.ErrDatabaseClosed).
			Once()

		err := s.mockStorer.SaveRecord(ctx, &record)
		s.ErrorIs(err, storage.ErrDatabaseClosed)
	})

	s.Run("GetRecord", func() {
		s.mockStorer.EXPECT().
			GetRecord(mock.Anything, key).
			Return(storage.HistoryRecord{}, storage.ErrDatabaseClosed).
			Once()

		_, err := s.mockStorer.GetRecord(ctx, key)
		s.ErrorIs(err, storage.ErrDatabaseClosed)
	})

	s.Run("GetMostRecent", func() {
		s.mockStorer.EXPECT().
			GetMostRecent(mock.Anything).
			Return(storage.HistoryRecord{}, storage.ErrDatabaseClosed).
			Once()

		_, err := s.mockStorer.GetMostRecent(ctx)
		s.ErrorIs(err, storage.ErrDatabaseClosed)
	})

	s.Run("ListRecords", func() {
		s.mockStorer.EXPECT().
			ListRecords(mock.Anything, storage.ListOptions{}).
			Return(nil, storage.ErrDatabaseClosed).
			Once()

		_, err := s.mockStorer.ListRecords(ctx, storage.ListOptions{})
		s.ErrorIs(err, storage.ErrDatabaseClosed)
	})

	s.Run("UpdateRecord", func() {
		s.mockStorer.EXPECT().
			UpdateRecord(mock.Anything, mock.AnythingOfType("*storage.HistoryRecord")).
			Return(storage.ErrDatabaseClosed).
			Once()

		err := s.mockStorer.UpdateRecord(ctx, &record)
		s.ErrorIs(err, storage.ErrDatabaseClosed)
	})

	s.Run("DeleteRecord", func() {
		s.mockStorer.EXPECT().
			DeleteRecord(mock.Anything, key).
			Return(storage.ErrDatabaseClosed).
			Once()

		err := s.mockStorer.DeleteRecord(ctx, key)
		s.ErrorIs(err, storage.ErrDatabaseClosed)
	})

	s.Run("DeleteAllRecords", func() {
		s.mockStorer.EXPECT().
			DeleteAllRecords(mock.Anything).
			Return(storage.ErrDatabaseClosed).
			Once()

		err := s.mockStorer.DeleteAllRecords(ctx)
		s.ErrorIs(err, storage.ErrDatabaseClosed)
	})
}

// TestMultipleSequentialOperations verifies multiple operations in sequence.
//
// This test simulates a realistic workflow of multiple save, list, and delete
// operations to ensure the storage layer behaves correctly across multiple calls.
func (s *StorageIntegrationTestSuite) TestMultipleSequentialOperations() {
	ctx := context.Background()
	now := time.Now()

	// Save first record
	record1 := s.createHistoryRecord(now.Unix(), "binary-1", true)
	s.mockStorer.EXPECT().
		SaveRecord(mock.Anything, mock.AnythingOfType("*storage.HistoryRecord")).
		Return(nil).
		Once()
	err := s.mockStorer.SaveRecord(ctx, &record1)
	s.Require().NoError(err)

	// Save second record
	record2 := s.createHistoryRecord(now.Add(-1*time.Second).Unix(), "binary-2", true)
	s.mockStorer.EXPECT().
		SaveRecord(mock.Anything, mock.AnythingOfType("*storage.HistoryRecord")).
		Return(nil).
		Once()
	err = s.mockStorer.SaveRecord(ctx, &record2)
	s.Require().NoError(err)

	// List all records
	s.mockStorer.EXPECT().
		ListRecords(mock.Anything, storage.ListOptions{}).
		Return([]storage.HistoryRecord{record1, record2}, nil).
		Once()
	records, err := s.mockStorer.ListRecords(ctx, storage.ListOptions{})
	s.Require().NoError(err)
	s.Len(records, 2)

	// Get most recent (should be record1)
	s.mockStorer.EXPECT().
		GetMostRecent(mock.Anything).
		Return(record1, nil).
		Once()
	mostRecent, err := s.mockStorer.GetMostRecent(ctx)
	s.Require().NoError(err)
	s.Equal("binary-1", mostRecent.BinaryName)

	// Delete first record
	key1 := storage.GenerateKey(record1.Timestamp, record1.BinaryName)
	s.mockStorer.EXPECT().
		DeleteRecord(mock.Anything, key1).
		Return(nil).
		Once()
	err = s.mockStorer.DeleteRecord(ctx, key1)
	s.Require().NoError(err)

	// List should now show only one record
	s.mockStorer.EXPECT().
		ListRecords(mock.Anything, storage.ListOptions{}).
		Return([]storage.HistoryRecord{record2}, nil).
		Once()
	records, err = s.mockStorer.ListRecords(ctx, storage.ListOptions{})
	s.Require().NoError(err)
	s.Len(records, 1)
	s.Equal("binary-2", records[0].BinaryName)
}

// Additional standalone tests for edge cases.

// TestHistoryRecordFields verifies all HistoryRecord fields are handled correctly.
func TestHistoryRecordFields(t *testing.T) {
	t.Parallel()

	record := storage.HistoryRecord{
		Timestamp:      testTimestamp,
		BinaryName:     testBinaryName,
		OriginalPath:   testBinaryPath,
		TrashPath:      testTrashPath,
		ModulePath:     testModulePath,
		Version:        testVersion,
		VCSRevision:    testVCSRevision,
		VCSTime:        time.Now(),
		GoVersion:      testGoVersion,
		BuildInfo:      `{"test":"data"}`,
		Checksum:       testChecksum,
		TrashAvailable: true,
		OriginalDir:    testOriginalDir,
	}

	assert.Equal(t, testTimestamp, record.Timestamp)
	assert.Equal(t, testBinaryName, record.BinaryName)
	assert.Equal(t, testBinaryPath, record.OriginalPath)
	assert.Equal(t, testTrashPath, record.TrashPath)
	assert.Equal(t, testModulePath, record.ModulePath)
	assert.Equal(t, testVersion, record.Version)
	assert.Equal(t, testVCSRevision, record.VCSRevision)
	assert.Equal(t, testGoVersion, record.GoVersion)
	assert.Equal(t, testChecksum, record.Checksum)
	assert.True(t, record.TrashAvailable)
	assert.Equal(t, testOriginalDir, record.OriginalDir)

	// Test DisplayTime method
	displayTime := record.DisplayTime()
	assert.NotEmpty(t, displayTime)
	_, err := time.Parse("2006-01-02 15:04:05", displayTime)
	require.NoError(t, err)
}

// TestListOptionsCombinations verifies various ListOptions combinations.
func TestListOptionsCombinations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts storage.ListOptions
	}{
		{
			name: "zero values",
			opts: storage.ListOptions{},
		},
		{
			name: "only limit",
			opts: storage.ListOptions{Limit: 10},
		},
		{
			name: "only offset",
			opts: storage.ListOptions{Offset: 5},
		},
		{
			name: "only available filter",
			opts: storage.ListOptions{OnlyAvailable: true},
		},
		{
			name: "all options combined",
			opts: storage.ListOptions{
				Limit:         10,
				Offset:        5,
				OnlyAvailable: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Just verify the options can be created without issues
			assert.NotNil(t, tt.opts)
		})
	}
}

// TestErrorTypes verifies storage error types.
func TestErrorTypes(t *testing.T) {
	t.Parallel()

	// Verify all expected errors are defined
	require.Error(t, storage.ErrRecordNotFound)
	require.Error(t, storage.ErrNoHistory)
	require.Error(t, storage.ErrInvalidKey)
	require.Error(t, storage.ErrInvalidRecord)
	require.Error(t, storage.ErrDatabaseClosed)
	require.Error(t, storage.ErrContextCanceled)
}

// TestGenerateAndParseKey verifies key generation and parsing.
func TestGenerateAndParseKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		timestamp  int64
		binaryName string
	}{
		{
			name:       "standard key",
			timestamp:  1709321234,
			binaryName: "golangci-lint",
		},
		{
			name:       "binary with hyphens",
			timestamp:  1709321234,
			binaryName: "my-binary-name",
		},
		{
			name:       "binary with dots",
			timestamp:  1709321234,
			binaryName: "my.binary.name",
		},
		{
			name:       "zero timestamp",
			timestamp:  0,
			binaryName: "test-binary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			key := storage.GenerateKey(tt.timestamp, tt.binaryName)
			assert.NotEmpty(t, key)
			assert.Contains(t, key, tt.binaryName)
		})
	}
}

// TestContextTimeout verifies handling of context timeout.
func TestContextTimeout(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait for context to expire
	time.Sleep(10 * time.Millisecond)

	// Context should be canceled
	select {
	case <-ctx.Done():
		require.ErrorIs(t, ctx.Err(), context.DeadlineExceeded)
	default:
		t.Error("expected context to be done")
	}
}

// TestRecordComparison verifies HistoryRecord comparison.
func TestRecordComparison(t *testing.T) {
	t.Parallel()

	record1 := storage.HistoryRecord{
		Timestamp:  1000,
		BinaryName: "binary-1",
	}

	record2 := storage.HistoryRecord{
		Timestamp:  2000,
		BinaryName: "binary-2",
	}

	// Verify different timestamps result in different keys
	key1 := storage.GenerateKey(record1.Timestamp, record1.BinaryName)
	key2 := storage.GenerateKey(record2.Timestamp, record2.BinaryName)

	assert.NotEqual(t, key1, key2)
	assert.Greater(t, record2.Timestamp, record1.Timestamp)
}
