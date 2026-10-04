//go:build windows

/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package trash

import (
	"errors"
	"syscall"

	"golang.org/x/sys/windows"
)

// isCrossDevice reports whether a move failed because the source and
// destination are on different filesystems.
//
// syscall.EXDEV is an invented value on Windows that no API ever returns; the
// real code for a cross-volume move is ERROR_NOT_SAME_DEVICE. Matching that one
// directly is what keeps a cross-volume move from being reported as a hard
// failure instead of falling back to a copy.
//
// Parameters:
//   - err: Error returned by the move.
//
// Returns:
//   - True if the move failed for that reason.
func isCrossDevice(err error) bool {
	return errors.Is(err, syscall.EXDEV) ||
		errors.Is(err, windows.ERROR_NOT_SAME_DEVICE)
}
