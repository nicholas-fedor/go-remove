//go:build !windows

/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package storage

import (
	"sync"
	"syscall"
)

// umaskFilePermissions keeps the group and world bits out of files created
// while the database is opened.
const umaskFilePermissions = 0o077

// umaskLock serializes every read of or change to the process file creation
// mask, which is shared by all goroutines.
//
// The mask is not goroutine-local, so a second caller reading it between the
// moment it is set and the moment the database files are created would clear
// it, and those files would land with the ambient permissions instead.
var umaskLock sync.Mutex

// currentUmask reports the process file creation mask, for tests.
//
// Returns:
//   - The mask's permission bits.
func currentUmask() int {
	umaskLock.Lock()
	defer umaskLock.Unlock()

	// Setting the mask to zero is the only way to read it, so the original is
	// restored straight away rather than left pending in a defer.
	saved := syscall.Umask(0)
	syscall.Umask(saved)

	return saved
}

// withRestrictiveUmask runs fn with a file creation mask that leaves new files
// readable only by their owner, then restores the previous mask.
//
// Badger opens its value log without an explicit mode, so that file inherits the
// process mask and would otherwise land world-readable. The other database files
// Badger creates are given 0600 by Badger itself.
//
// The mask is process-wide while open runs, so a file created elsewhere in
// the process at the same time may come out more restrictive than it otherwise
// would.
//
// Parameters:
//   - open: The work to run, which is expected to create the database files.
//
// Returns:
//   - Whatever fn returns.
func withRestrictiveUmask(open func() error) error {
	umaskLock.Lock()
	defer umaskLock.Unlock()

	previous := syscall.Umask(umaskFilePermissions)
	defer syscall.Umask(previous)

	return open()
}
