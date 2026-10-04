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

var (
	// ErrNoWritableStorage indicates no writable directory was found for storage.
	ErrNoWritableStorage = errors.New("no writable directory found for storage")

	// ErrGorootNotSet indicates that GOROOT is not set when required.
	ErrGorootNotSet = errors.New("GOROOT is not set")
)

// DataHomeCandidates returns candidate application data directories for the
// current platform, most preferred first.
//
// Unix-like systems follow the XDG base directory specification, macOS adds
// ~/Library/Application Support, and Windows uses %LOCALAPPDATA% then
// %USERPROFILE%.
//
// Returns:
//   - Ordered list of candidate directories. Empty entries are omitted.
func DataHomeCandidates() []string {
	var candidates []string

	switch runtime.GOOS {
	case windowsOS:
		if dataHome := os.Getenv("LOCALAPPDATA"); dataHome != "" {
			candidates = append(candidates, dataHome)
		}

		if userProfile := os.Getenv("USERPROFILE"); userProfile != "" {
			candidates = append(candidates, userProfile)
		}
	case "darwin":
		// XDG_DATA_HOME is honored here too, so one variable selects both the
		// application data directory and the trash root. A relative value is
		// skipped rather than joined, because joining it would resolve the data
		// directory against the working directory.
		if dataHome := os.Getenv("XDG_DATA_HOME"); dataHome != "" && filepath.IsAbs(dataHome) {
			candidates = append(candidates, dataHome)
		}

		if userHome, err := os.UserHomeDir(); err == nil {
			candidates = append(
				candidates,
				filepath.Join(userHome, "Library", "Application Support"),
			)
		}

	// Linux and other Unix-like systems.
	default:
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
// It tries the platform data directories first, then the user home directory.
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

	if userHome, err := os.UserHomeDir(); err == nil && IsDirWritable(userHome) {
		return userHome, nil
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
// A relative value is rejected at each of those steps, since it would resolve
// against the working directory rather than the toolchain. GOROOT is joined
// without that check, so a relative GOROOT yields a relative path.
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

	// DataHomeCandidates applies the same rule, so the two cannot disagree
	// about which paths count as resolved.
	gopath := os.Getenv("GOPATH")
	if gopath != "" {
		// Only the first entry is the search root. Validating the whole
		// string would accept "/abs/path:relative" as absolute.
		gopath, _, _ = strings.Cut(gopath, string(os.PathListSeparator))
	}

	if gopath != "" && !filepath.IsAbs(gopath) {
		// A relative GOPATH would place the binary directory in the working
		// directory, so treat it as unset.
		gopath = ""
	}

	if gopath == "" {
		// os.UserHomeDir returns $HOME or %USERPROFILE% verbatim, and an
		// unresolvable home is an error rather than a path.
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
