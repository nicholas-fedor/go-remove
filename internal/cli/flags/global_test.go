/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package flags

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/go-remove/internal/logger"
)

// TestNew verifies the defaults match the flag defaults.
func TestNew(t *testing.T) {
	t.Parallel()

	assert.Equal(t, &Global{LogLevel: "info", Verbose: false, Goroot: false}, New())
}

// TestBindGlobal verifies the persistent flags parse into the Global.
func TestBindGlobal(t *testing.T) {
	t.Parallel()

	global := New()
	command := &cobra.Command{Use: "test"}
	BindGlobal(command, global)

	require.NoError(t, command.ParseFlags([]string{"-v", "-l", "warn", "--goroot"}))

	assert.Equal(t, &Global{LogLevel: "warn", Verbose: true, Goroot: true}, global)
}

// TestGlobal_Resolve verifies level parsing and the verbose override.
func TestGlobal_Resolve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		global  Global
		want    Settings
		wantErr error
	}{
		{
			name:   "default level",
			global: Global{LogLevel: "info", Verbose: false, Goroot: false},
			want: Settings{
				LogLevel: "info",
				Level:    logger.InfoLevel,
				Verbose:  false,
				Goroot:   false,
			},
			wantErr: nil,
		},
		{
			name:   "level applies without verbose",
			global: Global{LogLevel: "ERROR", Verbose: false, Goroot: true},
			want: Settings{
				LogLevel: "ERROR",
				Level:    logger.ErrorLevel,
				Verbose:  false,
				Goroot:   true,
			},
			wantErr: nil,
		},
		{
			name:   "verbose raises the level to debug",
			global: Global{LogLevel: "warn", Verbose: true, Goroot: false},
			want: Settings{
				LogLevel: "warn",
				Level:    logger.DebugLevel,
				Verbose:  true,
				Goroot:   false,
			},
			wantErr: nil,
		},
		{
			name:    "unknown level",
			global:  Global{LogLevel: "banana", Verbose: true, Goroot: false},
			want:    Settings{},
			wantErr: logger.ErrInvalidLogLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.global.Resolve()
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Contains(t, err.Error(), "debug, info, warn, error",
					"the error must list the accepted values")

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
