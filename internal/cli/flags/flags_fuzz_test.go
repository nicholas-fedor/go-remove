/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package flags

import (
	"errors"
	"strings"
	"testing"
)

// FuzzValidateBinary verifies ValidateBinary never panics, never accepts a
// blank name, and returns an accepted name unchanged.
//
// The seed corpus covers an empty name, whitespace only, a plain name, and a
// name with surrounding whitespace.
func FuzzValidateBinary(f *testing.F) {
	f.Add("")
	f.Add("   ")
	f.Add("vhs")
	f.Add(" vhs\t")

	f.Fuzz(func(t *testing.T, name string) {
		got, err := ValidateBinary(name)
		if err != nil {
			if !errors.Is(err, ErrEmptyBinaryName) {
				t.Fatalf("ValidateBinary(%q) error %v does not wrap ErrEmptyBinaryName", name, err)
			}

			if strings.TrimSpace(name) != "" {
				t.Fatalf("ValidateBinary(%q) rejected a non-blank name", name)
			}

			return
		}

		if strings.TrimSpace(got) == "" {
			t.Fatalf("ValidateBinary(%q) accepted a blank name", name)
		}

		if got != name {
			t.Fatalf("ValidateBinary(%q) = %q, want the input unchanged", name, got)
		}
	})
}
