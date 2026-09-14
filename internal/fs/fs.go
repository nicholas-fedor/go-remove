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
)

// OS-specific constants for filesystem operations.
const (
	windowsOS  = "windows" // Operating system identifier for Windows
	windowsExt = ".exe"    // File extension for Windows executables
)

var goBinaryExtractor = &buildinfo.DefaultExtractor{}

// ErrGorootNotSet indicates that GOROOT is not set when required.
var ErrGorootNotSet = errors.New("GOROOT is not set")

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
	ListBinaries(dir string) []string
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
	if useGoroot {
		gorootDir := os.Getenv("GOROOT")
		if gorootDir == "" {
			return "", ErrGorootNotSet
		}

		return filepath.Join(gorootDir, "bin"), nil
	}

	goBin := os.Getenv("GOBIN")
	if goBin == "" {
		gopath := os.Getenv("GOPATH")
		if gopath == "" {
			home := os.Getenv("HOME")
			if runtime.GOOS == windowsOS && home == "" {
				home = os.Getenv("USERPROFILE")
			}

			gopath = filepath.Join(home, "go")
		}

		goBin = filepath.Join(gopath, "bin")
	}

	return goBin, nil
}

// AdjustBinaryPath constructs a full binary path, adding .exe on Windows if needed.
//
// Parameters:
//   - dir: Directory containing the binary.
//   - binary: Binary file name.
//
// Returns:
//   - Absolute path to the binary.
func (r *RealFS) AdjustBinaryPath(dir, binary string) string {
	path := filepath.Join(dir, binary)
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
//   - Names of Go binaries in the directory, or nil if the directory cannot be read.
func (r *RealFS) ListBinaries(dir string) []string {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var choices []string

	for _, file := range files {
		if isRemovableBinary(dir, file) {
			choices = append(choices, file.Name())
		}
	}

	return choices
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
