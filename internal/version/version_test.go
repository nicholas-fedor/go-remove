/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGetVersion_Stamped verifies a stamped version is returned as is.
//
// It changes the package-level Version, so it does not run in parallel.
func TestGetVersion_Stamped(t *testing.T) {
	original := Version

	t.Cleanup(func() { Version = original })

	Version = "v1.2.3"

	assert.Equal(t, "v1.2.3", GetVersion())
}

// TestGetVersion_Unstamped verifies a dev build reports a non-empty version
// and leaves Version unchanged.
//
// It changes the package-level Version, so it does not run in parallel.
func TestGetVersion_Unstamped(t *testing.T) {
	original := Version

	t.Cleanup(func() { Version = original })

	Version = devVersion

	assert.NotEmpty(t, GetVersion())
	assert.Equal(t, devVersion, Version, "GetVersion must not write back to Version")
}

// TestCurrent verifies Current carries the link-time variables.
//
// It changes package-level variables, so it does not run in parallel.
func TestCurrent(t *testing.T) {
	originalVersion, originalCommit, originalTime := Version, CommitSHA, BuildTime

	t.Cleanup(func() {
		Version, CommitSHA, BuildTime = originalVersion, originalCommit, originalTime
	})

	Version, CommitSHA, BuildTime = "v1.2.3", "abc123", "2026-10-06T00:00:00Z"

	assert.Equal(t, Info{
		Version:   "v1.2.3",
		CommitSHA: "abc123",
		BuildTime: "2026-10-06T00:00:00Z",
	}, Current())
}
