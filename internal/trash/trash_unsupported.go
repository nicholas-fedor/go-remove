//go:build !linux && !darwin && !windows

/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package trash

import (
	"fmt"
	"runtime"
)

// newTrasher reports that no trash implementation exists for this platform.
//
// Without this the package fails to compile with an undefined newTrasher
// pointing at the shared file, which gives no hint about the real cause.
//
// Returns:
//   - Always an error wrapping ErrUnsupportedPlatform.
func newTrasher() (Trasher, error) {
	return nil, fmt.Errorf("%w: %s", ErrUnsupportedPlatform, runtime.GOOS)
}

// newTrasherAt reports that no trash implementation exists for this platform.
//
// Returns:
//   - Always an error wrapping ErrUnsupportedPlatform.
func newTrasherAt(_ string) (Trasher, error) {
	return newTrasher()
}
