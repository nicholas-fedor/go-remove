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

// ErrInvalidKeyFormat indicates the key is not in the expected format.
var ErrInvalidKeyFormat = errors.New(
	"invalid key format: expected '<zero-padded-timestamp>:<binary_name>'",
)

// ErrEmptyBinaryName indicates the binary name in the key is empty.
var ErrEmptyBinaryName = errors.New("empty binary name in key")

// maxRecordKeyAttempts bounds the search for an unused record key.
const maxRecordKeyAttempts = 100

// randomKeyDiscriminator returns a short random string used to distinguish two
// records that would otherwise share a key.
//
// Parameters:
//   - None.
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
// The key format is "<zero-padded-timestamp>:<binary_name>". A random
// discriminator is appended as a further segment when that key is already in
// use, giving "<timestamp>:<binary_name>:<discriminator>".
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
// A key written with a discriminator carries it as a further segment, so the
// returned binary name includes that suffix. Callers that only need to reject
// malformed keys are unaffected.
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
