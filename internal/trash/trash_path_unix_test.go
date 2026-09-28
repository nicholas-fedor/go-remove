//go:build linux || darwin

/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package trash

import (
	"errors"
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestIsCrossDevice_Unix verifies the cross-device predicate on unix, where
// EXDEV is the real code a rename returns across filesystems.
func TestIsCrossDevice_Unix(t *testing.T) {
	t.Parallel()

	assert.True(t, isCrossDevice(syscall.EXDEV))
	assert.True(t, isCrossDevice(&os.LinkError{
		Op:  "rename",
		Old: "/one/volume/file",
		New: "/other/volume/file",
		Err: syscall.EXDEV,
	}))

	// Anything else is a real failure and must not reach the copy fallback,
	// which would duplicate the file and then fail to unlink the original.
	assert.False(t, isCrossDevice(nil))
	assert.False(t, isCrossDevice(syscall.EACCES))
	assert.False(t, isCrossDevice(syscall.EROFS))
	assert.False(t, isCrossDevice(errors.New("some other failure")))
}
