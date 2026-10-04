/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package storage

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	json "encoding/json/v2"
)

// testBinaryName is a constant for the test binary name to avoid magic strings.
const testBinaryName = "test-binary"

// setupTestStore creates a temporary Badger store for testing.
// Returns the store and a cleanup function.
func setupTestStore(t *testing.T) (*BadgerStore, func()) {
	t.Helper()

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

		ctx := t.Context()
		record := createTestRecord(time.Now().Unix(), testBinaryName)
		err := store.SaveRecord(ctx, &record)
		require.NoError(t, err)
	})

	t.Run("fails with zero timestamp", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := t.Context()
		record := createTestRecord(0, testBinaryName)
		err := store.SaveRecord(ctx, &record)
		require.ErrorIs(t, err, ErrInvalidRecord)
		assert.Contains(t, err.Error(), "timestamp is required")
	})

	t.Run("fails with empty binary name", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := t.Context()
		record := createTestRecord(time.Now().Unix(), "")
		err := store.SaveRecord(ctx, &record)
		require.ErrorIs(t, err, ErrInvalidRecord)
		assert.Contains(t, err.Error(), "binary_name is required")
	})

	t.Run("fails when database is closed", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		cleanup() // Close immediately

		ctx := t.Context()
		record := createTestRecord(time.Now().Unix(), testBinaryName)
		err := store.SaveRecord(ctx, &record)
		assert.ErrorIs(t, err, ErrDatabaseClosed)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx, cancel := context.WithCancel(t.Context())
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

		ctx := t.Context()
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

		ctx := t.Context()
		key := GenerateKey(time.Now().Unix(), "non-existent")
		_, err := store.GetRecord(ctx, key)
		assert.ErrorIs(t, err, ErrRecordNotFound)
	})

	t.Run("returns error for invalid key format", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := t.Context()
		_, err := store.GetRecord(ctx, "invalid-key")
		assert.ErrorIs(t, err, ErrInvalidKey)
	})

	t.Run("fails when database is closed", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		cleanup()

		ctx := t.Context()
		key := GenerateKey(time.Now().Unix(), "test")
		_, err := store.GetRecord(ctx, key)
		assert.ErrorIs(t, err, ErrDatabaseClosed)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		key := GenerateKey(time.Now().Unix(), "test")
		_, err := store.GetRecord(ctx, key)
		assert.ErrorIs(t, err, ErrContextCanceled)
	})
}

func TestBadgerStore_GetMostRecent(t *testing.T) {
	t.Run("skips malformed records and returns the newest valid one", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := t.Context()
		now := time.Now().Unix()

		valid := createTestRecord(now-100, "valid-binary")
		require.NoError(t, store.SaveRecord(ctx, &valid))

		// A value that cannot be decoded, stored under a key that sorts above
		// the valid record. Reverse iteration reaches it first.
		corruptKey := GenerateKey(now, "corrupt-binary")

		require.NoError(t, store.database.Update(func(txn *badger.Txn) error {
			return txn.Set([]byte(corruptKey), []byte("{not valid json"))
		}))

		mostRecent, err := store.GetMostRecent(ctx)
		require.NoError(t, err, "a malformed record must not block valid ones")
		assert.Equal(t, "valid-binary", mostRecent.BinaryName)
		assert.Equal(t, now-100, mostRecent.Timestamp)
		assert.Equal(t, valid.RecordKey(), mostRecent.RecordKey())
	})

	t.Run("returns error when every record is malformed", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := t.Context()

		corruptKey := GenerateKey(time.Now().Unix(), "corrupt-binary")

		require.NoError(t, store.database.Update(func(txn *badger.Txn) error {
			return txn.Set([]byte(corruptKey), []byte("{not valid json"))
		}))

		_, err := store.GetMostRecent(ctx)
		assert.ErrorIs(t, err, ErrNoHistory,
			"history of only malformed records reports no history")
	})

	t.Run("returns error when no records exist", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := t.Context()
		_, err := store.GetMostRecent(ctx)
		assert.ErrorIs(t, err, ErrNoHistory)
	})

	t.Run("returns most recent record", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := t.Context()
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

		ctx := t.Context()
		_, err := store.GetMostRecent(ctx)
		assert.ErrorIs(t, err, ErrDatabaseClosed)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := t.Context()
		record := createTestRecord(time.Now().Unix(), "test")
		require.NoError(t, store.SaveRecord(ctx, &record))

		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		_, err := store.GetMostRecent(ctx)
		assert.ErrorIs(t, err, ErrContextCanceled)
	})
}

func TestBadgerStore_ListRecords(t *testing.T) {
	t.Run("returns empty list when no records", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := t.Context()
		records, err := store.ListRecords(ctx, ListOptions{})
		require.NoError(t, err)
		assert.Empty(t, records)
	})

	t.Run("returns all records in reverse chronological order", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := t.Context()
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

		ctx := t.Context()
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

		ctx := t.Context()
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

		ctx := t.Context()
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

		ctx := t.Context()
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

		_, err := store.ListRecords(t.Context(), ListOptions{})
		assert.ErrorIs(t, err, ErrDatabaseClosed)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		// Save records first
		for i := range 5 {
			record := createTestRecord(time.Now().Unix()+int64(i), fmt.Sprintf("binary-%d", i))
			require.NoError(t, store.SaveRecord(t.Context(), &record))
		}

		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		_, err := store.ListRecords(ctx, ListOptions{})
		assert.ErrorIs(t, err, ErrContextCanceled)
	})
}

func TestBadgerStore_UpdateRecord(t *testing.T) {
	t.Run("successfully updates existing record", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := t.Context()
		timestamp := time.Now().Unix()
		binaryName := testBinaryName

		record := createTestRecord(timestamp, binaryName)
		require.NoError(t, store.SaveRecord(ctx, &record))

		// Update the record
		record.Version = "v2.0.0"
		record.TrashAvailable = false

		err := store.UpdateRecord(ctx, &record)
		require.NoError(t, err)

		key := GenerateKey(timestamp, binaryName)
		updated, err := store.GetRecord(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, "v2.0.0", updated.Version)
		assert.False(t, updated.TrashAvailable)
	})

	t.Run("fails for non-existent record", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := t.Context()
		record := createTestRecord(time.Now().Unix(), "non-existent")
		err := store.UpdateRecord(ctx, &record)
		assert.ErrorIs(t, err, ErrRecordNotFound)
	})

	t.Run("fails with zero timestamp", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := t.Context()
		record := createTestRecord(0, "test")
		err := store.UpdateRecord(ctx, &record)
		assert.ErrorIs(t, err, ErrInvalidRecord)
	})

	t.Run("fails with empty binary name", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := t.Context()
		record := createTestRecord(time.Now().Unix(), "")
		err := store.UpdateRecord(ctx, &record)
		assert.ErrorIs(t, err, ErrInvalidRecord)
	})

	t.Run("fails when database is closed", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		cleanup()

		ctx := t.Context()
		record := createTestRecord(time.Now().Unix(), "test")
		err := store.UpdateRecord(ctx, &record)
		assert.ErrorIs(t, err, ErrDatabaseClosed)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx, cancel := context.WithCancel(t.Context())
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

		ctx := t.Context()
		timestamp := time.Now().Unix()
		binaryName := testBinaryName

		record := createTestRecord(timestamp, binaryName)
		require.NoError(t, store.SaveRecord(ctx, &record))

		key := GenerateKey(timestamp, binaryName)
		err := store.DeleteRecord(ctx, key)
		require.NoError(t, err)

		_, err = store.GetRecord(ctx, key)
		assert.ErrorIs(t, err, ErrRecordNotFound)
	})

	t.Run("fails for non-existent key", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := t.Context()
		key := GenerateKey(time.Now().Unix(), "non-existent")
		err := store.DeleteRecord(ctx, key)
		assert.ErrorIs(t, err, ErrRecordNotFound)
	})

	t.Run("fails for invalid key format", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := t.Context()
		err := store.DeleteRecord(ctx, "invalid-key")
		assert.ErrorIs(t, err, ErrInvalidKey)
	})

	t.Run("fails when database is closed", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		cleanup()

		ctx := t.Context()
		key := GenerateKey(time.Now().Unix(), "test")
		err := store.DeleteRecord(ctx, key)
		assert.ErrorIs(t, err, ErrDatabaseClosed)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx, cancel := context.WithCancel(t.Context())
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

		ctx := t.Context()

		for i := range 5 {
			record := createTestRecord(time.Now().Unix()+int64(i), fmt.Sprintf("binary-%d", i))
			require.NoError(t, store.SaveRecord(ctx, &record))
		}

		records, err := store.ListRecords(ctx, ListOptions{})
		require.NoError(t, err)
		assert.Len(t, records, 5)

		// Delete all
		err = store.DeleteAllRecords(ctx)
		require.NoError(t, err)

		records, err = store.ListRecords(ctx, ListOptions{})
		require.NoError(t, err)
		assert.Empty(t, records)
	})

	t.Run("succeeds when no records exist", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := t.Context()
		err := store.DeleteAllRecords(ctx)
		require.NoError(t, err)
	})

	t.Run("fails when database is closed", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		cleanup()

		ctx := t.Context()
		err := store.DeleteAllRecords(ctx)
		assert.ErrorIs(t, err, ErrDatabaseClosed)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := t.Context()

		for i := range 3 {
			record := createTestRecord(time.Now().Unix()+int64(i), fmt.Sprintf("binary-%d", i))
			require.NoError(t, store.SaveRecord(ctx, &record))
		}

		ctx, cancel := context.WithCancel(t.Context())
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

// TestSaveRecord_KeyCollisionPreservesBothRecords verifies that two records
// sharing a timestamp and basename are both stored.
//
// The first record is the only index for a copy of the binary already sitting in
// trash, so overwriting it would make that copy unreachable.
func TestSaveRecord_KeyCollisionPreservesBothRecords(t *testing.T) {
	t.Parallel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := t.Context()
	timestamp := time.Now().Unix()

	first := HistoryRecord{
		Timestamp:  timestamp,
		BinaryName: "tool",
		OriginalPath: filepath.Join(
			t.TempDir(),
			"first",
			"tool",
		),
		TrashPath:      "/trash/files/tool_aaa",
		TrashAvailable: true,
		ModulePath:     "example.com/first",
		OriginalDir:    "/first/bin",
	}
	second := HistoryRecord{
		Timestamp:  timestamp,
		BinaryName: "tool",
		OriginalPath: filepath.Join(
			t.TempDir(),
			"second",
			"tool",
		),
		TrashPath:      "/trash/files/tool_bbb",
		TrashAvailable: true,
		ModulePath:     "example.com/second",
		OriginalDir:    "/second/bin",
	}

	require.NoError(t, store.SaveRecord(ctx, &first))
	require.NoError(t, store.SaveRecord(ctx, &second))

	// The colliding key must have been separated.
	assert.NotEqual(t, first.Key, second.Key,
		"colliding records were stored under the same key")
	assert.Equal(t, GenerateKey(timestamp, "tool"), first.Key,
		"the first record should keep the legacy key layout")

	// Both records must be retrievable, each with its own content intact.
	for _, want := range []HistoryRecord{first, second} {
		got, err := store.GetRecord(ctx, want.Key)
		require.NoErrorf(t, err, "record %s is unreachable", want.OriginalPath)
		assert.Equal(t, want.OriginalPath, got.OriginalPath)
		assert.Equal(t, want.ModulePath, got.ModulePath)
		assert.Equal(t, want.TrashPath, got.TrashPath)
	}

	// The history must report both entries.
	records, err := store.ListRecords(ctx, ListOptions{})
	require.NoError(t, err)
	assert.Len(t, records, 2, "a colliding record was lost")
}

// TestSaveRecord_UpdateUsesRecordedKey verifies that updating a record that was
// stored under a discriminated key updates that same entry.
func TestSaveRecord_UpdateUsesRecordedKey(t *testing.T) {
	t.Parallel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := t.Context()
	timestamp := time.Now().Unix()

	// Occupy the legacy key so the record below is forced onto a new one.
	require.NoError(t, store.SaveRecord(ctx, &HistoryRecord{
		Timestamp:      timestamp,
		BinaryName:     "tool",
		OriginalPath:   "/first/bin/tool",
		TrashAvailable: true,
	}))

	collided := HistoryRecord{
		Timestamp:      timestamp,
		BinaryName:     "tool",
		OriginalPath:   "/second/bin/tool",
		TrashPath:      "/trash/files/tool_bbb",
		TrashAvailable: true,
	}
	require.NoError(t, store.SaveRecord(ctx, &collided))
	require.NotEqual(t, GenerateKey(timestamp, "tool"), collided.Key)

	collided.TrashAvailable = false
	require.NoError(t, store.UpdateRecord(ctx, &collided))

	// The update must land on the recorded key, not create a duplicate at the
	// legacy key.
	got, err := store.GetRecord(ctx, collided.Key)
	require.NoError(t, err)
	assert.False(t, got.TrashAvailable)
	assert.Equal(t, "/second/bin/tool", got.OriginalPath)

	records, err := store.ListRecords(ctx, ListOptions{})
	require.NoError(t, err)
	assert.Len(t, records, 2, "update created a duplicate record")
}

// TestHistoryRecord_RecordKey verifies the legacy fallback for records written
// before the Key field existed.
func TestHistoryRecord_RecordKey(t *testing.T) {
	t.Parallel()

	legacy := HistoryRecord{Timestamp: 1709321234, BinaryName: "tool"}
	assert.Equal(t, GenerateKey(1709321234, "tool"), legacy.RecordKey())

	recorded := HistoryRecord{
		Timestamp:  1709321234,
		BinaryName: "tool",
		Key:        "00000000001709321234:tool:abcdef012345",
	}
	assert.Equal(t, recorded.Key, recorded.RecordKey(),
		"a recorded key must take precedence over the derived one")
}

// TestSaveRecord_ConcurrentCollision verifies that concurrent saves sharing a
// timestamp and basename all survive.
//
// Probing for a free key in a separate read transaction left a window in which
// two writers could both observe the key as free and then both write it, so the
// second silently replaced the first. Probing inside the write transaction lets
// Badger's conflict detection reject the loser, which retries on a new key.
func TestSaveRecord_ConcurrentCollision(t *testing.T) {
	t.Parallel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := t.Context()
	timestamp := time.Now().Unix()

	const writers = 32

	// Release every writer at once so they contend for the same key instead of
	// trickling in, which is what makes a probe-then-write race observable.
	start := make(chan struct{})

	var (
		waitGroup sync.WaitGroup
		mu        sync.Mutex
		keys      = make([]string, 0, writers)
		failures  []error
	)

	for i := range writers {
		waitGroup.Go(func() {
			<-start

			record := HistoryRecord{
				Timestamp:      timestamp,
				BinaryName:     "tool",
				OriginalPath:   fmt.Sprintf("/bin/tool-%d", i),
				TrashPath:      fmt.Sprintf("/trash/files/tool_%d", i),
				TrashAvailable: true,
			}

			if err := store.SaveRecord(ctx, &record); err != nil {
				mu.Lock()
				defer mu.Unlock()

				failures = append(failures, err)

				return
			}

			mu.Lock()
			defer mu.Unlock()

			keys = append(keys, record.Key)
		})
	}

	close(start)

	waitGroup.Wait()

	require.Empty(t, failures, "concurrent saves reported errors")
	require.Len(t, keys, writers, "every writer must receive a key")

	// No key may be handed to two writers.
	unique := make(map[string]struct{}, writers)
	for _, key := range keys {
		_, duplicate := unique[key]
		require.Falsef(t, duplicate, "two writers received key %s", key)

		unique[key] = struct{}{}
	}

	// Every record must be readable back under its own key.
	records, err := store.ListRecords(ctx, ListOptions{})
	require.NoError(t, err)
	assert.Len(t, records, writers, "a concurrent save was lost")
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
			_, err := time.Parse("2006-01-02 15:04:05", result)
			require.NoError(t, err)
		})
	}
}

// Integration test demonstrating full CRUD workflow.
func TestIntegration_CRUDWorkflow(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := t.Context()

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
		err := store.SaveRecord(t.Context(), &record)

		// Should be ErrDatabaseClosed
		assert.ErrorIs(t, err, ErrDatabaseClosed)
	})

	t.Run("errors can be checked with errors.Is", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		ctx := t.Context()

		// Test ErrRecordNotFound wrapping
		key := GenerateKey(time.Now().Unix(), "nonexistent")
		_, err := store.GetRecord(ctx, key)
		assert.ErrorIs(t, err, ErrRecordNotFound)
	})
}

// TestDeserialize_LegacyRecordWithBuildInfo verifies a record that still
// carries the BuildInfo field decodes.
//
// The field carried the full debug.BuildInfo, so a database written by an
// earlier version still holds it. Undecoded keys are ignored rather than
// rejected, so those records stay readable and are simply rewritten without it
// on their next save.
func TestDeserialize_LegacyRecordWithBuildInfo(t *testing.T) {
	t.Parallel()

	legacy := []byte(
		`{"binary_name":"vhs","checksum":"abc123",` +
			`"build_info":"{\"path\":\"github.com/test\",\"-ldflags\":\"-X main.token=s3cret\"}"}`,
	)

	var record HistoryRecord

	require.NoError(t, json.Unmarshal(legacy, &record))

	assert.Equal(t, "vhs", record.BinaryName)
	assert.Equal(t, "abc123", record.Checksum)
}
