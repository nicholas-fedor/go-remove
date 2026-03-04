/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package storage

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ErrInvalidKeyFormat indicates the key is not in the expected format.
var ErrInvalidKeyFormat = errors.New(
	"invalid key format: expected '<zero-padded-timestamp>:<binary_name>'",
)

// ErrEmptyBinaryName indicates the binary name in the key is empty.
var ErrEmptyBinaryName = errors.New("empty binary name in key")

// GenerateKey creates a composite key for Badger storage.
// The key format is "<zero-padded-timestamp>:<binary_name>" (e.g., "00000001709321234:golangci-lint").
// The timestamp is zero-padded to 20 digits to ensure lexicographic order equals chronological order.
// This format enables chronological sorting via Badger's key ordering.
//
// Parameters:
//   - timestamp: Unix timestamp (seconds since epoch)
//   - binaryName: Name of the binary file
//
// Returns:
//   - A composite key string suitable for Badger storage
func GenerateKey(timestamp int64, binaryName string) string {
	return fmt.Sprintf("%020d:%s", timestamp, binaryName)
}

// ParseKey extracts timestamp and binary name from a key.
// The key must be in the format "<timestamp>:<binary_name>".
//
// Parameters:
//   - key: The composite key to parse
//
// Returns:
//   - timestamp: The Unix timestamp extracted from the key
//   - binaryName: The binary name extracted from the key
//   - err: An error if the key format is invalid
//
//nolint:nonamedreturns // Named returns improve clarity for this function
func ParseKey(key string) (timestamp int64, binaryName string, err error) {
	//nolint:mnd // Split into 2 parts: timestamp and binary_name
	parts := strings.SplitN(key, ":", 2)
	//nolint:mnd // Expect exactly 2 parts
	if len(parts) != 2 {
		return 0, "", fmt.Errorf("%w: got %q", ErrInvalidKeyFormat, key)
	}

	timestamp, err = strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, "", fmt.Errorf("invalid timestamp in key %q: %w", key, err)
	}

	if parts[1] == "" {
		return 0, "", ErrEmptyBinaryName
	}

	return timestamp, parts[1], nil
}
