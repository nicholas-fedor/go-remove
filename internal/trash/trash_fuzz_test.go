/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package trash

import (
	"testing"
	"time"
)

// FuzzEncodeDecodeTrashPath round-trips paths through trash path encoding.
func FuzzEncodeDecodeTrashPath(f *testing.F) {
	f.Add("/home/user/bin/tool")
	f.Add("/tmp/file with spaces")
	f.Add("100%")
	f.Add("")
	f.Add("/tmp/\nnewline")
	f.Add("/tmp/\t tab")
	f.Add("/tmp/\x00null")
	f.Add("///")
	f.Add("%2F")

	f.Fuzz(func(t *testing.T, path string) {
		encoded := encodeTrashPath(path)

		decoded, err := decodeTrashPath(encoded)
		if err != nil {
			t.Fatalf("decodeTrashPath(%q) unexpected error: %v", encoded, err)
		}

		if decoded != path {
			t.Errorf("round-trip = %q, want %q", decoded, path)
		}
	})
}

// FuzzDecodeTrashPath verifies decodeTrashPath does not panic on arbitrary input.
func FuzzDecodeTrashPath(f *testing.F) {
	f.Add("/tmp/tool")
	f.Add("%20")
	f.Add("%")
	f.Add("%ZZ")
	f.Add("%2")
	f.Add("%%")
	f.Add("%E9")

	f.Fuzz(func(t *testing.T, encoded string) {
		_, _ = decodeTrashPath(encoded)
	})
}

// FuzzParseTrashInfo verifies parseTrashInfo does not panic on arbitrary content.
func FuzzParseTrashInfo(f *testing.F) {
	f.Add("[Trash Info]\nPath=/tmp/tool\nDeletionDate=2026-03-02T12:00:00\n")
	f.Add("")
	f.Add("Path=/tmp/tool")
	f.Add("[Trash Info]\nPath=%ZZ\n")
	f.Add("[Trash Info]\r\nPath=/tmp/tool\r\nDeletionDate=2026-03-02T12:00:00\r\n")
	f.Add("Path=\nDeletionDate=2026-03-02T12:00:00")
	f.Add("[Trash Info]\nPath=%20\nDeletionDate=not-a-date\n")
	f.Add("%")
	f.Add("Path=%")

	f.Fuzz(func(t *testing.T, content string) {
		_, _, _ = parseTrashInfo(content)
	})
}

// FuzzGenerateParseTrashInfo round-trips generated .trashinfo files for simple paths.
func FuzzGenerateParseTrashInfo(f *testing.F) {
	f.Add("/tmp/tool")
	f.Add("/home/user/bin/go")
	f.Add("/tmp/file with spaces")
	f.Add("/tmp/%percent")

	f.Fuzz(func(t *testing.T, path string) {
		if path == "" {
			return
		}

		now := time.Date(2026, 3, 2, 12, 0, 0, 0, time.Local)
		content := generateTrashInfo(path, now)

		gotPath, _, err := parseTrashInfo(content)
		if err != nil {
			t.Fatalf("parseTrashInfo(%q) unexpected error: %v", content, err)
		}

		if gotPath != path {
			t.Errorf("path = %q, want %q", gotPath, path)
		}
	})
}
