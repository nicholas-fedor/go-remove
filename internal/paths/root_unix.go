//go:build linux || darwin

/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package paths

import (
	"os"
	"path/filepath"
)

// TrashRoot returns the default trash root for the current environment.
//
// It follows the XDG Base Directory Specification, using $XDG_DATA_HOME/Trash
// when that value is absolute, and ~/.local/share/Trash otherwise. The same
// absolute-path rule is applied by DataHomeCandidates, so the trash root and the
// application data root cannot disagree about where state lives.
//
// Returns:
//   - Absolute trash directory path, or empty if the home directory cannot be
//     resolved.
func TrashRoot() string {
	xdgDataHome := os.Getenv("XDG_DATA_HOME")
	if xdgDataHome == "" || !filepath.IsAbs(xdgDataHome) {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}

		xdgDataHome = filepath.Join(home, ".local", "share")
	}

	return filepath.Join(xdgDataHome, "Trash")
}
