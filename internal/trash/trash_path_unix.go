//go:build linux || darwin

/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package trash

import (
	"errors"
	"syscall"
)

// isCrossDevice reports whether a move failed because the source and
// destination are on different filesystems.
//
// Parameters:
//   - err: Error returned by the move.
//
// Returns:
//   - True if the move failed for that reason.
func isCrossDevice(err error) bool {
	return errors.Is(err, syscall.EXDEV)
}
