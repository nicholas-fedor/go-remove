/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

// Package storage persists go-remove history records in a Badger key-value
// store under chronologically sortable keys.
package storage

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	badger "github.com/dgraph-io/badger/v4"

	"github.com/nicholas-fedor/go-remove/internal/logger"
)

// ValueLogSizeExponent defines the exponent for value log file size calculation (1 << 20 = 1MB).
const ValueLogSizeExponent = 20

// dbDirPermissions restricts the database directory to its owner, so the files
// created inside it cannot be reached by other users.
const dbDirPermissions = 0o700

// LevelZeroTablesStall defines the conservative stall threshold for L0 tables.
const LevelZeroTablesStall = 2

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

	// ErrRecordKeyExhausted indicates no unused storage key could be found.
	ErrRecordKeyExhausted = errors.New("no unused record key available")

	// ErrTransactionTooLarge indicates a transaction exceeded the database's
	// size limit, so the work it covered did not happen.
	ErrTransactionTooLarge = errors.New("transaction exceeds the database size limit")

	// errRecordKeyOccupied signals that a candidate storage key is already in use.
	errRecordKeyOccupied = errors.New("record key already occupied")
)

var _ Storer = (*BadgerStore)(nil)

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

	// Checksum is the SHA256 hash of the binary at deletion time.
	Checksum string `json:"checksum"`

	// TrashAvailable indicates whether the binary is still in trash.
	TrashAvailable bool `json:"trash_available"`

	// OriginalDir is the directory containing the binary (for restoration).
	OriginalDir string `json:"original_dir"`

	// Key is the storage key this record is stored under.
	//
	// A record written before this field existed leaves it empty, and RecordKey
	// derives the legacy key instead.
	Key string `json:"key,omitempty"`
}

// ListOptions provides filtering and pagination for record listing.
type ListOptions struct {
	// OnlyAvailable filters to records where TrashAvailable is true.
	OnlyAvailable bool

	// Limit is the maximum number of records to return (0 = no limit).
	Limit int

	// Offset is the number of records to skip.
	Offset int
}

// Storer defines operations for history record persistence.
type Storer interface {
	// SaveRecord persists a history record to Badger.
	//
	// Parameters:
	//   - ctx: Context for cancellation.
	//   - record: History record to persist.
	//
	// Returns:
	//   - An error if the record is invalid or the write fails.
	SaveRecord(ctx context.Context, record *HistoryRecord) error

	// GetRecord retrieves a history record by its composite key.
	//
	// Parameters:
	//   - ctx: Context for cancellation.
	//   - key: Composite record key.
	//
	// Returns:
	//   - Matching history record.
	//   - An error if the key is invalid or the record does not exist.
	GetRecord(ctx context.Context, key string) (HistoryRecord, error)

	// GetMostRecent returns the most recent history record.
	//
	// Parameters:
	//   - ctx: Context for cancellation.
	//
	// Returns:
	//   - Newest history record.
	//   - ErrNoHistory if no records exist.
	GetMostRecent(ctx context.Context) (HistoryRecord, error)

	// ListRecords returns history records matching the provided options.
	//
	// Records are returned newest first.
	//
	// Parameters:
	//   - ctx: Context for cancellation.
	//   - opts: Filtering and pagination options.
	//
	// Returns:
	//   - Matching history records.
	//   - An error if the query fails.
	ListRecords(ctx context.Context, opts ListOptions) ([]HistoryRecord, error)

	// UpdateRecord updates an existing history record.
	//
	// Parameters:
	//   - ctx: Context for cancellation.
	//   - record: History record to update.
	//
	// Returns:
	//   - An error if the record does not exist or the write fails.
	UpdateRecord(ctx context.Context, record *HistoryRecord) error

	// DeleteRecord removes a history record from storage.
	//
	// Parameters:
	//   - ctx: Context for cancellation.
	//   - key: Composite record key.
	//
	// Returns:
	//   - An error if the record does not exist or the delete fails.
	DeleteRecord(ctx context.Context, key string) error

	// DeleteAllRecords removes all history records.
	//
	// Parameters:
	//   - ctx: Context for cancellation.
	//
	// Returns:
	//   - An error if the delete fails.
	DeleteAllRecords(ctx context.Context) error

	// Close closes the Badger database.
	//
	// Returns:
	//   - An error if the database is already closed or close fails.
	Close() error
}

// BadgerStore implements Storer using Badger KV store.
type BadgerStore struct {
	database *badger.DB
	path     string
	logger   logger.Logger
	closed   atomic.Bool
}

// RecordKey returns the storage key for the record.
//
// Returns:
//   - The stored key, or the legacy derived key when none was recorded.
//
// translateWriteError marks a Badger transaction that outgrew the size limit, so
// callers can tell it apart from a genuine write failure.
//
// Parameters:
//   - err: Error returned by a Badger write transaction.
//
// Returns:
//   - ErrTransactionTooLarge wrapping err, or err unchanged.
func translateWriteError(err error) error {
	if errors.Is(err, badger.ErrTxnTooBig) {
		return fmt.Errorf("%w: %w", ErrTransactionTooLarge, err)
	}

	return err
}

func (r *HistoryRecord) RecordKey() string {
	if r.Key != "" {
		return r.Key
	}

	return GenerateKey(r.Timestamp, r.BinaryName)
}

// DisplayTime returns a formatted time string for TUI display.
//
// Returns:
//   - Timestamp formatted as "2006-01-02 15:04:05".
func (r *HistoryRecord) DisplayTime() string {
	return time.Unix(r.Timestamp, 0).Format("2006-01-02 15:04:05")
}

// NewBadgerStore creates a new Badger-based storage instance.
//
// The directory is created if it does not exist.
//
// Parameters:
//   - path: Directory where the database files will be stored.
//
// Returns:
//   - Opened Badger store.
//   - An error if the database cannot be opened.
func NewBadgerStore(path string, log logger.Logger) (*BadgerStore, error) {
	// Configure Badger for a desktop application.
	opts := badger.DefaultOptions(path).
		WithSyncWrites(true).
		WithLogger(nil).
		WithValueLogFileSize(1 << ValueLogSizeExponent).
		WithNumMemtables(1).
		WithNumLevelZeroTables(1).
		WithNumLevelZeroTablesStall(LevelZeroTablesStall)

	var database *badger.DB

	// The mask is process-wide for the duration of the open, so files created
	// elsewhere at the same time may come out more restrictive than intended.
	err := withRestrictiveUmask(func() error {
		opened, openErr := badger.Open(opts)
		database = opened

		//nolint:wrapcheck // The caller adds the database path to this error.
		return openErr
	})
	if err != nil {
		return nil, fmt.Errorf("opening badger database at %s: %w", path, err)
	}

	// Badger creates the directory owner-only, but a directory left behind by an
	// earlier install keeps whatever mode it had, and its mode is what later
	// files inherit.
	if err := os.Chmod(path, dbDirPermissions); err != nil {
		if closeErr := database.Close(); closeErr != nil {
			log.Warn(
				"Failed to close database after its directory could not be restricted",
				logger.Err(closeErr),
				logger.Str("path", path),
			)
		}

		return nil, fmt.Errorf("restricting database directory %s to owner access: %w", path, err)
	}

	return &BadgerStore{
		database: database,
		logger:   log,
		path:     path,
	}, nil
}

// checkReady returns an error if the store is closed or ctx is already done.
//
// Parameters:
//   - ctx: Context for cancellation.
//
// Returns:
//   - ErrDatabaseClosed or ErrContextCanceled when the store cannot be used.
func (s *BadgerStore) checkReady(ctx context.Context) error {
	if s.closed.Load() {
		return ErrDatabaseClosed
	}

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("%w: %w", ErrContextCanceled, err)
	}

	return nil
}

// validateRecord returns ErrInvalidRecord when required fields are missing.
//
// Parameters:
//   - record: History record to validate.
//
// Returns:
//   - An error if the record is nil or missing required fields.
func validateRecord(record *HistoryRecord) error {
	if record == nil {
		return fmt.Errorf("%w: record is nil", ErrInvalidRecord)
	}

	if record.Timestamp == 0 {
		return fmt.Errorf("%w: timestamp is required", ErrInvalidRecord)
	}

	if record.BinaryName == "" {
		return fmt.Errorf("%w: binary_name is required", ErrInvalidRecord)
	}

	return nil
}

// saveRecordExclusive writes the record under a key that is free at commit time.
//
// A collision or a conflict is retried with a fresh random discriminator, so two
// deletions of the same binary within one second are both stored instead of one
// replacing the other.
//
// Parameters:
//   - ctx: Context for cancellation.
//   - record: Record to persist. Its Key field is set only on success.
//
// Returns:
//   - An error if the record cannot be marshaled or written, if no key was
//     free, or if the context is canceled.
func (s *BadgerStore) saveRecordExclusive(ctx context.Context, record *HistoryRecord) error {
	base := GenerateKey(record.Timestamp, record.BinaryName)

	for attempt := range maxRecordKeyAttempts {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("%w: %w", ErrContextCanceled, err)
		}

		// The derived key is tried first so the common case keeps the legacy
		// layout.
		key := base
		if attempt > 0 {
			key = base + ":" + randomKeyDiscriminator()
		}

		// Marshal a copy so the caller's record only gains a key on success.
		stored := *record
		stored.Key = key

		value, err := json.Marshal(&stored)
		if err != nil {
			return fmt.Errorf("marshaling record: %w", err)
		}

		// The existence probe and the write share a single read-write
		// transaction. Badger arms conflict detection for write transactions, so
		// a writer that claims the key after this transaction read it turns the
		// commit into badger.ErrConflict rather than an overwrite.
		err = s.database.Update(func(txn *badger.Txn) error {
			_, getErr := txn.Get([]byte(key))

			switch {
			case getErr == nil:
				return errRecordKeyOccupied
			case !errors.Is(getErr, badger.ErrKeyNotFound):
				return fmt.Errorf("probing record key: %w", getErr)
			}

			return txn.Set([]byte(key), value)
		})

		switch {
		case err == nil:
			record.Key = key

			return nil
		case errors.Is(err, errRecordKeyOccupied), errors.Is(err, badger.ErrConflict):
			// Occupied, or claimed by a writer that committed after this
			// transaction read the key, so retry with a fresh discriminator.
			continue
		default:
			return fmt.Errorf("writing record: %w", translateWriteError(err))
		}
	}

	return fmt.Errorf("%w: %s", ErrRecordKeyExhausted, record.BinaryName)
}

// SaveRecord persists a history record to Badger.
//
// A key already held by another record is left alone: the new record is stored
// alongside it under a key carrying a random discriminator, since overwriting
// would destroy the only index for a copy of the binary already in trash.
//
// Parameters:
//   - ctx: Context for cancellation.
//   - record: History record to persist.
//
// Returns:
//   - An error if the record is invalid or the write fails.
func (s *BadgerStore) SaveRecord(ctx context.Context, record *HistoryRecord) error {
	if err := s.checkReady(ctx); err != nil {
		return err
	}

	if err := validateRecord(record); err != nil {
		return err
	}

	if err := s.saveRecordExclusive(ctx, record); err != nil {
		return err
	}

	return nil
}

// GetRecord retrieves a history record by its composite key.
//
// Parameters:
//   - ctx: Context for cancellation.
//   - key: Composite record key.
//
// Returns:
//   - Matching history record.
//   - An error if the key is invalid or the record does not exist.
func (s *BadgerStore) GetRecord(ctx context.Context, key string) (HistoryRecord, error) {
	var record HistoryRecord

	if err := s.checkReady(ctx); err != nil {
		return record, err
	}

	if _, _, err := ParseKey(key); err != nil {
		return record, fmt.Errorf("%w: %w", ErrInvalidKey, err)
	}

	err := s.database.View(func(txn *badger.Txn) error {
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
//
// Records that fail to decode are skipped rather than aborting the search, so a
// single corrupt value cannot deny access to every valid record beneath it.
//
// Parameters:
//   - ctx: Context for cancellation.
//
// Returns:
//   - Newest decodable history record.
//   - ErrNoHistory if no decodable records exist.
func (s *BadgerStore) GetMostRecent(ctx context.Context) (HistoryRecord, error) {
	var record HistoryRecord

	if err := s.checkReady(ctx); err != nil {
		return record, err
	}

	err := s.database.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Reverse = true
		opts.PrefetchValues = false

		iterator := txn.NewIterator(opts)
		defer iterator.Close()

		for iterator.Rewind(); iterator.Valid(); iterator.Next() {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("%w: %w", ErrContextCanceled, err)
			}

			var candidate HistoryRecord

			if err := iterator.Item().Value(func(val []byte) error {
				return json.Unmarshal(val, &candidate)
			}); err != nil {
				continue
			}

			record = candidate

			return nil
		}

		return ErrNoHistory
	})
	if err != nil {
		if errors.Is(err, ErrNoHistory) {
			return record, ErrNoHistory
		}

		return record, fmt.Errorf("getting most recent record: %w", err)
	}

	return record, nil
}

// ListRecords returns history records matching the provided options.
//
// Records are returned newest first and may be filtered or paginated.
//
// Parameters:
//   - ctx: Context for cancellation.
//   - opts: Filtering and pagination options.
//
// Returns:
//   - Matching history records.
//   - An error if the query fails.
func (s *BadgerStore) ListRecords(ctx context.Context, opts ListOptions) ([]HistoryRecord, error) {
	if err := s.checkReady(ctx); err != nil {
		return nil, err
	}

	var records []HistoryRecord

	err := s.database.View(func(txn *badger.Txn) error {
		iterOpts := badger.DefaultIteratorOptions
		iterOpts.Reverse = true

		iterator := txn.NewIterator(iterOpts)
		defer iterator.Close()

		skipped := 0
		count := 0

		for iterator.Rewind(); iterator.Valid(); iterator.Next() {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("%w: %w", ErrContextCanceled, err)
			}

			var record HistoryRecord

			if err := iterator.Item().Value(func(val []byte) error {
				return json.Unmarshal(val, &record)
			}); err != nil {
				continue
			}

			if opts.OnlyAvailable && !record.TrashAvailable {
				continue
			}

			if skipped < opts.Offset {
				skipped++

				continue
			}

			records = append(records, record)
			count++

			if opts.Limit > 0 && count >= opts.Limit {
				break
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listing records: %w", err)
	}

	return records, nil
}

// UpdateRecord updates an existing history record.
//
// Parameters:
//   - ctx: Context for cancellation.
//   - record: History record to update.
//
// Returns:
//   - An error if the record does not exist or the write fails.
func (s *BadgerStore) UpdateRecord(ctx context.Context, record *HistoryRecord) error {
	if err := s.checkReady(ctx); err != nil {
		return err
	}

	if err := validateRecord(record); err != nil {
		return err
	}

	value, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshaling record: %w", err)
	}

	// Use the recorded key so an update lands on the same entry the record was
	// saved under, including when a discriminator was appended.
	key := record.RecordKey()

	err = s.database.Update(func(txn *badger.Txn) error {
		if _, err := txn.Get([]byte(key)); err != nil {
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

		return fmt.Errorf("updating record: %w", translateWriteError(err))
	}

	return nil
}

// DeleteRecord removes a history record from storage.
//
// Parameters:
//   - ctx: Context for cancellation.
//   - key: Composite record key.
//
// Returns:
//   - An error if the record does not exist or the delete fails.
func (s *BadgerStore) DeleteRecord(ctx context.Context, key string) error {
	if err := s.checkReady(ctx); err != nil {
		return err
	}

	if _, _, err := ParseKey(key); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidKey, err)
	}

	err := s.database.Update(func(txn *badger.Txn) error {
		if _, err := txn.Get([]byte(key)); err != nil {
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

		return fmt.Errorf("deleting record: %w", translateWriteError(err))
	}

	return nil
}

// DeleteAllRecords removes all history records.
//
// This operation cannot be undone.
//
// Parameters:
//   - ctx: Context for cancellation.
//
// Returns:
//   - An error if the delete fails.
func (s *BadgerStore) DeleteAllRecords(ctx context.Context) error {
	if err := s.checkReady(ctx); err != nil {
		return err
	}

	err := s.database.Update(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false

		iterator := txn.NewIterator(opts)
		defer iterator.Close()

		var keys [][]byte

		for iterator.Rewind(); iterator.Valid(); iterator.Next() {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("%w: %w", ErrContextCanceled, err)
			}

			// An iterator's key is only valid until it moves, so the keys are
			// gathered here and deleted afterwards.
			key := make([]byte, len(iterator.Item().Key()))
			copy(key, iterator.Item().Key())
			keys = append(keys, key)
		}

		for _, key := range keys {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("%w: %w", ErrContextCanceled, err)
			}

			if err := txn.Delete(key); err != nil {
				return fmt.Errorf("deleting key %s: %w", key, err)
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("deleting all records: %w", translateWriteError(err))
	}

	return nil
}

// Close closes the Badger database.
//
// Returns:
//   - An error if the database is already closed or close fails.
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
