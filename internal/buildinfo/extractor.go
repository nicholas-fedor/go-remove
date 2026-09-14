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
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"os"
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
		"unsupported platform: only Linux, Windows, and Darwin are supported",
	)
)

// Extractor defines operations for extracting build information from Go binaries.
type Extractor interface {
	// Extract retrieves build information from a binary file.
	//
	// Parameters:
	//   - ctx: Context for cancellation.
	//   - binaryPath: Path to the Go binary.
	//
	// Returns:
	//   - Structured build info.
	//   - An error if extraction fails.
	Extract(ctx context.Context, binaryPath string) (*BuildInfoData, error)

	// CalculateChecksum computes the SHA256 checksum of a binary.
	//
	// Parameters:
	//   - binaryPath: Path to the file to hash.
	//
	// Returns:
	//   - Hex-encoded SHA256 digest.
	//   - An error if the file cannot be read.
	CalculateChecksum(binaryPath string) (string, error)

	// IsGoBinary reports whether a file contains valid Go build information.
	//
	// Parameters:
	//   - binaryPath: Path to the file to inspect.
	//
	// Returns:
	//   - True if the file is a Go binary with build info.
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

var _ Extractor = (*DefaultExtractor)(nil)

// NewExtractor creates a new build info extractor.
//
// Returns:
//   - Extractor implementation for the current platform.
//   - ErrUnsupportedPlatform if the OS is not Linux, Windows, or Darwin.
func NewExtractor() (*DefaultExtractor, error) {
	if !isSupportedPlatform() {
		return nil, ErrUnsupportedPlatform
	}

	return &DefaultExtractor{}, nil
}

// isSupportedPlatform reports whether the current OS is supported.
//
// Returns:
//   - True for Linux, Windows, or Darwin.
func isSupportedPlatform() bool {
	return runtime.GOOS == "linux" || runtime.GOOS == "windows" || runtime.GOOS == "darwin"
}

// Extract retrieves build information from a Go binary.
//
// It uses debug/buildinfo.ReadFile rather than invoking go version.
//
// Parameters:
//   - ctx: Context for cancellation.
//   - binaryPath: Path to the Go binary.
//
// Returns:
//   - Structured build info.
//   - An error if the file is missing, not a Go binary, or extraction fails.
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
//
// Parameters:
//   - binaryPath: Path to the file to hash.
//
// Returns:
//   - Hex-encoded SHA256 digest.
//   - An error if the file cannot be read.
func (e *DefaultExtractor) CalculateChecksum(binaryPath string) (string, error) {
	// Open the binary file
	file, err := os.Open(binaryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%w: %s", ErrPathNotFound, binaryPath)
		}

		return "", fmt.Errorf("opening binary file: %w", err)
	}

	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("hashing binary file: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// IsGoBinary reports whether a file contains valid Go build information.
//
// Parameters:
//   - binaryPath: Path to the file to inspect.
//
// Returns:
//   - True if the file is a regular Go binary with a Go version string.
func (e *DefaultExtractor) IsGoBinary(binaryPath string) bool {
	fileInfo, err := os.Stat(binaryPath)
	if err != nil || !fileInfo.Mode().IsRegular() {
		return false
	}

	info, err := buildinfo.ReadFile(binaryPath)
	if err != nil {
		return false
	}

	return info.GoVersion != ""
}
