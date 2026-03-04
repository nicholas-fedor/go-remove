/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

// Package history provides high-level undo/restore operations for go-remove.
//
// This package serves as the orchestration layer that coordinates trash management,
// persistent storage, and build info extraction to provide a unified interface
// for managing deletion history and restoration operations.
//
// The history package enables users to:
//   - Record binary deletions with full metadata
//   - Undo the most recent deletion
//   - Restore specific binaries by history entry ID
//   - View deletion history with filtering and pagination
//   - Permanently delete binaries from trash
//   - Clear history entries (with optional trash clearing)
//
// Architecture:
//
//	+-------------+     +-------------+     +-------------+
//	|   Manager   |---->|   Trash     |     |   Storage   |
//	|  (history)  |     |  (trash)    |<--->|  (storage)  |
//	+-------------+     +-------------+     +-------------+
//	       |
//	       v
//	+-------------+
//	|  BuildInfo  |
//	| (buildinfo) |
//	+-------------+
//
// Usage:
//
//	// Initialize dependencies
//	trasher, _ := trash.NewTrasher()
//	store, _ := storage.NewBadgerStore(dbPath)
//	extractor, _ := buildinfo.NewExtractor()
//	logger, _ := logger.NewLogger()
//
//	// Create history manager
//	:= history.NewManager(trasher, store, extractor, logger)
//	defer manager.Close()
//
//	// Record a deletion
//	entry, err := manager.RecordDeletion(ctx, "/usr/local/bin/myapp")
//
//	// Undo the most recent deletion
//	result, err := manager.UndoMostRecent(ctx)
//
//	// Restore a specific entry
//	result, err := manager.Restore(ctx, entry.ID)
//
//	// Get deletion history
//	history, err := manager.GetHistory(ctx, 10)
//
// Platform Support:
//   - Linux: Full support via XDG trash specification
//   - Windows: Full support via Windows Recycle Bin
//   - Darwin: Not supported
//
// Error Handling:
// The package defines specific error types for common failure scenarios.
// Errors are wrapped with context using fmt.Errorf and %w verb for
// error chain inspection with errors.Is().
package history
