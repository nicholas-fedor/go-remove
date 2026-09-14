//go:build !windows

/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package trash

import (
	"os"
	"path/filepath"
)

// getXDGTrashPath returns the XDG trash path for the current environment.
//
// It uses $XDG_DATA_HOME/Trash, falling back to ~/.local/share/Trash.
//
// Returns:
//   - Absolute trash directory path, or empty if the home directory cannot be resolved.
func getXDGTrashPath() string {
	xdgDataHome := os.Getenv("XDG_DATA_HOME")
	if xdgDataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}

		xdgDataHome = filepath.Join(home, ".local", "share")
	}

	return filepath.Join(xdgDataHome, "Trash")
}
