/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nicholas-fedor/go-remove/internal/buildinfo"
	"github.com/nicholas-fedor/go-remove/internal/history"
	"github.com/nicholas-fedor/go-remove/internal/logger"
	"github.com/nicholas-fedor/go-remove/internal/paths"
	"github.com/nicholas-fedor/go-remove/internal/storage"
	"github.com/nicholas-fedor/go-remove/internal/trash"
)

// openHistory creates a history manager backed by the trash and the on-disk
// history database.
//
// Parameters:
//   - log: logger for the manager and its store.
//
// Returns:
//   - history.Manager: the manager, which owns and closes the store.
//   - error: non-nil when any dependency fails to initialize.
func openHistory(log logger.Logger) (history.Manager, error) {
	trasher, err := trash.NewTrasher()
	if err != nil {
		return nil, fmt.Errorf("initializing trash: %w", err)
	}

	dbPath, err := paths.StoragePath()
	if err != nil {
		return nil, fmt.Errorf("determining storage path: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), paths.DirPermissions); err != nil {
		return nil, fmt.Errorf("creating storage directory: %w", err)
	}

	storer, err := storage.NewBadgerStore(dbPath, log)
	if err != nil {
		return nil, fmt.Errorf("initializing storage: %w", err)
	}

	// Close only on failure, since the manager closes the store on success. The
	// defer reads the one err every return path assigns, so nothing may shadow it.
	defer func() {
		if err != nil {
			if closeErr := storer.Close(); closeErr != nil {
				log.Warn(
					"Failed to close storage after initialization error",
					logger.Err(closeErr),
				)
			}
		}
	}()

	extractor, err := buildinfo.NewExtractor()
	if err != nil {
		return nil, fmt.Errorf("initializing build info extractor: %w", err)
	}

	return history.NewManager(trasher, storer, extractor, log), nil
}
