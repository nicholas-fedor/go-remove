/*
Copyright © 2025 Nicholas Fedor <nick@nickfedor.com>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
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

	"github.com/nicholas-fedor/go-remove/internal/logger"
)

// OS-specific constants for filesystem operations.
const (
	windowsOS  = "windows" // Operating system identifier for Windows
	windowsExt = ".exe"    // File extension for Windows executables
)

// ErrGorootNotSet indicates that GOROOT is not set when required.
var ErrGorootNotSet = errors.New("GOROOT is not set")

// ErrBinaryNotFound indicates that a binary does not exist at the specified path.
var ErrBinaryNotFound = errors.New("binary not found")

// FS defines filesystem operations for go-remove.
type FS interface {
	DetermineBinDir(useGoroot bool) (string, error)
	AdjustBinaryPath(dir, binary string) string
	RemoveBinary(binaryPath, name string, verbose bool, logger logger.Logger) error
	ListBinaries(dir string) []string
}

// RealFS implements the FS interface using real filesystem operations.
type RealFS struct{}

// NewRealFS creates a new RealFS instance.
func NewRealFS() FS {
	return &RealFS{}
}

// DetermineBinDir resolves the binary directory based on GOROOT or GOPATH/GOBIN.
func (r *RealFS) DetermineBinDir(useGoroot bool) (string, error) {
	// Use GOROOT/bin if specified and available.
	if useGoroot {
		gorootDir := os.Getenv("GOROOT")
		if gorootDir == "" {
			return "", ErrGorootNotSet
		}

		return filepath.Join(gorootDir, "bin"), nil
	}

	// Fall back to GOBIN or GOPATH/bin, defaulting to ~/go/bin if neither is set.
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
func (r *RealFS) AdjustBinaryPath(dir, binary string) string {
	// Join the directory and binary name into a single path.
	path := filepath.Join(dir, binary)

	// Append .exe extension on Windows if the binary lacks it.
	if binary != "" && runtime.GOOS == windowsOS && filepath.Ext(binary) != windowsExt {
		path += windowsExt
	}

	return path
}

// RemoveBinary deletes a binary file from the filesystem.
func (r *RealFS) RemoveBinary(binaryPath, name string, verbose bool, logger logger.Logger) error {
	// Verify the binary exists before attempting removal.
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return fmt.Errorf("%w: %s at %s", ErrBinaryNotFound, name, binaryPath)
	}

	// Log removal intent if verbose mode is enabled.
	sugar := logger.Sugar()
	if verbose {
		sugar.Infow("Removing binary", "path", binaryPath)
	}

	// Perform the removal operation and handle any errors.
	if err := os.Remove(binaryPath); err != nil {
		return fmt.Errorf("failed to remove %s: %w", binaryPath, err)
	}

	// Log success if verbose mode is enabled.
	if verbose {
		sugar.Infow("Successfully removed binary", "name", name, "path", binaryPath)
	}

	return nil
}

// ListBinaries retrieves a list of executable binaries from a directory.
func (r *RealFS) ListBinaries(dir string) []string {
	// Read directory contents, returning an empty list on error.
	files, err := os.ReadDir(dir)
	if err != nil {
		return []string{}
	}

	// Filter for executable files, including .exe on Windows.
	var choices []string

	for _, file := range files {
		if !file.IsDir() &&
			(runtime.GOOS != windowsOS || strings.HasSuffix(file.Name(), windowsExt)) {
			choices = append(choices, file.Name())
		}
	}

	return choices
}
