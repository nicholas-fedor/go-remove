//go:build linux || darwin

/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package paths

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTrashRoot verifies the XDG rule for the trash root.
//
// This function had no test coverage before it moved into this package: the
// test files that lived alongside it in internal/trash covered only the
// cross-device predicate. The absolute-path rule is the one that had already
// drifted between the command layer and the trash layer, so it is worth
// pinning down here.
func TestTrashRoot(t *testing.T) {
	t.Run("absolute XDG_DATA_HOME is used", func(t *testing.T) {
		home := isolateHome(t)
		xdg := filepath.Join(home, "custom", "data")
		t.Setenv("XDG_DATA_HOME", xdg)

		assert.Equal(t, filepath.Join(xdg, "Trash"), TrashRoot())
	})

	t.Run("relative XDG_DATA_HOME falls back to the home directory", func(t *testing.T) {
		home := isolateHome(t)
		t.Setenv("XDG_DATA_HOME", "relative/data")

		root := TrashRoot()
		require.NotEmpty(t, root)
		assert.Equal(t, filepath.Join(home, ".local", "share", "Trash"), root)
		assert.NotContains(t, root, filepath.Join("relative", "data"),
			"a relative XDG_DATA_HOME must not be joined")
	})

	t.Run("unset XDG_DATA_HOME falls back to the home directory", func(t *testing.T) {
		home := isolateHome(t)
		t.Setenv("XDG_DATA_HOME", "")

		assert.Equal(t, filepath.Join(home, ".local", "share", "Trash"), TrashRoot())
	})

	t.Run("agrees with the data home candidate", func(t *testing.T) {
		home := isolateHome(t)
		xdg := filepath.Join(home, "shared", "data")
		t.Setenv("XDG_DATA_HOME", xdg)

		// The trash root and the application data root must resolve against the
		// same base. When they disagreed, a relative XDG_DATA_HOME was rejected
		// by one layer and accepted by the other.
		require.NotEmpty(t, DataHomeCandidates())
		assert.Equal(t, xdg, DataHomeCandidates()[0])
		assert.Equal(t, filepath.Join(xdg, "Trash"), TrashRoot())
	})
}
