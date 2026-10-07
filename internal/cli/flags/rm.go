/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package flags

import (
	"errors"
	"strings"
)

// ErrEmptyBinaryName indicates the binary name argument was blank.
var ErrEmptyBinaryName = errors.New("binary name cannot be empty")

// ValidateBinary checks the binary name given to the rm command.
//
// A blank name would join to the bin directory itself, so it is rejected here
// rather than passed on.
//
// Parameters:
//   - name: the raw binary name argument.
//
// Returns:
//   - string: the name, unchanged.
//   - error: [ErrEmptyBinaryName] when the name is empty or only whitespace.
func ValidateBinary(name string) (string, error) {
	if strings.TrimSpace(name) == "" {
		return "", ErrEmptyBinaryName
	}

	return name, nil
}
