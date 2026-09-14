/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package storage

import (
	"testing"
)

// FuzzGenerateParseKey round-trips timestamps and binary names through key encoding.
func FuzzGenerateParseKey(f *testing.F) {
	f.Add(int64(1709321234), "golangci-lint")
	f.Add(int64(0), "tool")
	f.Add(int64(-1), "name:with:colons")
	f.Add(int64(-9223372036854775808), "min")
	f.Add(int64(9223372036854775807), "max")
	f.Add(int64(1), " ")
	f.Add(int64(1), "\n")
	f.Add(int64(1), "unicøde")

	f.Fuzz(func(t *testing.T, timestamp int64, binaryName string) {
		if binaryName == "" {
			return
		}

		key := GenerateKey(timestamp, binaryName)

		gotTS, gotName, err := ParseKey(key)
		if err != nil {
			t.Fatalf("ParseKey(%q) unexpected error: %v", key, err)
		}

		if gotTS != timestamp {
			t.Errorf("timestamp = %d, want %d", gotTS, timestamp)
		}

		if gotName != binaryName {
			t.Errorf("binaryName = %q, want %q", gotName, binaryName)
		}
	})
}

// FuzzParseKey verifies ParseKey does not panic on arbitrary keys.
func FuzzParseKey(f *testing.F) {
	f.Add("00000001709321234:golangci-lint")
	f.Add("")
	f.Add("not-a-key")
	f.Add("abc:name")
	f.Add("1:")

	f.Fuzz(func(t *testing.T, key string) {
		_, _, _ = ParseKey(key)
	})
}
