/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package storage

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testBinaryName is a constant for the test binary name to avoid magic strings.
const testBinaryName = "test-binary"

// setupTestStore creates a temporary Badger store for testing.
// Returns the store and a cleanup function.
func setupTestStore(t *testing.T) (*BadgerStore, func()) {
	t.Helper()

	// Create temporary directory for test database using t.TempDir()
	tempDir := t.TempDir()

	dbPath := filepath.Join(tempDir, "test.db")
	store, err := NewBadgerStore(dbPath)
	require.NoError(t, err, "Failed to create test store")

	cleanup := func() {
		store.Close()
	}

	return store, cleanup
}

// createTestRecord creates a HistoryRecord for testing.
func createTestRecord(timestamp int64, binaryName string) HistoryRecord {
	return HistoryRecord{
		Timestamp:      timestamp,
		BinaryName:     binaryName,
		OriginalPath:   "/usr/local/bin/" + binaryName,
		TrashPath:      "/tmp/trash/" + binaryName,
		ModulePath:     "github.com/test/" + binaryName,
		Version:        "v1.0.0",
		VCSRevision:    "abc123",
		VCSTime:        time.Now(),
		GoVersion:      "go1.22.0",
		BuildInfo:      `{"path":"github.com/test/` + binaryName + `","version":"v1.0.0"}`,
		Checksum:       "sha256:1234567890abcdef",
		TrashAvailable: true,
		OriginalDir:    "/usr/local/bin",
	}
}

func TestNewBadgerStore(t *testing.T) {
	t.Run("successfully creates store", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		assert.NotNil(t, store)
		assert.NotNil(t, store.database)
		assert.False(t, store.closed.Load())
	})

	t.Run("fails with invalid path", func(t *testing.T) {
		// Try to create store with an empty path, which is invalid on all platforms
		_, err := NewBadgerStore("")
		assert.Error(t, err)
	})
}

func TestBadgerStore_SaveRecord(t *testing.T) {
	t.Run("successfully saves record", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := context.Background()
		record := createTestRecord(time.Now().Unix(), testBinaryName)
		err := store.SaveRecord(ctx, &record)
		require.NoError(t, err)
	})

	t.Run("fails with zero timestamp", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := context.Background()
		record := createTestRecord(0, testBinaryName)
		err := store.SaveRecord(ctx, &record)
		require.ErrorIs(t, err, ErrInvalidRecord)
		assert.Contains(t, err.Error(), "timestamp is required")
	})

	t.Run("fails with empty binary name", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := context.Background()
		record := createTestRecord(time.Now().Unix(), "")
		err := store.SaveRecord(ctx, &record)
		require.ErrorIs(t, err, ErrInvalidRecord)
		assert.Contains(t, err.Error(), "binary_name is required")
	})

	t.Run("fails when database is closed", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		cleanup() // Close immediately

		ctx := context.Background()
		record := createTestRecord(time.Now().Unix(), testBinaryName)
		err := store.SaveRecord(ctx, &record)
		assert.ErrorIs(t, err, ErrDatabaseClosed)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		record := createTestRecord(time.Now().Unix(), testBinaryName)
		err := store.SaveRecord(ctx, &record)
		assert.ErrorIs(t, err, ErrContextCanceled)
	})
}

func TestBadgerStore_GetRecord(t *testing.T) {
	t.Run("successfully retrieves record", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := context.Background()
		timestamp := time.Now().Unix()
		binaryName := testBinaryName
		record := createTestRecord(timestamp, binaryName)

		err := store.SaveRecord(ctx, &record)
		require.NoError(t, err)

		key := GenerateKey(timestamp, binaryName)
		retrieved, err := store.GetRecord(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, record.BinaryName, retrieved.BinaryName)
		assert.Equal(t, record.Timestamp, retrieved.Timestamp)
		assert.Equal(t, record.ModulePath, retrieved.ModulePath)
	})

	t.Run("returns error for non-existent key", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := context.Background()
		key := GenerateKey(time.Now().Unix(), "non-existent")
		_, err := store.GetRecord(ctx, key)
		assert.ErrorIs(t, err, ErrRecordNotFound)
	})

	t.Run("returns error for invalid key format", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := context.Background()
		_, err := store.GetRecord(ctx, "invalid-key")
		assert.ErrorIs(t, err, ErrInvalidKey)
	})

	t.Run("fails when database is closed", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		cleanup()

		ctx := context.Background()
		key := GenerateKey(time.Now().Unix(), "test")
		_, err := store.GetRecord(ctx, key)
		assert.ErrorIs(t, err, ErrDatabaseClosed)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		key := GenerateKey(time.Now().Unix(), "test")
		_, err := store.GetRecord(ctx, key)
		assert.ErrorIs(t, err, ErrContextCanceled)
	})
}

func TestBadgerStore_GetMostRecent(t *testing.T) {
	t.Run("returns error when no records exist", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := context.Background()
		_, err := store.GetMostRecent(ctx)
		assert.ErrorIs(t, err, ErrNoHistory)
	})

	t.Run("returns most recent record", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := context.Background()
		now := time.Now().Unix()

		record1 := createTestRecord(now-100, "old-binary")
		record2 := createTestRecord(now, "new-binary")
		record3 := createTestRecord(now-50, "middle-binary")

		require.NoError(t, store.SaveRecord(ctx, &record1))
		require.NoError(t, store.SaveRecord(ctx, &record2))
		require.NoError(t, store.SaveRecord(ctx, &record3))

		mostRecent, err := store.GetMostRecent(ctx)
		require.NoError(t, err)
		assert.Equal(t, "new-binary", mostRecent.BinaryName)
		assert.Equal(t, now, mostRecent.Timestamp)
	})

	t.Run("fails when database is closed", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		cleanup()

		ctx := context.Background()
		_, err := store.GetMostRecent(ctx)
		assert.ErrorIs(t, err, ErrDatabaseClosed)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := context.Background()
		record := createTestRecord(time.Now().Unix(), "test")
		require.NoError(t, store.SaveRecord(ctx, &record))

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := store.GetMostRecent(ctx)
		assert.ErrorIs(t, err, ErrContextCanceled)
	})
}

func TestBadgerStore_ListRecords(t *testing.T) {
	t.Run("returns empty list when no records", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := context.Background()
		records, err := store.ListRecords(ctx, ListOptions{})
		require.NoError(t, err)
		assert.Empty(t, records)
	})

	t.Run("returns all records in reverse chronological order", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := context.Background()
		now := time.Now().Unix()

		record1 := createTestRecord(now-200, "oldest")
		record2 := createTestRecord(now-100, "middle")
		record3 := createTestRecord(now, "newest")

		require.NoError(t, store.SaveRecord(ctx, &record1))
		require.NoError(t, store.SaveRecord(ctx, &record2))
		require.NoError(t, store.SaveRecord(ctx, &record3))

		records, err := store.ListRecords(ctx, ListOptions{})
		require.NoError(t, err)
		assert.Len(t, records, 3)
		assert.Equal(t, "newest", records[0].BinaryName)
		assert.Equal(t, "middle", records[1].BinaryName)
		assert.Equal(t, "oldest", records[2].BinaryName)
	})

	t.Run("respects limit option", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := context.Background()
		now := time.Now().Unix()

		for i := range 5 {
			record := createTestRecord(now-int64(i*10), fmt.Sprintf("binary-%d", i))
			require.NoError(t, store.SaveRecord(ctx, &record))
		}

		records, err := store.ListRecords(ctx, ListOptions{Limit: 2})
		require.NoError(t, err)
		assert.Len(t, records, 2)
		assert.Equal(t, "binary-0", records[0].BinaryName)
		assert.Equal(t, "binary-1", records[1].BinaryName)
	})

	t.Run("respects offset option", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := context.Background()
		now := time.Now().Unix()

		for i := range 5 {
			record := createTestRecord(now-int64(i*10), fmt.Sprintf("binary-%d", i))
			require.NoError(t, store.SaveRecord(ctx, &record))
		}

		records, err := store.ListRecords(ctx, ListOptions{Offset: 2})
		require.NoError(t, err)
		assert.Len(t, records, 3)
		assert.Equal(t, "binary-2", records[0].BinaryName)
	})

	t.Run("respects only available filter", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := context.Background()
		now := time.Now().Unix()

		available := createTestRecord(now-100, "available")
		available.TrashAvailable = true

		unavailable := createTestRecord(now, "unavailable")
		unavailable.TrashAvailable = false

		require.NoError(t, store.SaveRecord(ctx, &available))
		require.NoError(t, store.SaveRecord(ctx, &unavailable))

		records, err := store.ListRecords(ctx, ListOptions{OnlyAvailable: true})
		require.NoError(t, err)
		assert.Len(t, records, 1)
		assert.Equal(t, "available", records[0].BinaryName)
	})

	t.Run("combines limit and offset", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := context.Background()
		now := time.Now().Unix()

		for i := range 10 {
			record := createTestRecord(now-int64(i*10), fmt.Sprintf("binary-%d", i))
			require.NoError(t, store.SaveRecord(ctx, &record))
		}

		records, err := store.ListRecords(ctx, ListOptions{Offset: 3, Limit: 3})
		require.NoError(t, err)
		assert.Len(t, records, 3)
		assert.Equal(t, "binary-3", records[0].BinaryName)
		assert.Equal(t, "binary-4", records[1].BinaryName)
		assert.Equal(t, "binary-5", records[2].BinaryName)
	})

	t.Run("fails when database is closed", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		cleanup()

		_, err := store.ListRecords(context.Background(), ListOptions{})
		assert.ErrorIs(t, err, ErrDatabaseClosed)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		// Save records first
		for i := range 5 {
			record := createTestRecord(time.Now().Unix()+int64(i), fmt.Sprintf("binary-%d", i))
			require.NoError(t, store.SaveRecord(context.Background(), &record))
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := store.ListRecords(ctx, ListOptions{})
		assert.ErrorIs(t, err, ErrContextCanceled)
	})
}

func TestBadgerStore_UpdateRecord(t *testing.T) {
	t.Run("successfully updates existing record", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := context.Background()
		timestamp := time.Now().Unix()
		binaryName := testBinaryName

		record := createTestRecord(timestamp, binaryName)
		require.NoError(t, store.SaveRecord(ctx, &record))

		// Update the record
		record.Version = "v2.0.0"
		record.TrashAvailable = false

		err := store.UpdateRecord(ctx, &record)
		require.NoError(t, err)

		// Verify update
		key := GenerateKey(timestamp, binaryName)
		updated, err := store.GetRecord(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, "v2.0.0", updated.Version)
		assert.False(t, updated.TrashAvailable)
	})

	t.Run("fails for non-existent record", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := context.Background()
		record := createTestRecord(time.Now().Unix(), "non-existent")
		err := store.UpdateRecord(ctx, &record)
		assert.ErrorIs(t, err, ErrRecordNotFound)
	})

	t.Run("fails with zero timestamp", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := context.Background()
		record := createTestRecord(0, "test")
		err := store.UpdateRecord(ctx, &record)
		assert.ErrorIs(t, err, ErrInvalidRecord)
	})

	t.Run("fails with empty binary name", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := context.Background()
		record := createTestRecord(time.Now().Unix(), "")
		err := store.UpdateRecord(ctx, &record)
		assert.ErrorIs(t, err, ErrInvalidRecord)
	})

	t.Run("fails when database is closed", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		cleanup()

		ctx := context.Background()
		record := createTestRecord(time.Now().Unix(), "test")
		err := store.UpdateRecord(ctx, &record)
		assert.ErrorIs(t, err, ErrDatabaseClosed)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		record := createTestRecord(time.Now().Unix(), "test")
		err := store.UpdateRecord(ctx, &record)
		assert.ErrorIs(t, err, ErrContextCanceled)
	})
}

func TestBadgerStore_DeleteRecord(t *testing.T) {
	t.Run("successfully deletes existing record", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := context.Background()
		timestamp := time.Now().Unix()
		binaryName := testBinaryName

		record := createTestRecord(timestamp, binaryName)
		require.NoError(t, store.SaveRecord(ctx, &record))

		key := GenerateKey(timestamp, binaryName)
		err := store.DeleteRecord(ctx, key)
		require.NoError(t, err)

		// Verify deletion
		_, err = store.GetRecord(ctx, key)
		assert.ErrorIs(t, err, ErrRecordNotFound)
	})

	t.Run("fails for non-existent key", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := context.Background()
		key := GenerateKey(time.Now().Unix(), "non-existent")
		err := store.DeleteRecord(ctx, key)
		assert.ErrorIs(t, err, ErrRecordNotFound)
	})

	t.Run("fails for invalid key format", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := context.Background()
		err := store.DeleteRecord(ctx, "invalid-key")
		assert.ErrorIs(t, err, ErrInvalidKey)
	})

	t.Run("fails when database is closed", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		cleanup()

		ctx := context.Background()
		key := GenerateKey(time.Now().Unix(), "test")
		err := store.DeleteRecord(ctx, key)
		assert.ErrorIs(t, err, ErrDatabaseClosed)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		key := GenerateKey(time.Now().Unix(), "test")
		err := store.DeleteRecord(ctx, key)
		assert.ErrorIs(t, err, ErrContextCanceled)
	})
}

func TestBadgerStore_DeleteAllRecords(t *testing.T) {
	t.Run("successfully deletes all records", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := context.Background()

		// Create multiple records
		for i := range 5 {
			record := createTestRecord(time.Now().Unix()+int64(i), fmt.Sprintf("binary-%d", i))
			require.NoError(t, store.SaveRecord(ctx, &record))
		}

		// Verify records exist
		records, err := store.ListRecords(ctx, ListOptions{})
		require.NoError(t, err)
		assert.Len(t, records, 5)

		// Delete all
		err = store.DeleteAllRecords(ctx)
		require.NoError(t, err)

		// Verify all deleted
		records, err = store.ListRecords(ctx, ListOptions{})
		require.NoError(t, err)
		assert.Empty(t, records)
	})

	t.Run("succeeds when no records exist", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := context.Background()
		err := store.DeleteAllRecords(ctx)
		require.NoError(t, err)
	})

	t.Run("fails when database is closed", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		cleanup()

		ctx := context.Background()
		err := store.DeleteAllRecords(ctx)
		assert.ErrorIs(t, err, ErrDatabaseClosed)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := context.Background()

		// Create some records first
		for i := range 3 {
			record := createTestRecord(time.Now().Unix()+int64(i), fmt.Sprintf("binary-%d", i))
			require.NoError(t, store.SaveRecord(ctx, &record))
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := store.DeleteAllRecords(ctx)
		assert.ErrorIs(t, err, ErrContextCanceled)
	})
}

func TestBadgerStore_Close(t *testing.T) {
	t.Run("successfully closes database", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		// Don't use defer cleanup() - we want to test Close explicitly

		err := store.Close()
		require.NoError(t, err)
		assert.True(t, store.closed.Load())

		// Cleanup temp directory
		cleanup()
	})

	t.Run("returns error when already closed", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		err := store.Close()
		require.NoError(t, err)

		err = store.Close()
		assert.ErrorIs(t, err, ErrDatabaseClosed)
	})
}

func TestGenerateKey(t *testing.T) {
	tests := []struct {
		name       string
		timestamp  int64
		binaryName string
		expected   string
	}{
		{
			name:       "standard key",
			timestamp:  1709321234,
			binaryName: "golangci-lint",
			expected:   "00000000001709321234:golangci-lint",
		},
		{
			name:       "binary with hyphen",
			timestamp:  1709321234,
			binaryName: "my-binary",
			expected:   "00000000001709321234:my-binary",
		},
		{
			name:       "binary with dot",
			timestamp:  1709321234,
			binaryName: "my.binary",
			expected:   "00000000001709321234:my.binary",
		},
		{
			name:       "zero timestamp",
			timestamp:  0,
			binaryName: "test",
			expected:   "00000000000000000000:test",
		},
		{
			name:       "empty binary name",
			timestamp:  1709321234,
			binaryName: "",
			expected:   "00000000001709321234:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateKey(tt.timestamp, tt.binaryName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseKey(t *testing.T) {
	tests := []struct {
		name               string
		key                string
		expectedTimestamp  int64
		expectedBinaryName string
		expectedErr        bool
	}{
		{
			name:               "valid key",
			key:                "00000000001709321234:golangci-lint",
			expectedTimestamp:  1709321234,
			expectedBinaryName: "golangci-lint",
			expectedErr:        false,
		},
		{
			name:               "binary with hyphen",
			key:                "00000000001709321234:my-binary",
			expectedTimestamp:  1709321234,
			expectedBinaryName: "my-binary",
			expectedErr:        false,
		},
		{
			name:               "multiple colons in binary name",
			key:                "00000000001709321234:my:binary:name",
			expectedTimestamp:  1709321234,
			expectedBinaryName: "my:binary:name",
			expectedErr:        false,
		},
		{
			name:        "missing colon",
			key:         "00000001709321234",
			expectedErr: true,
		},
		{
			name:        "empty key",
			key:         "",
			expectedErr: true,
		},
		{
			name:        "invalid timestamp",
			key:         "invalid:test",
			expectedErr: true,
		},
		{
			name:        "empty binary name",
			key:         "00000000001709321234:",
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timestamp, binaryName, err := ParseKey(tt.key)

			if tt.expectedErr {
				assert.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedTimestamp, timestamp)
			assert.Equal(t, tt.expectedBinaryName, binaryName)
		})
	}
}

func TestHistoryRecord_DisplayTime(t *testing.T) {
	tests := []struct {
		name      string
		timestamp int64
	}{
		{
			name:      "unix epoch",
			timestamp: 0,
		},
		{
			name:      "recent timestamp",
			timestamp: 1709321234,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := HistoryRecord{Timestamp: tt.timestamp}
			result := record.DisplayTime()
			assert.NotEmpty(t, result)
			// Verify format by parsing
			_, err := time.Parse("2006-01-02 15:04:05", result)
			require.NoError(t, err)
		})
	}
}

// Integration test demonstrating full CRUD workflow.
func TestIntegration_CRUDWorkflow(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Create
	record1 := createTestRecord(time.Now().Unix(), "binary1")
	record2 := createTestRecord(time.Now().Unix()+1, "binary2")

	require.NoError(t, store.SaveRecord(ctx, &record1))
	require.NoError(t, store.SaveRecord(ctx, &record2))

	// Read
	key1 := GenerateKey(record1.Timestamp, record1.BinaryName)
	retrieved1, err := store.GetRecord(ctx, key1)
	require.NoError(t, err)
	assert.Equal(t, record1.BinaryName, retrieved1.BinaryName)

	// List
	records, err := store.ListRecords(ctx, ListOptions{})
	require.NoError(t, err)
	assert.Len(t, records, 2)

	// Update
	retrieved1.Version = "v2.0.0"
	require.NoError(t, store.UpdateRecord(ctx, &retrieved1))

	updated, err := store.GetRecord(ctx, key1)
	require.NoError(t, err)
	assert.Equal(t, "v2.0.0", updated.Version)

	// Delete
	require.NoError(t, store.DeleteRecord(ctx, key1))

	_, err = store.GetRecord(ctx, key1)
	require.ErrorIs(t, err, ErrRecordNotFound)

	// Verify only one record remains
	records, err = store.ListRecords(ctx, ListOptions{})
	require.NoError(t, err)
	assert.Len(t, records, 1)
	assert.Equal(t, "binary2", records[0].BinaryName)
}

// Test error wrapping to ensure proper error chain.
func TestErrorWrapping(t *testing.T) {
	t.Run("SaveRecord wraps errors", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		cleanup() // Close immediately to cause errors

		record := createTestRecord(time.Now().Unix(), "test")
		err := store.SaveRecord(context.Background(), &record)

		// Should be ErrDatabaseClosed
		assert.ErrorIs(t, err, ErrDatabaseClosed)
	})

	t.Run("errors can be checked with errors.Is", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := context.Background()

		// Test ErrRecordNotFound wrapping
		key := GenerateKey(time.Now().Unix(), "nonexistent")
		_, err := store.GetRecord(ctx, key)
		assert.ErrorIs(t, err, ErrRecordNotFound)
	})
}
