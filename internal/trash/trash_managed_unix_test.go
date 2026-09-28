//go:build linux || darwin

/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package trash

import (
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestMoveToTrash_FifoIsRefused verifies that a named pipe is rejected rather
// than opened, which would block forever waiting for a writer.
//
// Creating a named pipe is a POSIX operation with no Windows equivalent, so
// this case lives in a unix-only file.
func TestMoveToTrash_FifoIsRefused(t *testing.T) {
	t.Parallel()

	trasher, ok := newTestTrasher(t).(*managedTrasher)
	require.True(t, ok)

	fifo := filepath.Join(t.TempDir(), "pipe")
	require.NoError(t, syscall.Mkfifo(fifo, 0o600))

	// The copy path is what would hang, so drive it directly rather than
	// through MoveToTrash, whose rename succeeds on a single filesystem.
	err := trasher.copyAndDelete(fifo, filepath.Join(t.TempDir(), "pipe-copy"))
	require.ErrorIs(t, err, ErrUnsupportedFileType)
}
