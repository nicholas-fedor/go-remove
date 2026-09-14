/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package fs

import (
	"os"
	"path/filepath"
	"testing"
)

// FuzzAdjustBinaryPath verifies AdjustBinaryPath does not panic on arbitrary names.
func FuzzAdjustBinaryPath(f *testing.F) {
	f.Add("/usr/local/bin", "go")
	f.Add("", "tool")
	f.Add("/tmp", "")
	f.Add("/tmp", "tool.exe")
	f.Add("/tmp", "gup.lock")
	f.Add("C:\\Go\\bin", "tool")

	f.Fuzz(func(t *testing.T, dir, binary string) {
		_ = (&RealFS{}).AdjustBinaryPath(dir, binary)
	})
}

// FuzzListBinaries verifies ListBinaries does not panic on temp directories.
func FuzzListBinaries(f *testing.F) {
	f.Add("gup.lock")
	f.Add("tool")
	f.Add("")

	f.Fuzz(func(t *testing.T, name string) {
		dir := t.TempDir()
		fs := &RealFS{}
		_ = fs.ListBinaries(dir)

		base := filepath.Base(name)
		if base == "" || base == "." || base == ".." {
			return
		}

		path := filepath.Join(dir, base)
		if err := os.WriteFile(path, []byte("not a go binary"), 0o644); err != nil {
			return
		}

		got := fs.ListBinaries(dir)
		for _, listed := range got {
			if listed == base {
				t.Errorf("non-Go file %q was listed as a binary", base)
			}
		}
	})
}
