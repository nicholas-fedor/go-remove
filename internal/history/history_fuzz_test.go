/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package history

import (
	"testing"

	"github.com/nicholas-fedor/go-remove/internal/storage"
)

// FuzzEntryFromRecord verifies entryFromRecord copies record fields without panicking.
func FuzzEntryFromRecord(f *testing.F) {
	f.Add(int64(1709321234), "golangci-lint", "/bin/golangci-lint", true)
	f.Add(int64(0), "tool", "", false)
	f.Add(int64(-1), "name:with:colons", "/tmp/a b", true)
	f.Add(int64(1), "", "/", false)

	f.Fuzz(func(t *testing.T, ts int64, name, path string, inTrash bool) {
		record := storage.HistoryRecord{
			Timestamp:      ts,
			BinaryName:     name,
			OriginalPath:   path,
			TrashAvailable: inTrash,
		}

		entry := entryFromRecord(&record)
		if entry.BinaryName != name {
			t.Errorf("BinaryName = %q, want %q", entry.BinaryName, name)
		}

		if entry.BinaryPath != path {
			t.Errorf("BinaryPath = %q, want %q", entry.BinaryPath, path)
		}

		if entry.InTrash != inTrash {
			t.Errorf("InTrash = %v, want %v", entry.InTrash, inTrash)
		}
	})
}
