/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

// Package fs provides filesystem operations for managing Go binaries.
package fs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/nicholas-fedor/go-remove/internal/buildinfo"
	"github.com/nicholas-fedor/go-remove/internal/logger"
	"github.com/nicholas-fedor/go-remove/internal/paths"
)

// OS-specific constants for filesystem operations.
const (
	windowsOS  = "windows"
	windowsExt = ".exe"
)

var goBinaryExtractor = &buildinfo.DefaultExtractor{}

var (
	// ErrGorootNotSet indicates that GOROOT is not set when required.
	//
	// It is the same sentinel as paths.ErrGorootNotSet, so errors.Is matches either
	// name.
	ErrGorootNotSet = paths.ErrGorootNotSet

	// ErrBinaryNotFound indicates that a binary does not exist at the specified path.
	ErrBinaryNotFound = errors.New("binary not found")
)

var (
	_ FS      = (*RealFS)(nil)
	_ Lister  = (*RealFS)(nil)
	_ Remover = (*RealFS)(nil)
)

// Lister reports the Go binaries that may be removed from a directory.
//
// It is the read-only half of the filesystem contract, so a consumer that only
// needs discovery does not have to satisfy the destructive half.
type Lister interface {
	// ListBinaries retrieves removable binaries from a directory.
	//
	// Directories, lock files, and non-executable files are omitted.
	//
	// Parameters:
	//   - dir: Directory to scan.
	//
	// Returns:
	//   - Names of Go binaries in the directory.
	//   - An error if the directory cannot be read, so a permission or I/O failure
	//     is not reported as an empty directory.
	ListBinaries(dir string) ([]string, error)
}

// Remover deletes a binary from the filesystem.
//
// It is the destructive half of the filesystem contract.
type Remover interface {
	// RemoveBinary deletes a binary file from the filesystem.
	//
	// Parameters:
	//   - binaryPath: Full path to the binary.
	//   - name: Binary name used in log messages.
	//   - verbose: When true, emit debug and info logs.
	//   - logger: Logger used for verbose output.
	//
	// Returns:
	//   - An error if the binary does not exist or cannot be removed.
	RemoveBinary(binaryPath, name string, verbose bool, logger logger.Logger) error
}

// FS defines filesystem operations for go-remove.
//
// It is the union of the narrow Lister and Remover contracts with path
// resolution, for consumers that need the whole surface.
type FS interface {
	Lister
	Remover

	// DetermineBinDir resolves the binary directory based on GOROOT or GOPATH/GOBIN.
	//
	// Parameters:
	//   - useGoroot: When true, use GOROOT/bin instead of GOBIN or GOPATH/bin.
	//
	// Returns:
	//   - Absolute path to the binary directory.
	//   - An error if GOROOT is requested but not set.
	DetermineBinDir(useGoroot bool) (string, error)

	// AdjustBinaryPath constructs a full binary path, adding .exe on Windows if needed.
	//
	// Parameters:
	//   - dir: Directory containing the binary.
	//   - binary: Binary file name.
	//
	// Returns:
	//   - Absolute path to the binary.
	AdjustBinaryPath(dir, binary string) string
}

// RealFS implements the FS interface using real filesystem operations.
type RealFS struct{}

// NewRealFS creates a RealFS instance.
//
// Returns:
//   - Filesystem implementation backed by the local OS.
func NewRealFS() FS {
	return &RealFS{}
}

// DetermineBinDir resolves the binary directory based on GOROOT or GOPATH/GOBIN.
//
// Parameters:
//   - useGoroot: When true, use GOROOT/bin instead of GOBIN or GOPATH/bin.
//
// Returns:
//   - Absolute path to the binary directory.
//   - An error if GOROOT is requested but not set.
func (r *RealFS) DetermineBinDir(useGoroot bool) (string, error) {
	//nolint:wrapcheck // paths.ErrGorootNotSet is the same sentinel as fs.ErrGorootNotSet, so wrapping it here would obscure the identity callers compare against.
	return paths.BinDir(useGoroot)
}

// AdjustBinaryPath constructs a full binary path, adding .exe on Windows if needed.
//
// The argument is a name from the scanned directory rather than a path, so a
// positional argument cannot address anything outside dir.
//
// Parameters:
//   - dir: Directory containing the binary.
//   - binary: Binary file name.
//
// Returns:
//   - Path of the binary within dir, or dir itself when ".." or "." names no
//     entry inside it.
func (r *RealFS) AdjustBinaryPath(dir, binary string) string {
	// filepath.Base discards separators and parent references, so an argument
	// given as a path still names only its last element.
	name := filepath.Base(binary)

	// filepath.Base leaves ".." unchanged, and neither it nor "." names an entry
	// inside dir, so return before the Windows suffix below, which would turn the
	// directory into "bin.exe".
	if name == ".." || name == "." {
		return dir
	}

	path := filepath.Join(dir, name)
	if binary != "" && runtime.GOOS == windowsOS && !hasExecutableSuffix(binary) {
		path += windowsExt
	}

	return path
}

// hasExecutableSuffix reports whether a name already ends in the Windows
// executable extension.
//
// Discovery and path construction have to agree on this rule, or a binary listed
// by ListBinaries would not be the one AdjustBinaryPath names.
//
// Parameters:
//   - name: File name or path to inspect.
//
// Returns:
//   - True when the extension matches .exe regardless of case.
func hasExecutableSuffix(name string) bool {
	// Windows treats "tool.exe" and "tool.EXE" as the same file, so a
	// case-sensitive test would append a second extension and name a path that
	// does not exist.
	return strings.EqualFold(filepath.Ext(name), windowsExt)
}

// RemoveBinary deletes a binary file from the filesystem.
//
// Parameters:
//   - binaryPath: Full path to the binary.
//   - name: Binary name used in log messages.
//   - verbose: When true, emit debug and info logs.
//   - log: Logger used for verbose output.
//
// Returns:
//   - An error if the binary does not exist or cannot be removed.
func (r *RealFS) RemoveBinary(binaryPath, name string, verbose bool, log logger.Logger) error {
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return fmt.Errorf("%w: %s at %s", ErrBinaryNotFound, name, binaryPath)
	}

	if verbose {
		log.Debug("Constructed binary path", logger.Str("path", binaryPath))
		log.Info("Removing binary", logger.Str("path", binaryPath))
	}

	if err := os.Remove(binaryPath); err != nil {
		return fmt.Errorf("removing %s: %w", binaryPath, err)
	}

	if verbose {
		log.Info("Successfully removed binary", logger.Str("binary", name))
	}

	return nil
}

// ListBinaries retrieves Go binaries from a directory.
//
// Only regular files with valid Go build information are included.
//
// Parameters:
//   - dir: Directory to scan.
//
// Returns:
//   - Names of Go binaries in the directory.
//   - An error if the directory cannot be read, so a permission or I/O failure
//     is not reported as an empty directory.
func (r *RealFS) ListBinaries(dir string) ([]string, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", dir, err)
	}

	var choices []string

	for _, file := range files {
		if isRemovableBinary(dir, file) {
			choices = append(choices, file.Name())
		}
	}

	return choices, nil
}

// isRemovableBinary reports whether a directory entry is a Go binary.
//
// Parameters:
//   - dir: Parent directory path.
//   - file: Directory entry to inspect.
//
// Returns:
//   - True if the file contains valid Go build information.
func isRemovableBinary(dir string, file os.DirEntry) bool {
	if file.IsDir() {
		return false
	}

	name := file.Name()
	if runtime.GOOS == windowsOS && !hasExecutableSuffix(name) {
		return false
	}

	return goBinaryExtractor.IsGoBinary(filepath.Join(dir, name))
}
