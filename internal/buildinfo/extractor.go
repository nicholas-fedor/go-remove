/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package buildinfo

import (
	"context"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime"
)

// Common errors for buildinfo operations.
var (
	// ErrNotGoBinary indicates the file is not a Go binary.
	ErrNotGoBinary = errors.New("file is not a Go binary")

	// ErrBuildInfoNotFound indicates build info could not be extracted.
	ErrBuildInfoNotFound = errors.New("build info not found in binary")

	// ErrPathNotFound indicates the specified path does not exist.
	ErrPathNotFound = errors.New("binary path not found")

	// ErrUnsupportedPlatform indicates the platform is not supported.
	ErrUnsupportedPlatform = errors.New(
		"unsupported platform: only Linux and Windows are supported",
	)
)

// Extractor defines operations for extracting build information from Go binaries.
type Extractor interface {
	// Extract retrieves build information from a binary file.
	// Returns structured build info or an error if extraction fails.
	Extract(ctx context.Context, binaryPath string) (*BuildInfoData, error)

	// CalculateChecksum computes SHA256 checksum of a binary.
	// Returns hex-encoded string representation of the hash.
	CalculateChecksum(binaryPath string) (string, error)

	// IsGoBinary checks if a file is a Go binary with build info.
	// Returns true if the file contains valid Go build information.
	IsGoBinary(binaryPath string) bool
}

// BuildInfoData holds structured build information extracted from a Go binary.
type BuildInfoData struct {
	// ModulePath is the Go module path (e.g., "github.com/user/repo").
	ModulePath string `json:"module_path"`

	// Version is the module version (e.g., "v1.2.3", "(devel)").
	Version string `json:"version"`

	// VCSRevision is the Git commit SHA from build settings.
	VCSRevision string `json:"vcs_revision"`

	// VCSTime is the VCS timestamp from build settings.
	VCSTime string `json:"vcs_time"`

	// GoVersion is the Go version used to build the binary.
	GoVersion string `json:"go_version"`

	// Settings contains all build settings from debug.BuildInfo.
	Settings map[string]string `json:"settings"`

	// RawJSON contains the full BuildInfo as JSON for future-proofing.
	RawJSON []byte `json:"raw_json"`
}

// DefaultExtractor implements the Extractor interface using debug/buildinfo.
type DefaultExtractor struct{}

// NewExtractor creates a new build info extractor.
// Returns an error if the current platform is not supported.
func NewExtractor() (*DefaultExtractor, error) {
	if !isSupportedPlatform() {
		return nil, ErrUnsupportedPlatform
	}

	return &DefaultExtractor{}, nil
}

// isSupportedPlatform checks if the current platform is supported.
func isSupportedPlatform() bool {
	return runtime.GOOS == "linux" || runtime.GOOS == "windows"
}

// Extract retrieves build information from a Go binary.
// Uses debug/buildinfo.ReadFile for direct extraction without os/exec.
func (e *DefaultExtractor) Extract(ctx context.Context, binaryPath string) (*BuildInfoData, error) {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled: %w", err)
	}

	// Verify file exists
	if _, err := os.Stat(binaryPath); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrPathNotFound, binaryPath)
		}

		return nil, fmt.Errorf("checking binary path: %w", err)
	}

	// Read build info directly from binary
	info, err := buildinfo.ReadFile(binaryPath)
	if err != nil {
		// Any error from ReadFile indicates this is not a valid Go binary
		// or build info could not be extracted
		return nil, fmt.Errorf("%w: %w", ErrNotGoBinary, err)
	}

	// Build the structured data
	data := &BuildInfoData{
		GoVersion: info.GoVersion,
		Settings:  make(map[string]string),
	}

	// Extract main module information
	if info.Main.Path != "" {
		data.ModulePath = info.Main.Path
		data.Version = info.Main.Version
	}

	// Parse build settings
	for _, setting := range info.Settings {
		data.Settings[setting.Key] = setting.Value

		// Extract VCS information from settings
		switch setting.Key {
		case "vcs.revision":
			data.VCSRevision = setting.Value
		case "vcs.time":
			data.VCSTime = setting.Value
		}
	}

	// Serialize full build info to JSON
	rawJSON, err := json.Marshal(info)
	if err != nil {
		return nil, fmt.Errorf("marshaling build info to JSON: %w", err)
	}

	data.RawJSON = rawJSON

	return data, nil
}

// CalculateChecksum computes the SHA256 hash of a binary file.
// Returns hex-encoded string representation of the hash.
func (e *DefaultExtractor) CalculateChecksum(binaryPath string) (string, error) {
	// Open the binary file
	file, err := os.Open(binaryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%w: %s", ErrPathNotFound, binaryPath)
		}

		return "", fmt.Errorf("opening binary file: %w", err)
	}

	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			// Log would go here if we had a logger
			_ = closeErr
		}
	}()

	// Create SHA256 hasher
	hasher := sha256.New()

	// Copy file content to hasher
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("hashing binary file: %w", err)
	}

	// Get hash sum and encode to hex
	hashSum := hasher.Sum(nil)
	checksum := hex.EncodeToString(hashSum)

	return checksum, nil
}

// IsGoBinary checks if a file is a Go binary by attempting to read its build info.
// Returns true if build information can be extracted, false otherwise.
func (e *DefaultExtractor) IsGoBinary(binaryPath string) bool {
	// Quick check: file must exist and be a regular file
	fileInfo, err := os.Stat(binaryPath)
	if err != nil {
		return false
	}

	// Must be a regular file (not directory, symlink, etc.)
	if !fileInfo.Mode().IsRegular() {
		return false
	}

	// Attempt to read build info
	info, err := buildinfo.ReadFile(binaryPath)
	if err != nil {
		return false
	}

	// Must have valid Go version
	if info.GoVersion == "" {
		return false
	}

	return true
}

// pseudoVersionRegex matches Go pseudo-version format:
// vX.Y.Z-yyyymmddhhmmss-abcdefabcdef or vX.Y.Z-0.yyyymmddhhmmss-abcdefabcdef.
var pseudoVersionRegex = regexp.MustCompile(`^v\d+\.\d+\.\d+-(?:0\.)?\d{14}-[a-f0-9]+$`)

// semanticVersionRegex matches semantic version with optional pre-release/build metadata.
var semanticVersionRegex = regexp.MustCompile(
	`^v\d+\.\d+\.\d+(?:-[a-zA-Z0-9._-]+)?(?:\+[a-zA-Z0-9._-]+)?$`,
)

// ParseVersionType determines the type of version string.
// Returns one of: "semantic", "pseudo", "devel", or "unknown".
func ParseVersionType(version string) string {
	if version == "" {
		return "unknown"
	}

	// Check for development build marker
	if version == "(devel)" {
		return "devel"
	}

	// Check for semantic version (vX.Y.Z)
	if len(version) > 1 && version[0] == 'v' {
		// Check for pseudo-version pattern: vX.Y.Z-yyyymmddhhmmss-abcdefabcdef
		if pseudoVersionRegex.MatchString(version) {
			return "pseudo"
		}

		// Check for semantic version format (allows pre-release like v1.2.3-beta)
		if semanticVersionRegex.MatchString(version) {
			return "semantic"
		}
	}

	return "unknown"
}

// IsReinstallable checks if the build info contains enough information
// for potential reinstallation from source.
// Requires both module path and VCS revision.
func (b *BuildInfoData) IsReinstallable() bool {
	return b.ModulePath != "" && b.VCSRevision != ""
}

// GetInstallCommand returns a go install command for reinstalling this binary
// if sufficient information is available.
// Returns empty string if not reinstallable.
func (b *BuildInfoData) GetInstallCommand() string {
	// Cannot construct install command without module path
	if b.ModulePath == "" {
		return ""
	}

	// Prefer tagged versions for reproducible installs
	if b.Version != "" && b.Version != "(devel)" {
		return fmt.Sprintf("go install %s@%s", b.ModulePath, b.Version)
	}

	// For pseudo-versions or specific commits, install at the revision
	if b.VCSRevision != "" {
		return fmt.Sprintf("go install %s@%s", b.ModulePath, b.VCSRevision)
	}

	// Fallback to latest if we have module path but no version/revision
	return fmt.Sprintf("go install %s@latest", b.ModulePath)
}
