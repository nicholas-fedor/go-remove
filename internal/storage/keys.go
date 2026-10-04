/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package storage

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// maxRecordKeyAttempts bounds the search for an unused record key.
const maxRecordKeyAttempts = 100

var (
	// ErrInvalidKeyFormat indicates the key is not in the expected format.
	ErrInvalidKeyFormat = errors.New(
		"invalid key format: expected '<zero-padded-timestamp>:<binary_name>'",
	)

	// ErrEmptyBinaryName indicates the binary name in the key is empty.
	ErrEmptyBinaryName = errors.New("empty binary name in key")
)

// randomKeyDiscriminator returns a short random string used to distinguish two
// records that would otherwise share a key.
//
// Returns:
//   - A hexadecimal string of 12 characters.
func randomKeyDiscriminator() string {
	var suffix [6]byte

	// crypto/rand.Read never fails and always fills the buffer.
	_, _ = rand.Read(suffix[:])

	return hex.EncodeToString(suffix[:])
}

// GenerateKey creates a composite Badger key.
//
// Parameters:
//   - timestamp: Unix timestamp of the deletion.
//   - binaryName: Name of the deleted binary.
//
// Returns:
//   - Composite storage key.
func GenerateKey(timestamp int64, binaryName string) string {
	return fmt.Sprintf("%020d:%s", timestamp, binaryName)
}

// ParseKey extracts the timestamp and binary name from a composite key.
//
// A key carrying a discriminator holds it as a further segment, so the returned
// binary name includes that suffix.
//
// Parameters:
//   - key: Composite storage key to parse.
//
// Returns:
//   - Unix timestamp encoded in the key.
//   - Binary name encoded in the key.
//   - An error if the key format is invalid.
func ParseKey(key string) (int64, string, error) {
	timestampPart, binaryName, ok := strings.Cut(key, ":")
	if !ok {
		return 0, "", fmt.Errorf("%w: got %q", ErrInvalidKeyFormat, key)
	}

	timestamp, err := strconv.ParseInt(timestampPart, 10, 64)
	if err != nil {
		return 0, "", fmt.Errorf("invalid timestamp in key %q: %w", key, err)
	}

	if binaryName == "" {
		return 0, "", ErrEmptyBinaryName
	}

	return timestamp, binaryName, nil
}
