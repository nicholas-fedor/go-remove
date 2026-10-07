/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package restore

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/go-remove/internal/cli/cmd/restore/mocks"
	"github.com/nicholas-fedor/go-remove/internal/cli/flags"
	"github.com/nicholas-fedor/go-remove/internal/logger"
)

// execute runs a standalone restore command.
//
// Parameters:
//   - t: the running test.
//   - browser: the service the command calls.
//   - global: the persistent flag values.
//   - args: the command arguments.
//
// Returns:
//   - error: the command error.
func execute(t *testing.T, browser Browser, global *flags.Global, args ...string) error {
	t.Helper()

	command := NewCommand(t.Context(), browser, global)
	command.SetOut(&bytes.Buffer{})
	command.SetErr(&bytes.Buffer{})
	command.SetArgs(args)

	return command.Execute()
}

// TestNewCommand_OpensHistory verifies the resolved settings reach the
// history view.
func TestNewCommand_OpensHistory(t *testing.T) {
	t.Parallel()

	browser := mocks.NewMockBrowser(t)
	browser.EXPECT().
		BrowseHistory(mock.Anything, flags.Settings{
			LogLevel: "info",
			Level:    logger.DebugLevel,
			Verbose:  true,
			Goroot:   true,
		}).
		Return(nil)

	err := execute(t, browser, &flags.Global{LogLevel: "info", Verbose: true, Goroot: true})

	require.NoError(t, err)
}

// TestNewCommand_BrowserError verifies a failed history view is wrapped.
func TestNewCommand_BrowserError(t *testing.T) {
	t.Parallel()

	errBrowse := errors.New("no terminal")

	browser := mocks.NewMockBrowser(t)
	browser.EXPECT().BrowseHistory(mock.Anything, mock.Anything).Return(errBrowse)

	err := execute(t, browser, flags.New())

	require.ErrorIs(t, err, errBrowse)
	assert.Contains(t, err.Error(), "opening history view")
}

// TestNewCommand_RejectsBeforeBrowsing verifies invalid input fails without
// calling the browser, which has no expectations and fails the test if called.
func TestNewCommand_RejectsBeforeBrowsing(t *testing.T) {
	t.Parallel()

	t.Run("argument", func(t *testing.T) {
		t.Parallel()

		require.Error(t, execute(t, mocks.NewMockBrowser(t), flags.New(), "vhs"))
	})

	t.Run("unknown log level", func(t *testing.T) {
		t.Parallel()

		global := &flags.Global{LogLevel: "banana", Verbose: false, Goroot: false}

		require.ErrorIs(t, execute(t, mocks.NewMockBrowser(t), global), logger.ErrInvalidLogLevel)
	})
}
