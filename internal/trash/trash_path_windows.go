//go:build windows

/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package trash

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/windows"
)

// platformTrashRoot returns the default trash root for the current environment.
//
// Windows has no XDG base directory, so the per-user application data
// directory is used, matching where the rest of the tool keeps its state.
//
// Returns:
//   - Absolute trash directory path, or empty if the local application data
//     directory cannot be resolved.
func platformTrashRoot() string {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" || !filepath.IsAbs(localAppData) {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}

		localAppData = filepath.Join(home, "AppData", "Local")
	}

	return filepath.Join(localAppData, "go-remove", "trash")
}

// isCrossDevice reports whether a move failed because the source and
// destination are on different filesystems.
//
// syscall.EXDEV is an invented value on Windows, declared as an offset into the
// application error range, and no Windows API ever returns it. The real code
// reported for a cross-volume move is ERROR_NOT_SAME_DEVICE, so it has to be
// matched directly or every cross-volume move is reported as a hard failure
// instead of falling back to a copy.
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
