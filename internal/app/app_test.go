/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package app

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestMain points HOME and the XDG data directory at a temporary directory,
// so no test reads or writes the real home directory.
//
// Parameters:
//   - m: the test runner.
func TestMain(m *testing.M) {
	os.Exit(runIsolated(m))
}

// runIsolated runs the tests with HOME and XDG_DATA_HOME in a temporary
// directory, and removes it afterwards.
//
// Parameters:
//   - m: the test runner.
//
// Returns:
//   - int: the test exit code.
func runIsolated(m *testing.M) int {
	home, err := os.MkdirTemp("", "go-remove-app-test-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "creating temporary home:", err)

		return 1
	}

	defer os.RemoveAll(home)

	for key, value := range map[string]string{
		"HOME":          home,
		"XDG_DATA_HOME": home + "/data",
	} {
		if err := os.Setenv(key, value); err != nil {
			fmt.Fprintln(os.Stderr, "setting", key+":", err)

			return 1
		}
	}

	return m.Run()
}

// TestRun_ExitCodes verifies success maps to 0 and any failure maps to 1,
// with the error reported once on stderr.
func TestRun_ExitCodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		setup      func(f *servicesFixture)
		name       string
		wantStdout string
		wantStderr string
		args       []string
		wantCode   int
	}{
		{
			name:       "version",
			args:       []string{"version"},
			setup:      func(*servicesFixture) {},
			wantCode:   exitOK,
			wantStdout: "go-remove ",
			wantStderr: "",
		},
		{
			name:       "invalid invocation",
			args:       []string{"--bogus"},
			setup:      func(*servicesFixture) {},
			wantCode:   exitFailure,
			wantStdout: "",
			wantStderr: "Error: unknown flag: --bogus\n",
		},
		{
			name: "operation failure",
			args: []string{"--goroot", "rm", "vhs"},
			setup: func(f *servicesFixture) {
				f.fs.EXPECT().DetermineBinDir(true).Return("", errBinDir)
			},
			wantCode:   exitFailure,
			wantStdout: "",
			wantStderr: "Error: removing vhs: determining binary directory: GOROOT is not set\n",
		},
		{
			name: "removal",
			args: []string{"rm", "vhs"},
			setup: func(f *servicesFixture) {
				f.log.EXPECT().Level(mock.Anything).Once()
				f.manager.EXPECT().Close().Return(nil).Once()
				f.fs.EXPECT().DetermineBinDir(false).Return("/go/bin", nil)
				f.fs.EXPECT().AdjustBinaryPath("/go/bin", "vhs").Return("/go/bin/vhs")
				f.manager.EXPECT().RecordDeletion(mock.Anything, "/go/bin/vhs").Return(nil, nil)
			},
			wantCode:   exitOK,
			wantStdout: "Successfully removed vhs\n",
			wantStderr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newServicesFixture(t)
			tt.setup(f)

			var stdout, stderr bytes.Buffer

			code := run(t.Context(), tt.args, &stdout, &stderr, f.svc)

			assert.Equal(t, tt.wantCode, code)
			assert.True(t, strings.HasPrefix(stdout.String(), tt.wantStdout),
				"stdout %q must start with %q", stdout.String(), tt.wantStdout)

			if tt.wantStderr == "" {
				assert.Empty(t, stderr.String())
			} else {
				assert.True(t, strings.HasSuffix(stderr.String(), tt.wantStderr),
					"stderr %q must end with %q", stderr.String(), tt.wantStderr)
				assert.Equal(t, 1, strings.Count(stderr.String(), "Error:"),
					"the error must be reported once")
			}
		})
	}
}
