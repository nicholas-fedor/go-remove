//go:build windows

/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package trash

import (
	"os"
	"path/filepath"
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
