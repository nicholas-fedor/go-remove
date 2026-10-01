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
	windowsOS  = "windows" // Operating system identifier for Windows
	windowsExt = ".exe"    // File extension for Windows executables
)

var goBinaryExtractor = &buildinfo.DefaultExtractor{}

// ErrGorootNotSet indicates that GOROOT is not set when required.
//
// It is the same sentinel as paths.ErrGorootNotSet, so errors.Is matches either
// name.
var ErrGorootNotSet = paths.ErrGorootNotSet

// ErrBinaryNotFound indicates that a binary does not exist at the specified path.
var ErrBinaryNotFound = errors.New("binary not found")

// FS defines filesystem operations for go-remove.
type FS interface {
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

// RealFS implements the FS interface using real filesystem operations.
type RealFS struct{}

var _ FS = (*RealFS)(nil)

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
// The argument is a name from the scanned directory, not a path. Separators and
// parent references are ignored rather than joined, because filepath.Join would
// resolve them and let a positional argument reach outside the binary
// directory, which is the one thing this tool is meant to prevent. A base of
// ".." is normalised to ".", since filepath.Base keeps ".." unchanged and
// joining it would step out of the directory.
//
// Parameters:
//   - dir: Directory containing the binary.
//   - binary: Binary file name.
//
// Returns:
//   - Path of the binary within dir, or dir itself when binary names no entry
//     inside it.
func (r *RealFS) AdjustBinaryPath(dir, binary string) string {
	name := filepath.Base(binary)

	// A base of ".." or "." names no entry inside dir, so the directory itself is
	// the answer. Return before the Windows suffix, which would otherwise turn
	// the directory into "bin.exe".
	if name == ".." || name == "." {
		return dir
	}

	path := filepath.Join(dir, name)
	if binary != "" && runtime.GOOS == windowsOS && filepath.Ext(binary) != windowsExt {
		path += windowsExt
	}

	return path
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
		log.Debug().Msgf("Constructed binary path: %s", binaryPath)
		log.Info().Msgf("Removing binary: %s", binaryPath)
	}

	if err := os.Remove(binaryPath); err != nil {
		return fmt.Errorf("removing %s: %w", binaryPath, err)
	}

	if verbose {
		log.Info().Msgf("Successfully removed binary: %s", name)
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
//   - An error if the directory cannot be read. A read failure used to be
//     reported as an empty directory, which sent the user looking for missing
//     binaries rather than a permission problem.
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
	if runtime.GOOS == windowsOS && !strings.EqualFold(filepath.Ext(name), windowsExt) {
		return false
	}

	return goBinaryExtractor.IsGoBinary(filepath.Join(dir, name))
}
