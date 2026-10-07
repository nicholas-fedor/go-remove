/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package version

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	buildversion "github.com/nicholas-fedor/go-remove/internal/version"
)

// errWrite is the error errWriter returns.
var errWrite = errors.New("write failed")

// testInfo is the build metadata the tests report.
var testInfo = buildversion.Info{
	Version:   "v1.2.3",
	CommitSHA: "abc123",
	BuildTime: "2026-10-06T00:00:00Z",
}

// errWriter is an io.Writer whose every write fails.
type errWriter struct{}

// Write fails every write.
//
// Returns:
//   - int: always 0.
//   - error: always errWrite.
func (errWriter) Write([]byte) (int, error) { return 0, errWrite }

// TestNewCommand_Prints verifies the version, commit, and build time lines.
func TestNewCommand_Prints(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer

	command := NewCommand(&stdout, testInfo)
	command.SetArgs([]string{})

	require.NoError(t, command.Execute())
	assert.Equal(t,
		"go-remove v1.2.3\n  Commit: abc123\n  Built:  2026-10-06T00:00:00Z\n",
		stdout.String(),
	)
}

// TestNewCommand_RejectsArguments verifies the command takes no arguments.
func TestNewCommand_RejectsArguments(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer

	command := NewCommand(&stdout, testInfo)
	command.SetOut(&bytes.Buffer{})
	command.SetErr(&bytes.Buffer{})
	command.SetArgs([]string{"extra"})

	require.Error(t, command.Execute())
	assert.Empty(t, stdout.String())
}

// TestNewCommand_WriteError verifies a failed write is reported.
func TestNewCommand_WriteError(t *testing.T) {
	t.Parallel()

	command := NewCommand(errWriter{}, testInfo)
	command.SetArgs([]string{})

	require.ErrorIs(t, command.Execute(), errWrite)
}
