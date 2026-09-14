/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package cmd

import (
	"path/filepath"
	"testing"
)

// FuzzIsDirWritable verifies isDirWritable does not panic on arbitrary paths.
func FuzzIsDirWritable(f *testing.F) {
	f.Add("")
	f.Add("subdir")
	f.Add("missing")
	f.Add(".")
	f.Add("..")
	f.Add("a/b")

	f.Fuzz(func(t *testing.T, rel string) {
		dir := t.TempDir()
		_ = isDirWritable(dir)

		name := filepath.Base(rel)
		if name == "" || name == "." || name == ".." {
			return
		}

		_ = isDirWritable(filepath.Join(dir, name))
	})
}
