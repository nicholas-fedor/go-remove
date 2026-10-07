/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/go-remove/internal/logger"
)

// TestOpenHistory verifies a real manager opens and closes against the
// temporary data directory that TestMain sets up.
//
// The history database takes a directory lock, so this does not run in
// parallel with anything else that opens it.
func TestOpenHistory(t *testing.T) {
	manager, err := openHistory(logger.NopLogger())

	require.NoError(t, err)
	require.NotNil(t, manager)

	entries, err := manager.GetHistory(t.Context(), 1)

	require.NoError(t, err)
	assert.Empty(t, entries)
	require.NoError(t, manager.Close())
}
