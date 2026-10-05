//go:build !windows

/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBadgerStore_FilesAreNotWorldReadable verifies the database files are
// created without group or world access.
//
// Badger opens its value log without an explicit mode, so that file inherits the
// process creation mask and would otherwise land world-readable. The records it
// holds carry filesystem paths, module graphs and checksums for every binary
// that was removed.
func TestBadgerStore_FilesAreNotWorldReadable(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "history.badger")

	store, err := NewBadgerStore(dbPath)
	require.NoError(t, err)
	require.NoError(t, store.Close())

	require.NoError(t, filepath.Walk(dbPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		assert.Zero(t, info.Mode().Perm()&0o077,
			"%s must not be accessible to group or world, got %04o",
			info.Name(), info.Mode().Perm())

		return nil
	}))
}

// TestWithRestrictiveUmask_RestoresPrevious verifies the mask is restored after
// the guarded work, so nothing else in the process inherits it.
func TestWithRestrictiveUmask_RestoresPrevious(t *testing.T) {
	before := currentUmask()

	require.NoError(t, withRestrictiveUmask(func() error { return nil }))

	assert.Equal(t, before, currentUmask(), "the mask must be restored")
}
