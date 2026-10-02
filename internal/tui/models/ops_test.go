/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	tea "charm.land/bubbletea/v2"

	mockFS "github.com/nicholas-fedor/go-remove/internal/fs/mocks"
	"github.com/nicholas-fedor/go-remove/internal/logger"
	"github.com/nicholas-fedor/go-remove/internal/tui/render"
)

// drainOperation runs the command an operation returned and applies its result,
// so a test can assert the outcome of work that now happens outside Update.
//
// Parameters:
//   - t: Test that owns the model.
//   - m: The model that produced the command.
//   - cmd: Command returned alongside the updated model.
//
// Returns:
//   - The model after the result has been applied.
func drainOperation(t *testing.T, m *Model, cmd tea.Cmd) *Model {
	t.Helper()

	if cmd == nil {
		return m
	}

	msg := cmd()

	// Commands that are not an operation, such as a quit, need no draining.
	switch msg.(type) {
	case opResultMsg:
	default:
		return m
	}

	result := msg.(opResultMsg)
	if result.operation == "" && result.err == nil && result.refresh == nil {
		return m
	}

	updated, _ := m.Update(result)
	got, ok := updated.(*Model)
	require.True(t, ok)

	return got
}

// TestCtrlC_KeepsInterruptedStatus verifies the status an interrupt showed is
// not replaced when the operation it stopped later reports success.
//
// The work is not necessarily context aware, so it can run to completion after
// the user has been told they stopped it.
func TestCtrlC_KeepsInterruptedStatus(t *testing.T) {
	t.Parallel()

	fsMock := mockFS.NewMockFS(t)
	fsMock.On("AdjustBinaryPath", "/bin", "vhs").Return("/bin/vhs")
	fsMock.On("RemoveBinary", "/bin/vhs", "vhs", false, mock.Anything).Return(nil)
	fsMock.On("ListBinaries", "/bin").Return([]string{"vhs"}, nil).Maybe()

	m := &Model{
		choices:       []string{"vhs"},
		dir:           "/bin",
		cols:          1,
		rows:          1,
		fs:            fsMock,
		logger:        logger.NopLogger(),
		logs:          make([]string, 0, maxLogLines),
		logChan:       make(chan LogMsg, maxLogLines),
		width:         80,
		height:        24,
		sortAscending: true,
		styles:        render.DefaultStyleConfig(),
	}

	// Start a removal, then interrupt it the way the key handler does.
	started, opCmd := m.Update(keyPressString(keyEnter))
	interrupting, _ := started.Update(keyPressString(keyCtrlC))

	interrupted, ok := interrupting.(*Model)
	require.True(t, ok)
	require.Equal(t, "Interrupted", interrupted.status)
	require.NotEmpty(t, opCmd, "the removal must still be in flight")

	// The work finishes without observing the cancellation.
	final := drainOperation(t, interrupted, opCmd)

	assert.Equal(t, "Interrupted", final.status,
		"a stopped operation must not then report that it completed")

	// The work is not context aware, so it still ran and the list was refreshed
	// from the store rather than left stale.
	fsMock.AssertExpectations(t)
}
