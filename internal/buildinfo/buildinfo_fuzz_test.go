/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package buildinfo

import (
	"os"
	"path/filepath"
	"testing"
)

const maxFuzzBinarySize = 1 << 16

// FuzzIsGoBinary verifies IsGoBinary does not panic on arbitrary file contents.
func FuzzIsGoBinary(f *testing.F) {
	f.Add([]byte("not a go binary"))
	f.Add([]byte{0x7f, 'E', 'L', 'F'})
	f.Add([]byte{})
	f.Add([]byte{0x00})
	f.Add([]byte("MZ"))

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > maxFuzzBinarySize {
			data = data[:maxFuzzBinarySize]
		}

		path := filepath.Join(t.TempDir(), "candidate")
		if err := os.WriteFile(path, data, 0o755); err != nil {
			t.Skip(err)
		}

		_ = (&DefaultExtractor{}).IsGoBinary(path)
	})
}

// FuzzCalculateChecksum verifies checksum calculation does not panic on arbitrary files.
func FuzzCalculateChecksum(f *testing.F) {
	f.Add([]byte("payload"))
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > maxFuzzBinarySize {
			data = data[:maxFuzzBinarySize]
		}

		path := filepath.Join(t.TempDir(), "candidate")
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Skip(err)
		}

		_, _ = (&DefaultExtractor{}).CalculateChecksum(path)
	})
}
