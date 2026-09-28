//go:build windows

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
	"golang.org/x/sys/windows"
)

// TestIsCrossDevice_Windows verifies the cross-device predicate on Windows.
//
// syscall.EXDEV is declared there as an offset into the application error range
// and is never returned by a Windows API, so matching it alone would report
// every cross-volume move as a hard failure. ERROR_NOT_SAME_DEVICE is the code
// MoveFileEx actually returns, and it is what must be recognised.
func TestIsCrossDevice_Windows(t *testing.T) {
	t.Parallel()

	assert.True(t, isCrossDevice(windows.ERROR_NOT_SAME_DEVICE),
		"ERROR_NOT_SAME_DEVICE is the code a cross-volume move reports")
	assert.True(t, isCrossDevice(&os.LinkError{
		Op:  "rename",
		Old: `D:\gopath\bin\tool`,
		New: `C:\Users\runneradmin\AppData\Local\go-remove\trash\files\tool`,
		Err: windows.ERROR_NOT_SAME_DEVICE,
	}))

	// A real failure must not reach the copy fallback, which would duplicate
	// the file and then fail to unlink the original.
	assert.False(t, isCrossDevice(nil))
	assert.False(t, isCrossDevice(syscall.ERROR_ACCESS_DENIED))
	assert.False(t, isCrossDevice(windows.ERROR_SHARING_VIOLATION))
	assert.False(t, isCrossDevice(errors.New("some other failure")))
}
