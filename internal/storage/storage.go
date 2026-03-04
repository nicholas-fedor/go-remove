/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

// Package storage provides KV storage for history records using Badger v4.
//
// This package implements persistent storage for go-remove history records,
// enabling undo and restore functionality. Records are stored with chronologically
// sortable keys for efficient time-based queries.
//
// Key Format:
//   - Keys use the format "<zero-padded-timestamp>:<binary_name>" (e.g., "00000001709321234:golangci-lint")
//   - The timestamp is zero-padded to 20 digits to ensure lexicographic order equals chronological order
//   - Unix timestamps ensure sortability and uniqueness
//
// Storage Location:
//   - Linux: $XDG_DATA_HOME/go-remove/history.badger (fallback: ~/.local/share/go-remove/history.badger)
//   - Windows: %LOCALAPPDATA%/go-remove/history.badger
//
// Usage:
//
//	store, err := storage.NewBadgerStore("/path/to/db")
//	if err != nil {
//	    return err
//	}
//	defer store.Close()
//
//	// Save a record
//	err = store.SaveRecord(ctx, record)
//
//	// Get the most recent record
//	record, err := store.GetMostRecent(ctx)
//
//	// List all records
//	records, err := store.ListRecords(ctx, opts)
package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"sync/atomic"
	"time"

	badger "github.com/dgraph-io/badger/v4"
)

// Common errors for storage operations.
var (
	// ErrRecordNotFound indicates the requested history record does not exist.
	ErrRecordNotFound = errors.New("history record not found")

	// ErrNoHistory indicates no deletion history exists.
	ErrNoHistory = errors.New("no deletion history found")

	// ErrInvalidKey indicates the provided key is malformed.
	ErrInvalidKey = errors.New("invalid record key")

	// ErrInvalidRecord indicates the record data is invalid.
	ErrInvalidRecord = errors.New("invalid record data")

	// ErrDatabaseClosed indicates the database connection is closed.
	ErrDatabaseClosed = errors.New("database connection is closed")

	// ErrContextCanceled indicates the operation was canceled.
	ErrContextCanceled = errors.New("operation canceled")
)

// HistoryRecord represents a single binary removal record.
type HistoryRecord struct {
	// Timestamp is the Unix timestamp when the binary was deleted.
	Timestamp int64 `json:"timestamp"`

	// BinaryName is the name of the binary file (e.g., "golangci-lint").
	BinaryName string `json:"binary_name"`

	// OriginalPath is the full path where the binary was located.
	OriginalPath string `json:"original_path"`

	// TrashPath is the path where the binary is stored in trash (if available).
	TrashPath string `json:"trash_path"`

	// ModulePath is the Go module path (e.g., "github.com/golangci/golangci-lint").
	ModulePath string `json:"module_path"`

	// Version is the semantic version of the binary (e.g., "v1.55.2").
	Version string `json:"version"`

	// VCSRevision is the git commit SHA used to build the binary.
	VCSRevision string `json:"vcs_revision"`

	// VCSTime is the timestamp of the VCS revision.
	VCSTime time.Time `json:"vcs_time"`

	// GoVersion is the Go version used to build the binary.
	GoVersion string `json:"go_version"`

	// BuildInfo contains the complete debug.BuildInfo as JSON.
	// Stored for future-proofing and complete reconstruction capability.
	BuildInfo string `json:"build_info"`

	// Checksum is the SHA256 hash of the binary at deletion time.
	Checksum string `json:"checksum"`

	// TrashAvailable indicates whether the binary is still in trash.
	TrashAvailable bool `json:"trash_available"`

	// OriginalDir is the directory containing the binary (for restoration).
	OriginalDir string `json:"original_dir"`
}

// DisplayTime returns a formatted time string for TUI display.
func (r *HistoryRecord) DisplayTime() string {
	return time.Unix(r.Timestamp, 0).Format("2006-01-02 15:04:05")
}

// ListOptions provides filtering and pagination for record listing.
type ListOptions struct {
	// OnlyAvailable filters to records where TrashAvailable is true.
	OnlyAvailable bool

	// Limit maximum number of records to return (0 = no limit).
	Limit int

	// Offset number of records to skip.
	Offset int
}

// Storer defines operations for history record persistence.
type Storer interface {
	// SaveRecord persists a history record to Badger.
	// The record's Timestamp and BinaryName fields are used to generate the key.
	SaveRecord(ctx context.Context, record *HistoryRecord) error

	// GetRecord retrieves a history record by its composite key.
	GetRecord(ctx context.Context, key string) (HistoryRecord, error)

	// GetMostRecent returns the most recent history record.
	// Returns ErrNoHistory if no records exist.
	GetMostRecent(ctx context.Context) (HistoryRecord, error)

	// ListRecords returns all history records matching the provided options.
	// Records are returned in reverse chronological order (newest first).
	ListRecords(ctx context.Context, opts ListOptions) ([]HistoryRecord, error)

	// UpdateRecord updates an existing history record.
	// The record must have a valid key derived from Timestamp and BinaryName.
	UpdateRecord(ctx context.Context, record *HistoryRecord) error

	// DeleteRecord removes a history record from storage.
	DeleteRecord(ctx context.Context, key string) error

	// DeleteAllRecords removes all history records.
	DeleteAllRecords(ctx context.Context) error

	// Close closes the Badger database.
	Close() error
}

// BadgerStore implements Storer using Badger KV store.
type BadgerStore struct {
	database *badger.DB
	path     string
	closed   atomic.Bool
}

// ValueLogSizeExponent defines the exponent for value log file size calculation (1 << 20 = 1MB).
const ValueLogSizeExponent = 20

// LevelZeroTablesStall defines the conservative stall threshold for L0 tables.
const LevelZeroTablesStall = 2

// NewBadgerStore creates a new Badger-based storage instance.
// The path parameter specifies the directory where the database files will be stored.
// The directory will be created if it does not exist.
func NewBadgerStore(path string) (*BadgerStore, error) {
	// Configure Badger with reasonable defaults for desktop application
	opts := badger.DefaultOptions(path).
		WithSyncWrites(true).                             // Ensure durability
		WithLogger(nil).                                  // Disable verbose logging
		WithValueLogFileSize(1 << ValueLogSizeExponent).  // 1MB value log files
		WithNumMemtables(1).                              // Minimal memory usage
		WithNumLevelZeroTables(1).                        // Minimal memory usage
		WithNumLevelZeroTablesStall(LevelZeroTablesStall) // Conservative stall threshold

	// Silence the unused import warning for runtime - may be used in future platform-specific code
	_ = runtime.GOOS

	database, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("opening badger database at %s: %w", path, err)
	}

	return &BadgerStore{
		database: database,
		path:     path,
	}, nil
}

// SaveRecord persists a history record to Badger.
// The record's Timestamp and BinaryName fields are used to generate the key.
// Returns ErrInvalidRecord if the record is missing required fields.
func (s *BadgerStore) SaveRecord(ctx context.Context, record *HistoryRecord) error {
	if s.closed.Load() {
		return ErrDatabaseClosed
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return fmt.Errorf("%w: %w", ErrContextCanceled, ctx.Err())
	default:
	}

	// Validate required fields
	if record == nil {
		return fmt.Errorf("%w: record is nil", ErrInvalidRecord)
	}

	if record.Timestamp == 0 {
		return fmt.Errorf("%w: timestamp is required", ErrInvalidRecord)
	}

	if record.BinaryName == "" {
		return fmt.Errorf("%w: binary_name is required", ErrInvalidRecord)
	}

	key := GenerateKey(record.Timestamp, record.BinaryName)

	value, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshaling record: %w", err)
	}

	err = s.database.Update(func(txn *badger.Txn) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		return txn.Set([]byte(key), value)
	})
	if err != nil {
		return fmt.Errorf("saving record: %w", err)
	}

	return nil
}

// GetRecord retrieves a history record by its composite key.
// Returns ErrRecordNotFound if the key does not exist.
// Returns ErrInvalidKey if the key format is invalid.
func (s *BadgerStore) GetRecord(ctx context.Context, key string) (HistoryRecord, error) {
	var record HistoryRecord

	if s.closed.Load() {
		return record, ErrDatabaseClosed
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return record, fmt.Errorf("%w: %w", ErrContextCanceled, ctx.Err())
	default:
	}

	// Validate key format
	if _, _, err := ParseKey(key); err != nil {
		return record, fmt.Errorf("%w: %w", ErrInvalidKey, err)
	}

	err := s.database.View(func(txn *badger.Txn) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		item, getErr := txn.Get([]byte(key))
		if getErr != nil {
			if errors.Is(getErr, badger.ErrKeyNotFound) {
				return ErrRecordNotFound
			}

			return fmt.Errorf("getting item from database: %w", getErr)
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &record)
		})
	})
	if err != nil {
		if errors.Is(err, ErrRecordNotFound) {
			return record, ErrRecordNotFound
		}

		return record, fmt.Errorf("getting record: %w", err)
	}

	return record, nil
}

// GetMostRecent returns the most recent history record.
// Returns ErrNoHistory if no records exist.
func (s *BadgerStore) GetMostRecent(ctx context.Context) (HistoryRecord, error) {
	var record HistoryRecord

	if s.closed.Load() {
		return record, ErrDatabaseClosed
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return record, fmt.Errorf("%w: %w", ErrContextCanceled, ctx.Err())
	default:
	}

	err := s.database.View(func(txn *badger.Txn) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		opts := badger.DefaultIteratorOptions
		opts.Reverse = true // Iterate in reverse order to get most recent first

		iterator := txn.NewIterator(opts)
		defer iterator.Close()

		iterator.Rewind()

		if !iterator.Valid() {
			return ErrNoHistory
		}

		item := iterator.Item()

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &record)
		})
	})
	if err != nil {
		if errors.Is(err, ErrNoHistory) {
			return record, ErrNoHistory
		}

		return record, fmt.Errorf("getting most recent record: %w", err)
	}

	return record, nil
}

// ListRecords returns all history records matching the provided options.
// Records are returned in reverse chronological order (newest first).
// Supports filtering by availability and pagination via limit/offset.
func (s *BadgerStore) ListRecords(ctx context.Context, opts ListOptions) ([]HistoryRecord, error) {
	if s.closed.Load() {
		return nil, ErrDatabaseClosed
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("%w: %w", ErrContextCanceled, ctx.Err())
	default:
	}

	var records []HistoryRecord

	err := s.database.View(func(txn *badger.Txn) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		iterOpts := badger.DefaultIteratorOptions
		iterOpts.Reverse = true // Newest first

		iterator := txn.NewIterator(iterOpts)
		defer iterator.Close()

		skipped := 0
		count := 0

		for iterator.Rewind(); iterator.Valid(); iterator.Next() {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			item := iterator.Item()

			var record HistoryRecord

			err := item.Value(func(val []byte) error {
				return json.Unmarshal(val, &record)
			})
			if err != nil {
				// Skip corrupted records but continue iteration
				continue
			}

			// Apply availability filter
			if opts.OnlyAvailable && !record.TrashAvailable {
				continue
			}

			// Apply offset
			if skipped < opts.Offset {
				skipped++

				continue
			}

			records = append(records, record)
			count++

			// Apply limit
			if opts.Limit > 0 && count >= opts.Limit {
				break
			}
		}

		return nil
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, fmt.Errorf("%w: %w", ErrContextCanceled, err)
		}

		return nil, fmt.Errorf("listing records: %w", err)
	}

	return records, nil
}

// UpdateRecord updates an existing history record.
// The record must have a valid key derived from Timestamp and BinaryName.
func (s *BadgerStore) UpdateRecord(ctx context.Context, record *HistoryRecord) error {
	if s.closed.Load() {
		return ErrDatabaseClosed
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return fmt.Errorf("%w: %w", ErrContextCanceled, ctx.Err())
	default:
	}

	// Validate required fields
	if record == nil {
		return fmt.Errorf("%w: record is nil", ErrInvalidRecord)
	}

	if record.Timestamp == 0 {
		return fmt.Errorf("%w: timestamp is required", ErrInvalidRecord)
	}

	if record.BinaryName == "" {
		return fmt.Errorf("%w: binary_name is required", ErrInvalidRecord)
	}

	key := GenerateKey(record.Timestamp, record.BinaryName)

	value, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshaling record: %w", err)
	}

	err = s.database.Update(func(txn *badger.Txn) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Check if key exists
		_, err := txn.Get([]byte(key))
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return ErrRecordNotFound
			}

			return fmt.Errorf("checking key existence: %w", err)
		}

		return txn.Set([]byte(key), value)
	})
	if err != nil {
		if errors.Is(err, ErrRecordNotFound) {
			return ErrRecordNotFound
		}

		return fmt.Errorf("updating record: %w", err)
	}

	return nil
}

// DeleteRecord removes a history record from storage.
// Returns ErrRecordNotFound if the key does not exist.
func (s *BadgerStore) DeleteRecord(ctx context.Context, key string) error {
	if s.closed.Load() {
		return ErrDatabaseClosed
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return fmt.Errorf("%w: %w", ErrContextCanceled, ctx.Err())
	default:
	}

	// Validate key format
	if _, _, err := ParseKey(key); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidKey, err)
	}

	err := s.database.Update(func(txn *badger.Txn) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Check if key exists
		_, err := txn.Get([]byte(key))
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return ErrRecordNotFound
			}

			return fmt.Errorf("checking key existence: %w", err)
		}

		return txn.Delete([]byte(key))
	})
	if err != nil {
		if errors.Is(err, ErrRecordNotFound) {
			return ErrRecordNotFound
		}

		return fmt.Errorf("deleting record: %w", err)
	}

	return nil
}

// DeleteAllRecords removes all history records.
// This operation cannot be undone.
func (s *BadgerStore) DeleteAllRecords(ctx context.Context) error {
	if s.closed.Load() {
		return ErrDatabaseClosed
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return fmt.Errorf("%w: %w", ErrContextCanceled, ctx.Err())
	default:
	}

	err := s.database.Update(func(txn *badger.Txn) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false // Keys only, we don't need values

		iterator := txn.NewIterator(opts)
		defer iterator.Close()

		keys := make([][]byte, 0)

		for iterator.Rewind(); iterator.Valid(); iterator.Next() {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			key := make([]byte, len(iterator.Item().Key()))
			copy(key, iterator.Item().Key())

			keys = append(keys, key)
		}

		for _, key := range keys {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if err := txn.Delete(key); err != nil {
				return fmt.Errorf("deleting key %s: %w", key, err)
			}
		}

		return nil
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return fmt.Errorf("%w: %w", ErrContextCanceled, err)
		}

		return fmt.Errorf("deleting all records: %w", err)
	}

	return nil
}

// Close closes the Badger database.
// Returns an error if the database is already closed.
func (s *BadgerStore) Close() error {
	if s.closed.Load() {
		return ErrDatabaseClosed
	}

	if err := s.database.Close(); err != nil {
		return fmt.Errorf("closing database: %w", err)
	}

	s.closed.Store(true)

	return nil
}
