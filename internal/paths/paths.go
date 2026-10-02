/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package paths

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// DirPermissions defines the permissions for directories this package creates.
//
// The application stores full filesystem paths, module graphs, VCS revisions and
// checksums, so the directory is not world readable.
const DirPermissions = 0o750

// windowsOS is the runtime.GOOS value for Windows.
const windowsOS = "windows"

// ErrNoWritableStorage indicates no writable directory was found for storage.
var ErrNoWritableStorage = errors.New("no writable directory found for storage")

// ErrGorootNotSet indicates that GOROOT is not set when required.
var ErrGorootNotSet = errors.New("GOROOT is not set")

// DataHomeCandidates returns candidate application data directories for the
// current platform, most preferred first.
//
// On Linux and other Unix-like systems the XDG base directory specification is
// followed: $XDG_DATA_HOME when it is set and absolute, then ~/.local/share. A
// relative $XDG_DATA_HOME is skipped rather than joined, because joining it
// would resolve the data directory against the working directory. The trash root
// applies the same rule, and both now come from here so they cannot drift.
//
// On macOS the location is ~/Library/Application Support. On Windows it is
// %LOCALAPPDATA%, then %USERPROFILE%.
//
// Returns:
//   - Ordered list of candidate directories. Empty entries are omitted.
func DataHomeCandidates() []string {
	var candidates []string

	switch runtime.GOOS {
	case windowsOS:
		// Try LOCALAPPDATA first, then USERPROFILE
		if dataHome := os.Getenv("LOCALAPPDATA"); dataHome != "" {
			candidates = append(candidates, dataHome)
		}

		if userProfile := os.Getenv("USERPROFILE"); userProfile != "" {
			candidates = append(candidates, userProfile)
		}
	case "darwin":
		// An explicit XDG_DATA_HOME is honoured here as well, so the same
		// variable selects both the application data directory and the trash
		// root. Application Support remains the default when it is unset.
		if dataHome := os.Getenv("XDG_DATA_HOME"); dataHome != "" && filepath.IsAbs(dataHome) {
			candidates = append(candidates, dataHome)
		}

		// Try ~/Library/Application Support
		if userHome, err := os.UserHomeDir(); err == nil {
			candidates = append(
				candidates,
				filepath.Join(userHome, "Library", "Application Support"),
			)
		}
	default: // Linux and other Unix-like systems
		// Try $XDG_DATA_HOME first, then ~/.local/share.
		if dataHome := os.Getenv("XDG_DATA_HOME"); dataHome != "" && filepath.IsAbs(dataHome) {
			candidates = append(candidates, dataHome)
		}

		if userHome, err := os.UserHomeDir(); err == nil {
			candidates = append(candidates, filepath.Join(userHome, ".local", "share"))
		}
	}

	return candidates
}

// WritableDataHome finds a writable directory for application data.
//
// It tries the platform data directories first, then the user home directory,
// then the directory holding the running executable.
//
// Returns:
//   - Writable data directory path.
//   - ErrNoWritableStorage if no writable directory can be found.
func WritableDataHome() (string, error) {
	for _, candidate := range DataHomeCandidates() {
		if candidate == "" {
			continue
		}

		if IsDirWritable(candidate) {
			return candidate, nil
		}
	}

	// Fallback: try user home directory
	if userHome, err := os.UserHomeDir(); err == nil && IsDirWritable(userHome) {
		return userHome, nil
	}

	// Last resort: try executable directory
	if exePath, err := os.Executable(); err == nil {
		if exeDir := filepath.Dir(exePath); IsDirWritable(exeDir) {
			return exeDir, nil
		}
	}

	return "", ErrNoWritableStorage
}

// StoragePath returns the path for the history storage database.
//
// It uses platform-specific data directories:
//   - Linux: $XDG_DATA_HOME/go-remove/history.badger or ~/.local/share/go-remove/history.badger.
//   - macOS: ~/Library/Application Support/go-remove/history.badger.
//   - Windows: %LOCALAPPDATA%/go-remove/history.badger.
//
// Returns:
//   - Absolute path to the Badger database directory.
//   - An error if no writable storage directory can be found.
func StoragePath() (string, error) {
	dataHome, err := WritableDataHome()
	if err != nil {
		return "", fmt.Errorf("finding writable storage directory: %w", err)
	}

	return filepath.Join(dataHome, "go-remove", "history.badger"), nil
}

// IsDirWritable reports whether a directory can be written to, creating it when
// it does not yet exist.
//
// Parameters:
//   - dir: Directory path to test.
//
// Returns:
//   - True if a temporary file can be created in the directory.
func IsDirWritable(dir string) bool {
	if err := os.MkdirAll(dir, DirPermissions); err != nil {
		return false
	}

	tmpFile, err := os.CreateTemp(dir, ".write_test_*")
	if err != nil {
		return false
	}

	_ = tmpFile.Close()
	_ = os.Remove(tmpFile.Name())

	return true
}

// BinDir resolves the Go toolchain binary directory.
//
// Without useGoroot the order is $GOBIN, then $GOPATH/bin, then ~/go/bin.
//
// A relative $GOBIN or $GOPATH is treated as unset, because a relative binary
// directory would resolve against the working directory, which is wherever the
// tool happens to be invoked from rather than where the toolchain lives. The
// same absolute-path rule is applied by DataHomeCandidates, so the binary
// directory and the application data root cannot disagree about which paths
// count as resolved.
//
// Parameters:
//   - useGoroot: Whether to target $GOROOT/bin instead.
//
// Returns:
//   - Absolute path to the binary directory.
//   - ErrGorootNotSet if GOROOT is requested but not set.
//   - An error if neither GOPATH nor the home directory can be resolved.
func BinDir(useGoroot bool) (string, error) {
	if useGoroot {
		gorootDir := os.Getenv("GOROOT")
		if gorootDir == "" {
			return "", ErrGorootNotSet
		}

		return filepath.Join(gorootDir, "bin"), nil
	}

	goBin := os.Getenv("GOBIN")
	if goBin != "" && filepath.IsAbs(goBin) {
		return goBin, nil
	}

	gopath := os.Getenv("GOPATH")
	if gopath != "" {
		// GOPATH is a list, and only its first entry is the search root.
		// Validating the whole string would accept "/abs/path:relative" as
		// absolute on the strength of its first character and then join the
		// binary directory onto the entire list.
		gopath, _, _ = strings.Cut(gopath, string(os.PathListSeparator))
	}

	if gopath != "" && !filepath.IsAbs(gopath) {
		// A relative GOPATH would place the binary directory inside the working
		// directory, so it falls through to the home directory exactly as an
		// unset value does.
		gopath = ""
	}

	if gopath == "" {
		// os.UserHomeDir reads $HOME on Unix and %USERPROFILE% on Windows, and
		// returns either verbatim. An unresolvable home is an error rather than
		// a path, and a relative one is rejected for the same reason a relative
		// GOPATH is, because joining it would resolve against the working
		// directory.
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolving home directory: %w", err)
		}

		if !filepath.IsAbs(home) {
			return "", fmt.Errorf("%w: %q is not an absolute path", ErrNoWritableStorage, home)
		}

		gopath = filepath.Join(home, "go")
	}

	return filepath.Join(gopath, "bin"), nil
}
